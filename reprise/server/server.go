package server

import (
	"context"
	"reprise/core/cache"
	vector_index "reprise/core/vector-index"
	"reprise/mrpc_generated/semantic"
	"reprise/pkg/config"
	"reprise/pkg/log"
)

const (
	scoreThreshold = 0.85
)

type repriseService struct {
	cnf         *config.Config
	log         log.ILogger
	cahce       cache.CacheObject
	vectorIndex vector_index.VectorIndex
}

func NewRepriseService(cnf *config.Config, log log.ILogger) (*repriseService, error) {

	// 初始化向量索引
	vectorIndex, err := initHNSW(cnf.Embedding.VectorDimensions)
	if err != nil {
		return nil, err
	}

	// 获取缓存连接
	cache, err := cache.NewCache(cnf)
	if err != nil {
		return nil, err
	}

	return &repriseService{
		cnf:         cnf,
		log:         log,
		vectorIndex: vectorIndex,
		cahce:       cache,
	}, nil
}

func (s *repriseService) VSet(ctx context.Context, req *semantic.VSetRequest) (*semantic.VSetResponse, error) {

	app := s.NewApp(s.cnf, s.cahce, s.vectorIndex)

	// 文本转向量
	vector, err := app.convertTextToVector(req.Text)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	// 保存向量索引
	err = app.updateVectorIndex(req.Text, vector)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	// 缓存更新
	cacheEntry := &cache.CacheEntry{
		Answer:      req.Answer,
		TotalTokens: req.TotalTokens,
		Vector:      vector,
	}
	err = app.saveCacheEntry(req.Text, cacheEntry)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &semantic.VSetResponse{}, nil
}

func (s *repriseService) VGet(ctx context.Context, req *semantic.VGetRequest) (*semantic.VGetResponse, error) {

	app := s.NewApp(s.cnf, s.cahce, s.vectorIndex)

	// 文本转向量
	vector, err := app.convertTextToVector(req.Text)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	// 向量索引搜索
	recallResults, err := app.vectorSearchTopK(vector)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	if recallResults == nil || len(recallResults.Querys) == 0 {
		return nil, err
	}

	// rerank 重排
	bestResult, err := app.recallResultRerank(req.Text, recallResults)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	if bestResult == nil {
		return nil, err
	}
	//fmt.Printf("recall: %f, rerank: %f\n", bestResult.RecallScore, bestResult.RerankScore)
	// 分数验证(0.3 recall + 0.7 rerank >? score threshold ) + 元数据对比
	if bestResult.RecallScore != 1 && bestResult.RerankScore != 1 {
		if 0.3*bestResult.RecallScore+0.7*bestResult.RerankScore < scoreThreshold {
			return nil, nil
		}
		if compareMetaData(req.Text, bestResult.Query) == false {
			return nil, nil
		}
	}

	// 搜索缓存
	result, err := app.searchCacheEntry(bestResult.Query)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	if result == nil {
		return nil, err
	}
	// 缓存命中！

	// 保存缓存命中结果
	saveHitLog(req.Text, bestResult)

	return &semantic.VGetResponse{
		Answer:      result.Answer,
		TotalTokens: result.TotalTokens,
	}, nil
}

func (s *repriseService) SaveVectorIndex() error {
	return s.vectorIndex.Save(vectorIndexPath + hnswFile)
}
