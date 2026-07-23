package chat_round

type TextSearch interface {
	SimilarTextSearch(rounds []*ChatRound, round *RoundMessage) (string, error)
	InsertRound(rounds []*ChatRound, round *RoundMessage) error
	GetRounds(round *RoundMessage) ([]*ChatRound, error)
	BuildIndex(vector []float32) uint64
}

type ChatRound struct {
	Messages []*RoundMessage
}

type RoundMessage struct {
	EmbeddingVector []float32
	Simhash         uint64
	Question        string
	AnswerID        string
}

type RetrievalResult struct {
	Round  *RoundMessage
	Rounds []*ChatRound
	Answer string
}
