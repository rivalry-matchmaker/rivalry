package service

import (
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/rs/zerolog/log"
)

const matchFunctionName = "pairs"

// PairsMatchmaker conforms to the MatchMakerServiceServer interface
type PairsMatchmaker struct {
	pb.UnimplementedMatchMakerServiceServer
	backfill bool
}

// NewPairsMatchmaker returns a MatchMakerServiceServer that matches tickets into pairs
func NewPairsMatchmaker(backfill bool) pb.MatchMakerServiceServer {
	return &PairsMatchmaker{
		backfill: backfill,
	}
}

// MakeMatches matches tickets into pairs
func (m *PairsMatchmaker) MakeMatches(req *pb.MakeMatchesRequest, svr pb.MatchMakerService_MakeMatchesServer) error {
	tickets := make([]*pb.Ticket, 2)
	noTickets := 0
	var t *pb.Ticket
	for name, pool := range req.PoolTickets {
		if backfillPool, ok := req.PoolBackfills[name]; ok {
			for _, backfill := range backfillPool.Backfill {
				if backfill.Extensions == nil {
					backfill.Extensions = new(pb.Extensions)
				}
				if backfill.Extensions.DoubleExtensions == nil {
					backfill.Extensions.DoubleExtensions = make(map[string]float64)
				}
				if _, ok := backfill.Extensions.DoubleExtensions["spaces"]; !ok {
					backfill.Extensions.DoubleExtensions["spaces"] = 1
				}
				if len(pool.Tickets) == 0 {
					break
				}
				if backfill.Extensions.DoubleExtensions["spaces"] > 0 {
					backfill.Extensions.DoubleExtensions["spaces"] = 0
					t, pool.Tickets = pool.Tickets[0], pool.Tickets[1:]
					err := svr.Send(&pb.MakeMatchesResponse{Match: &pb.Match{
						MatchProfile:  req.MatchProfile.Name,
						MatchFunction: matchFunctionName,
						Tickets:       []*pb.Ticket{t},
						Backfill:      backfill,
					}})
					if err != nil {
						return err
					}
					log.Info().Str("backfill_id", backfill.Id).Str("ticket_id", t.Id).Msg("adding ticket to backfill")
				}
			}
		}

		for _, t := range pool.Tickets {
			tickets[noTickets%2] = t
			if noTickets%2 == 1 {
				err := svr.Send(&pb.MakeMatchesResponse{Match: &pb.Match{
					MatchProfile:  req.MatchProfile.Name,
					MatchFunction: matchFunctionName,
					Tickets:       tickets,
				}})
				if err != nil {
					return err
				}
				log.Info().Strs("ticket_ids", []string{tickets[0].Id, tickets[1].Id}).Msg("adding tickets to match")
				tickets[0], tickets[1] = nil, nil
			}
			noTickets++
		}
	}
	if m.backfill && tickets[0] != nil && tickets[1] == nil {
		tickets = tickets[:1]
		ext := tickets[0].Extensions
		if ext == nil {
			ext = new(pb.Extensions)
		}
		if ext.DoubleExtensions == nil {
			ext.DoubleExtensions = make(map[string]float64)
		}
		ext.DoubleExtensions["spaces"] = 1
		err := svr.Send(&pb.MakeMatchesResponse{Match: &pb.Match{
			MatchProfile:  req.MatchProfile.Name,
			MatchFunction: matchFunctionName,
			Tickets:       tickets,
			Backfill: &pb.Backfill{
				SearchFields: tickets[0].SearchFields,
				Extensions:   ext,
			},
		}})
		if err != nil {
			return err
		}
		log.Info().Str("ticket_id", tickets[0].Id).Msg("adding ticket to new backfill")
	}
	return nil
}
