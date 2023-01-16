package backend

import (
	"context"

	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/matches"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/rs/zerolog/log"
)

// Dispenser defines the interface for the dispenser backend
type Dispenser interface {
	Run(ctx context.Context) error
}

type dispenser struct {
	matchesManager     matches.Manager
	ticketsManager     tickets.Manager
	customLogicManager customlogic.AssignmentManager
}

// NewDispenser returns a new dispenser
func NewDispenser(matchesManager matches.Manager, ticketsManager tickets.Manager, customlogicManager customlogic.AssignmentManager) Dispenser {
	return &dispenser{
		matchesManager:     matchesManager,
		ticketsManager:     ticketsManager,
		customLogicManager: customlogicManager,
	}
}

// Run is the main method for the service
func (d *dispenser) Run(ctx context.Context) error {
	err := d.matchesManager.StreamMatches(ctx, func(ctx context.Context, match *pb.Match) {
		if match.Backfill != nil {
			// TODO
		} else {
			assignment, err := d.makeAssignment(ctx, match)
			if err != nil {
				log.Err(err).Str("match_id", match.MatchId).Msg("failed to make assignment")
				d.requeueMatch(ctx, match)
				return
			}
			d.addAssignmentToTickets(ctx, match, assignment)
		}
	})
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (d *dispenser) makeAssignment(ctx context.Context, match *pb.Match) (assignment *pb.Assignment, err error) {
	err = backoff.Retry(ctx, func() error {
		var err error
		assignment, err = d.customLogicManager.MakeAssignment(ctx, match)
		log.Err(err).Msg("make assignment failed, retrying")
		return err
	}, backoff.Exponential())
	return
}

func (d *dispenser) requeueMatch(ctx context.Context, match *pb.Match) {
	if err := backoff.Retry(ctx, func() error {
		return d.matchesManager.Requeue(match.MatchId)
	}, backoff.Exponential()); err != nil {
		log.Err(err).Str("match_id", match.MatchId).Msg("failed to requeue match")
	}
}

func (d *dispenser) addAssignmentToTickets(ctx context.Context, match *pb.Match, assignment *pb.Assignment) {
	err := backoff.Retry(ctx, func() error {
		return d.ticketsManager.AddAssignmentToTickets(ctx, assignment, match.Tickets)
	}, backoff.Exponential())
	if err != nil {
		log.Err(err).Str("match_id", match.MatchId).Msg("failed to add assignment to ticket")
	}
}
