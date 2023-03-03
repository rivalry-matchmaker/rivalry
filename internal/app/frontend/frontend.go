package frontend

import (
	"context"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/pkg/errors"
	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	db "github.com/rivalry-matchmaker/rivalry/pkg/pb/db/v1"
	"github.com/rs/xid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the FrontendService APIs
type Service struct {
	queues         mapset.Set[string]
	ticketsManager tickets.Manager
}

// NewService returns a new Service
func NewService(queues []string, ticketsManager tickets.Manager) *Service {
	return &Service{
		queues:         mapset.NewSet[string](queues...),
		ticketsManager: ticketsManager,
	}
}

// Close shuts down database connections
func (s *Service) Close() {
	s.ticketsManager.Close()
}

// Match assigns a unique TicketId to the input Ticket and record it in state storage.
// A ticket is considered as ready for matchmaking once it is created.
//   - The ticket is tested against Match Profiles for membership and is added to a stream for the relevant profile,
//     if no profile is matched an error is returned.
func (s *Service) Match(req *pb.MatchRequest, svr pb.RivalryService_MatchServer) error {
	// Perform input validation.
	if len(req.Id) == 0 {
		req.Id = xid.New().String()
	}
	// TODO: frontend should know full list of matchmaking queues and reject invalid queues
	if !s.queues.Contains(req.MatchmakingQueue) {
		return status.Error(codes.InvalidArgument, "invalid matchmaking queue")
	}

	err := s.ticketsManager.CreateMatchRequest(svr.Context(), req)
	if err != nil {
		if errors.Is(err, tickets.ErrDuplicateTicketID) {
			return status.Error(codes.AlreadyExists, err.Error())
		}
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := false
	processTicket := func(ctx context.Context, t *db.MatchRequest) {
		if t.GameServer != nil {
			_ = backoff.Retry(
				ctx,
				func() error {
					ports := make([]*pb.GameServerPort, len(t.GameServer.GameServerPorts))
					for i, p := range t.GameServer.GameServerPorts {
						ports[i] = &pb.GameServerPort{Name: p.Name, Port: p.Port}
					}
					return svr.Send(&pb.MatchResponse{GameServer: &pb.GameServer{
						GameServerIp:    t.GameServer.GameServerIp,
						GameServerPorts: ports,
					}})
				},
				backoff.Constant(time.Second/10),
			)
			cancel()
			done = true
		}
	}

	t, err := s.ticketsManager.GetMatchRequest(svr.Context(), req.Id)
	if err != nil {
		return err
	}
	processTicket(svr.Context(), t)
	if done {
		return nil
	}

	return s.ticketsManager.WatchMatchRequest(ctx, req.Id, processTicket)
}
