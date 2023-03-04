package backend_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rivalry-matchmaker/rivalry/internal/app/backend"
	ticketsMock "github.com/rivalry-matchmaker/rivalry/internal/managers/tickets/mock"
	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	stream "github.com/rivalry-matchmaker/rivalry/pkg/pb/stream/v1"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var profileName = "everybody"

type AccumulatorTestSuite struct {
	suite.Suite
	accumulator    backend.Accumulator
	ctx            context.Context
	cancel         func()
	ticketsManager *ticketsMock.MockManager
	config         *backend.AccumulatorConfig
}

func (s *AccumulatorTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.ticketsManager = ticketsMock.NewMockManager(gomock.NewController(s.T()))
	s.config = &backend.AccumulatorConfig{
		MaxTickets: 2,
		MaxDelay:   time.Second / 2,
	}
	s.accumulator = backend.NewAccumulator(
		s.ctx, profileName, s.ticketsManager, s.config)
}

func (s *AccumulatorTestSuite) TestAccumulatorMatchMadeMaxDelay() {
	t := &pb.MatchRequest{Id: xid.New().String()}
	s.ticketsManager.EXPECT().StreamMatchRequests(s.ctx, profileName, gomock.Any()).Do(
		func(ctx context.Context, profile string, f func(ctx context.Context, st *stream.StreamTicket)) {
			f(ctx, &stream.StreamTicket{MatchmakingQueue: profileName, MatchRequestId: t.Id})
		})

	s.ticketsManager.EXPECT().PublishAccumulatedMatchRequests(s.ctx, gomock.Any(), gomock.Any())
	go func() {
		time.Sleep(time.Second)
		s.cancel()
	}()
	assert.NoError(s.T(), s.accumulator.Run())
}

func (s *AccumulatorTestSuite) TestAccumulatorMatchMadeMaxMatchRequests() {
	t := &pb.MatchRequest{Id: xid.New().String()}
	t2 := &pb.MatchRequest{Id: xid.New().String()}
	s.ticketsManager.EXPECT().StreamMatchRequests(s.ctx, profileName, gomock.Any()).Do(
		func(ctx context.Context, profile string, f func(ctx context.Context, st *stream.StreamTicket)) {
			f(ctx, &stream.StreamTicket{MatchmakingQueue: profileName, MatchRequestId: t.Id})
			// send the same ticket again to test deduplication
			f(ctx, &stream.StreamTicket{MatchmakingQueue: profileName, MatchRequestId: t.Id})
			f(ctx, &stream.StreamTicket{MatchmakingQueue: profileName, MatchRequestId: t2.Id})
		})
	s.ticketsManager.EXPECT().PublishAccumulatedMatchRequests(s.ctx, gomock.Any(), gomock.Any())
	go func() {
		time.Sleep(time.Second / 2)
		s.cancel()
	}()
	assert.NoError(s.T(), s.accumulator.Run())
}

func TestAccumulatorTestSuite(t *testing.T) {
	suite.Run(t, new(AccumulatorTestSuite))
}
