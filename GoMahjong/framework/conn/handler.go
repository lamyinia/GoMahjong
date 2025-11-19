package conn

type HandlerFunc func(body []byte) (any, error)

type LogicHandler map[string]HandlerFunc
