package chat_round

type TextSearch interface {
	SimilarTextSearch(round *RoundMessage) (bool, string, error)
	InsertText(round *RoundMessage) error
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
	Round      *RoundMessage
	Answer     string
	IsSameText bool
}
