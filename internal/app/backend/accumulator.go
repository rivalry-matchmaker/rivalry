package backend

import (
	"context"
	"sync"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/managers/backfill"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/matches"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	mapset "github.com/deckarep/golang-set"
	"github.com/rs/zerolog/log"
)

// Accumulator defines the interface for the accumulator backend
type Accumulator interface {
	Run() error
}

// AccumulatorConfig holds the settings for the Accumulator service
type AccumulatorConfig struct {
	MaxTickets  int64
	MaxBackfill int64
	MaxDelay    time.Duration
}

type accumulator struct {
	profile             *pb.MatchProfile
	profiles            []*pb.MatchProfile
	ticketsManager      tickets.Manager
	backfillManager     backfill.Manager
	matchesManager      matches.Manager
	customlogicManager  customlogic.MatchmakerManager
	matchFunctionTarget string
	filterManager       filter.Manager
	config              *AccumulatorConfig
	updateMux           sync.Mutex
	ctx                 context.Context
	ticketsSeen         mapset.Set
	poolTickets         map[string]*pb.MakeMatchesRequest_PoolTickets
	timeout             *time.Ticker
}

// NewAccumulator returns a new accumulator
func NewAccumulator(ctx context.Context, profileName string, profiles []*pb.MatchProfile, ticketsManager tickets.Manager,
	backfillManager backfill.Manager, matchesManager matches.Manager, customlogicManager customlogic.MatchmakerManager,
	config *AccumulatorConfig) Accumulator {
	var profile *pb.MatchProfile
	for _, p := range profiles {
		if p.Name == profileName {
			profile = p
			break
		}
	}

	a := &accumulator{
		ctx:                ctx,
		profile:            profile,
		profiles:           profiles,
		ticketsManager:     ticketsManager,
		backfillManager:    backfillManager,
		matchesManager:     matchesManager,
		customlogicManager: customlogicManager,
		filterManager:      filter.NewManager(profiles),
		config:             config,
		ticketsSeen:        mapset.NewSet(),
		poolTickets:        make(map[string]*pb.MakeMatchesRequest_PoolTickets),
		timeout:            time.NewTicker(config.MaxDelay),
	}
	go a.watchTimeout()
	return a
}

// Run is the main method for the service
func (a *accumulator) Run() error {
	err := a.ticketsManager.StreamTickets(a.ctx, a.profile.Name, func(ctx context.Context, st *pb.StreamTicket, t *pb.Ticket) {
		log.Trace().Str("ticket_id", t.Id).Msg("StreamTickets")
		a.updateMux.Lock()
		defer a.updateMux.Unlock()

		// deduplicate tickets
		if !a.ticketsSeen.Add(t.Id) {
			log.Trace().Str("ticket_id", t.Id).Msg("deduplicated ticket")
			return
		}

		// add ticket to pool tickets
		if _, ok := a.poolTickets[st.Pool]; !ok {
			a.poolTickets[st.Pool] = new(pb.MakeMatchesRequest_PoolTickets)
		}
		a.poolTickets[st.Pool].Tickets = append(a.poolTickets[st.Pool].Tickets, t)
		log.Trace().Str("pool", st.Pool).Str("ticket_id", t.Id).Msg("ticket added to pool")

		// if we hit max tickets the make matches
		if int64(a.ticketsSeen.Cardinality()) >= a.config.MaxTickets {
			log.Trace().Msg("accumulator hit max tickets")
			a.timeout.Reset(a.config.MaxDelay)
			go a.makeMatches(ctx, a.poolTickets)
			a.resetRun()
			return
		}
	})
	if err != nil {
		return err
	}
	<-a.ctx.Done()
	a.timeout.Stop()
	return nil
}

func (a *accumulator) makeMatches(ctx context.Context, poolTickets map[string]*pb.MakeMatchesRequest_PoolTickets) {
	log.Trace().Msg("makeMatches")
	ticketSet := mapset.NewSet()
	for _, p := range poolTickets {
		for _, t := range p.Tickets {
			ticketSet.Add(t.Id)
		}
	}

	poolBackfills := make(map[string]*pb.MakeMatchesRequest_PoolBackfills)
	backfills, err := a.backfillManager.GetAvailableBackfill(ctx, a.profile.Name, a.config.MaxBackfill)
	if err != nil {
		log.Err(err).Msg("failed to make matches")
		a.requeueAllTickets(ctx, poolTickets)
		return
	}
	for _, b := range backfills {
		_, pool, err := a.filterManager.ProfileMembershipTest(b)
		if err != nil {
			log.Err(err).Interface("backfill", b).Msg("failed to filter backfill")
			continue
		}
		if _, ok := poolBackfills[pool]; !ok {
			poolBackfills[pool] = new(pb.MakeMatchesRequest_PoolBackfills)
		}
		poolBackfills[pool].Backfill = append(poolBackfills[pool].Backfill, b)
	}

	err = a.customlogicManager.MakeMatches(ctx, &pb.MakeMatchesRequest{
		MatchProfile:  a.profile,
		PoolTickets:   poolTickets,
		PoolBackfills: poolBackfills,
	}, func(ctx context.Context, match *pb.Match) {
		// we need to allocate a game server if the backfill is not set or is new
		match.AllocateGameserver = match.Backfill == nil || match.Backfill.Id == ""

		if match.Backfill != nil {
			ticketsIDs := make([]string, len(match.Tickets))
			for i, v := range match.Tickets {
				ticketsIDs[i] = v.Id
			}
			// make sure the backfill object has the tickets from the match
			match.Backfill.Tickets = match.Tickets
			err := a.backfillManager.MakeMatchWithBackfill(ctx, match.Backfill, ticketsIDs, a.filterManager)
			if err != nil {
				log.Err(err).Str("backfill", match.Backfill.Id).Msg("make matches with backfill failed")
				a.ticketsManager.RequeueTickets(ctx, match.Tickets, a.filterManager)
				return
			}
		}

		successful, err := a.matchesManager.CreateMatch(ctx, match)
		if err != nil || !successful {
			log.Err(err).Msg("failed to create match")
			a.ticketsManager.RequeueTickets(ctx, match.Tickets, a.filterManager)
			return
		}

		for _, t := range match.Tickets {
			ticketSet.Remove(t.Id)
		}
		log.Trace().Interface("match", match).Msg("match created")
	})
	if err != nil {
		log.Err(err).Msg("failed to make matches")
		a.requeueAllTickets(ctx, poolTickets)
	}
	if ticketSet.Cardinality() > 0 {
		requeue := make([]*pb.Ticket, 0, ticketSet.Cardinality())
		for _, p := range poolTickets {
			for _, t := range p.Tickets {
				if ticketSet.Contains(t.Id) {
					requeue = append(requeue, t)
				}
			}
		}
		log.Trace().Int("number_of_tickets", len(requeue)).Msg("re-queuing unmatched tickets")
		a.ticketsManager.RequeueTickets(ctx, requeue, a.filterManager)
	}
}

func (a *accumulator) watchTimeout() {
	for range a.timeout.C {
		a.updateMux.Lock()
		if len(a.poolTickets) > 0 {
			go a.makeMatches(a.ctx, a.poolTickets)
			a.resetRun()
		}
		a.updateMux.Unlock()
	}
}

func (a *accumulator) resetRun() {
	a.poolTickets = make(map[string]*pb.MakeMatchesRequest_PoolTickets)
	a.ticketsSeen = mapset.NewSet()
}

func (a *accumulator) requeueAllTickets(ctx context.Context, poolTickets map[string]*pb.MakeMatchesRequest_PoolTickets) {
	totalTickets := 0
	for _, v := range poolTickets {
		totalTickets += len(v.Tickets)
	}
	allTickets := make([]*pb.Ticket, totalTickets)
	i := 0
	for _, v := range poolTickets {
		for _, t := range v.Tickets {
			allTickets[i] = t
			i++
		}
	}
	log.Trace().Int("total_tickets", totalTickets).Msg("requeueAllTickets")
	a.ticketsManager.RequeueTickets(ctx, allTickets, a.filterManager)
}
