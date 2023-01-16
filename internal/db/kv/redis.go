package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis is a store implementation of the KVStore interface
type Redis struct {
	redisCli *redis.Client
	prefix   string
}

// NewRedis is a helper function that returns a new Redis
func NewRedis(prefix string, opt *redis.Options) Store {
	return &Redis{
		redisCli: redis.NewClient(opt),
		prefix:   prefix,
	}
}

// Set sets a value in the store
func (r *Redis) Set(ctx context.Context, collection string, key string, value interface{}, expiration time.Duration) error {
	return r.redisCli.Set(ctx, fmt.Sprintf("%s:%s:%s", r.prefix, collection, key), value, expiration).Err()
}

// MSet sets multiple values in the store
func (r *Redis) MSet(ctx context.Context, collection string, values map[string]interface{}) error {
	setValues := make(map[string]interface{})
	for k, v := range values {
		setValues[fmt.Sprintf("%s:%s:%s", r.prefix, collection, k)] = v
	}
	return r.redisCli.MSet(ctx, setValues).Err()
}

// MSetNX sets a collection of keys with values provided none already exist
func (r *Redis) MSetNX(ctx context.Context, collection string, values map[string]interface{}) (bool, error) {
	setValues := make(map[string]interface{})
	for k, v := range values {
		setValues[fmt.Sprintf("%s:%s:%s", r.prefix, collection, k)] = v
	}
	return r.redisCli.MSetNX(ctx, setValues).Result()
}

// SetNX sets a value in the KVStore if it doesn't already exist
func (r *Redis) SetNX(ctx context.Context, collection string, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.redisCli.SetNX(ctx, fmt.Sprintf("%s:%s:%s", r.prefix, collection, key), value, expiration).Result()
}

// Keys lists all the keys in a collection
func (r *Redis) Keys(ctx context.Context, collection string) ([]string, error) {
	keys, err := r.redisCli.Keys(ctx, fmt.Sprintf("%s:%s:*", r.prefix, collection)).Result()
	if err != nil {
		return nil, err
	}
	totalPrefixLen := len(fmt.Sprintf("%s:%s:", r.prefix, collection))
	for i := range keys {
		keys[i] = keys[i][totalPrefixLen:]
	}
	return keys, nil
}

// Get reads a key from the store
func (r *Redis) Get(ctx context.Context, collection string, key string) ([]byte, error) {
	b, err := r.redisCli.Get(ctx, fmt.Sprintf("%s:%s:%s", r.prefix, collection, key)).Bytes()
	if err == redis.Nil {
		return b, ErrNotFound
	}
	return b, err
}

// MGet reads multiple keys from the store
func (r *Redis) MGet(ctx context.Context, collection string, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	collectionKeys := make([]string, len(keys))
	for i, v := range keys {
		collectionKeys[i] = fmt.Sprintf("%s:%s:%s", r.prefix, collection, v)
	}

	result, err := r.redisCli.MGet(ctx, collectionKeys...).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]byte)
	for i, v := range keys {
		// missing keys will return as nil, in which case interface conversion will fail
		strResult, ok := result[i].(string)
		if !ok {
			continue
		}

		resp[v] = []byte(strResult)
	}
	return resp, nil
}

// Del removes a key from the store
func (r *Redis) Del(ctx context.Context, collection string, keys ...string) error {
	toDel := make([]string, len(keys))
	for i, k := range keys {
		toDel[i] = fmt.Sprintf("%s:%s:%s", r.prefix, collection, k)
	}
	return r.redisCli.Del(ctx, toDel...).Err()
}

// Close disconnects from the database
func (r *Redis) Close() {
	r.redisCli.Close()
}

// ZAdd adds members to a sorted set key
func (r *Redis) ZAdd(ctx context.Context, key string, members ...*Z) error {
	m := make([]*redis.Z, len(members))
	for i, v := range members {
		m[i] = &redis.Z{
			Score:  v.Score,
			Member: v.Member,
		}
	}
	return r.redisCli.ZAdd(ctx, key, m...).Err()
}

// ZRangeByScore returns set members that match the given filter
func (r *Redis) ZRangeByScore(ctx context.Context, key string, opt *ZRangeBy) ([]string, error) {
	return r.redisCli.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: opt.Min, Max: opt.Max, Offset: opt.Offset, Count: opt.Count}).Result()
}

// ZRem removes members from a given set
func (r *Redis) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.redisCli.ZRem(ctx, key, members...).Err()
}

// SAdd adds members to a set
func (r *Redis) SAdd(ctx context.Context, key string, members ...string) error {
	interfaceMembers := make([]interface{}, len(members))
	for i, v := range members {
		interfaceMembers[i] = v
	}
	return r.redisCli.SAdd(ctx, key, interfaceMembers...).Err()
}

// SMembers returns a sets members
func (r *Redis) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.redisCli.SMembers(ctx, key).Result()
}

// SRem removes set members
func (r *Redis) SRem(ctx context.Context, key string, members ...string) error {
	interfaceMembers := make([]interface{}, len(members))
	for i, v := range members {
		interfaceMembers[i] = v
	}
	return r.redisCli.SRem(ctx, key, interfaceMembers...).Err()
}

// SPopN pops a given number of members from a set
func (r *Redis) SPopN(ctx context.Context, key string, count int64) ([]string, error) {
	return r.redisCli.SPopN(ctx, key, count).Result()
}
