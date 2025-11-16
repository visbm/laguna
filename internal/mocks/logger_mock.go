package mocks

import "laguna/common/logger"

type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields ...logger.Field) {}
func (m *MockLogger) Info(msg string, fields ...logger.Field)  {}
func (m *MockLogger) Warn(msg string, fields ...logger.Field)  {}
func (m *MockLogger) Error(msg string, fields ...logger.Field) {}
func (m *MockLogger) Fatal(msg string, fields ...logger.Field) {}
func (m *MockLogger) Sync() error                              { return nil }
