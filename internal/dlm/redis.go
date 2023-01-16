package dlm

import (
	"fmt"
	"time"

	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

// RedisDLM is a Redis backed implementation of the Distributed Lock Manager interface
type RedisDLM struct {
	redSync *redsync.Redsync
	prefix  string
}

// NewRedisDLM is a helper function for creating a RedisDLM
func NewRedisDLM(prefix string, opts *goredislib.Options) *RedisDLM {
	r := new(RedisDLM)

	client := goredislib.NewClient(opts)
	pool := goredis.NewPool(client)

	r.redSync = redsync.New(pool)
	r.prefix = prefix
	return r
}

// Lock locks a given name until the expiry duration has passed or Unlock is called
func (r *RedisDLM) Lock(name string, expiry time.Duration) error {
	return r.redSync.NewMutex(fmt.Sprintf("%s:%s", r.prefix, name), redsync.WithExpiry(expiry)).Lock()
}

// Unlock removes a lock from a given name
func (r *RedisDLM) Unlock(name string) (bool, error) {
	return r.redSync.NewMutex(fmt.Sprintf("%s:%s", r.prefix, name)).Unlock()
}
