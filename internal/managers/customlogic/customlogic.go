package customlogic

//go:generate mockgen -package mock -destination=mock/customlogic.go . FrontendManager,MatchmakerManager,AssignmentManager

import (
	"context"
	"io"

	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FrontendManager provides the interface for calling custom frontend logic
type FrontendManager interface {
	Validate(ctx context.Context, ticket *pb.Ticket) error
	GatherData(ctx context.Context, ticket *pb.Ticket) (*pb.Ticket, error)
}

type frontendManager struct {
	validationClient pb.ValidationServiceClient
	dataClient       pb.DataServiceClient
}

// NewFrontendManager returns a new FrontendManager
func NewFrontendManager(validationClient pb.ValidationServiceClient,
	dataClient pb.DataServiceClient) FrontendManager {
	return &frontendManager{
		validationClient: validationClient,
		dataClient:       dataClient,
	}
}

// Validate calls the Validate custom logic
func (m *frontendManager) Validate(ctx context.Context, ticket *pb.Ticket) error {
	if m.validationClient == nil {
		return nil
	}
	_, err := m.validationClient.Validate(ctx, ticket)
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return err
		}
		if s.Code() != codes.Unimplemented {
			return err
		}
	}
	return nil
}

// GatherData calls the GatherData custom logic
func (m *frontendManager) GatherData(ctx context.Context, ticket *pb.Ticket) (*pb.Ticket, error) {
	if m.dataClient == nil {
		return ticket, nil
	}
	gatherNotImplemented := false
	gatherDataResp, err := m.dataClient.GatherData(ctx, ticket)
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return nil, err
		}
		switch s.Code() {
		case codes.Unimplemented:
			gatherNotImplemented = true
		case codes.FailedPrecondition:
			return nil, err
		default:
			log.Err(err).Msg("failed to call custom gather data")
			return nil, err
		}
	}

	if gatherNotImplemented {
		return ticket, nil
	}

	return &pb.Ticket{
		Id:           ticket.Id,
		SearchFields: gatherDataResp.SearchFields,
		Extensions:   gatherDataResp.Extensions,
		CreateTime:   ticket.CreateTime,
	}, nil
}

// MatchmakerManager provides the interface for calling custom matchmaker logic
type MatchmakerManager interface {
	MakeMatches(ctx context.Context, req *pb.MakeMatchesRequest, f func(ctx context.Context, match *pb.Match)) error
}

type matchmakerManager struct {
	matchMakerClient pb.MatchMakerServiceClient
}

// NewMatchmakerManager returns a new MatchmakerManager
func NewMatchmakerManager(matchMakerClient pb.MatchMakerServiceClient) MatchmakerManager {
	return &matchmakerManager{
		matchMakerClient: matchMakerClient,
	}
}

// MakeMatches calls the MakeMatches custom logic
func (m *matchmakerManager) MakeMatches(ctx context.Context, req *pb.MakeMatchesRequest, f func(ctx context.Context, match *pb.Match)) error {
	cli, err := m.matchMakerClient.MakeMatches(ctx, req)
	if err != nil {
		return err
	}
	for {
		resp, err := cli.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		f(ctx, resp.Match)
	}
}

// AssignmentManager provides the interface for calling custom assignment logic
type AssignmentManager interface {
	MakeAssignment(ctx context.Context, match *pb.Match) (*pb.Assignment, error)
}

type assignmentManager struct {
	assignmentClient pb.AssignmentServiceClient
}

// NewAssignmentManager returns a new AssignmentManager
func NewAssignmentManager(assignmentClient pb.AssignmentServiceClient) AssignmentManager {
	return &assignmentManager{
		assignmentClient: assignmentClient,
	}
}

// MakeAssignment calls the MakeAssignment custom logic
func (m *assignmentManager) MakeAssignment(ctx context.Context, match *pb.Match) (*pb.Assignment, error) {
	return m.assignmentClient.MakeAssignment(ctx, match)
}
