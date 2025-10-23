// Package logger provides a small structured JSON logger used across the app.
package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Level represents the log level.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Logger provides structured logging functionality.
type Logger struct {
	level  Level
	logger *log.Logger
}

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     Level                  `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// New creates a new Logger instance.
func New(level Level) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// shouldLog determines if a message should be logged based on the logger's level.
func (l *Logger) shouldLog(level Level) bool {
	levels := map[Level]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}

	return levels[level] >= levels[l.level]
}

// log writes a structured log entry.
func (l *Logger) log(level Level, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonData))
}

// Debug logs a debug message with optional fields.
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelDebug, message, f)
}

// Info logs an info message with optional fields.
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelInfo, message, f)
}

// Warn logs a warning message with optional fields.
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelWarn, message, f)
}

// Error logs an error message with optional fields.
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelError, message, f)
}

// WithFields returns a new Logger with additional context fields.
func (l *Logger) WithFields(fields map[string]interface{}) *ContextLogger {
	return &ContextLogger{
		logger: l,
		fields: fields,
	}
}

// ContextLogger wraps Logger with additional context fields.
type ContextLogger struct {
	logger *Logger
	fields map[string]interface{}
}

// mergeFields combines context fields with additional fields.
func (cl *ContextLogger) mergeFields(additional map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy context fields
	for k, v := range cl.fields {
		merged[k] = v
	}

	// Add additional fields (override context fields if needed)
	for k, v := range additional {
		merged[k] = v
	}

	return merged
}

// Debug logs a debug message with context and optional additional fields.
func (cl *ContextLogger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = cl.mergeFields(fields[0])
	} else {
		f = cl.fields
	}
	cl.logger.log(LevelDebug, message, f)
}

// Info logs an info message with context and optional additional fields.
func (cl *ContextLogger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = cl.mergeFields(fields[0])
	} else {
		f = cl.fields
	}
	cl.logger.log(LevelInfo, message, f)
}

// Warn logs a warning message with context and optional additional fields.
func (cl *ContextLogger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = cl.mergeFields(fields[0])
	} else {
		f = cl.fields
	}
	cl.logger.log(LevelWarn, message, f)
}

// Error logs an error message with context and optional additional fields.
func (cl *ContextLogger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = cl.mergeFields(fields[0])
	} else {
		f = cl.fields
	}
	cl.logger.log(LevelError, message, f)
}

// Global logger instance
var defaultLogger = New(LevelInfo)

// SetLevel sets the global logger level.
func SetLevel(level Level) {
	defaultLogger.level = level
}

// Global logging functions
func Debug(message string, fields ...map[string]interface{}) {
	defaultLogger.Debug(message, fields...)
}

func Info(message string, fields ...map[string]interface{}) {
	defaultLogger.Info(message, fields...)
}

func Warn(message string, fields ...map[string]interface{}) {
	defaultLogger.Warn(message, fields...)
}

func Error(message string, fields ...map[string]interface{}) {
	defaultLogger.Error(message, fields...)
}

func WithFields(fields map[string]interface{}) *ContextLogger {
	return defaultLogger.WithFields(fields)
}
