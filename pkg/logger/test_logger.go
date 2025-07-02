package logger

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

// TestLogger is a logger implementation for testing that captures all log messages
type TestLogger struct {
	mu       sync.Mutex
	messages []LogMessage
	buffer   *bytes.Buffer
	zerolog  *zerolog.Logger
}

// LogMessage represents a captured log message
type LogMessage struct {
	Level   string
	Message string
	Fields  map[string]interface{}
	Error   error
}

// NewTestLogger creates a new test logger
func NewTestLogger() *TestLogger {
	nopLogger := zerolog.Nop()
	return &TestLogger{
		messages: make([]LogMessage, 0),
		buffer:   &bytes.Buffer{},
		zerolog:  &nopLogger,
	}
}

// Debug logs a debug message
func (l *TestLogger) Debug(msg string) {
	l.log("DEBUG", msg, nil, nil)
}

// DebugWithFields logs a debug message with fields
func (l *TestLogger) DebugWithFields(msg string, fields map[string]interface{}) {
	l.log("DEBUG", msg, fields, nil)
}

// Info logs an info message
func (l *TestLogger) Info(msg string) {
	l.log("INFO", msg, nil, nil)
}

// InfoWithFields logs an info message with fields
func (l *TestLogger) InfoWithFields(msg string, fields map[string]interface{}) {
	l.log("INFO", msg, fields, nil)
}

// Warn logs a warning message
func (l *TestLogger) Warn(msg string) {
	l.log("WARN", msg, nil, nil)
}

// WarnWithFields logs a warning message with fields
func (l *TestLogger) WarnWithFields(msg string, fields map[string]interface{}) {
	l.log("WARN", msg, fields, nil)
}

// Error logs an error message
func (l *TestLogger) Error(msg string) {
	l.log("ERROR", msg, nil, nil)
}

// ErrorWithFields logs an error message with fields
func (l *TestLogger) ErrorWithFields(msg string, fields map[string]interface{}) {
	l.log("ERROR", msg, fields, nil)
}

// WithError adds an error to the logger context
func (l *TestLogger) WithError(err error) Logger {
	return &testLoggerWithError{TestLogger: l, err: err}
}

// WithField adds a field to the logger context
func (l *TestLogger) WithField(key string, value interface{}) Logger {
	return &testLoggerWithFields{
		TestLogger: l,
		fields:     map[string]interface{}{key: value},
	}
}

// WithFields adds multiple fields to the logger context
func (l *TestLogger) WithFields(fields map[string]interface{}) Logger {
	return &testLoggerWithFields{
		TestLogger: l,
		fields:     fields,
	}
}

// Fatal logs a fatal message
func (l *TestLogger) Fatal(msg string) {
	l.log("FATAL", msg, nil, nil)
}

// FatalWithFields logs a fatal message with fields
func (l *TestLogger) FatalWithFields(msg string, fields map[string]interface{}) {
	l.log("FATAL", msg, fields, nil)
}

// WithContext adds context to the logger
func (l *TestLogger) WithContext(ctx context.Context) Logger {
	return l // For testing, we don't need to handle context
}

// GetZerolog returns the underlying zerolog instance
func (l *TestLogger) GetZerolog() *zerolog.Logger {
	return l.zerolog
}

// log captures a log message
func (l *TestLogger) log(level, msg string, fields map[string]interface{}, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	logMsg := LogMessage{
		Level:   level,
		Message: msg,
		Fields:  fields,
		Error:   err,
	}
	
	l.messages = append(l.messages, logMsg)
	
	// Also write to buffer for debugging
	fmt.Fprintf(l.buffer, "[%s] %s", level, msg)
	if fields != nil && len(fields) > 0 {
		fmt.Fprintf(l.buffer, " fields=%v", fields)
	}
	if err != nil {
		fmt.Fprintf(l.buffer, " error=%v", err)
	}
	fmt.Fprintln(l.buffer)
}

// GetMessages returns all captured log messages
func (l *TestLogger) GetMessages() []LogMessage {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Return a copy to avoid race conditions
	messages := make([]LogMessage, len(l.messages))
	copy(messages, l.messages)
	return messages
}

// GetMessagesByLevel returns all messages of a specific level
func (l *TestLogger) GetMessagesByLevel(level string) []LogMessage {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	var filtered []LogMessage
	for _, msg := range l.messages {
		if msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// HasMessage checks if a message with the given text was logged
func (l *TestLogger) HasMessage(text string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for _, msg := range l.messages {
		if msg.Message == text {
			return true
		}
	}
	return false
}

// HasError checks if an error was logged
func (l *TestLogger) HasError() bool {
	return len(l.GetMessagesByLevel("ERROR")) > 0
}

// Clear clears all captured messages
func (l *TestLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.messages = l.messages[:0]
	l.buffer.Reset()
}

// String returns all log messages as a string
func (l *TestLogger) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	return l.buffer.String()
}

// testLoggerWithError is a test logger with an error context
type testLoggerWithError struct {
	*TestLogger
	err error
}

func (l *testLoggerWithError) Debug(msg string) {
	l.log("DEBUG", msg, nil, l.err)
}

func (l *testLoggerWithError) DebugWithFields(msg string, fields map[string]interface{}) {
	l.log("DEBUG", msg, fields, l.err)
}

func (l *testLoggerWithError) Info(msg string) {
	l.log("INFO", msg, nil, l.err)
}

func (l *testLoggerWithError) InfoWithFields(msg string, fields map[string]interface{}) {
	l.log("INFO", msg, fields, l.err)
}

func (l *testLoggerWithError) Warn(msg string) {
	l.log("WARN", msg, nil, l.err)
}

func (l *testLoggerWithError) WarnWithFields(msg string, fields map[string]interface{}) {
	l.log("WARN", msg, fields, l.err)
}

func (l *testLoggerWithError) Error(msg string) {
	l.log("ERROR", msg, nil, l.err)
}

func (l *testLoggerWithError) ErrorWithFields(msg string, fields map[string]interface{}) {
	l.log("ERROR", msg, fields, l.err)
}

func (l *testLoggerWithError) WithError(err error) Logger {
	return &testLoggerWithError{TestLogger: l.TestLogger, err: err}
}

func (l *testLoggerWithError) WithField(key string, value interface{}) Logger {
	return &testLoggerWithFields{
		TestLogger: l.TestLogger,
		fields:     map[string]interface{}{key: value},
		err:        l.err,
	}
}

func (l *testLoggerWithError) WithFields(fields map[string]interface{}) Logger {
	return &testLoggerWithFields{
		TestLogger: l.TestLogger,
		fields:     fields,
		err:        l.err,
	}
}

func (l *testLoggerWithError) Fatal(msg string) {
	l.log("FATAL", msg, nil, l.err)
}

func (l *testLoggerWithError) FatalWithFields(msg string, fields map[string]interface{}) {
	l.log("FATAL", msg, fields, l.err)
}

func (l *testLoggerWithError) WithContext(ctx context.Context) Logger {
	return l
}

func (l *testLoggerWithError) GetZerolog() *zerolog.Logger {
	return l.TestLogger.zerolog
}

// testLoggerWithFields is a test logger with fields context
type testLoggerWithFields struct {
	*TestLogger
	fields map[string]interface{}
	err    error
}

func (l *testLoggerWithFields) Debug(msg string) {
	l.log("DEBUG", msg, l.fields, l.err)
}

func (l *testLoggerWithFields) DebugWithFields(msg string, fields map[string]interface{}) {
	merged := l.mergeFields(fields)
	l.log("DEBUG", msg, merged, l.err)
}

func (l *testLoggerWithFields) Info(msg string) {
	l.log("INFO", msg, l.fields, l.err)
}

func (l *testLoggerWithFields) InfoWithFields(msg string, fields map[string]interface{}) {
	merged := l.mergeFields(fields)
	l.log("INFO", msg, merged, l.err)
}

func (l *testLoggerWithFields) Warn(msg string) {
	l.log("WARN", msg, l.fields, l.err)
}

func (l *testLoggerWithFields) WarnWithFields(msg string, fields map[string]interface{}) {
	merged := l.mergeFields(fields)
	l.log("WARN", msg, merged, l.err)
}

func (l *testLoggerWithFields) Error(msg string) {
	l.log("ERROR", msg, l.fields, l.err)
}

func (l *testLoggerWithFields) ErrorWithFields(msg string, fields map[string]interface{}) {
	merged := l.mergeFields(fields)
	l.log("ERROR", msg, merged, l.err)
}

func (l *testLoggerWithFields) WithError(err error) Logger {
	return &testLoggerWithFields{
		TestLogger: l.TestLogger,
		fields:     l.fields,
		err:        err,
	}
}

func (l *testLoggerWithFields) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	
	return &testLoggerWithFields{
		TestLogger: l.TestLogger,
		fields:     newFields,
		err:        l.err,
	}
}

func (l *testLoggerWithFields) WithFields(fields map[string]interface{}) Logger {
	return &testLoggerWithFields{
		TestLogger: l.TestLogger,
		fields:     l.mergeFields(fields),
		err:        l.err,
	}
}

func (l *testLoggerWithFields) Fatal(msg string) {
	l.log("FATAL", msg, l.fields, l.err)
}

func (l *testLoggerWithFields) FatalWithFields(msg string, fields map[string]interface{}) {
	merged := l.mergeFields(fields)
	l.log("FATAL", msg, merged, l.err)
}

func (l *testLoggerWithFields) WithContext(ctx context.Context) Logger {
	return l
}

func (l *testLoggerWithFields) GetZerolog() *zerolog.Logger {
	return l.TestLogger.zerolog
}

func (l *testLoggerWithFields) mergeFields(additional map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range l.fields {
		merged[k] = v
	}
	for k, v := range additional {
		merged[k] = v
	}
	return merged
}