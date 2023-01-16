package systemtests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/stvp/tempredis"
	"google.golang.org/grpc"
)

type SimpleEndToEndTestSuite struct {
	suite.Suite
	redis             *tempredis.Server
	natsServer        *server.Server
	frontend          *cobra.Command
	accumulator       *cobra.Command
	dispenser         *cobra.Command
	matchmaker        *cobra.Command
	assignmentService *cobra.Command
}

func (s *SimpleEndToEndTestSuite) SetupSuite() {
	s.redis = initRedis(&s.Suite)
	s.natsServer = initNats(&s.Suite)
	var (
		redisAddr = s.redis.Socket()
		natsAddr  = nats.DefaultURL
	)
	s.frontend = initFrontend(&s.Suite, redisAddr, natsAddr)
	s.accumulator = initAccumulator(&s.Suite, redisAddr, natsAddr)
	s.dispenser = initDispenser(&s.Suite, redisAddr, natsAddr)
	s.matchmaker = initMatchmaker(false)
	s.assignmentService = initAssignmentService()
	time.Sleep(time.Second / 2)
}

func (s *SimpleEndToEndTestSuite) TearDownSuite() {
	s.natsServer.Shutdown()
	s.redis.Term()
	stopFrontend()
	stopAccumulator()
	stopDispenser()
	stopMatchmaker()
	stopAssignmentService()
}

func (s *SimpleEndToEndTestSuite) TestTwoPlayerMatch() {
	frontendConn, err := grpc.Dial(frontendTarget, grpc.WithInsecure())
	require.NoError(s.T(), err)
	frontendServiceClient := pb.NewFrontendServiceClient(frontendConn)
	ctx := context.Background()
	var wg sync.WaitGroup

	p1 := newPlayer(s.Suite, &wg, frontendServiceClient, "Player 1")
	p2 := newPlayer(s.Suite, &wg, frontendServiceClient, "Player 2")
	wg.Add(2)
	go p1.playTheGame(ctx)
	go p2.playTheGame(ctx)
	wg.Wait()

	assert.Equal(s.T(), p1.assignment.Connection, p2.assignment.Connection)
}

func (s *SimpleEndToEndTestSuite) TestTwoPlayerMatchWithPauseBetweenSubmissions() {
	frontendConn, err := grpc.Dial(frontendTarget, grpc.WithInsecure())
	require.NoError(s.T(), err)
	frontendServiceClient := pb.NewFrontendServiceClient(frontendConn)
	ctx := context.Background()
	var wg sync.WaitGroup
	p1 := newPlayer(s.Suite, &wg, frontendServiceClient, "Player 1")
	wg.Add(1)
	go p1.playTheGame(ctx)

	// wait for a few passes of the match function timing out
	log.Info().Msg("sleep for a second")
	time.Sleep(time.Second)
	log.Info().Msg("finished sleeping")

	p2 := newPlayer(s.Suite, &wg, frontendServiceClient, "Player 2")
	wg.Add(1)
	go p2.playTheGame(ctx)
	wg.Wait()

	assert.Equal(s.T(), p1.assignment.Connection, p2.assignment.Connection)
}

func TestSimpleEndToEndTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleEndToEndTestSuite))
}
