package backend_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/app/backend"
	backfill "github.com/rivalry-matchmaker/rivalry/internal/managers/backfill/mock"
	customlogic "github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic/mock"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	matches "github.com/rivalry-matchmaker/rivalry/internal/managers/matches/mock"
	ticketsMock "github.com/rivalry-matchmaker/rivalry/internal/managers/tickets/mock"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var profileName = "everybody"
var profiles = []*pb.MatchProfile{{Name: profileName, Pools: []*pb.Pool{{Name: "pool_everybody"}}}}

type AccumulatorTestSuite struct {
	suite.Suite
	accumulator        backend.Accumulator
	ctx                context.Context
	cancel             func()
	ticketsManager     *ticketsMock.MockManager
	backfillManager    *backfill.MockManager
	matchesManager     *matches.MockManager
	customlogicManager *customlogic.MockMatchmakerManager
	filterManager      filter.Manager
	config             *backend.AccumulatorConfig
}

func (s *AccumulatorTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.ticketsManager = ticketsMock.NewMockManager(gomock.NewController(s.T()))
	s.backfillManager = backfill.NewMockManager(gomock.NewController(s.T()))
	s.matchesManager = matches.NewMockManager(gomock.NewController(s.T()))
	s.filterManager = filter.NewManager(profiles)
	s.customlogicManager = customlogic.NewMockMatchmakerManager(gomock.NewController(s.T()))
	s.config = &backend.AccumulatorConfig{
		MaxTickets:  2,
		MaxDelay:    time.Second / 2,
		MaxBackfill: 1,
	}
	s.accumulator = backend.NewAccumulator(
		s.ctx, profileName, profiles, s.ticketsManager, s.backfillManager, s.matchesManager, s.customlogicManager, s.config)
}

func (s *AccumulatorTestSuite) TestAccumulatorMatchMadeMaxDelay() {
	t := &pb.Ticket{Id: xid.New().String()}
	profileName, pool, err := s.filterManager.ProfileMembershipTest(t)
	require.NoError(s.T(), err)
	s.ticketsManager.EXPECT().StreamTickets(s.ctx, profileName, gomock.Any()).Do(
		func(ctx context.Context, profile string, f func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket)) {
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t.Id}, t)
		})
	s.backfillManager.EXPECT().GetAvailableBackfill(s.ctx, profileName, s.config.MaxBackfill).Return(
		[]*pb.Backfill{{Id: xid.New().String()}}, nil)
	s.customlogicManager.EXPECT().MakeMatches(s.ctx, gomock.Any(), gomock.Any()).Do(
		func(ctx context.Context, req *pb.MakeMatchesRequest, f func(ctx context.Context, match *pb.Match)) {
			f(ctx, &pb.Match{
				Backfill: &pb.Backfill{Id: xid.New().String()},
				Tickets:  []*pb.Ticket{t},
			})
		})
	s.backfillManager.EXPECT().MakeMatchWithBackfill(s.ctx, gomock.Any(), gomock.Any(), s.filterManager)
	s.matchesManager.EXPECT().CreateMatch(s.ctx, gomock.Any()).Return(true, nil)
	go func() {
		time.Sleep(time.Second)
		s.cancel()
	}()
	assert.NoError(s.T(), s.accumulator.Run())
}

func (s *AccumulatorTestSuite) TestAccumulatorMatchMadeMaxTickets() {
	t := &pb.Ticket{Id: xid.New().String()}
	t2 := &pb.Ticket{Id: xid.New().String()}
	profileName, pool, err := s.filterManager.ProfileMembershipTest(t)
	require.NoError(s.T(), err)
	s.ticketsManager.EXPECT().StreamTickets(s.ctx, profileName, gomock.Any()).Do(
		func(ctx context.Context, profile string, f func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket)) {
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t.Id}, t)
			// send the same ticket again to test deduplication
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t.Id}, t)
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t2.Id}, t2)
		})
	s.backfillManager.EXPECT().GetAvailableBackfill(s.ctx, profileName, s.config.MaxBackfill).Return(
		[]*pb.Backfill{{Id: xid.New().String()}}, nil)
	s.customlogicManager.EXPECT().MakeMatches(s.ctx, gomock.Any(), gomock.Any()).Do(
		func(ctx context.Context, req *pb.MakeMatchesRequest, f func(ctx context.Context, match *pb.Match)) {
			// assert 2 tickets remain and therefore deduplication worked
			assert.Equal(s.T(), 2, len(req.PoolTickets[pool].Tickets))
			f(ctx, &pb.Match{
				Backfill: &pb.Backfill{Id: xid.New().String()},
				Tickets:  []*pb.Ticket{t, t2},
			})
		})
	s.backfillManager.EXPECT().MakeMatchWithBackfill(s.ctx, gomock.Any(), gomock.Any(), s.filterManager)
	s.matchesManager.EXPECT().CreateMatch(s.ctx, gomock.Any()).Return(true, nil)
	go func() {
		time.Sleep(time.Second / 2)
		s.cancel()
	}()
	assert.NoError(s.T(), s.accumulator.Run())
}

func (s *AccumulatorTestSuite) TestAccumulatorRequeueAllTickets() {
	t := &pb.Ticket{Id: xid.New().String()}
	t2 := &pb.Ticket{Id: xid.New().String()}
	profileName, pool, err := s.filterManager.ProfileMembershipTest(t)
	require.NoError(s.T(), err)
	s.ticketsManager.EXPECT().StreamTickets(s.ctx, profileName, gomock.Any()).Do(
		func(ctx context.Context, profile string, f func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket)) {
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t.Id}, t)
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t2.Id}, t2)
		})
	s.backfillManager.EXPECT().GetAvailableBackfill(s.ctx, profileName, s.config.MaxBackfill).Return(
		nil, fmt.Errorf("fail"))
	s.ticketsManager.EXPECT().RequeueTickets(s.ctx, gomock.Any(), s.filterManager)
	go func() {
		time.Sleep(time.Second / 2)
		s.cancel()
	}()
	assert.NoError(s.T(), s.accumulator.Run())
}

func (s *AccumulatorTestSuite) TestAccumulatorNotAllTicketsMatched() {
	t := &pb.Ticket{Id: xid.New().String()}
	t2 := &pb.Ticket{Id: xid.New().String()}
	profileName, pool, err := s.filterManager.ProfileMembershipTest(t)
	require.NoError(s.T(), err)
	// we stream 2 tickets
	s.ticketsManager.EXPECT().StreamTickets(s.ctx, profileName, gomock.Any()).Do(
		func(ctx context.Context, profile string, f func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket)) {
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t.Id}, t)
			f(ctx, &pb.StreamTicket{Profile: profileName, Pool: pool, TicketId: t2.Id}, t2)
		})
	s.backfillManager.EXPECT().GetAvailableBackfill(s.ctx, profileName, s.config.MaxBackfill).Return(
		[]*pb.Backfill{}, nil)
	// MakeMatches only matches 1 of the 2 tickets
	s.customlogicManager.EXPECT().MakeMatches(s.ctx, gomock.Any(), gomock.Any()).Do(
		func(ctx context.Context, req *pb.MakeMatchesRequest, f func(ctx context.Context, match *pb.Match)) {
			f(ctx, &pb.Match{
				Tickets: []*pb.Ticket{t},
			})
		})
	s.matchesManager.EXPECT().CreateMatch(s.ctx, gomock.Any()).Return(true, nil)
	// expect the accumulator to requeue the ticket that was not matched
	s.ticketsManager.EXPECT().RequeueTickets(s.ctx, gomock.Any(), s.filterManager).Do(
		func(ctx context.Context, tickets []*pb.Ticket, filterManager filter.Manager) {
			assert.Contains(s.T(), tickets, t2)
		})
	go func() {
		time.Sleep(time.Second / 2)
		s.cancel()
	}()
	assert.NoError(s.T(), s.accumulator.Run())
}

func TestAccumulatorTestSuite(t *testing.T) {
	suite.Run(t, new(AccumulatorTestSuite))
}
