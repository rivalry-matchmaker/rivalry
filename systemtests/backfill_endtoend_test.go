package systemtests

import (
	"context"
	"fmt"
	"math/rand"
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

type BackfillEndToEndTestSuite struct {
	suite.Suite
	redis             *tempredis.Server
	natsServer        *server.Server
	frontend          *cobra.Command
	accumulator       *cobra.Command
	dispenser         *cobra.Command
	matchmaker        *cobra.Command
	assignmentService *cobra.Command
}

func (s *BackfillEndToEndTestSuite) SetupSuite() {
	s.redis = initRedis(&s.Suite)
	s.natsServer = initNats(&s.Suite)
	var (
		redisAddr = s.redis.Socket()
		natsAddr  = nats.DefaultURL
	)
	s.frontend = initFrontend(&s.Suite, redisAddr, natsAddr)
	s.accumulator = initAccumulator(&s.Suite, redisAddr, natsAddr)
	s.dispenser = initDispenser(&s.Suite, redisAddr, natsAddr)
	s.matchmaker = initMatchmaker(true)
	s.assignmentService = initAssignmentService()
	time.Sleep(time.Second / 2)
}

func (s *BackfillEndToEndTestSuite) TearDownSuite() {
	s.natsServer.Shutdown()
	s.redis.Term()
	stopFrontend()
	stopAccumulator()
	stopDispenser()
	stopMatchmaker()
	stopAssignmentService()
}

func (s *BackfillEndToEndTestSuite) TestTwoPlayerMatchWithPauseBetweenSubmissions() {
	frontendConn, err := grpc.Dial(frontendTarget, grpc.WithInsecure())
	require.NoError(s.T(), err)
	frontendServiceClient := pb.NewFrontendServiceClient(frontendConn)
	ctx := context.Background()
	var (
		wgP1 sync.WaitGroup
		wgP2 sync.WaitGroup
	)

	p1 := newPlayer(s.Suite, &wgP1, frontendServiceClient, "Player 1")
	wgP1.Add(1)
	go p1.playTheGame(ctx)
	wgP1.Wait()

	time.Sleep(time.Second / 2)

	wgP2.Add(1)
	p2 := newPlayer(s.Suite, &wgP2, frontendServiceClient, "Player 2")
	go p2.playTheGame(ctx)
	wgP2.Wait()

	assert.Equal(s.T(), p1.assignment.Connection, p2.assignment.Connection)
}

func (s *BackfillEndToEndTestSuite) TestServerCreatedBackfill() {
	// create our server connection string
	connection := "127.0.0.1:" + fmt.Sprint(1024+rand.Intn(65535-1024))

	// create a connection to the frontend
	frontendConn, err := grpc.Dial(frontendTarget, grpc.WithInsecure())
	require.NoError(s.T(), err)
	cli := pb.NewFrontendServiceClient(frontendConn)
	ctx, cancel := context.WithCancel(context.Background())

	// create a backfill
	backfill, err := cli.CreateBackfill(ctx,
		&pb.CreateBackfillRequest{
			Backfill: &pb.Backfill{Extensions: &pb.Extensions{DoubleExtensions: map[string]float64{"spaces": 1}}}})
	require.NoError(s.T(), err)

	// loop acknowledging the backfill
	go func() {
		for range time.Tick(time.Second) {
			select {
			case <-ctx.Done():
				return
			default:
				_, err = cli.AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{
					BackfillId: backfill.Id, Assignment: &pb.Assignment{Connection: connection}})
				if err != nil {
					log.Err(err).Msg("failed to acknowledge backfill")
				}
			}
		}
	}()

	// player 1 plays the game
	var wg sync.WaitGroup
	wg.Add(1)
	p1 := newPlayer(s.Suite, &wg, cli, "Player 1")
	go p1.playTheGame(ctx)
	wg.Wait()

	// assert player 1 fill the backfill space in our server
	assert.Equal(s.T(), p1.assignment.Connection, connection)
	cancel()
}

func TestBackfillEndToEndTestSuite(t *testing.T) {
	suite.Run(t, new(BackfillEndToEndTestSuite))
}
