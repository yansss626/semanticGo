package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reprise/core/cache"
	"reprise/core/embedding"
	meta_data "reprise/core/meta-data"
	"reprise/core/reranker"
	vector_index "reprise/core/vector-index"
	"reprise/pkg/config"
	"time"
)

const (
	vectorIndexPath = "./data/"
	cacheHitLogPath = "./data/"
	cacheHitLog     = "cache.log"
	hnswFile        = "hnsw.snapshot"
)

type app struct {
	cnf       *config.Config
	cache     cache.CacheObject
	embedding embedding.EmbeddingObject
	rerank    reranker.ReRanker
	index     vector_index.VectorIndex
}

func (s *repriseService) NewApp(cnf *config.Config, cache cache.CacheObject, index vector_index.VectorIndex) *app {
	return &app{
		cnf:   cnf,
		cache: cache,
		embedding: &embedding.Qwen3TextEmbedding{
			Model: cnf.Embedding.Model,
			Dim:   cnf.Embedding.VectorDimensions,
			Client: &http.Client{
				Timeout: 5 * time.Second,
			},
			ApiKey:          cnf.Embedding.ApiKey,
			BaseUrl:         cnf.Embedding.BaseUrl,
			TopK:            cnf.Embedding.TopK,
			RecallThreshold: cnf.Embedding.RecallScore,
		},
		rerank: &reranker.AlibabaRerank{
			Model: cnf.Rerank.Model,
			Client: &http.Client{
				Timeout: 5 * time.Second,
			},
			ApiKey:          cnf.Rerank.ApiKey,
			BaseUrl:         cnf.Rerank.BaseUrl,
			Instruct:        cnf.Rerank.Instruct,
			RerankThreshold: cnf.Rerank.RerankScore,
		},
		index: index,
	}
}

// ******************* VSET *********************
// 文本转向量
func (a *app) convertTextToVector(text string) ([]float32, error) {

	resp, err := a.embedding.Get([]string{text})
	if err != nil {
		return nil, err
	}
	return resp.Data[0].Embedding, nil
}

// 更新向量索引

func (a *app) updateVectorIndex(text string, vector []float32) error {
	return a.index.Add(text, vector)
}

// 缓存更新
func (a *app) saveCacheEntry(text string, newCacheEntry *cache.CacheEntry) error {
	if newCacheEntry == nil || newCacheEntry.Answer == "" || newCacheEntry.Vector == nil || newCacheEntry.TotalTokens <= 0 {
		return fmt.Errorf("invalid cache entry for text [%s]: entry or required fields (answer, vector, totalTokens) are invalid", text)
	}

	return a.cache.Set(text, newCacheEntry)
}

// ******************* VGET *********************

//1. 文本转向量
//2. topk --> 向量索引搜索，返回 K 个最近邻向量 （粗筛）

func (a *app) vectorSearchTopK(vector []float32) (*vector_index.RecallResult, error) {
	return a.index.SearchK(vector, a.cnf.Embedding.TopK)
}

// 3. rerank 重排 （精筛）
func (a *app) recallResultRerank(text string, recallResult *vector_index.RecallResult) (*reranker.TopCandidate, error) {
	return a.rerank.Rerank(text, recallResult)
}

// 4. 搜索缓存
func (a *app) searchCacheEntry(text string) (*cache.CacheEntry, error) {
	return a.cache.Get(text)
}

// 5. 元数据对比
func compareMetaData(query, matchQuery string) bool {
	queryMeta := meta_data.ExtractMetaData(query)
	matchQueryMeta := meta_data.ExtractMetaData(matchQuery)
	return meta_data.MatchTwoMeta(queryMeta, matchQueryMeta)
}

// 文件保存/加载
func saveHitLog(text string, textHit *reranker.TopCandidate) {
	path := cacheHitLogPath + cacheHitLog
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	log := fmt.Sprintf("User Query: %s\nHit Query: %s\nreacll score: %f, rerank score: %f\n--------------------\n",
		text, textHit.Query, textHit.RecallScore, textHit.RerankScore)

	file.WriteString(log)
}

func initHNSW(dimensions int) (*vector_index.HNSWIndex, error) {
	path := vectorIndexPath + hnswFile
	index, err := vector_index.LoadHNSWIndex(path)
	if err == nil {
		return index, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	index = vector_index.NewHNSWIndex(vector_index.DefaultHNSWConfig(dimensions))

	return index, nil

}
