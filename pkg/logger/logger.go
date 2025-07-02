package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"igscraper/pkg/config"
)

// Logger defines the interface for logging operations
type Logger interface {
	// Basic logging methods
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)

	// Logging with fields
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger
	WithContext(ctx context.Context) Logger

	// Structured logging methods with fields
	DebugWithFields(msg string, fields map[string]interface{})
	InfoWithFields(msg string, fields map[string]interface{})
	WarnWithFields(msg string, fields map[string]interface{})
	ErrorWithFields(msg string, fields map[string]interface{})
	FatalWithFields(msg string, fields map[string]interface{})

	// Get the underlying zerolog instance (for advanced usage)
	GetZerolog() *zerolog.Logger
}

// zerologLogger implements the Logger interface using zerolog
type zerologLogger struct {
	logger *zerolog.Logger
	fields map[string]interface{}
}

// New creates a new Logger instance based on the provided configuration
func New(cfg *config.LoggingConfig) (Logger, error) {
	// Set up the log level
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Create the base logger with pretty console output
	var output io.Writer = os.Stdout
	
	// If console output, use pretty formatting
	if cfg.File == "" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			FieldsExclude: []string{},
			FormatLevel: func(i interface{}) string {
				if i == nil {
					return ""
				}
				level := strings.ToUpper(fmt.Sprintf("%s", i))
				switch level {
				case "DEBUG":
					return "\033[37mDEBG\033[0m" // White
				case "INFO":
					return "\033[32mINFO\033[0m"  // Green
				case "WARN":
					return "\033[33mWARN\033[0m"  // Yellow
				case "ERROR":
					return "\033[31mERRO\033[0m"  // Red
				case "FATAL":
					return "\033[35mFATL\033[0m"  // Magenta
				default:
					return level
				}
			},
			FormatMessage: func(i interface{}) string {
				if i == nil {
					return ""
				}
				return fmt.Sprintf("| %s", i)
			},
			FormatFieldName: func(i interface{}) string {
				return fmt.Sprintf("\033[36m%s\033[0m:", i) // Cyan for field names
			},
			FormatFieldValue: func(i interface{}) string {
				return fmt.Sprintf("%s", i)
			},
		}
	} else {
		// Set up file output if configured
		fileOutput, err := setupFileOutput(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to setup file output: %w", err)
		}
		
		// If both file and console output are needed, use multi-writer
		if cfg.File != "" {
			consoleWriter := zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: "15:04:05",
			}
			output = zerolog.MultiLevelWriter(consoleWriter, fileOutput)
		} else {
			output = fileOutput
		}
	}

	// Create the logger
	zlog := zerolog.New(output).With().Timestamp().Logger()

	// Add default fields
	zlog = zlog.With().
		Str("app", "igscraper").
		Str("version", "1.0.0").
		Logger()

	return &zerologLogger{
		logger: &zlog,
		fields: make(map[string]interface{}),
	}, nil
}

// setupFileOutput creates a file writer for logging
func setupFileOutput(cfg *config.LoggingConfig) (io.Writer, error) {
	// Create log directory if it doesn't exist
	dir := filepath.Dir(cfg.File)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open or create the log file
	file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Note: In a production environment, you might want to use a more sophisticated
	// file rotation mechanism like lumberjack. For now, we'll use a simple file.
	return file, nil
}

// parseLogLevel converts string log level to zerolog.Level
func parseLogLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	case "fatal":
		return zerolog.FatalLevel, nil
	case "panic":
		return zerolog.PanicLevel, nil
	case "disabled":
		return zerolog.Disabled, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// Debug logs a debug message
func (l *zerologLogger) Debug(msg string) {
	event := l.logger.Debug()
	l.addFields(event).Msg(msg)
}

// Info logs an info message
func (l *zerologLogger) Info(msg string) {
	event := l.logger.Info()
	l.addFields(event).Msg(msg)
}

// Warn logs a warning message
func (l *zerologLogger) Warn(msg string) {
	event := l.logger.Warn()
	l.addFields(event).Msg(msg)
}

// Error logs an error message
func (l *zerologLogger) Error(msg string) {
	event := l.logger.Error()
	l.addFields(event).Msg(msg)
}

// Fatal logs a fatal message and exits the application
func (l *zerologLogger) Fatal(msg string) {
	event := l.logger.Fatal()
	l.addFields(event).Msg(msg)
}

// WithField adds a single field to the logger
func (l *zerologLogger) WithField(key string, value interface{}) Logger {
	newLogger := &zerologLogger{
		logger: l.logger,
		fields: make(map[string]interface{}),
	}
	
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	
	// Add new field
	newLogger.fields[key] = value
	
	return newLogger
}

// WithFields adds multiple fields to the logger
func (l *zerologLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &zerologLogger{
		logger: l.logger,
		fields: make(map[string]interface{}),
	}
	
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	
	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	
	return newLogger
}

// WithError adds an error field to the logger
func (l *zerologLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// WithContext adds context to the logger
func (l *zerologLogger) WithContext(ctx context.Context) Logger {
	ctxLogger := l.logger.With().Ctx(ctx).Logger()
	return &zerologLogger{
		logger: &ctxLogger,
		fields: l.fields,
	}
}

// DebugWithFields logs a debug message with fields
func (l *zerologLogger) DebugWithFields(msg string, fields map[string]interface{}) {
	event := l.logger.Debug()
	l.addFieldsFromMap(event, fields).Msg(msg)
}

// InfoWithFields logs an info message with fields
func (l *zerologLogger) InfoWithFields(msg string, fields map[string]interface{}) {
	event := l.logger.Info()
	l.addFieldsFromMap(event, fields).Msg(msg)
}

// WarnWithFields logs a warning message with fields
func (l *zerologLogger) WarnWithFields(msg string, fields map[string]interface{}) {
	event := l.logger.Warn()
	l.addFieldsFromMap(event, fields).Msg(msg)
}

// ErrorWithFields logs an error message with fields
func (l *zerologLogger) ErrorWithFields(msg string, fields map[string]interface{}) {
	event := l.logger.Error()
	l.addFieldsFromMap(event, fields).Msg(msg)
}

// FatalWithFields logs a fatal message with fields and exits
func (l *zerologLogger) FatalWithFields(msg string, fields map[string]interface{}) {
	event := l.logger.Fatal()
	l.addFieldsFromMap(event, fields).Msg(msg)
}

// GetZerolog returns the underlying zerolog instance
func (l *zerologLogger) GetZerolog() *zerolog.Logger {
	return l.logger
}

// addFields adds stored fields to a zerolog event
func (l *zerologLogger) addFields(event *zerolog.Event) *zerolog.Event {
	for key, value := range l.fields {
		event = addFieldToEvent(event, key, value)
	}
	return event
}

// addFieldsFromMap adds fields from a map to a zerolog event
func (l *zerologLogger) addFieldsFromMap(event *zerolog.Event, fields map[string]interface{}) *zerolog.Event {
	// First add stored fields
	event = l.addFields(event)
	
	// Then add provided fields
	for key, value := range fields {
		event = addFieldToEvent(event, key, value)
	}
	return event
}

// addFieldToEvent adds a single field to a zerolog event with type checking
func addFieldToEvent(event *zerolog.Event, key string, value interface{}) *zerolog.Event {
	switch v := value.(type) {
	case string:
		return event.Str(key, v)
	case int:
		return event.Int(key, v)
	case int64:
		return event.Int64(key, v)
	case float64:
		return event.Float64(key, v)
	case bool:
		return event.Bool(key, v)
	case time.Time:
		return event.Time(key, v)
	case time.Duration:
		return event.Dur(key, v)
	case error:
		return event.Err(v)
	case []string:
		return event.Strs(key, v)
	case []int:
		return event.Ints(key, v)
	default:
		return event.Interface(key, v)
	}
}

// Global logger instance
var globalLogger Logger

// Initialize sets up the global logger
func Initialize(cfg *config.LoggingConfig) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	
	// Also set the global zerolog logger
	log.Logger = *logger.GetZerolog()
	
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() Logger {
	if globalLogger == nil {
		// Create a default logger if not initialized
		cfg := &config.LoggingConfig{
			Level: "info",
		}
		globalLogger, _ = New(cfg)
	}
	return globalLogger
}

// Convenience functions for the global logger

// Debug logs a debug message using the global logger
func Debug(msg string) {
	GetLogger().Debug(msg)
}

// Info logs an info message using the global logger
func Info(msg string) {
	GetLogger().Info(msg)
}

// Warn logs a warning message using the global logger
func Warn(msg string) {
	GetLogger().Warn(msg)
}

// Error logs an error message using the global logger
func Error(msg string) {
	GetLogger().Error(msg)
}

// Fatal logs a fatal message using the global logger and exits
func Fatal(msg string) {
	GetLogger().Fatal(msg)
}

// WithField adds a field to the global logger
func WithField(key string, value interface{}) Logger {
	return GetLogger().WithField(key, value)
}

// WithFields adds multiple fields to the global logger
func WithFields(fields map[string]interface{}) Logger {
	return GetLogger().WithFields(fields)
}

// WithError adds an error to the global logger
func WithError(err error) Logger {
	return GetLogger().WithError(err)
}