package backend

import (
	"context"

	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/matches"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	db "github.com/rivalry-matchmaker/rivalry/pkg/pb/db/v1"
	"github.com/rs/zerolog/log"
)

// Matcher defines the interface for the Matcher backend
type Matcher interface {
	Run() error
}

type matcher struct {
	ctx                context.Context
	matchmakingQueue   string
	customLogicManager customlogic.MatchmakerManager
	ticketsManager     tickets.Manager
	matchesManager     matches.Manager
}

func NewMatcher(ctx context.Context, matchmakingQueue string, ticketsManager tickets.Manager,
	matchesManager matches.Manager, customLogicManager customlogic.MatchmakerManager) Matcher {
	return &matcher{
		ctx:                ctx,
		matchmakingQueue:   matchmakingQueue,
		ticketsManager:     ticketsManager,
		matchesManager:     matchesManager,
		customLogicManager: customLogicManager,
	}
}

func (m *matcher) Run() error {
	err := m.ticketsManager.StreamAccumulatedMatchRequests(m.ctx, m.matchmakingQueue, func(ctx context.Context, sts []*api.StreamTicket) {
		ids := make([]string, len(sts))
		for i, v := range sts {
			ids[i] = v.MatchRequestId
		}
		reqs, err := m.ticketsManager.GetMatchRequests(ctx, ids)
		if err != nil {
			log.Err(err).Int("total", len(sts)).Msg("requeue match requests, because we failed to read them")
			m.ticketsManager.RequeueMatchRequests(ctx, sts)
		}
		m.makeMatches(ctx, reqs)
	})
	if err != nil {
		return err
	}
	<-m.ctx.Done()
	return nil
}

func (m *matcher) makeMatches(ctx context.Context, matchRequestSlice []*db.MatchRequest) {
	log.Trace().Msg("makeMatches")

	matchRequestMap := make(map[string]*api.StreamTicket)
	for _, mr := range matchRequestSlice {
		matchRequestMap[mr.Id] = tickets.StreamTicketFromDBRequest(mr)
	}

	subs := make([]*api.Submission, len(matchRequestSlice))
	for i, v := range matchRequestSlice {
		subs[i] = tickets.SubmissionFromDBRequest(v)
	}

	matchRequast := &api.MakeMatchesRequest{
		Submissions: subs,
	}

	if err := m.customLogicManager.MakeMatches(ctx, matchRequast, func(ctx context.Context, match *api.Match) {

		successful, err := m.matchesManager.CreateMatch(ctx, match)
		if err != nil || !successful {
			log.Err(err).Msg("failed to create match")
			matchTickets := make([]*api.StreamTicket, len(match.MatchRequestIds))
			for i, id := range match.MatchRequestIds {
				matchTickets[i] = matchRequestMap[id]
			}
			m.ticketsManager.RequeueMatchRequests(ctx, matchTickets)
			return
		}

		for _, t := range match.MatchRequestIds {
			delete(matchRequestMap, t)
		}
		log.Trace().Interface("match", match).Msg("match created")
	}); err != nil {
		log.Err(err).Msg("failed to make matches")
		m.requeueAll(ctx, matchRequestSlice)
	}

	if len(matchRequestMap) > 0 {
		requeue := make([]*api.StreamTicket, 0, len(matchRequestMap))
		for _, v := range matchRequestMap {
			requeue = append(requeue, v)
		}
		log.Trace().Int("number_of_tickets", len(requeue)).Msg("re-queuing unmatched tickets")
		m.ticketsManager.RequeueMatchRequests(ctx, requeue)
	}
}

func (m *matcher) requeueAll(ctx context.Context, matchRequestSlice []*db.MatchRequest) {
	q := make([]*api.StreamTicket, len(matchRequestSlice))
	for i, v := range matchRequestSlice {
		q[i] = tickets.StreamTicketFromDBRequest(v)
	}
	log.Trace().Int("total", len(matchRequestSlice)).Msg("requeueAll")
	m.ticketsManager.RequeueMatchRequests(ctx, q)
}
