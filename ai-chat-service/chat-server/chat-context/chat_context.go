package chat_context

type ChatMessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatMessage struct {
	//当前记录ID
	ID string `json:"id,omitempty"`
	//上一条记录ID
	PID string `json:"pid,omitempty"`
	//消息内容
	Message ChatMessageContent `json:"message"`
	//该消息tokens数
	Tokens int `json:"tokens,omitempty"`
}

type ContextCache interface {
	Get(key string) (*ChatMessage, error)
	Set(key string, value *ChatMessage, ttl int) error
	Close()
}
