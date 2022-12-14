// Code generated by MockGen. DO NOT EDIT.
// Source: rivalry/internal/managers/customlogic (interfaces: FrontendManager,MatchmakerManager,AssignmentManager)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockFrontendManager is a mock of FrontendManager interface.
type MockFrontendManager struct {
	ctrl     *gomock.Controller
	recorder *MockFrontendManagerMockRecorder
}

// MockFrontendManagerMockRecorder is the mock recorder for MockFrontendManager.
type MockFrontendManagerMockRecorder struct {
	mock *MockFrontendManager
}

// NewMockFrontendManager creates a new mock instance.
func NewMockFrontendManager(ctrl *gomock.Controller) *MockFrontendManager {
	mock := &MockFrontendManager{ctrl: ctrl}
	mock.recorder = &MockFrontendManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFrontendManager) EXPECT() *MockFrontendManagerMockRecorder {
	return m.recorder
}

// GatherData mocks base method.
func (m *MockFrontendManager) GatherData(arg0 context.Context, arg1 *pb.Ticket) (*pb.Ticket, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GatherData", arg0, arg1)
	ret0, _ := ret[0].(*pb.Ticket)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GatherData indicates an expected call of GatherData.
func (mr *MockFrontendManagerMockRecorder) GatherData(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GatherData", reflect.TypeOf((*MockFrontendManager)(nil).GatherData), arg0, arg1)
}

// Validate mocks base method.
func (m *MockFrontendManager) Validate(arg0 context.Context, arg1 *pb.Ticket) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Validate", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Validate indicates an expected call of Validate.
func (mr *MockFrontendManagerMockRecorder) Validate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Validate", reflect.TypeOf((*MockFrontendManager)(nil).Validate), arg0, arg1)
}

// MockMatchmakerManager is a mock of MatchmakerManager interface.
type MockMatchmakerManager struct {
	ctrl     *gomock.Controller
	recorder *MockMatchmakerManagerMockRecorder
}

// MockMatchmakerManagerMockRecorder is the mock recorder for MockMatchmakerManager.
type MockMatchmakerManagerMockRecorder struct {
	mock *MockMatchmakerManager
}

// NewMockMatchmakerManager creates a new mock instance.
func NewMockMatchmakerManager(ctrl *gomock.Controller) *MockMatchmakerManager {
	mock := &MockMatchmakerManager{ctrl: ctrl}
	mock.recorder = &MockMatchmakerManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMatchmakerManager) EXPECT() *MockMatchmakerManagerMockRecorder {
	return m.recorder
}

// MakeMatches mocks base method.
func (m *MockMatchmakerManager) MakeMatches(arg0 context.Context, arg1 *pb.MakeMatchesRequest, arg2 func(context.Context, *pb.Match)) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeMatches", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// MakeMatches indicates an expected call of MakeMatches.
func (mr *MockMatchmakerManagerMockRecorder) MakeMatches(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeMatches", reflect.TypeOf((*MockMatchmakerManager)(nil).MakeMatches), arg0, arg1, arg2)
}

// MockAssignmentManager is a mock of AssignmentManager interface.
type MockAssignmentManager struct {
	ctrl     *gomock.Controller
	recorder *MockAssignmentManagerMockRecorder
}

// MockAssignmentManagerMockRecorder is the mock recorder for MockAssignmentManager.
type MockAssignmentManagerMockRecorder struct {
	mock *MockAssignmentManager
}

// NewMockAssignmentManager creates a new mock instance.
func NewMockAssignmentManager(ctrl *gomock.Controller) *MockAssignmentManager {
	mock := &MockAssignmentManager{ctrl: ctrl}
	mock.recorder = &MockAssignmentManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAssignmentManager) EXPECT() *MockAssignmentManagerMockRecorder {
	return m.recorder
}

// MakeAssignment mocks base method.
func (m *MockAssignmentManager) MakeAssignment(arg0 context.Context, arg1 *pb.Match) (*pb.Assignment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeAssignment", arg0, arg1)
	ret0, _ := ret[0].(*pb.Assignment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MakeAssignment indicates an expected call of MakeAssignment.
func (mr *MockAssignmentManagerMockRecorder) MakeAssignment(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeAssignment", reflect.TypeOf((*MockAssignmentManager)(nil).MakeAssignment), arg0, arg1)
}
