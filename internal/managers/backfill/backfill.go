package backfill

//go:generate mockgen -package mock -destination=mock/backfill.go . Manager

import (
	"context"
	"fmt"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/dlm"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	collectionBackfill  = "backfill"
	backfillLastAckTime = "backfill_last_ack_time"
	pendingTickets      = "pending_tickets"
	availableBackfill   = "available_backfill"
)

// Manager specifies the interface for managing Backfill
type Manager interface {
	CreateBackfill(ctx context.Context, backfill *pb.Backfill, filterManager filter.Manager) (*pb.Backfill, error)
	UpdateBackfill(ctx context.Context, backfill *pb.Backfill, filterManager filter.Manager) (*pb.Backfill, error)
	MakeMatchWithBackfill(ctx context.Context, backfill *pb.Backfill, ticketIDs []string, filterManager filter.Manager) error
	GetBackfill(ctx context.Context, backfillID string) (*pb.Backfill, error)
	GetAvailableBackfill(ctx context.Context, profile string, max int64) ([]*pb.Backfill, error)
	AcknowledgeBackfill(ctx context.Context, backfillID string, assignment *pb.Assignment, filterManager filter.Manager) (*pb.Backfill, error)
	DeleteBackfill(ctx context.Context, backfillID string, filterManager filter.Manager) error
}

type manager struct {
	kvStore            kv.Store
	sortedSet          kv.SortedSet
	set                kv.Set
	lockManager        dlm.DLM
	ticketsManager     tickets.Manager
	backfillLockExpiry time.Duration
}

// NewManager returns a new Backfill Manager
func NewManager(kvStore kv.Store, sortedSet kv.SortedSet, set kv.Set, lockManager dlm.DLM, ticketsManager tickets.Manager, backfillLockExpirySeconds int64) Manager {
	return &manager{
		kvStore:            kvStore,
		sortedSet:          sortedSet,
		set:                set,
		lockManager:        lockManager,
		ticketsManager:     ticketsManager,
		backfillLockExpiry: time.Duration(backfillLockExpirySeconds) * time.Second,
	}
}

// CreateBackfill creates a new backfill
func (m *manager) CreateBackfill(ctx context.Context, backfill *pb.Backfill, filterManager filter.Manager) (*pb.Backfill, error) {
	if len(backfill.Id) == 0 {
		backfill.Id = xid.New().String()
	}
	backfill.CreateTime = timestamppb.Now()
	backfill.Generation = 1

	// check the backfill passes the profile membership test
	profile, _, err := filterManager.ProfileMembershipTest(backfill)
	if err != nil {
		return nil, err
	}

	// marshal backfill into bytes
	backfillBytes, err := proto.Marshal(backfill)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal backfill")
	}

	// persist the backfill
	set, err := m.kvStore.SetNX(ctx, collectionBackfill, backfill.Id, backfillBytes, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to persist backfill")
	}
	if !set {
		return nil, errors.Errorf("backfill already exists, id: %s", backfill.GetId())
	}

	// Update acknowledgment timestamp
	err = m.sortedSet.ZAdd(ctx, backfillLastAckTime, &kv.Z{Score: float64(time.Now().UnixNano()), Member: backfill.Id})
	if err != nil {
		return nil, errors.Wrap(err, "failed to index backfill ack time")
	}

	// add backfill to available backfill index
	err = m.markBackfillAsAvailable(ctx, profile, backfill.Id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to index backfill")
	}

	log.Info().Str("backfill_id", backfill.Id).Msg("Backfill Created")

	return backfill, nil
}

// GetAvailableBackfill returns available backfill for a given profile, up to a specified maximum quantity
func (m *manager) GetAvailableBackfill(ctx context.Context, profile string, max int64) ([]*pb.Backfill, error) {
	ids, err := m.set.SPopN(ctx, availableBackfill+":"+profile, max)
	if err != nil {
		return nil, err
	}
	// defer, if we return an error then we re-indexes these ids
	defer func() {
		if err != nil {
			if err := m.markBackfillAsAvailable(ctx, profile, ids...); err != nil {
				log.Err(err).Str("profile", profile).Strs("backfill_ids", ids).Msg("failed to reindex backfill")
			}
		}
	}()
	data, err := m.kvStore.MGet(ctx, collectionBackfill, ids)
	if err != nil {
		return nil, err
	}
	backfill := make([]*pb.Backfill, 0, len(ids))
	for _, v := range data {
		b := new(pb.Backfill)
		err = proto.Unmarshal(v, b)
		if err != nil {
			return nil, err
		}
		backfill = append(backfill, b)
	}
	return backfill, nil
}

// UpdateBackfill updates a Backfill
func (m *manager) UpdateBackfill(ctx context.Context, backfill *pb.Backfill, filterManager filter.Manager) (*pb.Backfill, error) {
	// check the backfill ID is given
	if len(backfill.Id) == 0 {
		return nil, fmt.Errorf("backfill id is required")
	}

	// check the backfill passes the profile membership test
	_, _, err := filterManager.ProfileMembershipTest(backfill)
	if err != nil {
		return nil, err
	}

	// lock the backfill
	unlock, err := m.lockBackfill(backfill.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to lock backfill %s", backfill.Id)
	}
	defer unlock()

	// check for pending tickets
	err = m.requeuePendingTickets(ctx, backfill.Id, filterManager)
	if err != nil {
		return nil, err
	}

	// marshal backfill into bytes
	backfillBytes, err := proto.Marshal(backfill)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal backfill")
	}

	// persist the backfill
	err = m.kvStore.Set(ctx, collectionBackfill, backfill.Id, backfillBytes, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to persist backfill")
	}

	// Update acknowledgment timestamp
	err = m.sortedSet.ZAdd(ctx, backfillLastAckTime, &kv.Z{Score: float64(time.Now().UnixNano()), Member: backfill.Id})
	if err != nil {
		return nil, errors.Wrap(err, "failed to index Backfill")
	}
	return backfill, nil
}

// MakeMatchWithBackfill is called after backfill has had tickets matched into it
func (m *manager) MakeMatchWithBackfill(ctx context.Context, backfill *pb.Backfill, ticketIDs []string, filterManager filter.Manager) error {
	var err error
	if backfill.Id == "" {
		backfill, err = m.CreateBackfill(ctx, backfill, filterManager)
		if err != nil {
			return err
		}
		return m.set.SAdd(ctx, pendingTickets+":"+backfill.Id, ticketIDs...)
	}

	// ensure the backfill still matches a profile
	_, _, err = filterManager.ProfileMembershipTest(backfill)
	if err != nil {
		return err
	}

	// lock the backfill
	unlock, err := m.lockBackfill(backfill.Id)
	if err != nil {
		return errors.Wrapf(err, "unable to lock backfill %s", backfill.Id)
	}
	defer unlock()

	// read backfill from the database
	storedBackfill, err := m.GetBackfill(ctx, backfill.Id)
	if err != nil {
		return err
	}

	// check the persisted backfill generation matches the generation of the given backfill
	if storedBackfill.Generation != backfill.Generation {
		return errors.New("backfill generation mismatch")
	}

	// copy updates search
	storedBackfill.SearchFields = backfill.SearchFields
	storedBackfill.Extensions = backfill.Extensions
	storedBackfill.Generation++

	// marshal backfill into bytes
	backfillBytes, err := proto.Marshal(storedBackfill)
	if err != nil {
		return errors.Wrap(err, "failed to marshal backfill")
	}

	// persist the backfill
	err = m.kvStore.Set(ctx, collectionBackfill, backfill.Id, backfillBytes, 0)
	if err != nil {
		return errors.Wrap(err, "failed to persist backfill")
	}

	// add tickets into pending
	err = m.set.SAdd(ctx, pendingTickets+":"+backfill.Id, ticketIDs...)
	if err != nil {
		return errors.Wrap(err, "failed to add tickets into pending set")
	}
	log.Info().Str("backfill_id", backfill.Id).Msg("Make Match With Backfill")
	return nil
}

// GetBackfill returns the current state of a backfill
func (m *manager) GetBackfill(ctx context.Context, backfillID string) (*pb.Backfill, error) {
	data, err := m.kvStore.Get(ctx, collectionBackfill, backfillID)
	b := new(pb.Backfill)
	err = proto.Unmarshal(data, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// AcknowledgeBackfill updates a backfills last acknowledge time and adds pending tickets to the backfill
func (m *manager) AcknowledgeBackfill(ctx context.Context, backfillID string, assignment *pb.Assignment, filterManager filter.Manager) (*pb.Backfill, error) {
	unlock, err := m.lockBackfill(backfillID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to lock backfill %s", backfillID)
	}
	defer unlock()

	// get pending ticket ids for backfill
	ticketIDs, err := m.getPendingTicketIDs(ctx, backfillID)
	tickets, err := m.ticketsManager.GetTickets(ctx, ticketIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read pending tickets")
	}

	// read the backfill object
	backfill, err := m.GetBackfill(ctx, backfillID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read backfill")
	}
	// add pending tickets to backfill obj
	backfill.Tickets = append(backfill.Tickets, tickets...)

	// marshal backfill into bytes
	backfillBytes, err := proto.Marshal(backfill)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal backfill")
	}

	// store backfill in the db
	err = m.kvStore.Set(ctx, collectionBackfill, backfillID, backfillBytes, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to persist backfill")
	}

	// remove tickets from the pending set
	err = m.removePendingTickets(ctx, backfillID, ticketIDs)
	if err != nil {
		return nil, err
	}

	// get the profile for this backfill
	profile, _, err := filterManager.ProfileMembershipTest(backfill)
	if err != nil {
		return nil, err
	}

	// mark this backfill as available
	err = m.markBackfillAsAvailable(ctx, profile, backfill.Id)
	if err != nil {
		return nil, err
	}

	// add assignment to tickets
	err = m.ticketsManager.AddAssignmentToTickets(ctx, assignment, tickets)
	if err != nil {
		return nil, err
	}
	log.Info().Str("backfill_id", backfillID).Msg("backfill acknowledged")
	return backfill, nil
}

// DeleteBackfill removes a backfill
func (m *manager) DeleteBackfill(ctx context.Context, backfillID string, filterManager filter.Manager) error {
	unlock, err := m.lockBackfill(backfillID)
	if err != nil {
		return errors.Wrapf(err, "unable to lock backfill %s", backfillID)
	}
	defer unlock()

	// remove the backfill acknowledgment timestamp
	err = m.sortedSet.ZRem(ctx, backfillLastAckTime, backfillID)
	if err != nil {
		return err
	}

	// remove the backfill document from the store
	err = m.kvStore.Del(ctx, collectionBackfill, backfillID)
	if err != nil {
		return err
	}

	// remove tickets associated with this backfill
	err = m.requeuePendingTickets(ctx, backfillID, filterManager)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) lockBackfill(backfillID string) (func(), error) {
	log.Trace().Str("backfill_id", backfillID).Msg("locking backfill")
	err := m.lockManager.Lock(backfillID, m.backfillLockExpiry)
	if err != nil {
		log.Err(err).Str("backfill_id", backfillID).Msg("unable to lock backfill")
		return nil, err
	}
	log.Trace().Str("backfill_id", backfillID).Msg("backfill locked")

	return func() {
		log.Trace().Str("backfill_id", backfillID).Msg("unlocking backfill")
		_, err = m.lockManager.Unlock(backfillID)
		if err != nil {
			log.Err(err).Str("backfill_id", backfillID).Msg("unable to unlock backfill")
			return
		}
		log.Trace().Str("backfill_id", backfillID).Msg("backfill unlocked")
	}, nil
}

func (m *manager) requeuePendingTickets(ctx context.Context, backfillID string, filterManager filter.Manager) error {
	ticketIDs, err := m.getPendingTicketIDs(ctx, backfillID)
	if len(ticketIDs) > 0 {
		// remove tickets pending acknowledgement
		err = m.removePendingTickets(ctx, backfillID, ticketIDs)
		if err != nil {
			return err
		}
		// requeue  tickets
		tickets, err := m.ticketsManager.GetTickets(ctx, ticketIDs)
		if err != nil {
			return err
		}
		m.ticketsManager.RequeueTickets(ctx, tickets, filterManager)
	}
	return nil
}

func (m *manager) getPendingTicketIDs(ctx context.Context, backfillID string) ([]string, error) {
	ticketIDs, err := m.set.SMembers(ctx, pendingTickets+":"+backfillID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read pending ticket ids")
	}
	return ticketIDs, nil
}

func (m *manager) removePendingTickets(ctx context.Context, backfillID string, ticketIDs []string) error {
	if len(ticketIDs) == 0 {
		return nil
	}
	err := m.set.SRem(ctx, pendingTickets+":"+backfillID, ticketIDs...)
	if err != nil {
		return errors.Wrap(err, "failed to tidy pending ticket list")
	}
	return nil
}

func (m *manager) markBackfillAsAvailable(ctx context.Context, profile string, backfillIDs ...string) error {
	return m.set.SAdd(ctx, availableBackfill+":"+profile, backfillIDs...)
}
