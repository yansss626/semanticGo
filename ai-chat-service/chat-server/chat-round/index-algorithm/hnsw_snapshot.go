package index_algorithm

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fogfish/hnsw"
	vectorSurface "github.com/kshard/vector"
)

type hnswSnapshot struct {
	Config    HNSWConfig
	Nodes     hnsw.Nodes[hnswItem]
	NextId    uint64
	IdToQuery map[uint64]string
	QueryToId map[string]uint64
}

func (h *HNSWIndex) SaveSnapshot(path string) error {
	if path == "" {
		return fmt.Errorf("snapshot path is empty")
	}

	h.snapShotMu.Lock()
	defer h.snapShotMu.Unlock()

	h.mu.RLock()
	snapShot := hnswSnapshot{
		Config:    h.config,
		Nodes:     h.grap.Nodes(),
		NextId:    h.nextID,
		QueryToId: cloneQueryToIdy(h.queryToID),
		IdToQuery: cloneIdToQuery(h.idToQuery),
	}
	h.mu.RUnlock()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil
	}

	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	err = gob.NewEncoder(file).Encode(&snapShot)
	if err != nil {
		file.Close()
		return err
	}

	err = file.Sync()
	if err != nil {
		file.Close()
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tmpPath, path)
	if err != nil {
		return err
	}

	return nil
}

func LoadHNSWIndexSnapshot(path string) (*HNSWIndex, error) {
	if path == "" {
		return nil, fmt.Errorf("snapshot path is empty")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var snapShot hnswSnapshot
	err = gob.NewDecoder(file).Decode(&snapShot)
	if err != nil {
		return nil, err
	}

	vectorInterface := hnswItemSurce{
		cosine: vectorSurface.Cosine(),
	}
	grap := hnsw.FromNodes(
		vectorInterface,
		snapShot.Nodes,
		hnsw.WithM(snapShot.Config.M),
		hnsw.WithM0(snapShot.Config.M0),
		hnsw.WithEfConstruction(snapShot.Config.EfConstruction),
	)

	if snapShot.QueryToId == nil {
		snapShot.QueryToId = make(map[string]uint64)
	}

	if snapShot.IdToQuery == nil {
		snapShot.IdToQuery = make(map[uint64]string)
	}

	return &HNSWIndex{
		config:    snapShot.Config,
		grap:      grap,
		queryToID: snapShot.QueryToId,
		idToQuery: snapShot.IdToQuery,
		nextID:    snapShot.NextId,
	}, nil
}

func cloneIdToQuery(IdToQuery map[uint64]string) map[uint64]string {

	copy := make(map[uint64]string, len(IdToQuery))

	for key, value := range IdToQuery {
		copy[key] = value
	}
	return copy
}

func cloneQueryToIdy(QueryToId map[string]uint64) map[string]uint64 {

	copy := make(map[string]uint64, len(QueryToId))

	for key, value := range QueryToId {
		copy[key] = value
	}
	return copy
}
