package stream

import (
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// NATS implements the stream Client interface
type NATS struct {
	nc    *nats.Conn
	subs  []*nats.Subscription
	group string
}

// NewNATS returns a new stream Client instance
func NewNATS(url, group string, options ...nats.Option) Client {
	nc, err := nats.Connect(url, options...)
	if err != nil {
		panic(err)
	}
	return &NATS{
		nc:    nc,
		group: group,
	}
}

// SendMessage sends a stream message to NATS
func (n *NATS) SendMessage(topic string, value []byte) error {
	return n.nc.Publish(topic, value)
}

// Subscribe listens to messages on a NATS stream
func (n *NATS) Subscribe(topic string, f func([]byte)) error {
	sub, err := n.nc.QueueSubscribe(topic, n.group, func(m *nats.Msg) {
		log.Trace().Str("topic", topic).Str("data", string(m.Data)).Msg("message from NATS")
		f(m.Data)
	})
	if err != nil {
		log.Err(err).Msg("Subscribe failed")
		return err
	}
	n.subs = append(n.subs, sub)
	return nil
}

// Close drains the NATS subscriptions and connection
func (n *NATS) Close() {
	for _, sub := range n.subs {
		sub.Unsubscribe()
		sub.Drain()
	}
	n.nc.Drain()
}
