package tickets_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	kvMock "github.com/rivalry-matchmaker/rivalry/internal/db/kv/mock"
	pubsubMock "github.com/rivalry-matchmaker/rivalry/internal/db/pubsub/mock"
	streamMock "github.com/rivalry-matchmaker/rivalry/internal/db/stream/mock"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	ctx     = context.Background()
	profile = &pb.MatchProfile{Name: "everybody", Pools: []*pb.Pool{{Name: "pool_everybody"}}}
)

type TicketManagerTestSuite struct {
	suite.Suite
	manager       tickets.Manager
	store         *kvMock.MockStore
	streamClient  *streamMock.MockClient
	pubsubClient  *pubsubMock.MockClient
	filterManager filter.Manager
}

func (s *TicketManagerTestSuite) SetupTest() {
	s.store = kvMock.NewMockStore(gomock.NewController(s.T()))
	s.streamClient = streamMock.NewMockClient(gomock.NewController(s.T()))
	s.pubsubClient = pubsubMock.NewMockClient(gomock.NewController(s.T()))
	s.manager = tickets.NewManager(s.store, s.streamClient, s.pubsubClient)
	s.filterManager = filter.NewManager([]*pb.MatchProfile{profile})
}

func (s *TicketManagerTestSuite) TearDownTest() {
	s.store.EXPECT().Close()
	s.streamClient.EXPECT().Close()
	s.pubsubClient.EXPECT().Close()
	s.manager.Close()
}

func (s *TicketManagerTestSuite) TestCreateTicket() {
	tIn := &pb.Ticket{}
	s.store.EXPECT().SetNX(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	s.streamClient.EXPECT().SendMessage(tickets.GetMatchRequestTopic(profile.Name), gomock.Any())
	tOut, err := s.manager.CreateTicket(ctx, tIn, s.filterManager)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), tOut.Id)
}

func (s *TicketManagerTestSuite) TestCreateTicketFailSendCausesDelete() {
	tIn := &pb.Ticket{}
	s.store.EXPECT().SetNX(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	s.streamClient.EXPECT().SendMessage(tickets.GetMatchRequestTopic(profile.Name), gomock.Any()).Return(backoff.Permanent(fmt.Errorf("fail")))
	s.store.EXPECT().Del(ctx, gomock.Any(), gomock.Any())
	_, err := s.manager.CreateTicket(ctx, tIn, s.filterManager)
	require.Error(s.T(), err)
}

func (s *TicketManagerTestSuite) TestDeleteTicket() {
	ticketID := xid.New().String()
	s.store.EXPECT().Del(ctx, gomock.Any(), ticketID)
	assert.NoError(s.T(), s.manager.DeleteTicket(ctx, ticketID))
}

func (s *TicketManagerTestSuite) TestGetTicket() {
	ticketID := xid.New().String()
	t := &pb.Ticket{Id: ticketID}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.store.EXPECT().Get(ctx, gomock.Any(), ticketID).Return(tBytes, nil)
	_, err = s.manager.GetTicket(ctx, ticketID)
	require.NoError(s.T(), err)
}

func (s *TicketManagerTestSuite) TestGetTickets() {
	ticketID := xid.New().String()
	t := &pb.Ticket{Id: ticketID}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.store.EXPECT().MGet(ctx, gomock.Any(), []string{ticketID}).Return(map[string][]byte{
		ticketID: tBytes,
	}, nil)
	ts, err := s.manager.GetTickets(ctx, []string{ticketID})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(ts))
	assert.Equal(s.T(), ts[0].Id, t.Id)
}

func (s *TicketManagerTestSuite) TestWatchTicket() {
	ticketID := xid.New().String()
	s.pubsubClient.EXPECT().Subscribe(ctx, tickets.GetTicketSubject(ticketID), gomock.Any()).Do(
		func(ctx context.Context, topic string, f func([]byte)) error {
			f([]byte{})
			return nil
		})
	t := &pb.Ticket{Id: ticketID}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.store.EXPECT().Get(ctx, gomock.Any(), ticketID).Return(tBytes, nil)
	err = s.manager.WatchTicket(ctx, ticketID, func(ctx context.Context, t *pb.Ticket) {
		assert.Equal(s.T(), ticketID, t.Id)
	})
	assert.NoError(s.T(), err)
}

func (s *TicketManagerTestSuite) TestStreamTickets() {
	st := &pb.StreamTicket{
		TicketId: xid.New().String(),
		Profile:  "profile",
		Pool:     "pool",
	}
	stData, err := proto.Marshal(st)
	t := &pb.Ticket{Id: st.TicketId}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.streamClient.EXPECT().Subscribe(tickets.GetMatchRequestTopic(profile.Name), gomock.Any()).Do(
		func(topic string, f func([]byte)) error {
			f(stData)
			return nil
		})
	s.store.EXPECT().Get(ctx, gomock.Any(), st.TicketId).Return(tBytes, nil)
	err = s.manager.StreamTickets(ctx, profile.Name, func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket) {
		assert.Equal(s.T(), st.TicketId, t.Id)
	})
	assert.NoError(s.T(), err)
}

func (s *TicketManagerTestSuite) TestAssignTicketsToMatch() {
	// test assigning match with no tickets
	success, err := s.manager.AssignTicketsToMatch(ctx, &pb.Match{})
	assert.NoError(s.T(), err)
	assert.True(s.T(), success)

	// test assigning match with tickets
	s.store.EXPECT().MSetNX(ctx, gomock.Any(), gomock.Any()).Return(true, nil)
	success, err = s.manager.AssignTicketsToMatch(ctx, &pb.Match{Tickets: []*pb.Ticket{
		{Id: xid.New().String()}}})
	assert.NoError(s.T(), err)
	assert.True(s.T(), success)
}

func (s *TicketManagerTestSuite) TestReleaseTicketsFromMatch() {
	// test releasing match with no tickets
	err := s.manager.ReleaseTicketsFromMatch(ctx, &pb.Match{})
	assert.NoError(s.T(), err)

	// test releasing match with tickets
	s.store.EXPECT().Del(ctx, gomock.Any(), gomock.Any()).Return(nil)
	err = s.manager.ReleaseTicketsFromMatch(ctx, &pb.Match{Tickets: []*pb.Ticket{
		{Id: xid.New().String()}}})
	assert.NoError(s.T(), err)
}

func (s *TicketManagerTestSuite) TestAddAssignmentToTickets() {
	a := &pb.Assignment{Connection: xid.New().String()}
	t := &pb.Ticket{Id: xid.New().String()}
	s.store.EXPECT().MSet(ctx, gomock.Any(), gomock.Any())
	s.pubsubClient.EXPECT().Publish(ctx, tickets.GetTicketSubject(t.Id), gomock.Any())
	assert.NoError(s.T(), s.manager.AddAssignmentToTickets(ctx, a, []*pb.Ticket{t}))
}

func (s *TicketManagerTestSuite) TestRequeueTickets() {
	ticketID := xid.New().String()
	t1 := &pb.Ticket{Id: ticketID}
	allocatedTicketID := xid.New().String()
	t2 := &pb.Ticket{Id: allocatedTicketID}
	t2Bytes, err := proto.Marshal(t2)
	require.NoError(s.T(), err)
	ts := []*pb.Ticket{t1, t2}

	s.store.EXPECT().MGet(ctx, gomock.Any(), gomock.Any()).Return(map[string][]byte{
		allocatedTicketID: t2Bytes,
	}, nil)
	s.streamClient.EXPECT().SendMessage(tickets.GetMatchRequestTopic(profile.Name), gomock.Any())
	s.manager.RequeueTickets(ctx, ts, s.filterManager)
}

func TestTicketManagerTestSuite(t *testing.T) {
	suite.Run(t, new(TicketManagerTestSuite))
}
