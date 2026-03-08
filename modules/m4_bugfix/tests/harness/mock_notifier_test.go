//go:build m4harness

// Code generated for test harness. DO NOT EDIT.
package result

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockRiskNotifier is a mock of RiskNotifier interface.
type MockRiskNotifier struct {
	ctrl     *gomock.Controller
	recorder *MockRiskNotifierMockRecorder
}

// MockRiskNotifierMockRecorder is the mock recorder for MockRiskNotifier.
type MockRiskNotifierMockRecorder struct {
	mock *MockRiskNotifier
}

// NewMockRiskNotifier creates a new mock instance.
func NewMockRiskNotifier(ctrl *gomock.Controller) *MockRiskNotifier {
	mock := &MockRiskNotifier{ctrl: ctrl}
	mock.recorder = &MockRiskNotifierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRiskNotifier) EXPECT() *MockRiskNotifierMockRecorder {
	return m.recorder
}

// NotifyHighRisk mocks base method.
func (m *MockRiskNotifier) NotifyHighRisk(ctx context.Context, day string, amount int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NotifyHighRisk", ctx, day, amount)
	ret0, _ := ret[0].(error)
	return ret0
}

// NotifyHighRisk indicates an expected call of NotifyHighRisk.
func (mr *MockRiskNotifierMockRecorder) NotifyHighRisk(ctx, day, amount any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"NotifyHighRisk",
		reflect.TypeOf((*MockRiskNotifier)(nil).NotifyHighRisk),
		ctx,
		day,
		amount,
	)
}
