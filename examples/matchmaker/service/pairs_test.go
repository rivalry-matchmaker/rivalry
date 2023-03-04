package service_test

import (
	"testing"

	"github.com/rivalry-matchmaker/rivalry/examples/matchmaker/service"
	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	mock_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1/mock"

	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	queue = "default"
	t1    = &pb.Submission{Id: xid.New().String(), Members: []string{xid.New().String()}}
	t2    = &pb.Submission{Id: xid.New().String(), Members: []string{xid.New().String()}}
)

type PairsTestSuite struct {
	suite.Suite
	service pb.MatchMakerServiceServer
}

func (s *PairsTestSuite) SetupTest() {
	s.service = service.NewPairsMatchmaker()
}

func (s *PairsTestSuite) TestPairs() {
	svr := mock_pb.NewMockMatchMakerService_MakeMatchesServer(gomock.NewController(s.T()))
	svr.EXPECT().Send(gomock.Any()).Do(func(resp *pb.MakeMatchesResponse) error {
		assert.Equal(s.T(), 2, len(resp.Match.MatchRequestIds))
		return nil
	})
	err := s.service.MakeMatches(&pb.MakeMatchesRequest{
		Submissions: []*pb.Submission{t1, t2},
	}, svr)
	assert.NoError(s.T(), err)
}

func (s *PairsTestSuite) TestPairsSingleTicketNoMatch() {
	svr := mock_pb.NewMockMatchMakerService_MakeMatchesServer(gomock.NewController(s.T()))
	err := s.service.MakeMatches(&pb.MakeMatchesRequest{
		Submissions: []*pb.Submission{t1},
	}, svr)
	assert.NoError(s.T(), err)
}

func TestPairsTestSuite(t *testing.T) {
	suite.Run(t, new(PairsTestSuite))
}
