// Package logging records categorized, leveled diagnostics for the game and
// its tools, and always keeps warnings and errors.
package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger is the main logging interface that handles categorized, leveled logging
type Logger struct {
	mu             sync.RWMutex
	enabled        bool            // Global logging enabled/disabled
	categoryStates map[string]bool // Per-category enabled/disabled state
	minLevel       LogLevel        // Minimum level to log
	outputToStdout bool            // Whether to output to stdout (controlled by debug command)
	// diagnostics receives every WARN and ERROR line, whatever the switches say,
	// so a player's bug report can include them.
	diagnostics  io.Writer
	entries      []Entry
	entryStart   int
	entryCount   int
	nextSequence uint64
}

// NewLogger creates a new logger instance with default settings
func NewLogger() *Logger {
	return newLoggerWithCapacity(defaultEntryCapacity)
}

func newLoggerWithCapacity(capacity int) *Logger {
	if capacity <= 0 {
		panic("logging entry capacity must be positive")
	}

	return &Logger{
		mu:             sync.RWMutex{},
		enabled:        false, // Disabled by default as requested
		categoryStates: make(map[string]bool),
		minLevel:       DEBUG, // Allow all levels by default
		outputToStdout: false, // Controlled by debug command
		diagnostics:    nil,
		entries:        make([]Entry, capacity),
		entryStart:     0,
		entryCount:     0,
		nextSequence:   0,
	}
}

// Global logger instance
//
//nolint:gochecknoglobals // The single process-wide sink behind the package logging functions.
var globalLogger = NewLogger()

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}

// SetEnabled sets the global logging enabled state
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// IsEnabled returns the global logging enabled state
func (l *Logger) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.enabled
}

// SetCategoryEnabled sets the enabled state for a specific category
func (l *Logger) SetCategoryEnabled(category string, enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.categoryStates[category] = enabled
}

// SetStdoutEnabled sets whether logs should be output to stdout
func (l *Logger) SetStdoutEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outputToStdout = enabled
}

// IsStdoutEnabled returns whether stdout logging is enabled
func (l *Logger) IsStdoutEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.outputToStdout
}

// SetMinLevel sets the minimum log level to output
func (l *Logger) SetMinLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// GetMinLevel returns the current minimum log level
func (l *Logger) GetMinLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.minLevel
}

// SetDiagnosticsWriter selects where WARN and ERROR lines are always written.
func (l *Logger) SetDiagnosticsWriter(writer io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.diagnostics = writer
}

// Logf logs a message with the specified level and category. WARN and ERROR
// are always recorded; the switches only filter DEBUG and INFO and stdout.
func (l *Logger) Logf(level LogLevel, category string, format string, args ...interface{}) {
	timestamp := time.Now()
	l.mu.Lock()
	selected := l.enabled && level >= l.minLevel && l.categoryStates[category]
	if !selected && level < WARN {
		l.mu.Unlock()

		return
	}
	message := fmt.Sprintf(format, args...)
	l.appendEntryLocked(timestamp, level, category, message)
	logLine := fmt.Sprintf("[%s] %s [%s] %s",
		timestamp.Format("2006-01-02 15:04:05.000"), level.String(), strings.ToUpper(category), message)
	if level >= WARN && l.diagnostics != nil {
		if _, err := fmt.Fprintln(l.diagnostics, logLine); err != nil {
			// Stop writing to a failed log rather than disturbing every caller.
			l.diagnostics = nil
		}
	}
	outputToStdout := selected && l.outputToStdout
	l.mu.Unlock()

	if outputToStdout {
		if _, err := fmt.Fprintln(os.Stdout, logLine); err != nil {
			l.SetStdoutEnabled(false) // Stop echoing to an unusable stdout.
		}
	}
}

// Debugf logs a debug message
func (l *Logger) Debugf(category string, format string, args ...interface{}) {
	l.Logf(DEBUG, category, format, args...)
}

// Infof logs an info message
func (l *Logger) Infof(category string, format string, args ...interface{}) {
	l.Logf(INFO, category, format, args...)
}

// Warnf logs a warning message
func (l *Logger) Warnf(category string, format string, args ...interface{}) {
	l.Logf(WARN, category, format, args...)
}

// Errorf logs an error message
func (l *Logger) Errorf(category string, format string, args ...interface{}) {
	l.Logf(ERROR, category, format, args...)
}

// GetCategoryStates returns the current state of all categories for saving
func (l *Logger) GetCategoryStates() map[string]bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	states := make(map[string]bool)
	for category, enabled := range l.categoryStates {
		states[category] = enabled
	}

	return states
}

// SetCategoryStates sets the state of all categories (for loading from config)
func (l *Logger) SetCategoryStates(states map[string]bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Clear existing states
	l.categoryStates = make(map[string]bool)

	// Set new states
	for category, enabled := range states {
		l.categoryStates[category] = enabled
	}
}

// Package-level convenience functions that use the global logger

// SetGlobalEnabled sets the global logging enabled state
func SetGlobalEnabled(enabled bool) {
	globalLogger.SetEnabled(enabled)
}

// SetCategoryEnabled sets a category's enabled state
func SetCategoryEnabled(category string, enabled bool) {
	globalLogger.SetCategoryEnabled(category, enabled)
}

// SetDiagnosticsWriter selects where the global logger always writes WARN and ERROR.
func SetDiagnosticsWriter(writer io.Writer) {
	globalLogger.SetDiagnosticsWriter(writer)
}

// SetStdoutEnabled sets stdout logging state
func SetStdoutEnabled(enabled bool) {
	globalLogger.SetStdoutEnabled(enabled)
}

// Debugf logs a debug message using the global logger
func Debugf(category string, format string, args ...interface{}) {
	globalLogger.Debugf(category, format, args...)
}

// Infof logs an info message using the global logger
func Infof(category string, format string, args ...interface{}) {
	globalLogger.Infof(category, format, args...)
}

// Warnf logs a warning message using the global logger
func Warnf(category string, format string, args ...interface{}) {
	globalLogger.Warnf(category, format, args...)
}

// Errorf logs an error message using the global logger
func Errorf(category string, format string, args ...interface{}) {
	globalLogger.Errorf(category, format, args...)
}
