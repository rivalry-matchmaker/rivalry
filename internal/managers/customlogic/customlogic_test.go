package customlogic_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	mock_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/mock"
	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ctx     = context.Background()
	profile = &pb.MatchProfile{Name: "everybody", Pools: []*pb.Pool{{Name: "pool_everybody"}}}
)

type FrontendTestSuite struct {
	suite.Suite
	manager          customlogic.FrontendManager
	validationClient *mock_pb.MockValidationServiceClient
	dataClient       *mock_pb.MockDataServiceClient
}

func (s *FrontendTestSuite) SetupTest() {
	s.validationClient = mock_pb.NewMockValidationServiceClient(gomock.NewController(s.T()))
	s.dataClient = mock_pb.NewMockDataServiceClient(gomock.NewController(s.T()))
	s.manager = customlogic.NewFrontendManager(s.validationClient, s.dataClient)
}

func (s *FrontendTestSuite) TestValidateNoError() {
	ticket := &pb.Ticket{Id: xid.New().String()}
	s.validationClient.EXPECT().Validate(ctx, ticket)
	assert.NoError(s.T(), s.manager.Validate(ctx, ticket))
}

func (s *FrontendTestSuite) TestValidateNonGRPCError() {
	ticket := &pb.Ticket{Id: xid.New().String()}
	s.validationClient.EXPECT().Validate(ctx, ticket).Return(nil, fmt.Errorf("fail"))
	assert.Error(s.T(), s.manager.Validate(ctx, ticket))
}

func (s *FrontendTestSuite) TestValidateGRPCError() {
	ticket := &pb.Ticket{Id: xid.New().String()}
	s.validationClient.EXPECT().Validate(ctx, ticket).Return(nil, status.Error(codes.InvalidArgument, "fail"))
	assert.Error(s.T(), s.manager.Validate(ctx, ticket))
}

func (s *FrontendTestSuite) TestValidateUnimplemented() {
	ticket := &pb.Ticket{Id: xid.New().String()}
	s.validationClient.EXPECT().Validate(ctx, ticket).Return(nil, status.Error(codes.Unimplemented, "unimplemented"))
	assert.NoError(s.T(), s.manager.Validate(ctx, ticket))
}

func (s *FrontendTestSuite) TestGatherData() {
	ticket := &pb.Ticket{Id: xid.New().String()}
	s.dataClient.EXPECT().GatherData(ctx, ticket).Return(&pb.GatherDataResponse{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"rank": 42,
			},
		},
	}, nil)
	t, err := s.manager.GatherData(ctx, ticket)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), ticket.Id, t.Id)
	assert.Equal(s.T(), float64(42), t.SearchFields.DoubleArgs["rank"])
}

func TestFrontendTestSuite(t *testing.T) {
	suite.Run(t, new(FrontendTestSuite))
}

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
	req := &pb.MakeMatchesRequest{
		MatchProfile: profile,
	}
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
	conn := xid.New().String()
	s.assignmentClient.EXPECT().MakeAssignment(ctx, match).Return(&pb.Assignment{Connection: conn}, nil)
	assignment, err := s.manager.MakeAssignment(ctx, match)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), conn, assignment.Connection)
}

func TestAssignmentTestSuite(t *testing.T) {
	suite.Run(t, new(AssignmentTestSuite))
}
