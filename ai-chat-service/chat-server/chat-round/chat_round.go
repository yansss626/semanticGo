package chat_round

type Round interface {
	Retrieval(query string, vector []float32) (string, error)
	Update(vector []float32, query string, answer string) error
}
