package conn

type HandlerFunc func(session *Session, body []byte) (any, error)

type MessageTypeHandler map[string]HandlerFunc
