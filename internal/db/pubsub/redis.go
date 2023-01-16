package pubsub

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// RedisPubSub is a redis implementation of the PubSub interface
type RedisPubSub struct {
	redisCli *redis.Client
	prefix   string
}

// NewRedis is a helper function used to create a new RedisPubSub
func NewRedis(prefix string, opt *redis.Options) Client {
	return &RedisPubSub{
		redisCli: redis.NewClient(opt),
		prefix:   prefix,
	}
}

// Publish send a message out to subscribers of a given subject
func (r *RedisPubSub) Publish(ctx context.Context, topic string, value []byte) error {
	return r.redisCli.Publish(ctx, fmt.Sprintf("%s:%s", r.prefix, topic), string(value)).Err()
}

// Subscribe starts listening for messages on a channel, call the done function when you're finished listening.
func (r *RedisPubSub) Subscribe(ctx context.Context, topic string, f func([]byte)) error {
	ps := r.redisCli.Subscribe(ctx, fmt.Sprintf("%s:%s", r.prefix, topic))
	psCh := ps.Channel()
	for {
		select {
		case msg := <-psCh:
			f([]byte(msg.Payload))
		case <-ctx.Done():
			return ps.Close()
		}
	}
}

// Close stops he redis client
func (r *RedisPubSub) Close() {
	r.redisCli.Close()
}
