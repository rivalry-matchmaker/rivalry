package pubsub_test

import (
	"context"
	"testing"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type NATSTestSuite struct {
	suite.Suite
	server *server.Server
	cli    pubsub.Client
}

func (suite *NATSTestSuite) SetupTest() {
	var err error
	suite.server, err = server.NewServer(&server.Options{
		Host: "127.0.0.1",
		Port: nats.DefaultPort,
	})
	require.NoError(suite.T(), err)
	err = server.Run(suite.server)
	require.NoError(suite.T(), err)
	suite.cli = pubsub.NewNATS(nats.DefaultURL)
}

func (suite *NATSTestSuite) TearDownTest() {
	suite.cli.Close()
	suite.server.Shutdown()
}

func (suite *NATSTestSuite) TestNATS() {
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

func TestNATSTestSuite(t *testing.T) {
	suite.Run(t, new(NATSTestSuite))
}
