package service_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rivalry-matchmaker/rivalry/examples/assignment/service"
	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	mock_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1/mock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FakeAssignmentServiceTestSuite struct {
	suite.Suite
	service        api.AssignmentServiceServer
	frontendClient *mock_pb.MockRivalryServiceClient
}

func (s *FakeAssignmentServiceTestSuite) SetupTest() {
	s.frontendClient = mock_pb.NewMockRivalryServiceClient(gomock.NewController(s.T()))
	s.service = service.NewFakeServerAssignmentServiceWithClient(s.frontendClient)
}

func (s *FakeAssignmentServiceTestSuite) TestFakeAssignmentService() {
	assignment, err := s.service.MakeAssignment(context.Background(), &api.MakeAssignmentRequest{Match: &api.Match{
		MatchId: xid.New().String(),
	}})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), assignment.GameServer)
}

func TestFakeAssignmentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(FakeAssignmentServiceTestSuite))
}
