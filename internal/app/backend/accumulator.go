package backend

import (
	"context"
	"sync"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	stream "github.com/rivalry-matchmaker/rivalry/pkg/pb/stream/v1"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/rs/zerolog/log"
)

// Accumulator defines the interface for the accumulator backend
type Accumulator interface {
	Run() error
}

// AccumulatorConfig holds the settings for the Accumulator service
type AccumulatorConfig struct {
	MaxTickets int64
	MaxDelay   time.Duration
}

type accumulator struct {
	matchmakingQueue  string
	ticketsManager    tickets.Manager
	config            *AccumulatorConfig
	updateMux         sync.Mutex
	ctx               context.Context
	matchRequestsSeen mapset.Set[string]
	matchRequestSlice []*stream.StreamTicket
	timeout           *time.Ticker
}

// NewAccumulator returns a new accumulator
func NewAccumulator(ctx context.Context, matchmakingQueue string, ticketsManager tickets.Manager, config *AccumulatorConfig) Accumulator {

	a := &accumulator{
		ctx:               ctx,
		matchmakingQueue:  matchmakingQueue,
		ticketsManager:    ticketsManager,
		config:            config,
		matchRequestsSeen: mapset.NewSet[string](),
		timeout:           time.NewTicker(config.MaxDelay),
	}
	go a.watchTimeout()
	return a
}

// Run is the main method for the service
func (a *accumulator) Run() error {
	err := a.ticketsManager.StreamMatchRequests(a.ctx, a.matchmakingQueue, func(ctx context.Context, st *stream.StreamTicket) {
		log.Trace().Str("match_request_id", st.MatchRequestId).Msg("StreamTickets")
		a.updateMux.Lock()
		defer a.updateMux.Unlock()

		// deduplicate tickets
		if !a.matchRequestsSeen.Add(st.MatchRequestId) {
			log.Trace().Str("match_request_id", st.MatchRequestId).Msg("deduplicated match request")
			return
		}

		// add ticket to pool tickets
		a.matchRequestSlice = append(a.matchRequestSlice, st)
		log.Trace().Str("match_request_id", st.MatchRequestId).Msg("match request added to pool")

		// if we hit max tickets the make matches
		if int64(a.matchRequestsSeen.Cardinality()) >= a.config.MaxTickets {
			log.Trace().Msg("accumulator hit max tickets")
			a.timeout.Reset(a.config.MaxDelay)
			go a.forwardAccumulatedRequests(ctx, a.matchRequestSlice)
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

func (a *accumulator) watchTimeout() {
	for range a.timeout.C {
		a.updateMux.Lock()
		if len(a.matchRequestSlice) > 0 {
			go a.forwardAccumulatedRequests(a.ctx, a.matchRequestSlice)
			a.resetRun()
		}
		a.updateMux.Unlock()
	}
}

func (a *accumulator) forwardAccumulatedRequests(ctx context.Context, matchRequestSlice []*stream.StreamTicket) {
	log.Trace().Int("stream_tickets", len(matchRequestSlice)).Msg("PublishAccumulatedMatchRequests")
	err := a.ticketsManager.PublishAccumulatedMatchRequests(ctx, a.matchmakingQueue, matchRequestSlice)
	if err != nil {
		log.Err(err).Msg("failed to publish accumulated match requests")
	}
}

func (a *accumulator) resetRun() {
	a.matchRequestSlice = make([]*stream.StreamTicket, 0)
	a.matchRequestsSeen = mapset.NewSet[string]()
}
