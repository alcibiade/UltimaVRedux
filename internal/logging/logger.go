package logging

import (
	"fmt"
	"log"
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
	outputToFile   bool            // Whether to output to file (future feature)
	entries        []Entry
	entryStart     int
	entryCount     int
	nextSequence   uint64
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
		enabled:        false, // Disabled by default as requested
		categoryStates: make(map[string]bool),
		minLevel:       DEBUG, // Allow all levels by default
		outputToStdout: false, // Controlled by debug command
		outputToFile:   false, // Future feature
		entries:        make([]Entry, capacity),
		entryStart:     0,
		entryCount:     0,
		nextSequence:   0,
	}
}

// Global logger instance
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

// Log logs a message with the specified level and category
func (l *Logger) Log(level LogLevel, category string, format string, args ...interface{}) {
	timestamp := time.Now()
	l.mu.Lock()
	if !l.enabled || level < l.minLevel || !l.categoryStates[category] {
		l.mu.Unlock()

		return
	}
	message := fmt.Sprintf(format, args...)
	l.appendEntryLocked(timestamp, level, category, message)
	outputToStdout := l.outputToStdout
	l.mu.Unlock()
	logLine := fmt.Sprintf("[%s] %s [%s] %s", timestamp.Format("2006-01-02 15:04:05.000"), level.String(), strings.ToUpper(category), message)

	// Output to stdout if enabled
	if outputToStdout {
		fmt.Println(logLine)
	}

	// For FATAL level, also use the standard log package for immediate visibility
	if level == FATAL {
		log.Printf("FATAL [%s] %s", strings.ToUpper(category), message)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(category string, format string, args ...interface{}) {
	l.Log(DEBUG, category, format, args...)
}

// Info logs an info message
func (l *Logger) Info(category string, format string, args ...interface{}) {
	l.Log(INFO, category, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(category string, format string, args ...interface{}) {
	l.Log(WARN, category, format, args...)
}

// Error logs an error message
func (l *Logger) Error(category string, format string, args ...interface{}) {
	l.Log(ERROR, category, format, args...)
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

// SetStdoutEnabled sets stdout logging state
func SetStdoutEnabled(enabled bool) {
	globalLogger.SetStdoutEnabled(enabled)
}

// Debug logs a debug message using the global logger
func Debug(category string, format string, args ...interface{}) {
	globalLogger.Debug(category, format, args...)
}

// Info logs an info message using the global logger
func Info(category string, format string, args ...interface{}) {
	globalLogger.Info(category, format, args...)
}

// Warn logs a warning message using the global logger
func Warn(category string, format string, args ...interface{}) {
	globalLogger.Warn(category, format, args...)
}

// Error logs an error message using the global logger
func Error(category string, format string, args ...interface{}) {
	globalLogger.Error(category, format, args...)
}
