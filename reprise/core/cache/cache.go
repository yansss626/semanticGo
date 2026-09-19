package cache

type CacheEntry struct {
	Answer string `json:"answer"`

	TotalTokens int32 `json:"total_tokens"`

	Vector []float32 `json:"vector"`
}

type CacheObject interface {
	Set(key string, enrty *CacheEntry) error
	Get(key string) (*CacheEntry, error)
	Del(key string) error
}
