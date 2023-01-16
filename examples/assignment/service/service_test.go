package service_test

import (
	"context"
	"testing"

	"github.com/rivalry-matchmaker/rivalry/examples/assignment/service"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	mock_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/mock"
	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type FakeAssignmentServiceTestSuite struct {
	suite.Suite
	service        pb.AssignmentServiceServer
	frontendClient *mock_pb.MockFrontendServiceClient
	ctx            context.Context
	cancel         func()
}

func (s *FakeAssignmentServiceTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.frontendClient = mock_pb.NewMockFrontendServiceClient(gomock.NewController(s.T()))
	s.service = service.NewFakeServerAssignmentServiceWithClient(s.ctx, s.frontendClient)
}

func (s *FakeAssignmentServiceTestSuite) TestFakeAssignmentService() {
	// we will pass a match with a backfill, so we will expect an AcknowledgeBackfill
	s.frontendClient.EXPECT().AcknowledgeBackfill(gomock.Any(), gomock.Any()).Do(
		func(ctx context.Context, in *pb.AcknowledgeBackfillRequest, opts ...grpc.CallOption) {
			// cancel the context here to break out of the go routine
			s.cancel()
		})
	assignment, err := s.service.MakeAssignment(s.ctx, &pb.Match{
		MatchId:  xid.New().String(),
		Backfill: &pb.Backfill{Id: xid.New().String()},
	})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), assignment.Connection)
	// block the test ending until the acknowledge backfill endpoint has been called
	<-s.ctx.Done()
}

func TestFakeAssignmentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(FakeAssignmentServiceTestSuite))
}
