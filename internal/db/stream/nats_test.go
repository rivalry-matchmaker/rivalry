package stream_test

import (
	"sync"
	"testing"

	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type NATSTestSuite struct {
	suite.Suite
	server *server.Server
	cli    stream.Client
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
	suite.cli = stream.NewNATS(nats.DefaultURL, "group")
}

func (suite *NATSTestSuite) TearDownTest() {
	suite.cli.Close()
	suite.server.Shutdown()
}

func (suite *NATSTestSuite) TestNATS() {
	topic := "topic"
	message := []byte("hello world")
	var wg sync.WaitGroup
	wg.Add(1)
	suite.cli.Subscribe(topic, func(bytes []byte) {
		assert.Equal(suite.T(), message, bytes)
		wg.Done()
	})
	suite.cli.SendMessage(topic, message)
	wg.Wait()
}

func TestNATSTestSuite(t *testing.T) {
	suite.Run(t, new(NATSTestSuite))
}
