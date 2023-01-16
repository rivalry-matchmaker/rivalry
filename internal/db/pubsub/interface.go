package pubsub

//go:generate mockgen -package mock -destination=mock/pubsub.go . Client

import "context"

// Client specifies the interface for pubsub actions
type Client interface {
	Publish(ctx context.Context, topic string, value []byte) error
	Subscribe(ctx context.Context, topic string, f func([]byte)) error
	Close()
}
