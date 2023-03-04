package tickets

//go:generate mockgen -package mock -destination=mock/tickets.go . Manager

import (
	"context"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	db "github.com/rivalry-matchmaker/rivalry/pkg/pb/db/v1"
	stream_pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/stream/v1"
	"github.com/rs/zerolog/log"
)

const (
	collectionTickets                  = "tickets"
	collectionTicketAssignments        = "ticket_assignments"
	topicMatchRequestPrefix            = "match_request_"
	topicAccumulatedMatchRequestPrefix = "accumulated_match_request_"
	subjectTicketUpdatesPrefix         = "ticket_updates_"
)

// ErrDuplicateTicketID is an error returned when the ticket id given when creating a new ticket already exists
var ErrDuplicateTicketID = errors.New("duplicate ticket id")

// GetMatchRequestTopic returns a topic name based on the Match Profile a Ticket aligns with
func GetMatchRequestTopic(queueName string) string {
	return topicMatchRequestPrefix + queueName
}

// GetAccumulatedMatchRequestTopic returns a topic name based on the Match Profile a Ticket aligns with
func GetAccumulatedMatchRequestTopic(queueName string) string {
	return topicAccumulatedMatchRequestPrefix + queueName
}

// GetMatchRequestSubject returns a subject to be used with ticket pub-sub
func GetMatchRequestSubject(ticketID string) string {
	return subjectTicketUpdatesPrefix + ticketID
}

// Manager interface specifies our interactions with tickets
type Manager interface {
	CreateMatchRequest(ctx context.Context, ticket *api.MatchRequest) error
	DeleteMatchRequest(ctx context.Context, ticketID string) error
	GetMatchRequest(ctx context.Context, ticketID string) (*db.MatchRequest, error)
	GetMatchRequests(ctx context.Context, ticketIDs []string) ([]*db.MatchRequest, error)
	WatchMatchRequest(ctx context.Context, ticketID string, f func(ctx context.Context, t *db.MatchRequest)) error
	StreamMatchRequests(ctx context.Context, queue string, f func(ctx context.Context, st *stream_pb.StreamTicket)) error
	PublishAccumulatedMatchRequests(ctx context.Context, queue string, sts []*stream_pb.StreamTicket) error
	StreamAccumulatedMatchRequests(ctx context.Context, queue string, f func(ctx context.Context, sts []*stream_pb.StreamTicket)) error
	AssignMatchRequestsToMatch(ctx context.Context, match *api.Match) (bool, error)
	ReleaseMatchRequestsFromMatch(ctx context.Context, match *api.Match) error
	AddAssignmentToMatchRequests(ctx context.Context, assignment *api.GameServer, match *api.Match) error
	RequeueMatchRequests(ctx context.Context, tickets []*stream_pb.StreamTicket)
	Close()
}

type manager struct {
	kvStore      kv.Store
	streamClient stream.Client
	pubsubClient pubsub.Client
}

// NewManager returns a new ticket Manager
func NewManager(kvStore kv.Store, streamClient stream.Client, pubsubClient pubsub.Client) Manager {
	return &manager{
		kvStore:      kvStore,
		streamClient: streamClient,
		pubsubClient: pubsubClient,
	}
}

func (m *manager) publishStreamTicket(ctx context.Context, st *stream_pb.StreamTicket) error {
	return backoff.Retry(
		ctx,
		func() error {
			streamTicketBytes, err := proto.Marshal(st)
			if err != nil {
				return errors.Wrap(err, "failed to marshal stream ticket")
			}
			err = m.streamClient.SendMessage(GetMatchRequestTopic(st.MatchmakingQueue), streamTicketBytes)
			if err != nil {
				return errors.Wrap(err, "failed to add ticket to stream")
			}
			return nil
		},
		backoff.Exponential(),
	)
}

func StreamTicketFromDBRequest(request *db.MatchRequest) *stream_pb.StreamTicket {
	return &stream_pb.StreamTicket{
		MatchmakingQueue: request.MatchmakingQueue,
		MatchRequestId:   request.Id,
		NumberOfPlayers:  int32(len(request.Members)),
	}
}

func SubmissionFromDBRequest(request *db.MatchRequest) *api.Submission {
	var (
		rtts = make([]*api.RTT, len(request.Rtts))
		md   = new(api.MatchRequestData)
	)

	for j, rtt := range request.Rtts {
		rtts[j] = &api.RTT{
			Host:         rtt.Host,
			Milliseconds: rtt.Milliseconds,
		}
	}

	if request.MatchRequestData != nil {
		md.Strings = request.MatchRequestData.Strings
		md.Doubles = request.MatchRequestData.Doubles
		md.Tags = request.MatchRequestData.Tags
		md.ExtraData = request.MatchRequestData.ExtraData
	}

	return &api.Submission{
		Id:               request.Id,
		Members:          request.Members,
		MatchRequestData: md,
		MatchmakingQueue: request.MatchmakingQueue,
		Rtts:             rtts,
		CreateTime:       request.CreateTime,
	}
}

// CreateMatchRequest creates a new ticket
func (m *manager) CreateMatchRequest(ctx context.Context, req *api.MatchRequest) error {
	// generate a req id if one doesn't exist
	if len(req.Id) == 0 {
		return errors.New("Missing ID")
	}

	members := mapset.NewSet[string]()
	if req.PartyData != nil {
		for _, pm := range req.PartyData.PartyMembers {
			members.Add(pm)
		}
	}
	members.Add(req.PlayerId)

	dbRTTs := make([]*db.RTT, len(req.Rtts))
	for i, v := range req.Rtts {
		dbRTTs[i] = &db.RTT{
			Host:         v.Host,
			Milliseconds: v.Milliseconds,
		}
	}

	if req.MatchRequestData == nil {
		req.MatchRequestData = &api.MatchRequestData{}
	}

	kvReq := &db.MatchRequest{
		Id:      req.Id,
		Members: members.ToSlice(),
		MatchRequestData: &db.MatchRequestData{
			Doubles:   req.MatchRequestData.Doubles,
			Strings:   req.MatchRequestData.Strings,
			Tags:      req.MatchRequestData.Tags,
			ExtraData: req.MatchRequestData.ExtraData,
		},
		MatchmakingQueue: req.MatchmakingQueue,
		Rtts:             dbRTTs,
		CreateTime:       time.Now().UTC().Unix(),
	}

	streamTicket := StreamTicketFromDBRequest(kvReq)

	// marshal req into bytes
	reqBytes, err := proto.Marshal(kvReq)
	if err != nil {
		return errors.Wrap(err, "failed to marshal req")
	}

	// persist the req
	set, err := m.kvStore.SetNX(ctx, collectionTickets, req.Id, reqBytes, 0)
	if err != nil {
		return errors.Wrap(err, "failed to persist req")
	}
	if !set {
		return ErrDuplicateTicketID
	}
	// if anything from here onwards fails we want to remove the req from the database
	defer func() {
		if err != nil {
			_ = m.kvStore.Del(ctx, collectionTickets, req.Id)
		}
	}()

	// stream req details
	err = m.publishStreamTicket(ctx, streamTicket)
	if err != nil {
		return err
	}
	return nil
}

// DeleteMatchRequest deletes a ticket
func (m *manager) DeleteMatchRequest(ctx context.Context, ticketID string) error {
	return m.kvStore.Del(ctx, collectionTickets, ticketID)
}

// GetMatchRequest returns a Match Request
func (m *manager) GetMatchRequest(ctx context.Context, ticketID string) (*db.MatchRequest, error) {
	ticketBytes, err := m.kvStore.Get(ctx, collectionTickets, ticketID)
	if err != nil {
		return nil, err
	}
	t := new(db.MatchRequest)
	err = proto.Unmarshal(ticketBytes, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetMatchRequests returns a list of Match Requests
func (m *manager) GetMatchRequests(ctx context.Context, ticketIDs []string) ([]*db.MatchRequest, error) {
	data, err := m.kvStore.MGet(ctx, collectionTickets, ticketIDs)
	if err != nil {
		return nil, err
	}
	tickets := make([]*db.MatchRequest, 0, len(data))
	for _, v := range data {
		t := new(db.MatchRequest)
		err = proto.Unmarshal(v, t)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// StreamMatchRequests listens to a stream of tickets that have been submitted and match a given queue
func (m *manager) StreamMatchRequests(ctx context.Context, queue string, f func(ctx context.Context, st *stream_pb.StreamTicket)) error {
	return m.streamClient.Subscribe(
		GetMatchRequestTopic(queue),
		func(b []byte) {
			st := new(stream_pb.StreamTicket)
			err := proto.Unmarshal(b, st)
			if err != nil {
				log.Err(err).Msg("unable to unmarshal ticket")
				return
			}
			f(ctx, st)
		})
}

func (m *manager) PublishAccumulatedMatchRequests(ctx context.Context, queue string, sts []*stream_pb.StreamTicket) error {
	return backoff.Retry(
		ctx,
		func() error {
			streamTicketBytes, err := proto.Marshal(&stream_pb.AccumulatedStreamTicket{StreamTickets: sts})
			if err != nil {
				return errors.Wrap(err, "failed to marshal stream ticket")
			}
			err = m.streamClient.SendMessage(GetAccumulatedMatchRequestTopic(queue), streamTicketBytes)
			if err != nil {
				return errors.Wrap(err, "failed to add ticket to stream")
			}
			return nil
		},
		backoff.Exponential(),
	)
}

func (m *manager) StreamAccumulatedMatchRequests(ctx context.Context, queue string, f func(ctx context.Context, sts []*stream_pb.StreamTicket)) error {
	return m.streamClient.Subscribe(
		GetAccumulatedMatchRequestTopic(queue),
		func(b []byte) {
			st := new(stream_pb.AccumulatedStreamTicket)
			err := proto.Unmarshal(b, st)
			if err != nil {
				log.Err(err).Msg("unable to unmarshal ticket")
				return
			}
			f(ctx, st.StreamTickets)
		})
}

// WatchMatchRequest listens to pubsub events for a given ticket
func (m *manager) WatchMatchRequest(ctx context.Context, ticketID string, f func(ctx context.Context, t *db.MatchRequest)) error {
	err := m.pubsubClient.Subscribe(
		ctx, GetMatchRequestSubject(ticketID),
		func(b []byte) {
			t, err := m.GetMatchRequest(ctx, ticketID)
			if err != nil {
				return
			}
			f(ctx, t)
		})
	return err
}

// AssignMatchRequestsToMatch tries to link all the ticket IDs in a match to the match's ID
func (m *manager) AssignMatchRequestsToMatch(ctx context.Context, match *api.Match) (bool, error) {
	if len(match.MatchRequestIds) == 0 {
		return true, nil
	}
	allocations := make(map[string]interface{})
	for _, t := range match.MatchRequestIds {
		allocations[t] = match.MatchId
	}
	return m.kvStore.MSetNX(ctx, collectionTicketAssignments, allocations)
}

// ReleaseMatchRequestsFromMatch removes the link between all the ticket IDs in a match to the match's ID
func (m *manager) ReleaseMatchRequestsFromMatch(ctx context.Context, match *api.Match) error {
	if len(match.MatchRequestIds) == 0 {
		return nil
	}
	var ts = make([]string, len(match.MatchRequestIds))
	for i, v := range match.MatchRequestIds {
		ts[i] = v
	}
	return m.kvStore.Del(ctx, collectionTicketAssignments, ts...)
}

// AddAssignmentToMatchRequests updates a list of tickets adding an assignment to each
func (m *manager) AddAssignmentToMatchRequests(ctx context.Context, assignment *api.GameServer, match *api.Match) error {
	if len(match.MatchRequestIds) == 0 {
		return nil
	}
	matchRequests, err := m.GetMatchRequests(ctx, match.MatchRequestIds)
	if err != nil {
		return err
	}

	gs := &db.GameServer{
		GameServerIp: assignment.GameServerIp,
	}
	for _, p := range assignment.GameServerPorts {
		gs.GameServerPorts = append(gs.GameServerPorts, &db.GameServerPort{Name: p.Name, Port: p.Port})
	}

	data := make(map[string]interface{})
	for _, mr := range matchRequests {
		mr.GameServer = gs
		reqBytes, err := proto.Marshal(mr)
		if err != nil {
			return errors.Wrap(err, "failed to marshal match request")
		}
		data[mr.Id] = reqBytes
	}

	if err := m.kvStore.MSet(ctx, collectionTickets, data); err != nil {
		return err
	}

	log.Debug().Strs("tickets", match.MatchRequestIds).Msg("successfully added assignment to tickets")

	var wg sync.WaitGroup
	for _, t := range match.MatchRequestIds {
		wg.Add(1)
		go func(matchRequestID string) {
			backoff.Retry(ctx, func() error {
				return m.pubsubClient.Publish(ctx, GetMatchRequestSubject(matchRequestID), []byte("assigned"))
			}, backoff.Exponential())
			if err != nil {
				log.Err(err).Str("match_request_id", matchRequestID).Msg("failed to publish assignment")
			}
			wg.Done()
		}(t)
	}
	wg.Wait()
	log.Debug().Strs("tickets", match.MatchRequestIds).Msg("successfully published ticket assignments")
	return nil
}

// RequeueMatchRequests resubmits match requests to the stream of submitted tickets
func (m *manager) RequeueMatchRequests(ctx context.Context, tickets []*stream_pb.StreamTicket) {
	// get ticket ids from the match
	ids := make([]string, len(tickets))
	for i, t := range tickets {
		ids[i] = t.MatchRequestId
	}
	// search for allocations
	var allocations map[string][]byte

	err := backoff.Retry(ctx, func() error {
		var err error
		allocations, err = m.kvStore.MGet(ctx, collectionTicketAssignments, ids)
		return err
	}, backoff.Exponential())
	if err != nil {
		log.Err(err).Msg("failed to read ticket assignment")
		return
	}
	// we should requeue all the keys that don't have an allocation
	requeueIDs := mapset.NewSet[string]()
	for _, k := range ids {
		v := allocations[k]
		if v == nil {
			requeueIDs.Add(k)
		}
	}
	// if we don't need to requeue any tickets then we are done
	log.Trace().Interface("allocations", allocations).Int("tickets_in", len(tickets)).Int("tickets_requeued", requeueIDs.Cardinality()).Msg("RequeueTickets")
	if requeueIDs.Cardinality() == 0 {
		return
	}
	for _, t := range tickets {
		if requeueIDs.Contains(t.MatchRequestId) {
			err = m.publishStreamTicket(ctx, t)
			if err != nil {
				log.Err(err).Msg("stream ticket failed")
				continue
			}
		}
	}
}

// Close disconnects the kv store, stream and pub-sub clients
func (m *manager) Close() {
	m.kvStore.Close()
	m.streamClient.Close()
	m.pubsubClient.Close()
}
