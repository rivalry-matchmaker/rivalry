package tickets_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	kvMock "github.com/rivalry-matchmaker/rivalry/internal/db/kv/mock"
	pubsubMock "github.com/rivalry-matchmaker/rivalry/internal/db/pubsub/mock"
	streamMock "github.com/rivalry-matchmaker/rivalry/internal/db/stream/mock"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	db "github.com/rivalry-matchmaker/rivalry/pkg/pb/db/v1"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = context.Background()
)

type MatchRequestManagerTestSuite struct {
	suite.Suite
	manager      tickets.Manager
	store        *kvMock.MockStore
	streamClient *streamMock.MockClient
	pubsubClient *pubsubMock.MockClient
}

func (s *MatchRequestManagerTestSuite) SetupTest() {
	s.store = kvMock.NewMockStore(gomock.NewController(s.T()))
	s.streamClient = streamMock.NewMockClient(gomock.NewController(s.T()))
	s.pubsubClient = pubsubMock.NewMockClient(gomock.NewController(s.T()))
	s.manager = tickets.NewManager(s.store, s.streamClient, s.pubsubClient)
}

func (s *MatchRequestManagerTestSuite) TearDownTest() {
	s.store.EXPECT().Close()
	s.streamClient.EXPECT().Close()
	s.pubsubClient.EXPECT().Close()
	s.manager.Close()
}

func (s *MatchRequestManagerTestSuite) TestCreateMatchRequest() {
	tIn := &api.MatchRequest{
		Id:               xid.New().String(),
		MatchmakingQueue: "everybody",
	}
	s.store.EXPECT().SetNX(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	s.streamClient.EXPECT().SendMessage(tickets.GetMatchRequestTopic(tIn.MatchmakingQueue), gomock.Any())
	err := s.manager.CreateMatchRequest(ctx, tIn)
	require.NoError(s.T(), err)
}

func (s *MatchRequestManagerTestSuite) TestCreateMatchRequestFailSendCausesDelete() {
	tIn := &api.MatchRequest{
		Id:               xid.New().String(),
		MatchmakingQueue: "everybody",
	}
	s.store.EXPECT().SetNX(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	s.streamClient.EXPECT().SendMessage(tickets.GetMatchRequestTopic(tIn.MatchmakingQueue), gomock.Any()).Return(backoff.Permanent(fmt.Errorf("fail")))
	s.store.EXPECT().Del(ctx, gomock.Any(), gomock.Any())
	err := s.manager.CreateMatchRequest(ctx, tIn)
	require.Error(s.T(), err)
}

func (s *MatchRequestManagerTestSuite) TestDeleteMatchRequest() {
	ticketID := xid.New().String()
	s.store.EXPECT().Del(ctx, gomock.Any(), ticketID)
	assert.NoError(s.T(), s.manager.DeleteMatchRequest(ctx, ticketID))
}

func (s *MatchRequestManagerTestSuite) TestGetMatchRequest() {
	ticketID := xid.New().String()
	t := &db.MatchRequest{Id: ticketID, MatchmakingQueue: "everybody"}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.store.EXPECT().Get(ctx, gomock.Any(), ticketID).Return(tBytes, nil)
	_, err = s.manager.GetMatchRequest(ctx, ticketID)
	require.NoError(s.T(), err)
}

func (s *MatchRequestManagerTestSuite) TestGetMatchRequests() {
	ticketID := xid.New().String()
	t := &db.MatchRequest{Id: ticketID, MatchmakingQueue: "everybody"}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.store.EXPECT().MGet(ctx, gomock.Any(), []string{ticketID}).Return(map[string][]byte{
		ticketID: tBytes,
	}, nil)
	ts, err := s.manager.GetMatchRequests(ctx, []string{ticketID})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(ts))
	assert.Equal(s.T(), ts[0].Id, t.Id)
}

func (s *MatchRequestManagerTestSuite) TestWatchMatchRequest() {
	ticketID := xid.New().String()
	s.pubsubClient.EXPECT().Subscribe(ctx, tickets.GetMatchRequestSubject(ticketID), gomock.Any()).Do(
		func(ctx context.Context, topic string, f func([]byte)) error {
			f([]byte{})
			return nil
		})
	t := &api.MatchRequest{Id: ticketID, MatchmakingQueue: "everybody"}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.store.EXPECT().Get(ctx, gomock.Any(), ticketID).Return(tBytes, nil)
	err = s.manager.WatchMatchRequest(ctx, ticketID, func(ctx context.Context, t *db.MatchRequest) {
		assert.Equal(s.T(), ticketID, t.Id)
	})
	assert.NoError(s.T(), err)
}

func (s *MatchRequestManagerTestSuite) TestStreamMatchRequests() {
	st := &api.StreamTicket{
		MatchRequestId:   xid.New().String(),
		MatchmakingQueue: "everybody",
		NumberOfPlayers:  1,
	}
	stData, err := proto.Marshal(st)
	t := &api.MatchRequest{Id: st.MatchRequestId}
	tBytes, err := proto.Marshal(t)
	require.NoError(s.T(), err)
	s.streamClient.EXPECT().Subscribe(tickets.GetMatchRequestTopic(st.MatchmakingQueue), gomock.Any()).Do(
		func(topic string, f func([]byte)) error {
			f(stData)
			return nil
		})
	s.store.EXPECT().Get(ctx, gomock.Any(), st.MatchRequestId).Return(tBytes, nil)
	err = s.manager.StreamMatchRequests(ctx, st.MatchmakingQueue, func(ctx context.Context, st *api.StreamTicket, t *db.MatchRequest) {
		assert.Equal(s.T(), st.MatchRequestId, t.Id)
	})
	assert.NoError(s.T(), err)
}

func (s *MatchRequestManagerTestSuite) TestAssignMatchRequestsToMatch() {
	// test assigning match with no tickets
	success, err := s.manager.AssignMatchRequestsToMatch(ctx, &api.Match{})
	assert.NoError(s.T(), err)
	assert.True(s.T(), success)

	// test assigning match with tickets
	s.store.EXPECT().MSetNX(ctx, gomock.Any(), gomock.Any()).Return(true, nil)
	success, err = s.manager.AssignMatchRequestsToMatch(ctx, &api.Match{MatchRequestIds: []string{
		xid.New().String()}})
	assert.NoError(s.T(), err)
	assert.True(s.T(), success)
}

func (s *MatchRequestManagerTestSuite) TestReleaseMatchRequestsFromMatch() {
	// test releasing match with no tickets
	err := s.manager.ReleaseMatchRequestsFromMatch(ctx, &api.Match{})
	assert.NoError(s.T(), err)

	// test releasing match with tickets
	s.store.EXPECT().Del(ctx, gomock.Any(), gomock.Any()).Return(nil)
	err = s.manager.ReleaseMatchRequestsFromMatch(ctx, &api.Match{MatchRequestIds: []string{
		xid.New().String()}})
	assert.NoError(s.T(), err)
}

func (s *MatchRequestManagerTestSuite) TestAddAssignmentToMatchRequests() {
	a := &api.GameServer{}
	t := &api.MatchRequest{Id: xid.New().String(), MatchmakingQueue: "everybody"}
	s.store.EXPECT().MSet(ctx, gomock.Any(), gomock.Any())
	s.store.EXPECT().MGet(ctx, gomock.Any(), gomock.Any())
	s.pubsubClient.EXPECT().Publish(ctx, tickets.GetMatchRequestSubject(t.Id), gomock.Any())
	assert.NoError(s.T(), s.manager.AddAssignmentToMatchRequests(ctx, a, &api.Match{MatchRequestIds: []string{t.Id}}))
}

func (s *MatchRequestManagerTestSuite) TestRequeueMatchRequests() {
	ticketID := xid.New().String()
	t1 := &api.MatchRequest{Id: ticketID, MatchmakingQueue: "everybody"}
	allocatedMatchRequestID := xid.New().String()
	t2 := &api.MatchRequest{Id: allocatedMatchRequestID, MatchmakingQueue: "everybody"}
	t2Bytes, err := proto.Marshal(t2)
	require.NoError(s.T(), err)
	ts := []*api.StreamTicket{
		{
			MatchRequestId:   t1.Id,
			MatchmakingQueue: t1.MatchmakingQueue,
			NumberOfPlayers:  1,
		},
		{
			MatchRequestId:   t2.Id,
			MatchmakingQueue: t2.MatchmakingQueue,
			NumberOfPlayers:  1,
		},
	}

	s.store.EXPECT().MGet(ctx, gomock.Any(), gomock.Any()).Return(map[string][]byte{
		allocatedMatchRequestID: t2Bytes,
	}, nil)
	s.streamClient.EXPECT().SendMessage(tickets.GetMatchRequestTopic(t1.MatchmakingQueue), gomock.Any())
	s.manager.RequeueMatchRequests(ctx, ts)
}

func TestMatchRequestManagerTestSuite(t *testing.T) {
	suite.Run(t, new(MatchRequestManagerTestSuite))
}
