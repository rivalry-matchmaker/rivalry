package pubsub_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RedisTestSuite struct {
	suite.Suite
	miniredis *miniredis.Miniredis
	cli       pubsub.Client
}

func (suite *RedisTestSuite) SetupTest() {
	var err error
	suite.miniredis, err = miniredis.Run()
	require.NoError(suite.T(), err)
	suite.cli = pubsub.NewRedis("prefix", &redis.Options{Addr: suite.miniredis.Addr()})
}

func (suite *RedisTestSuite) TearDownTest() {
	suite.cli.Close()
	suite.miniredis.Close()
}

func (suite *RedisTestSuite) TestRedis() {
	topic := "topic"
	message := []byte("hello world")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second / 10)
		suite.cli.Publish(context.Background(), topic, message)
	}()
	suite.cli.Subscribe(ctx, topic, func(bytes []byte) {
		assert.Equal(suite.T(), message, bytes)
		cancel()
	})
}

func TestRedisTestSuite(t *testing.T) {
	suite.Run(t, new(RedisTestSuite))
}
