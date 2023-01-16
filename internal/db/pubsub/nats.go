package pubsub

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// NATS implements the pubsub Client interface
type NATS struct {
	nc *nats.Conn
}

// NewNATS returns a new nats pubsub Client
func NewNATS(url string, options ...nats.Option) Client {
	nc, err := nats.Connect(url, options...)
	if err != nil {
		panic(err)
	}
	return &NATS{
		nc: nc,
	}
}

// Publish submits a message to NATS to be published on a given topic
func (n *NATS) Publish(_ context.Context, topic string, value []byte) error {
	return n.nc.Publish(topic, value)
}

// Subscribe reads messages from NATS on a given topic
func (n *NATS) Subscribe(ctx context.Context, topic string, f func([]byte)) error {
	sub, err := n.nc.Subscribe(topic, func(m *nats.Msg) {
		log.Trace().Str("data", string(m.Data)).Msg("message from NATS")
		f(m.Data)
	})
	if err != nil {
		return err
	}
	<-ctx.Done()
	_ = sub.Unsubscribe()
	return nil
}

// Close drains the NATS client
func (n *NATS) Close() {
	n.nc.Drain()
}
