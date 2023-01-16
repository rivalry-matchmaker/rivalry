package stream

//go:generate mockgen -package mock -destination=mock/stream.go . Client

// Client provides the interface for streaming messages
type Client interface {
	SendMessage(topic string, value []byte) error
	Subscribe(topic string, f func([]byte)) error
	Close()
}
