package chat_round

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"encoding/json"
)

func (c *cacheLidis) Update(vector []float32, query string, answer string) error {

	err := c.index.Add(query, vector)
	if err != nil {
		return err
	}

	cand := &index_algorithm.Candidate{
		Query:  query,
		Answer: answer,
	}

	value, err := json.Marshal(cand)
	if err != nil {
		return err
	}

	_, err = c.cache.Set(query, string(value))

	return err
}

//另外我看你当前 update.go 还有一个小地方建议你之后处理：现在是先 Index.Add()，再写 Cache。
//如果索引成功但 Cache.Set() 失败，就会出现“索引里有这个 Query，但缓存中找不到”的不一致
//状态。你现在先跑通逻辑没问题，等基本架构完成后，再专门设计 update.go 的写入顺序、失败回滚和索引/
//缓存一致性。这个问题比现在继续完善 Rerank 细节更值得作为下一阶段处理。
