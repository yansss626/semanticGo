package index_algorithm

import (
	"fmt"
	"sort"
	"sync"

	"github.com/fogfish/hnsw"
	vectorSurface "github.com/kshard/vector"
)

const (
	defaultHNSWM              = 16
	defaultHNSWM0             = 32
	defaultHNSWEfConstruction = 200
	defaultEfSearch           = 64
	defaultOverFetch          = 3
	defaultHNSWK              = 3
)

type HNSWConfig struct {
	Dim            int // 向量维度
	M              int // 非零层外其余层中节点所拥有的最大邻居数
	M0             int // 零层中节点所拥有的最大邻居数
	EfConstruction int // 插入新节点时，候选节点的个数
	EfSearch       int // 搜索节点时，候选节点的个数
	OverFetch      int
}

type hnswItem struct {
	ID     uint64
	Vector []float32
}

type hnswItemSurce struct {
	cosine vectorSurface.Surface[[]float32]
}

func (s hnswItemSurce) Distance(a, b hnswItem) float32 {
	return s.cosine.Distance(a.Vector, b.Vector)
}

func (s hnswItemSurce) Equal(a, b hnswItem) bool {
	return a.ID == b.ID
}

type HNSWIndex struct {
	grap       *hnsw.HNSW[hnswItem] // 图索引算法的具体实现：github.com/fogfish/hnsw
	config     HNSWConfig           // 使用参数
	mu         sync.RWMutex         // 读写锁，防止并发竞争,主要是为了防止修改map时的竞争
	snapShotMu sync.RWMutex         // 读写锁，在保存索引的快照文件使用，保证快照文件中的信息是同一个时间点的状态
	nextID     uint64               // 节点ID
	queryToID  map[string]uint64    // 问题-->ID 哈希映射
	idToQuery  map[uint64]string    // ID-->问题 哈希映射
}

func DefaultHNSWConfig(dim int) HNSWConfig {
	return HNSWConfig{
		Dim:            dim,
		M:              defaultHNSWM,
		M0:             defaultHNSWM0,
		EfConstruction: defaultHNSWEfConstruction,
		EfSearch:       defaultEfSearch,
		OverFetch:      defaultOverFetch,
	}
}

func NewHNSWIndex(config HNSWConfig) *HNSWIndex {

	surface := hnswItemSurce{
		cosine: vectorSurface.Cosine(),
	}

	grap := hnsw.New(
		surface,
		hnsw.WithM(config.M),
		hnsw.WithM0(config.M0),
		hnsw.WithEfConstruction(config.EfConstruction),
	)

	return &HNSWIndex{
		grap:      grap,
		config:    config,
		nextID:    1,
		queryToID: make(map[string]uint64),
		idToQuery: make(map[uint64]string),
	}
}

func (h *HNSWIndex) validateVector(vector []float32) error {
	if len(vector) != h.config.Dim {
		return fmt.Errorf("invalid vector dimension: expected=%d, actual=%d", h.config.Dim, len(vector))
	}
	return nil
}

func (h *HNSWIndex) Add(query string, Vector []float32) error {
	if query == "" {
		return fmt.Errorf("error: query is empty")
	}

	if err := h.validateVector(Vector); err != nil {
		return err
	}

	vectorCopy := append([]float32(nil), Vector...)
	h.snapShotMu.RLock()
	defer h.snapShotMu.RUnlock()
	h.mu.Lock()
	id := h.nextID
	h.nextID++
	h.mu.Unlock()

	item := hnswItem{
		ID:     id,
		Vector: vectorCopy,
	}

	h.grap.Insert(item)
	h.mu.Lock()
	defer h.mu.Unlock()

	if currentId, exists := h.queryToID[query]; exists && currentId > id {
		return nil
	}

	if oldId, ok := h.queryToID[query]; ok {
		delete(h.idToQuery, oldId)
	}

	h.queryToID[query] = id
	h.idToQuery[id] = query

	return nil
}

func (h *HNSWIndex) SearchK(Vector []float32, k int) ([]*SemanticCacheRecallResult, error) {

	if err := h.validateVector(Vector); err != nil {
		return nil, err
	}
	if k <= 0 {
		k = defaultHNSWK
	}

	if h.grap.Size() == 0 {
		return nil, nil
	}

	candidateK := k * h.config.OverFetch

	if candidateK > h.grap.Size() {
		candidateK = h.grap.Size()
	}

	efSearch := h.config.EfSearch
	if efSearch < candidateK {
		efSearch = candidateK
	}

	item := hnswItem{
		ID:     0,
		Vector: Vector,
	}

	neighbours := h.grap.Search(
		item,
		candidateK,
		efSearch,
	)

	results := make([]*SemanticCacheRecallResult, 0, k)
	for _, neighbour := range neighbours {
		h.mu.RLock()
		query, active := h.idToQuery[neighbour.ID]
		h.mu.RUnlock()

		if !active {
			continue
		}

		result := &SemanticCacheRecallResult{
			Query: query,
			Score: CosineSimilarity(Vector, neighbour.Vector),
		}

		results = append(results, result)
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > k {
		results = results[:k]
	}

	return results, nil
}

func (h *HNSWIndex) Delete(query string) error {

	if query == "" {
		return nil
	}
	h.snapShotMu.RLock()
	defer h.snapShotMu.RUnlock()
	h.mu.Lock()
	defer h.mu.Unlock()

	id, ok := h.queryToID[query]
	if !ok {
		return nil
	}

	delete(h.queryToID, query)
	delete(h.idToQuery, id)

	return nil
}
