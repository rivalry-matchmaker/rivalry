package filter_test

import (
	"testing"

	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var profile = &pb.MatchProfile{Name: "everybody", Pools: []*pb.Pool{
	{Name: "pool_everybody", TagPresentFilters: []*pb.TagPresentFilter{{Tag: "test"}}}}}

type FilterTestSuite struct {
	suite.Suite
	manager filter.Manager
}

func (s *FilterTestSuite) SetupTest() {
	s.manager = filter.NewManager([]*pb.MatchProfile{profile})
}

func (s *FilterTestSuite) TestFilterMatch() {
	t := &pb.Ticket{Id: xid.New().String(), SearchFields: &pb.SearchFields{
		Tags: []string{"test"}}}
	profileName, pool, err := s.manager.ProfileMembershipTest(t)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "everybody", profileName)
	assert.Equal(s.T(), "pool_everybody", pool)
}

func (s *FilterTestSuite) TestFilterNoMatch() {
	t := &pb.Ticket{Id: xid.New().String()}
	profileName, pool, err := s.manager.ProfileMembershipTest(t)
	assert.Error(s.T(), err)
	assert.Empty(s.T(), profileName)
	assert.Empty(s.T(), pool)
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}
