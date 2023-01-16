package filter

import (
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Manager provides an interface for filtering FilteredEntity into MatchProfile and Pools
type Manager interface {
	ProfileMembershipTest(fe FilteredEntity) (string, string, error)
}

type manager struct {
	profiles []*pb.MatchProfile
}

// NewManager returns a filter Manager
func NewManager(profiles []*pb.MatchProfile) Manager {
	return &manager{
		profiles: profiles,
	}
}

// ProfileMembershipTest tests for MatchProfile and Pools membership
func (m *manager) ProfileMembershipTest(fe FilteredEntity) (string, string, error) {
	for _, profile := range m.profiles {
		for _, pool := range profile.Pools {
			f, err := NewPoolFilter(pool)
			if err != nil {
				return "", "", err
			}
			if f.In(fe) {
				return profile.Name, pool.Name, nil
			}
		}
	}
	return "", "", status.Error(codes.InvalidArgument, "ticket does not match criteria")
}
