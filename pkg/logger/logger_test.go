package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"igscraper/pkg/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.LoggingConfig
		wantErr bool
	}{
		{
			name: "valid config with info level",
			cfg: &config.LoggingConfig{
				Level: "info",
			},
			wantErr: false,
		},
		{
			name: "valid config with debug level",
			cfg: &config.LoggingConfig{
				Level: "debug",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			cfg: &config.LoggingConfig{
				Level: "invalid",
			},
			wantErr: true,
		},
		{
			name: "config with file output",
			cfg: &config.LoggingConfig{
				Level: "info",
				File:  "/tmp/test.log",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("New() returned nil logger")
			}
			
			// Clean up test files
			if tt.cfg.File != "" {
				os.Remove(tt.cfg.File)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected zerolog.Level
		wantErr  bool
	}{
		{"debug", zerolog.DebugLevel, false},
		{"DEBUG", zerolog.DebugLevel, false},
		{"info", zerolog.InfoLevel, false},
		{"INFO", zerolog.InfoLevel, false},
		{"warn", zerolog.WarnLevel, false},
		{"warning", zerolog.WarnLevel, false},
		{"error", zerolog.ErrorLevel, false},
		{"fatal", zerolog.FatalLevel, false},
		{"panic", zerolog.PanicLevel, false},
		{"disabled", zerolog.Disabled, false},
		{"invalid", zerolog.InfoLevel, true},
		{"", zerolog.InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			level, err := parseLogLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLogLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if level != tt.expected {
				t.Errorf("parseLogLevel() = %v, want %v", level, tt.expected)
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	
	// Create a custom logger that writes to buffer with debug level
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zlog := zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	logger := &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}

	// Test basic logging methods
	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug message")
		if !strings.Contains(buf.String(), "debug message") {
			t.Error("Debug message not found in output")
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info("info message")
		if !strings.Contains(buf.String(), "info message") {
			t.Error("Info message not found in output")
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		logger.Warn("warn message")
		if !strings.Contains(buf.String(), "warn message") {
			t.Error("Warn message not found in output")
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		logger.Error("error message")
		if !strings.Contains(buf.String(), "error message") {
			t.Error("Error message not found in output")
		}
	})
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}

	// Test adding a single field
	newLogger := logger.WithField("key", "value")
	newLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Error("Field not found in output")
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}

	// Test adding multiple fields
	fields := map[string]interface{}{
		"string":   "value",
		"int":      42,
		"bool":     true,
		"float":    3.14,
	}
	
	newLogger := logger.WithFields(fields)
	newLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, `"string":"value"`) {
		t.Error("String field not found in output")
	}
	if !strings.Contains(output, `"int":42`) {
		t.Error("Int field not found in output")
	}
	if !strings.Contains(output, `"bool":true`) {
		t.Error("Bool field not found in output")
	}
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}

	// Test with nil error
	logger1 := logger.WithError(nil)
	if logger1 != logger {
		t.Error("WithError(nil) should return the same logger")
	}

	// Test with actual error
	testErr := &testError{msg: "test error"}
	logger2 := logger.WithError(testErr)
	logger2.Error("error occurred")

	output := buf.String()
	if !strings.Contains(output, "error occurred") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Error message not found in output")
	}
}

func TestStructuredLogging(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}

	fields := map[string]interface{}{
		"username": "john_doe",
		"action":   "download",
		"count":    10,
	}

	logger.InfoWithFields("operation completed", fields)

	output := buf.String()
	if !strings.Contains(output, "operation completed") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, `"username":"john_doe"`) {
		t.Error("Username field not found in output")
	}
	if !strings.Contains(output, `"action":"download"`) {
		t.Error("Action field not found in output")
	}
	if !strings.Contains(output, `"count":10`) {
		t.Error("Count field not found in output")
	}
}

func TestFieldTypes(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}

	// Test various field types
	fields := map[string]interface{}{
		"string":   "test",
		"int":      123,
		"int64":    int64(456),
		"float":    3.14,
		"bool":     true,
		"time":     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		"duration": time.Second * 5,
		"strings":  []string{"a", "b", "c"},
		"ints":     []int{1, 2, 3},
		"custom":   struct{ Name string }{Name: "test"},
	}

	logger.WithFields(fields).Info("test all types")

	output := buf.String()
	if !strings.Contains(output, "test all types") {
		t.Error("Message not found in output")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Initialize global logger
	cfg := &config.LoggingConfig{
		Level: "debug",
	}
	
	err := Initialize(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Test global logger functions
	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger() returned nil")
	}

	// Test convenience functions (just ensure they don't panic)
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	
	WithField("key", "value").Info("with field")
	WithFields(map[string]interface{}{"k1": "v1", "k2": "v2"}).Info("with fields")
	WithError(&testError{msg: "test"}).Error("with error")
}

func TestFieldChaining(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}

	// Test field chaining
	logger.
		WithField("field1", "value1").
		WithField("field2", "value2").
		WithFields(map[string]interface{}{
			"field3": "value3",
			"field4": 4,
		}).
		Info("chained fields")

	output := buf.String()
	if !strings.Contains(output, "chained fields") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, `"field1":"value1"`) {
		t.Error("Field1 not found in output")
	}
	if !strings.Contains(output, `"field2":"value2"`) {
		t.Error("Field2 not found in output")
	}
	if !strings.Contains(output, `"field3":"value3"`) {
		t.Error("Field3 not found in output")
	}
	if !strings.Contains(output, `"field4":4`) {
		t.Error("Field4 not found in output")
	}
}

// Helper error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}