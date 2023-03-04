package frontend_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rivalry-matchmaker/rivalry/internal/app/frontend"
	ticketsMock "github.com/rivalry-matchmaker/rivalry/internal/managers/tickets/mock"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = context.Background()
)

type FrontendTestSuite struct {
	suite.Suite
	service        *frontend.Service
	ticketsManager *ticketsMock.MockManager
}

func (s *FrontendTestSuite) SetupTest() {
	s.ticketsManager = ticketsMock.NewMockManager(gomock.NewController(s.T()))
	s.service = frontend.NewService([]string{"default"}, s.ticketsManager)
}

func (s *FrontendTestSuite) TestClose() {
	s.ticketsManager.EXPECT().Close()
	s.service.Close()
}

func TestFrontendTestSuite(t *testing.T) {
	suite.Run(t, new(FrontendTestSuite))
}
