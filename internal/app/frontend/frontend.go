package frontend

import (
	"context"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/backoff"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/backfill"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the FrontendService APIs
type Service struct {
	ticketsManager     tickets.Manager
	backfillManager    backfill.Manager
	customlogicManager customlogic.FrontendManager
	filterManager      filter.Manager
}

// NewService returns a new Service
func NewService(ticketsManager tickets.Manager, backfillManager backfill.Manager, customlogicManager customlogic.FrontendManager, filterManager filter.Manager) *Service {
	return &Service{
		ticketsManager:     ticketsManager,
		backfillManager:    backfillManager,
		customlogicManager: customlogicManager,
		filterManager:      filterManager,
	}
}

// Close shuts down database connections
func (s *Service) Close() {
	s.ticketsManager.Close()
}

// CreateTicket assigns an unique TicketId to the input Ticket and record it in state storage.
// A ticket is considered as ready for matchmaking once it is created.
//   - If a TicketId exists in a Ticket request, an auto-generated TicketId will override this field.
//   - The ticket is tested against Match Profiles for membership and is added to a stream for the relevant profile,
//     if no profile is matched an error is returned.
func (s *Service) CreateTicket(ctx context.Context, req *pb.CreateTicketRequest) (*pb.Ticket, error) {
	// Perform input validation.
	if req.Ticket == nil {
		return nil, status.Error(codes.InvalidArgument, ".ticket is required")
	}
	if req.Ticket.Assignment != nil {
		return nil, status.Error(codes.InvalidArgument, "tickets cannot be created with an assignment")
	}
	if req.Ticket.CreateTime != nil {
		return nil, status.Error(codes.InvalidArgument, "tickets cannot be created with create time set")
	}

	if len(req.Ticket.Id) == 0 {
		req.Ticket.Id = xid.New().String()
	}

	// Call out to validate
	err := s.customlogicManager.Validate(ctx, req.Ticket)
	if err != nil {
		return nil, err
	}

	// Call out to gather data
	t, err := s.customlogicManager.GatherData(ctx, req.Ticket)
	if err != nil {
		return nil, err
	}

	t, err = s.ticketsManager.CreateTicket(ctx, t, s.filterManager)
	if err != nil {
		if errors.Is(err, tickets.ErrDuplicateTicketID) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, err
	}

	return t, nil
}

// DeleteTicket immediately stops Open Match from using the Ticket for matchmaking and removes the Ticket from state storage.
func (s *Service) DeleteTicket(ctx context.Context, req *pb.DeleteTicketRequest) (*empty.Empty, error) {
	err := s.ticketsManager.DeleteTicket(ctx, req.TicketId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// GetTicket get the Ticket associated with the specified TicketId.
func (s *Service) GetTicket(ctx context.Context, req *pb.GetTicketRequest) (*pb.Ticket, error) {
	return s.ticketsManager.GetTicket(ctx, req.TicketId)
}

// WatchAssignments stream back Assignment of the specified TicketId if it is updated.
func (s *Service) WatchAssignments(req *pb.WatchAssignmentsRequest, svr pb.FrontendService_WatchAssignmentsServer) error {
	ctx, cancel := context.WithCancel(context.Background())
	done := false
	processTicket := func(ctx context.Context, t *pb.Ticket) {
		if t.Assignment != nil {
			_ = backoff.Retry(
				ctx,
				func() error {
					return svr.Send(&pb.WatchAssignmentsResponse{Assignment: t.Assignment})
				},
				backoff.Constant(time.Second/10),
			)
			cancel()
			done = true
		}
	}

	t, err := s.ticketsManager.GetTicket(svr.Context(), req.TicketId)
	if err != nil {
		return err
	}
	processTicket(svr.Context(), t)
	if done {
		return nil
	}

	return s.ticketsManager.WatchTicket(ctx, req.TicketId, processTicket)
}

// AcknowledgeBackfill is used to notify OpenMatch about GameServer connection info
func (s *Service) AcknowledgeBackfill(ctx context.Context, req *pb.AcknowledgeBackfillRequest) (*pb.Backfill, error) {
	return s.backfillManager.AcknowledgeBackfill(ctx, req.BackfillId, req.Assignment, s.filterManager)
}

// CreateBackfill creates a new Backfill object.
func (s *Service) CreateBackfill(ctx context.Context, req *pb.CreateBackfillRequest) (*pb.Backfill, error) {
	return s.backfillManager.CreateBackfill(ctx, req.Backfill, s.filterManager)
}

// DeleteBackfill receives a backfill ID and deletes its resource.
func (s *Service) DeleteBackfill(ctx context.Context, req *pb.DeleteBackfillRequest) (*empty.Empty, error) {
	err := s.backfillManager.DeleteBackfill(ctx, req.BackfillId, s.filterManager)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// GetBackfill returns a backfill object by its ID.
func (s *Service) GetBackfill(ctx context.Context, req *pb.GetBackfillRequest) (*pb.Backfill, error) {
	return s.backfillManager.GetBackfill(ctx, req.BackfillId)
}

// UpdateBackfill updates search_fields and extensions for the backfill with the provided id.
func (s *Service) UpdateBackfill(ctx context.Context, req *pb.UpdateBackfillRequest) (*pb.Backfill, error) {
	return s.backfillManager.UpdateBackfill(ctx, req.Backfill, s.filterManager)
}
