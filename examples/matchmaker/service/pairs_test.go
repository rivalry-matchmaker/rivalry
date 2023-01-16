package service_test

import (
	"testing"

	"github.com/rivalry-matchmaker/rivalry/examples/matchmaker/service"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	mock_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/mock"
	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	poolEverybody = "pool_everybody"
	profile       = &pb.MatchProfile{Name: "everybody", Pools: []*pb.Pool{{Name: poolEverybody}}}
	t1            = &pb.Ticket{Id: xid.New().String()}
	t2            = &pb.Ticket{Id: xid.New().String()}
)

type PairsTestSuite struct {
	suite.Suite
	service pb.MatchMakerServiceServer
}

func (s *PairsTestSuite) SetupTest() {
	s.service = service.NewPairsMatchmaker(false)
}

func (s *PairsTestSuite) TestPairs() {
	svr := mock_pb.NewMockMatchMakerService_MakeMatchesServer(gomock.NewController(s.T()))
	svr.EXPECT().Send(gomock.Any()).Do(func(resp *pb.MakeMatchesResponse) error {
		assert.Equal(s.T(), 2, len(resp.Match.Tickets))
		return nil
	})
	err := s.service.MakeMatches(&pb.MakeMatchesRequest{
		MatchProfile: profile,
		PoolTickets: map[string]*pb.MakeMatchesRequest_PoolTickets{
			poolEverybody: {Tickets: []*pb.Ticket{t1, t2}},
		},
	}, svr)
	assert.NoError(s.T(), err)
}

func (s *PairsTestSuite) TestPairsSingleTicketNoMatch() {
	svr := mock_pb.NewMockMatchMakerService_MakeMatchesServer(gomock.NewController(s.T()))
	err := s.service.MakeMatches(&pb.MakeMatchesRequest{
		MatchProfile: profile,
		PoolTickets: map[string]*pb.MakeMatchesRequest_PoolTickets{
			poolEverybody: {Tickets: []*pb.Ticket{t1}},
		},
	}, svr)
	assert.NoError(s.T(), err)
}

func (s *PairsTestSuite) TestPairsMatchSingleTicketAndBackfill() {
	svr := mock_pb.NewMockMatchMakerService_MakeMatchesServer(gomock.NewController(s.T()))
	svr.EXPECT().Send(gomock.Any()).Do(func(resp *pb.MakeMatchesResponse) error {
		assert.Equal(s.T(), 1, len(resp.Match.Tickets))
		assert.NotNil(s.T(), resp.Match.Backfill)
		return nil
	})
	err := s.service.MakeMatches(&pb.MakeMatchesRequest{
		MatchProfile: profile,
		PoolTickets: map[string]*pb.MakeMatchesRequest_PoolTickets{
			poolEverybody: {Tickets: []*pb.Ticket{t1}},
		},
		PoolBackfills: map[string]*pb.MakeMatchesRequest_PoolBackfills{
			poolEverybody: {Backfill: []*pb.Backfill{{Id: xid.New().String()}}},
		},
	}, svr)
	assert.NoError(s.T(), err)
}

func TestPairsTestSuite(t *testing.T) {
	suite.Run(t, new(PairsTestSuite))
}

type BackfillPairsTestSuite struct {
	suite.Suite
	service pb.MatchMakerServiceServer
}

func (s *BackfillPairsTestSuite) SetupTest() {
	s.service = service.NewPairsMatchmaker(true)
}

func (s *BackfillPairsTestSuite) TestPairsSingleTicketMakesBackfill() {
	svr := mock_pb.NewMockMatchMakerService_MakeMatchesServer(gomock.NewController(s.T()))
	svr.EXPECT().Send(gomock.Any()).Do(func(resp *pb.MakeMatchesResponse) error {
		assert.Equal(s.T(), 1, len(resp.Match.Tickets))
		assert.NotNil(s.T(), resp.Match.Backfill)
		return nil
	})
	err := s.service.MakeMatches(&pb.MakeMatchesRequest{
		MatchProfile: profile,
		PoolTickets: map[string]*pb.MakeMatchesRequest_PoolTickets{
			poolEverybody: {Tickets: []*pb.Ticket{t1}},
		},
	}, svr)
	assert.NoError(s.T(), err)
}

func TestBackfillPairsTestSuite(t *testing.T) {
	suite.Run(t, new(BackfillPairsTestSuite))
}
