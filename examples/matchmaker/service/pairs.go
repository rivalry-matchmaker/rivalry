package service

import (
	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	"github.com/rs/xid"
)

const matchFunctionName = "pairs"

// PairsMatchmaker conforms to the MatchMakerServiceServer interface
type PairsMatchmaker struct {
	api.UnimplementedMatchMakerServiceServer
}

// NewPairsMatchmaker returns a MatchMakerServiceServer that matches tickets into pairs
func NewPairsMatchmaker() api.MatchMakerServiceServer {
	return &PairsMatchmaker{}
}

// MakeMatches matches tickets into pairs
func (m *PairsMatchmaker) MakeMatches(req *api.MakeMatchesRequest, svr api.MatchMakerService_MakeMatchesServer) error {
	tickets := make([]string, 2)
	players := make([]string, 2)
	rtts := make(map[string]int64)
	for i, sub := range req.Submissions {
		tickets[i%2] = sub.Id
		players[i%2] = sub.Members[0]
		for _, r := range sub.Rtts {
			if _, ok := rtts[r.Host]; !ok {
				rtts[r.Host] = r.Milliseconds
			} else {
				rtts[r.Host] = (rtts[r.Host] + r.Milliseconds) / 2
			}
		}
		if i%2 == 1 {
			svr.Send(&api.MakeMatchesResponse{
				Match: &api.Match{
					MatchId:          xid.New().String(),
					MatchmakingQueue: "",
					MatchRequestIds:  tickets,
					PlayerIds:        players,
					Rtts: func() []*api.RTT {
						out := make([]*api.RTT, 0, len(rtts))
						for k, v := range rtts {
							out = append(out, &api.RTT{Host: k, Milliseconds: v})
						}
						return out
					}(),
				},
			})
			rtts = make(map[string]int64)
		}
	}
	return nil
}
