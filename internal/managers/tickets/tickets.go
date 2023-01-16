package tickets

//go:generate mockgen -package mock -destination=mock/tickets.go . Manager

import (
	"context"
	"sync"

	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	mapset "github.com/deckarep/golang-set"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	collectionTickets           = "tickets"
	collectionTicketAssignments = "ticket_assignments"
	topicMatchRequestPrefix     = "match_request_"
	subjectTicketUpdatesPrefix  = "ticket_updates_"
)

// ErrDuplicateTicketID is an error returned when the ticket id given when creating a new ticket already exists
var ErrDuplicateTicketID = errors.New("duplicate ticket id")

// GetMatchRequestTopic returns a topic name based on the Match Profile a Ticket aligns with
func GetMatchRequestTopic(profileName string) string {
	return topicMatchRequestPrefix + profileName
}

// GetTicketSubject returns a subject to be used with ticket pub-sub
func GetTicketSubject(ticketID string) string {
	return subjectTicketUpdatesPrefix + ticketID
}

// Manager interface specifies our interactions with tickets
type Manager interface {
	CreateTicket(ctx context.Context, ticket *pb.Ticket, filterManager filter.Manager) (*pb.Ticket, error)
	DeleteTicket(ctx context.Context, ticketID string) error
	GetTicket(ctx context.Context, ticketID string) (*pb.Ticket, error)
	GetTickets(ctx context.Context, ticketIDs []string) ([]*pb.Ticket, error)
	WatchTicket(ctx context.Context, ticketID string, f func(ctx context.Context, t *pb.Ticket)) error
	StreamTickets(ctx context.Context, profile string, f func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket)) error
	AssignTicketsToMatch(ctx context.Context, match *pb.Match) (bool, error)
	ReleaseTicketsFromMatch(ctx context.Context, match *pb.Match) error
	AddAssignmentToTickets(ctx context.Context, assignment *pb.Assignment, tickets []*pb.Ticket) error
	RequeueTickets(ctx context.Context, tickets []*pb.Ticket, filterManager filter.Manager)
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

func (m *manager) sendMessage(ctx context.Context, st *pb.StreamTicket) error {
	return backoff.Retry(
		ctx,
		func() error {
			streamTicketBytes, err := proto.Marshal(st)
			if err != nil {
				return errors.Wrap(err, "failed to marshal stream ticket")
			}
			err = m.streamClient.SendMessage(GetMatchRequestTopic(st.Profile), streamTicketBytes)
			if err != nil {
				return errors.Wrap(err, "failed to add ticket to stream")
			}
			return nil
		},
		backoff.Exponential(),
	)
}

func streamTicketFromTicket(ticket *pb.Ticket, filterManager filter.Manager) (*pb.StreamTicket, error) {
	// find the match profile this ticket belongs to
	profile, pool, err := filterManager.ProfileMembershipTest(ticket)
	if err != nil {
		return nil, err
	}
	return &pb.StreamTicket{
		Profile:  profile,
		Pool:     pool,
		TicketId: ticket.Id,
	}, nil
}

// CreateTicket creates a new ticket
func (m *manager) CreateTicket(ctx context.Context, ticket *pb.Ticket, filterManager filter.Manager) (*pb.Ticket, error) {
	// generate a ticket id if one doesn't exist
	if len(ticket.Id) == 0 {
		ticket.Id = xid.New().String()
	}
	ticket.CreateTime = timestamppb.Now()

	// filter ticket
	streamTicket, err := streamTicketFromTicket(ticket, filterManager)
	if err != nil {
		return nil, err
	}

	// marshal ticket into bytes
	ticketBytes, err := proto.Marshal(ticket)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ticket")
	}

	// persist the ticket
	set, err := m.kvStore.SetNX(ctx, collectionTickets, ticket.Id, ticketBytes, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to persist ticket")
	}
	if !set {
		return nil, ErrDuplicateTicketID
	}
	// if anything from here onwards fails we want to remove the ticket from the database
	defer func() {
		if err != nil {
			_ = m.kvStore.Del(ctx, collectionTickets, ticket.Id)
		}
	}()

	// stream ticket details
	err = m.sendMessage(ctx, streamTicket)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

// DeleteTicket deletes a ticket
func (m *manager) DeleteTicket(ctx context.Context, ticketID string) error {
	return m.kvStore.Del(ctx, collectionTickets, ticketID)
}

// GetTicket returns a ticket
func (m *manager) GetTicket(ctx context.Context, ticketID string) (*pb.Ticket, error) {
	ticketBytes, err := m.kvStore.Get(ctx, collectionTickets, ticketID)
	if err != nil {
		return nil, err
	}
	t := new(pb.Ticket)
	err = proto.Unmarshal(ticketBytes, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetTickets returns a list of tickets
func (m *manager) GetTickets(ctx context.Context, ticketIDs []string) ([]*pb.Ticket, error) {
	data, err := m.kvStore.MGet(ctx, collectionTickets, ticketIDs)
	if err != nil {
		return nil, err
	}
	tickets := make([]*pb.Ticket, 0, len(data))
	for _, v := range data {
		t := new(pb.Ticket)
		err = proto.Unmarshal(v, t)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// StreamTickets listens to a stream of tickets that have been submitted and match a given profile
func (m *manager) StreamTickets(ctx context.Context, profile string, f func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket)) error {
	return m.streamClient.Subscribe(
		GetMatchRequestTopic(profile),
		func(b []byte) {
			st := new(pb.StreamTicket)
			err := proto.Unmarshal(b, st)
			if err != nil {
				log.Err(err).Msg("unable to unmarshal ticket")
				return
			}
			t, err := m.GetTicket(ctx, st.TicketId)
			if err != nil {
				log.Err(err).Str("ticket_id", st.TicketId).Msg("unable to read ticket")
				return
			}
			f(ctx, st, t)
		})
}

// WatchTicket listens to pubsub events for a given ticket
func (m *manager) WatchTicket(ctx context.Context, ticketID string, f func(ctx context.Context, t *pb.Ticket)) error {
	err := m.pubsubClient.Subscribe(
		ctx, GetTicketSubject(ticketID),
		func(b []byte) {
			t, err := m.GetTicket(ctx, ticketID)
			if err != nil {
				return
			}
			f(ctx, t)
		})
	return err
}

// AssignTicketsToMatch tries to link all the ticket IDs in a match to the match's ID
func (m *manager) AssignTicketsToMatch(ctx context.Context, match *pb.Match) (bool, error) {
	if len(match.Tickets) == 0 {
		return true, nil
	}
	allocations := make(map[string]interface{})
	for _, t := range match.Tickets {
		allocations[t.Id] = match.MatchId
	}
	return m.kvStore.MSetNX(ctx, collectionTicketAssignments, allocations)
}

// ReleaseTicketsFromMatch removes the link between all the ticket IDs in a match to the match's ID
func (m *manager) ReleaseTicketsFromMatch(ctx context.Context, match *pb.Match) error {
	if len(match.Tickets) == 0 {
		return nil
	}
	var ts = make([]string, len(match.Tickets))
	for i, v := range match.Tickets {
		ts[i] = v.Id
	}
	return m.kvStore.Del(ctx, collectionTicketAssignments, ts...)
}

func idsFromTickets(tickets []*pb.Ticket) []string {
	ids := make([]string, len(tickets))
	for i, v := range tickets {
		ids[i] = v.Id
	}
	return ids
}

// AddAssignmentToTickets updates a list of tickets adding an assignment to each
func (m *manager) AddAssignmentToTickets(ctx context.Context, assignment *pb.Assignment, tickets []*pb.Ticket) error {
	if len(tickets) == 0 {
		return nil
	}
	data := make(map[string]interface{})
	for _, t := range tickets {
		t.Assignment = assignment
		tBytes, err := proto.Marshal(t)
		if err != nil {
			log.Err(err).Str("ticket_id", t.Id).Msg("failed to marshal Ticket")
			continue
		}
		data[t.Id] = tBytes
	}
	err := m.kvStore.MSet(ctx, collectionTickets, data)
	if err != nil {
		return err
	}

	ticketIDs := idsFromTickets(tickets)
	log.Debug().Strs("tickets", ticketIDs).Msg("successfully added assignment to tickets")

	var wg sync.WaitGroup
	for _, t := range tickets {
		wg.Add(1)
		go func(ticketID string) {
			backoff.Retry(ctx, func() error {
				return m.pubsubClient.Publish(ctx, GetTicketSubject(ticketID), []byte("assigned"))
			}, backoff.Exponential())
			if err != nil {
				log.Err(err).Str("ticket_id", ticketID).Msg("failed to publish assignment")
			}
			wg.Done()
		}(t.Id)
	}
	wg.Wait()
	log.Debug().Strs("tickets", ticketIDs).Msg("successfully published ticket assignments")
	return nil
}

// RequeueTickets resubmits tickets to the stream of submitted tickets
func (m *manager) RequeueTickets(ctx context.Context, tickets []*pb.Ticket, filterManager filter.Manager) {
	// get ticket ids from the match
	ids := make([]string, len(tickets))
	for i, t := range tickets {
		ids[i] = t.Id
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
	requeueIDs := mapset.NewSet()
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
		if requeueIDs.Contains(t.Id) {
			streamTicket, err := streamTicketFromTicket(t, filterManager)
			if err != nil {
				log.Err(err).Interface("ticket", t).Msg("failed to filter ticket")
				continue
			}
			err = m.sendMessage(ctx, streamTicket)
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
