package backfill_test

import (
	"context"
	"fmt"
	"testing"

	kvMock "github.com/rivalry-matchmaker/rivalry/internal/db/kv/mock"
	dlmMock "github.com/rivalry-matchmaker/rivalry/internal/dlm/mock"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/backfill"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	ticketsMock "github.com/rivalry-matchmaker/rivalry/internal/managers/tickets/mock"
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

type ExampleTestSuite struct {
	suite.Suite
	manager       backfill.Manager
	store         *kvMock.MockStore
	sortedSet     *kvMock.MockSortedSet
	set           *kvMock.MockSet
	dlm           *dlmMock.MockDLM
	ticketManager *ticketsMock.MockManager
	filterManager filter.Manager
}

func (s *ExampleTestSuite) SetupTest() {
	s.store = kvMock.NewMockStore(gomock.NewController(s.T()))
	s.sortedSet = kvMock.NewMockSortedSet(gomock.NewController(s.T()))
	s.set = kvMock.NewMockSet(gomock.NewController(s.T()))
	s.dlm = dlmMock.NewMockDLM(gomock.NewController(s.T()))
	s.ticketManager = ticketsMock.NewMockManager(gomock.NewController(s.T()))
	s.filterManager = filter.NewManager([]*pb.MatchProfile{profile})
	s.manager = backfill.NewManager(s.store, s.sortedSet, s.set, s.dlm, s.ticketManager, 1)
}

func (s *ExampleTestSuite) TestCreateBackfill() {
	s.store.EXPECT().SetNX(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	s.sortedSet.EXPECT().ZAdd(ctx, gomock.Any(), gomock.Any())
	s.set.EXPECT().SAdd(ctx, gomock.Any(), gomock.Any())
	b, err := s.manager.CreateBackfill(ctx, &pb.Backfill{}, s.filterManager)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), b.Id)
}

func (s *ExampleTestSuite) TestCreateBackfillFailAlreadyExists() {
	s.store.EXPECT().SetNX(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
	_, err := s.manager.CreateBackfill(ctx, &pb.Backfill{}, s.filterManager)
	require.Error(s.T(), err)
}

func (s *ExampleTestSuite) TestGetAvailableBackfill() {
	max := int64(100)
	backfillID := xid.New().String()
	bids := []string{backfillID}
	b := &pb.Backfill{Id: backfillID}
	bBytes, err := proto.Marshal(b)
	require.NoError(s.T(), err)
	s.set.EXPECT().SPopN(ctx, gomock.Any(), max).Return(bids, nil)
	s.store.EXPECT().MGet(ctx, gomock.Any(), bids).Return(map[string][]byte{
		backfillID: bBytes,
	}, nil)
	bs, err := s.manager.GetAvailableBackfill(ctx, profile.Name, max)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, len(bs))
	assert.Equal(s.T(), backfillID, bs[0].Id)
}

func (s *ExampleTestSuite) TestGetAvailableBackfillFailAfterPop() {
	max := int64(100)
	backfillID := xid.New().String()
	bids := []string{backfillID}
	s.set.EXPECT().SPopN(ctx, gomock.Any(), max).Return(bids, nil)
	s.store.EXPECT().MGet(ctx, gomock.Any(), bids).Return(nil, fmt.Errorf("fail"))
	s.set.EXPECT().SAdd(ctx, gomock.Any(), backfillID)
	_, err := s.manager.GetAvailableBackfill(ctx, profile.Name, max)
	require.Error(s.T(), err)
}

func (s *ExampleTestSuite) TestUpdateBackfill() {
	backfillID := xid.New().String()
	ticketID := xid.New().String()
	b := &pb.Backfill{Id: backfillID}
	s.dlm.EXPECT().Lock(backfillID, gomock.Any())
	s.dlm.EXPECT().Unlock(backfillID)

	// requeuePendingTickets
	s.set.EXPECT().SMembers(ctx, gomock.Any()).Return([]string{ticketID}, nil)
	s.set.EXPECT().SRem(ctx, gomock.Any(), ticketID)
	s.ticketManager.EXPECT().GetTickets(ctx, gomock.Any()).Return([]*pb.Ticket{{Id: ticketID}}, nil)
	s.ticketManager.EXPECT().RequeueTickets(ctx, gomock.Any(), s.filterManager)

	s.store.EXPECT().Set(ctx, gomock.Any(), backfillID, gomock.Any(), gomock.Any())
	s.sortedSet.EXPECT().ZAdd(ctx, gomock.Any(), gomock.Any())
	bOut, err := s.manager.UpdateBackfill(ctx, b, s.filterManager)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), b.Id, bOut.Id)
}

func (s *ExampleTestSuite) TestMakeMatchWithNewBackfill() {
	ticketID := xid.New().String()
	s.store.EXPECT().SetNX(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	s.sortedSet.EXPECT().ZAdd(ctx, gomock.Any(), gomock.Any())
	s.set.EXPECT().SAdd(ctx, gomock.Any(), gomock.Any())
	s.set.EXPECT().SAdd(ctx, gomock.Any(), ticketID)
	err := s.manager.MakeMatchWithBackfill(ctx, &pb.Backfill{}, []string{ticketID}, s.filterManager)
	require.NoError(s.T(), err)
}

func (s *ExampleTestSuite) TestMakeMatchWithExistingBackfill() {
	ticketID := xid.New().String()
	backfillID := xid.New().String()
	s.dlm.EXPECT().Lock(backfillID, gomock.Any())
	s.dlm.EXPECT().Unlock(backfillID)
	bBytes, err := proto.Marshal(&pb.Backfill{Id: backfillID})
	require.NoError(s.T(), err)
	s.store.EXPECT().Get(ctx, gomock.Any(), backfillID).Return(bBytes, nil)
	s.store.EXPECT().Set(ctx, gomock.Any(), backfillID, gomock.Any(), gomock.Any())
	s.set.EXPECT().SAdd(ctx, gomock.Any(), ticketID)
	err = s.manager.MakeMatchWithBackfill(ctx, &pb.Backfill{Id: backfillID}, []string{ticketID}, s.filterManager)
	require.NoError(s.T(), err)
}

func (s *ExampleTestSuite) TestAcknowledgeBackfill() {
	backfillID := xid.New().String()
	ticketID := xid.New().String()
	assignment := &pb.Assignment{Connection: "test"}
	// lock and unlock this backfill
	s.dlm.EXPECT().Lock(backfillID, gomock.Any())
	s.dlm.EXPECT().Unlock(backfillID)
	// Get tickets pending acknowledgement
	s.set.EXPECT().SMembers(ctx, gomock.Any()).Return([]string{ticketID}, nil)
	s.ticketManager.EXPECT().GetTickets(ctx, gomock.Any()).Return([]*pb.Ticket{{Id: ticketID}}, nil)
	// Read Backfill from DB
	bBytes, err := proto.Marshal(&pb.Backfill{Id: backfillID})
	require.NoError(s.T(), err)
	s.store.EXPECT().Get(ctx, gomock.Any(), backfillID).Return(bBytes, nil)
	// Update backfill with new data
	s.store.EXPECT().Set(ctx, gomock.Any(), backfillID, gomock.Any(), gomock.Any())
	// remove ticket from pending set
	s.set.EXPECT().SRem(ctx, gomock.Any(), ticketID)
	// add back fill to available set
	s.set.EXPECT().SAdd(ctx, gomock.Any(), backfillID)
	// update all the tickets with the assignment
	s.ticketManager.EXPECT().AddAssignmentToTickets(ctx, assignment, gomock.Any())

	b, err := s.manager.AcknowledgeBackfill(ctx, backfillID, assignment, s.filterManager)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), backfillID, b.Id)
}

func (s *ExampleTestSuite) TestDeleteBackfill() {
	backfillID := xid.New().String()
	// lock and unlock this backfill
	s.dlm.EXPECT().Lock(backfillID, gomock.Any())
	s.dlm.EXPECT().Unlock(backfillID)
	// remove backfill from the last ack times
	s.sortedSet.EXPECT().ZRem(ctx, gomock.Any(), backfillID)
	// delete backfill from db
	s.store.EXPECT().Del(ctx, gomock.Any(), backfillID)
	// requeuePendingTickets
	s.set.EXPECT().SMembers(ctx, gomock.Any()).Return([]string{}, nil)

	assert.NoError(s.T(), s.manager.DeleteBackfill(ctx, backfillID, s.filterManager))
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleTestSuite))
}
