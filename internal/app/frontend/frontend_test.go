package frontend_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rivalry-matchmaker/rivalry/internal/app/frontend"
	backfill "github.com/rivalry-matchmaker/rivalry/internal/managers/backfill/mock"
	customlogic "github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic/mock"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	ticketsMock "github.com/rivalry-matchmaker/rivalry/internal/managers/tickets/mock"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	mock_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/mock"
	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ctx     = context.Background()
	profile = &pb.MatchProfile{Name: "everybody", Pools: []*pb.Pool{{Name: "pool_everybody"}}}
)

type FrontendTestSuite struct {
	suite.Suite
	service            *frontend.Service
	ticketsManager     *ticketsMock.MockManager
	backfillManager    *backfill.MockManager
	customlogicManager *customlogic.MockFrontendManager
	filterManager      filter.Manager
}

func (s *FrontendTestSuite) SetupTest() {
	s.ticketsManager = ticketsMock.NewMockManager(gomock.NewController(s.T()))
	s.backfillManager = backfill.NewMockManager(gomock.NewController(s.T()))
	s.customlogicManager = customlogic.NewMockFrontendManager(gomock.NewController(s.T()))
	s.filterManager = filter.NewManager([]*pb.MatchProfile{profile})
	s.service = frontend.NewService(s.ticketsManager, s.backfillManager, s.customlogicManager, s.filterManager)
}

func (s *FrontendTestSuite) TestClose() {
	s.ticketsManager.EXPECT().Close()
	s.service.Close()
}

func (s *FrontendTestSuite) TestCreateTicketValidation() {
	for _, tt := range []struct {
		name    string
		req     *pb.CreateTicketRequest
		code    codes.Code
		message string
	}{
		{
			name:    "ticket is nil",
			req:     &pb.CreateTicketRequest{Ticket: nil},
			code:    codes.InvalidArgument,
			message: ".ticket is required",
		},
		{
			name:    "ticket is assigned",
			req:     &pb.CreateTicketRequest{Ticket: &pb.Ticket{Assignment: &pb.Assignment{Connection: "fake"}}},
			code:    codes.InvalidArgument,
			message: "tickets cannot be created with an assignment",
		},
		{
			name:    "ticket create time is not nil",
			req:     &pb.CreateTicketRequest{Ticket: &pb.Ticket{CreateTime: timestamppb.Now()}},
			code:    codes.InvalidArgument,
			message: "tickets cannot be created with create time set",
		},
	} {
		s.Run(tt.name, func() {
			_, err := s.service.CreateTicket(ctx, tt.req)
			assert.Error(s.T(), err)
			st, ok := status.FromError(err)
			require.True(s.T(), ok)
			assert.Equal(s.T(), tt.code, st.Code())
			assert.Equal(s.T(), tt.message, st.Message())
		})
	}
}

func (s *FrontendTestSuite) TestCreateTicketFailValidation() {
	t := &pb.Ticket{}
	s.customlogicManager.EXPECT().Validate(ctx, t).Return(fmt.Errorf("fail"))
	_, err := s.service.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: t})
	assert.Error(s.T(), err)
}

func (s *FrontendTestSuite) TestCreateTicketFailGatherData() {
	t := &pb.Ticket{}
	s.customlogicManager.EXPECT().Validate(ctx, t)
	s.customlogicManager.EXPECT().GatherData(ctx, t).Return(nil, fmt.Errorf("fail"))
	_, err := s.service.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: t})
	assert.Error(s.T(), err)
}

func (s *FrontendTestSuite) TestCreateTicketFailCreateTicket() {
	t := &pb.Ticket{}
	s.customlogicManager.EXPECT().Validate(ctx, t)
	s.customlogicManager.EXPECT().GatherData(ctx, t).Return(t, nil)
	s.ticketsManager.EXPECT().CreateTicket(ctx, t, s.filterManager).Return(nil, fmt.Errorf("fail"))
	_, err := s.service.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: t})
	assert.Error(s.T(), err)
}

func (s *FrontendTestSuite) TestCreateTicketFailDuplicateTicket() {
	t := &pb.Ticket{}
	s.customlogicManager.EXPECT().Validate(ctx, t)
	s.customlogicManager.EXPECT().GatherData(ctx, t).Return(t, nil)
	s.ticketsManager.EXPECT().CreateTicket(ctx, t, s.filterManager).Return(nil, tickets.ErrDuplicateTicketID)
	_, err := s.service.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: t})
	assert.Error(s.T(), err)
	st, ok := status.FromError(err)
	require.True(s.T(), ok)
	assert.Equal(s.T(), codes.AlreadyExists, st.Code())
}

func (s *FrontendTestSuite) TestCreateTicket() {
	t := &pb.Ticket{}
	s.customlogicManager.EXPECT().Validate(ctx, t)
	s.customlogicManager.EXPECT().GatherData(ctx, t).Return(t, nil)
	s.ticketsManager.EXPECT().CreateTicket(ctx, t, s.filterManager).Return(t, nil)
	resp, err := s.service.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: t})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), t, resp)
}

func (s *FrontendTestSuite) TestDeleteTicket() {
	ticketID := xid.New().String()
	{
		// delete without error
		s.ticketsManager.EXPECT().DeleteTicket(ctx, ticketID)
		_, err := s.service.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticketID})
		assert.NoError(s.T(), err)
	}
	{
		// fail to delete
		s.ticketsManager.EXPECT().DeleteTicket(ctx, ticketID).Return(fmt.Errorf("fail"))
		_, err := s.service.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticketID})
		assert.Error(s.T(), err)
	}
}

func (s *FrontendTestSuite) TestGetTicket() {
	t := &pb.Ticket{Id: xid.New().String()}
	s.ticketsManager.EXPECT().GetTicket(ctx, t.Id).Return(t, nil)
	resp, err := s.service.GetTicket(ctx, &pb.GetTicketRequest{TicketId: t.Id})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), t, resp)
}

func (s *FrontendTestSuite) TestWatchAssignmentsAlreadyAssigned() {
	t := &pb.Ticket{Id: xid.New().String(), Assignment: &pb.Assignment{Connection: "foo"}}
	svr := mock_pb.NewMockFrontendService_WatchAssignmentsServer(gomock.NewController(s.T()))
	svr.EXPECT().Context().Return(ctx).AnyTimes()
	s.ticketsManager.EXPECT().GetTicket(ctx, t.Id).Return(t, nil)
	svr.EXPECT().Send(gomock.Any())
	err := s.service.WatchAssignments(&pb.WatchAssignmentsRequest{TicketId: t.Id}, svr)
	require.NoError(s.T(), err)
}

func (s *FrontendTestSuite) TestWatchAssignmentsNotAssigned() {
	t := &pb.Ticket{Id: xid.New().String()}
	svr := mock_pb.NewMockFrontendService_WatchAssignmentsServer(gomock.NewController(s.T()))
	svr.EXPECT().Context().Return(ctx).AnyTimes()
	s.ticketsManager.EXPECT().GetTicket(ctx, t.Id).Return(t, nil)
	s.ticketsManager.EXPECT().WatchTicket(gomock.Any(), t.Id, gomock.Any())
	err := s.service.WatchAssignments(&pb.WatchAssignmentsRequest{TicketId: t.Id}, svr)
	require.NoError(s.T(), err)
}

func (s *FrontendTestSuite) TestAcknowledgeBackfill() {
	backfillObject := &pb.Backfill{Id: xid.New().String()}
	assignment := &pb.Assignment{Connection: "foo"}
	s.backfillManager.EXPECT().AcknowledgeBackfill(
		ctx, backfillObject.Id, assignment, s.filterManager).Return(backfillObject, nil)
	b, err := s.service.AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{
		BackfillId: backfillObject.Id,
		Assignment: assignment,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), backfillObject, b)
}

func (s *FrontendTestSuite) TestCreateBackfill() {
	backfillObject := &pb.Backfill{Id: xid.New().String()}
	s.backfillManager.EXPECT().CreateBackfill(ctx, backfillObject, s.filterManager).Return(backfillObject, nil)
	b, err := s.service.CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: backfillObject})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), backfillObject, b)
}

func (s *FrontendTestSuite) TestDeleteBackfill() {
	id := xid.New().String()
	{
		s.backfillManager.EXPECT().DeleteBackfill(ctx, id, s.filterManager)
		_, err := s.service.DeleteBackfill(ctx, &pb.DeleteBackfillRequest{BackfillId: id})
		require.NoError(s.T(), err)
	}
	{
		s.backfillManager.EXPECT().DeleteBackfill(ctx, id, s.filterManager).Return(fmt.Errorf("fail"))
		_, err := s.service.DeleteBackfill(ctx, &pb.DeleteBackfillRequest{BackfillId: id})
		require.Error(s.T(), err)
	}
}

func (s *FrontendTestSuite) TestGetBackfill() {
	backfillObject := &pb.Backfill{Id: xid.New().String()}
	s.backfillManager.EXPECT().GetBackfill(ctx, backfillObject.Id).Return(backfillObject, nil)
	b, err := s.service.GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: backfillObject.Id})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), backfillObject, b)
}

func (s *FrontendTestSuite) TestUpdateBackfill() {
	backfillObject := &pb.Backfill{Id: xid.New().String()}
	s.backfillManager.EXPECT().UpdateBackfill(ctx, backfillObject, s.filterManager).Return(backfillObject, nil)
	b, err := s.service.UpdateBackfill(ctx, &pb.UpdateBackfillRequest{Backfill: backfillObject})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), backfillObject, b)
}

func TestFrontendTestSuite(t *testing.T) {
	suite.Run(t, new(FrontendTestSuite))
}
