package customlogic_test

import (
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	mock_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1/mock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = context.Background()
)

type MatchmakerTestSuite struct {
	suite.Suite
	manager          customlogic.MatchmakerManager
	matchmakerClient *mock_pb.MockMatchMakerServiceClient
}

func (s *MatchmakerTestSuite) SetupTest() {
	s.matchmakerClient = mock_pb.NewMockMatchMakerServiceClient(gomock.NewController(s.T()))
	s.manager = customlogic.NewMatchmakerManager(s.matchmakerClient)
}

func (s *MatchmakerTestSuite) TestMakeMatches() {
	req := &pb.MakeMatchesRequest{}
	match := &pb.Match{MatchId: xid.New().String()}
	cli := mock_pb.NewMockMatchMakerService_MakeMatchesClient(gomock.NewController(s.T()))
	s.matchmakerClient.EXPECT().MakeMatches(ctx, req).Return(cli, nil)
	cli.EXPECT().Recv().Return(&pb.MakeMatchesResponse{Match: match}, nil)
	cli.EXPECT().Recv().Return(nil, io.EOF)
	s.manager.MakeMatches(ctx, req, func(ctx context.Context, m *pb.Match) {
		assert.Equal(s.T(), match.MatchId, m.MatchId)
	})
}

func TestMatchmakerTestSuite(t *testing.T) {
	suite.Run(t, new(MatchmakerTestSuite))
}

type AssignmentTestSuite struct {
	suite.Suite
	manager          customlogic.AssignmentManager
	assignmentClient *mock_pb.MockAssignmentServiceClient
}

func (s *AssignmentTestSuite) SetupTest() {
	s.assignmentClient = mock_pb.NewMockAssignmentServiceClient(gomock.NewController(s.T()))
	s.manager = customlogic.NewAssignmentManager(s.assignmentClient)
}

func (s *AssignmentTestSuite) TestMakeAssignment() {
	match := &pb.Match{MatchId: xid.New().String()}
	addr := "127.0.0.1"
	s.assignmentClient.EXPECT().MakeAssignment(ctx, gomock.Any()).Return(&pb.MakeAssignmentResponse{GameServer: &pb.GameServer{GameServerIp: addr}}, nil)
	assignment, err := s.manager.MakeAssignment(ctx, match)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), addr, assignment.GameServerIp)
}

func TestAssignmentTestSuite(t *testing.T) {
	suite.Run(t, new(AssignmentTestSuite))
}
