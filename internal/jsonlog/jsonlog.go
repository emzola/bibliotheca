// Package jsonlog implements structured JSON log entries with different severity levels.
// Only entries above a minimum severity level are logged.
package jsonlog

import (
	"encoding/json"
	"io"
	"runtime/debug"
	"sync"
	"time"
)

// Level represents the type for the severity of a log entry.
type Level int8

// The severity type is one of these levels.
const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

// String returns a human-friendly string for the severity level.
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Logger holds the output destination that the log entries will be written to,
// the minimum severity level that log entries will be wrtten for,
// and a mutex for coordinating the writes.
type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

// New returns a new logger instance which writes logs at or above a severity level
// to a specific output destination.
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	if level < l.minLevel {
		return 0, nil
	}
	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC1123),
		Message:    message,
		Properties: properties,
	}
	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message: " + err.Error())
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out.Write(append(line, '\n'))
}

// Write is a method implemented so that the Logger type satisfies the io.Writer interface.
// This writes a log entry at the ERROR level with no additional properties.
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelError, string(message), nil)
}

// PrintInfo writes log entries at the INFO level.
func (l *Logger) PrintInfo(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

// PrintError writes log entries at the ERROR level.
func (l *Logger) PrintError(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}

// PrintFatal writes log entries at the FATAL level.
func (l *Logger) PrintFatal(err error, properties map[string]string) {
	l.print(LevelFatal, err.Error(), properties)
}
