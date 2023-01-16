package matches

//go:generate mockgen -package mock -destination=mock/matches.go . Manager

import (
	"context"

	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/golang/protobuf/proto"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
)

const (
	collectionMatch        = "matches"
	topicUnassignedMatches = "match"
)

// GetUnassignedMatchesTopic returns the topic for match proposals
func GetUnassignedMatchesTopic() string {
	return topicUnassignedMatches
}

// Manager interface specifies the methods needed to manage match state
type Manager interface {
	CreateMatch(ctx context.Context, match *pb.Match) (bool, error)
	StreamMatches(ctx context.Context, f func(ctx context.Context, match *pb.Match)) error
	Requeue(matchID string) error
	Close()
}

type manager struct {
	ticketsManager tickets.Manager
	kvStore        kv.Store
	streamClient   stream.Client
}

// NewManager returns a match Manager
func NewManager(ticketsManager tickets.Manager, kvStore kv.Store, streamClient stream.Client) Manager {
	return &manager{
		ticketsManager: ticketsManager,
		kvStore:        kvStore,
		streamClient:   streamClient,
	}
}

// GetMatch returns a Match
func (m *manager) GetMatch(ctx context.Context, matchID string) (*pb.Match, error) {
	data, err := m.kvStore.Get(ctx, collectionMatch, matchID)
	if err != nil {
		return nil, err
	}
	match := new(pb.Match)
	err = proto.Unmarshal(data, match)
	if err != nil {
		return nil, err
	}
	return match, nil
}

// CreateMatch creates a Match
func (m *manager) CreateMatch(ctx context.Context, match *pb.Match) (bool, error) {
	if len(match.MatchId) == 0 {
		match.MatchId = xid.New().String()
	}

	// try to take ownership of all the tickets in this match
	successful, err := m.ticketsManager.AssignTicketsToMatch(ctx, match)
	if err != nil {
		return false, err
	}
	if !successful {
		return false, nil
	}
	// if anything from here onwards fails we want to release tickets from this match
	defer func() {
		if err != nil {
			releaseErr := m.ticketsManager.ReleaseTicketsFromMatch(ctx, match)
			if releaseErr != nil {
				log.Err(releaseErr).Msg("failed to release tickets from match")
			}
		}
	}()

	// marshal match into bytes
	matchBytes, err := proto.Marshal(match)
	if err != nil {
		return false, err
	}

	// write to the database
	err = m.kvStore.Set(ctx, collectionMatch, match.MatchId, matchBytes, 0)
	if err != nil {
		return false, err
	}

	// if anything from here onwards fails we want to remove the match from the database
	defer func() {
		if err != nil {
			_ = m.kvStore.Del(ctx, collectionMatch, match.MatchId)
		}
	}()

	if match.AllocateGameserver {
		err = m.streamClient.SendMessage(GetUnassignedMatchesTopic(), []byte(match.MatchId))
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// StreamMatches listens on a stream for unassigned matches
func (m *manager) StreamMatches(ctx context.Context, f func(ctx context.Context, match *pb.Match)) error {
	return m.streamClient.Subscribe(GetUnassignedMatchesTopic(), func(data []byte) {
		var match *pb.Match
		err := backoff.Retry(ctx, func() error {
			var err error
			match, err = m.GetMatch(ctx, string(data))
			return err
		}, backoff.Exponential())
		if err != nil {
			log.Err(err).Str("match_id", string(data)).Msg("failed to get match")
		}
		f(ctx, match)
	})
}

// Requeue takes a match and resubmits it into the stream for unassigned matches
func (m *manager) Requeue(matchID string) error {
	return m.streamClient.SendMessage(GetUnassignedMatchesTopic(), []byte(matchID))
}

// Close shuts down the kv store and stream clients
func (m *manager) Close() {
	m.kvStore.Close()
	m.streamClient.Close()
}
