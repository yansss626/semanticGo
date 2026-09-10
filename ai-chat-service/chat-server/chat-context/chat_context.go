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
	GetContext(key string) (*ChatMessage, error)
	SetContext(key string, message *ChatMessage) error
	DelContext(key string) error
}
