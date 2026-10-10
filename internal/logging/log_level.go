package logging

// LogLevel represents the importance/severity of a log message
type LogLevel int

const (
	// DEBUG level for detailed diagnostic information, typically only visible during development
	DEBUG LogLevel = iota

	// INFO level for general informational messages that highlight application progress
	INFO

	// WARN level for potentially harmful situations that don't prevent operation
	WARN

	// ERROR level for error events that still allow the application to continue
	ERROR

	// FATAL level for severe error events that will likely abort the application
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO // Default to INFO for unknown levels
	}
}
