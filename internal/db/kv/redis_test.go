package kv_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/deckarep/golang-set"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	ctx        = context.Background()
	collection = "collection"
	key        = "key"
	value      = "value"
	otherValue = "otherValue"

	keys    = []string{"foo", "ben"}
	keysSet = mapset.NewSet([]interface{}{"foo", "ben"}...)
	values  = map[string]interface{}{
		"foo": "bar",
		"ben": "bob",
	}
)

type RedisTestSuite struct {
	suite.Suite
	miniredis *miniredis.Miniredis
	store     kv.Store
	sortedSet kv.SortedSet
	set       kv.Set
}

func (suite *RedisTestSuite) SetupTest() {
	var err error
	suite.miniredis, err = miniredis.Run()
	require.NoError(suite.T(), err)
	suite.store = kv.NewRedis("our_app", &redis.Options{Addr: suite.miniredis.Addr()})
	suite.sortedSet = suite.store.(kv.SortedSet)
	suite.set = suite.store.(kv.Set)
}

func (suite *RedisTestSuite) TearDownTest() {
	suite.store.Close()
	suite.miniredis.Close()
}

// Store interface

func (suite *RedisTestSuite) TestGetSet() {
	// Get a value that we have not set
	_, err := suite.store.Get(ctx, collection, key)
	assert.Equal(suite.T(), err, kv.ErrNotFound)

	// Use Set to writes the value
	err = suite.store.Set(ctx, collection, key, otherValue, 0)
	require.NoError(suite.T(), err)

	// Check the value is set correctly
	valueBytes, err := suite.store.Get(ctx, collection, key)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), otherValue, string(valueBytes))
}

func (suite *RedisTestSuite) TestSetNX() {
	// Set the value requiring it to not exist
	success, err := suite.store.SetNX(ctx, collection, key, value, 0)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), success)

	// Set the value requiring it to not exist
	success, err = suite.store.SetNX(ctx, collection, key, value, 0)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), success)

	// Check the value is set correctly
	valueBytes, err := suite.store.Get(ctx, collection, key)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), value, string(valueBytes))
}

func (suite *RedisTestSuite) TestMSetNX() {
	success, err := suite.store.MSetNX(ctx, collection, values)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), success)

	success, err = suite.store.MSetNX(ctx, collection, values)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), success)

	redisKeys, err := suite.store.Keys(ctx, collection)
	redisKeySet := mapset.NewSet()
	for _, v := range redisKeys {
		redisKeySet.Add(v)
	}
	require.True(suite.T(), keysSet.Equal(redisKeySet))

	valueBytes, err := suite.store.MGet(ctx, collection, redisKeys)
	require.NoError(suite.T(), err)
	for _, v := range redisKeys {
		assert.Equal(suite.T(), values[v], string(valueBytes[v]))
	}
}

func (suite *RedisTestSuite) TestMGetMSetKeys() {
	ctx := context.Background()
	collection := "collection"

	err := suite.store.MSet(ctx, collection, values)
	require.NoError(suite.T(), err)

	redisKeys, err := suite.store.Keys(ctx, collection)
	redisKeySet := mapset.NewSet()
	for _, v := range redisKeys {
		redisKeySet.Add(v)
	}
	require.True(suite.T(), keysSet.Equal(redisKeySet))

	valueBytes, err := suite.store.MGet(ctx, collection, redisKeys)
	require.NoError(suite.T(), err)
	for _, v := range redisKeys {
		assert.Equal(suite.T(), values[v], string(valueBytes[v]))
	}
}

func (suite *RedisTestSuite) TestDel() {
	// Use Set to writes the value
	err := suite.store.Set(ctx, collection, key, otherValue, 0)
	require.NoError(suite.T(), err)

	// Check the value is set correctly
	valueBytes, err := suite.store.Get(ctx, collection, key)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), otherValue, string(valueBytes))

	// delete the record
	err = suite.store.Del(ctx, collection, key)
	require.NoError(suite.T(), err)

	// assert Get returns the not found error
	_, err = suite.store.Get(ctx, collection, key)
	assert.Equal(suite.T(), kv.ErrNotFound, err)
}

// SortedSet interface

func (suite *RedisTestSuite) TestSortedSet() {
	// Add 3 members to the set
	member1 := &kv.Z{Score: 1, Member: "foo"}
	member2 := &kv.Z{Score: 2, Member: "bar"}
	member3 := &kv.Z{Score: 3, Member: "boo"}

	err := suite.sortedSet.ZAdd(ctx, key, member1, member2, member3)
	require.NoError(suite.T(), err)

	// retrieve the set ordered by sore
	results, err := suite.sortedSet.ZRangeByScore(ctx, key, &kv.ZRangeBy{
		Min: "1", Max: "3",
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), member1.Member.(string), results[0])
	assert.Equal(suite.T(), member2.Member.(string), results[1])
	assert.Equal(suite.T(), member3.Member.(string), results[2])

	// remove member 2
	err = suite.sortedSet.ZRem(ctx, key, member2.Member)
	require.NoError(suite.T(), err)

	// check member 2 is removed
	results, err = suite.sortedSet.ZRangeByScore(ctx, key, &kv.ZRangeBy{
		Min: "1", Max: "3",
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), member1.Member.(string), results[0])
	assert.Equal(suite.T(), member3.Member.(string), results[1])
}

// Set interface
func (suite *RedisTestSuite) TestSet() {
	// add keys to set
	err := suite.set.SAdd(ctx, key, keys...)
	require.NoError(suite.T(), err)

	// read set members and assert equal to key set
	members, err := suite.set.SMembers(ctx, key)
	require.NoError(suite.T(), err)
	membersSet := mapset.NewSet()
	for _, member := range members {
		membersSet.Add(member)
	}
	assert.True(suite.T(), keysSet.Equal(membersSet))

	// remove a key
	err = suite.set.SRem(ctx, key, keys[0])
	require.NoError(suite.T(), err)

	// construct key set that does not contain the removed key
	remKeySet := keysSet.Clone()
	remKeySet.Remove(keys[0])

	// read set members and assert equal to the new key set
	members, err = suite.set.SMembers(ctx, key)
	require.NoError(suite.T(), err)
	membersSet = mapset.NewSet()
	for _, member := range members {
		membersSet.Add(member)
	}
	assert.True(suite.T(), remKeySet.Equal(membersSet))

	// Pop upto 1k items from the set and assert that equals our new key set
	members, err = suite.set.SPopN(ctx, key, 1000)
	require.NoError(suite.T(), err)
	membersSet = mapset.NewSet()
	for _, member := range members {
		membersSet.Add(member)
	}
	assert.True(suite.T(), remKeySet.Equal(membersSet))

	// assert the member set is now empty
	members, err = suite.set.SMembers(ctx, key)
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), members)
}

func TestRedisTestSuite(t *testing.T) {
	suite.Run(t, new(RedisTestSuite))
}
