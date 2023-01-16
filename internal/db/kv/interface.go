package kv

//go:generate mockgen -package mock -destination=mock/kv.go . Store,SortedSet,Set

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// ErrNotFound is returned when the implementation of one of the Store, SortedSet of Set do not find a record
var ErrNotFound = errors.New("not found")

// Store defines the interface for persisting data in a key value store
type Store interface {
	Keys(ctx context.Context, collection string) ([]string, error)
	Set(ctx context.Context, collection string, key string, value interface{}, expiration time.Duration) error
	MSet(ctx context.Context, collection string, values map[string]interface{}) error
	SetNX(ctx context.Context, collection string, key string, value interface{}, expiration time.Duration) (bool, error)
	MSetNX(ctx context.Context, collection string, values map[string]interface{}) (bool, error)
	Get(ctx context.Context, collection string, key string) ([]byte, error)
	MGet(ctx context.Context, collection string, keys []string) (map[string][]byte, error)
	Del(ctx context.Context, collection string, keys ...string) error
	Close()
}

// Z represents sorted set member.
type Z struct {
	Score  float64
	Member interface{}
}

// ZRangeBy represents a sorted set query
type ZRangeBy struct {
	Min, Max      string
	Offset, Count int64
}

// SortedSet defines the interface for persisting data in a sorted set store
type SortedSet interface {
	ZAdd(ctx context.Context, key string, members ...*Z) error
	ZRangeByScore(ctx context.Context, key string, opt *ZRangeBy) ([]string, error)
	ZRem(ctx context.Context, key string, members ...interface{}) error
	Close()
}

// Set defines the interface for persisting data in a set store
type Set interface {
	SAdd(ctx context.Context, key string, members ...string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SRem(ctx context.Context, key string, members ...string) error
	SPopN(ctx context.Context, key string, count int64) ([]string, error)
	Close()
}
