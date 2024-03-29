// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic (interfaces: MatchmakerManager,AssignmentManager)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
)

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
func (m *MockMatchmakerManager) MakeMatches(arg0 context.Context, arg1 *v1.MakeMatchesRequest, arg2 func(context.Context, *v1.Match)) error {
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
func (m *MockAssignmentManager) MakeAssignment(arg0 context.Context, arg1 *v1.Match) (*v1.GameServer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeAssignment", arg0, arg1)
	ret0, _ := ret[0].(*v1.GameServer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MakeAssignment indicates an expected call of MakeAssignment.
func (mr *MockAssignmentManagerMockRecorder) MakeAssignment(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeAssignment", reflect.TypeOf((*MockAssignmentManager)(nil).MakeAssignment), arg0, arg1)
}
