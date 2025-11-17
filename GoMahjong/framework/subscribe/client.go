package subscribe

type Client interface {
	Run() error
	SendMessage(string, []byte) error
	Close() error
}
