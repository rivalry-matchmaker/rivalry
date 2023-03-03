package customlogic

//go:generate mockgen -package mock -destination=mock/customlogic.go . MatchmakerManager,AssignmentManager

import (
	"context"
	"io"

	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
)

// MatchmakerManager provides the interface for calling custom matchmaker logic
type MatchmakerManager interface {
	MakeMatches(ctx context.Context, req *api.MakeMatchesRequest, f func(ctx context.Context, match *api.Match)) error
}

type matchmakerManager struct {
	matchMakerClient api.MatchMakerServiceClient
}

// NewMatchmakerManager returns a new MatchmakerManager
func NewMatchmakerManager(matchMakerClient api.MatchMakerServiceClient) MatchmakerManager {
	return &matchmakerManager{
		matchMakerClient: matchMakerClient,
	}
}

// MakeMatches calls the MakeMatches custom logic
func (m *matchmakerManager) MakeMatches(ctx context.Context, req *api.MakeMatchesRequest, f func(ctx context.Context, match *api.Match)) error {
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
	MakeAssignment(ctx context.Context, match *api.Match) (*api.GameServer, error)
}

type assignmentManager struct {
	assignmentClient api.AssignmentServiceClient
}

// NewAssignmentManager returns a new AssignmentManager
func NewAssignmentManager(assignmentClient api.AssignmentServiceClient) AssignmentManager {
	return &assignmentManager{
		assignmentClient: assignmentClient,
	}
}

// MakeAssignment calls the MakeAssignment custom logic
func (m *assignmentManager) MakeAssignment(ctx context.Context, match *api.Match) (*api.GameServer, error) {
	resp, err := m.assignmentClient.MakeAssignment(ctx, &api.MakeAssignmentRequest{Match: match})
	if err != nil {
		return nil, err
	}
	return resp.GameServer, nil
}
