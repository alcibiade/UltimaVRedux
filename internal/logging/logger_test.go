package logging

import (
	"testing"
)

func TestBasicLoggingFunctionality(t *testing.T) {
	// Test creating and configuring logger
	logger := NewLogger()

	// Test initial state
	if logger.IsEnabled() {
		t.Error("Logger should be disabled by default")
	}

	// Test enabling global logging
	logger.SetEnabled(true)
	if !logger.IsEnabled() {
		t.Error("Logger should be enabled after SetEnabled(true)")
	}

	// Test category management
	if logger.IsCategoryEnabled(CategoryMoongate) {
		t.Error("Category should be disabled initially")
	}

	logger.SetCategoryEnabled(CategoryMoongate, true)
	if !logger.IsCategoryEnabled(CategoryMoongate) {
		t.Error("Category should be enabled after SetCategoryEnabled(true)")
	}

	// Test stdout control
	logger.SetStdoutEnabled(true)
	if !logger.IsStdoutEnabled() {
		t.Error("Stdout should be enabled after SetStdoutEnabled(true)")
	}

	// Test level filtering
	logger.SetMinLevel(WARN)
	if logger.GetMinLevel() != WARN {
		t.Error("Min level should be WARN after SetMinLevel(WARN)")
	}
}

func TestCategoryConstants(t *testing.T) {
	// Test that all category constants are available
	categories := GetAllCategories()
	if len(categories) == 0 {
		t.Error("GetAllCategories should return non-empty slice")
	}

	// Check that moongate category is included
	found := false
	for _, category := range categories {
		if category == CategoryMoongate {
			found = true
			break
		}
	}
	if !found {
		t.Error("CategoryMoongate should be included in GetAllCategories")
	}
}

func TestConfigurationSaveLoad(t *testing.T) {
	logger := NewLogger()

	// Configure logger
	logger.SetEnabled(true)
	logger.SetStdoutEnabled(true)
	logger.SetMinLevel(INFO)
	logger.SetCategoryEnabled(CategoryMoongate, true)
	logger.SetCategoryEnabled(CategoryConversation, false)

	// Save configuration
	config := logger.SaveConfigurationToMap()

	// Create new logger and load configuration
	newLogger := NewLogger()
	newLogger.LoadConfigurationFromMap(config)

	// Verify configuration was loaded correctly
	if !newLogger.IsEnabled() {
		t.Error("Enabled state should be preserved")
	}
	if !newLogger.IsStdoutEnabled() {
		t.Error("Stdout enabled state should be preserved")
	}
	if newLogger.GetMinLevel() != INFO {
		t.Error("Min level should be preserved")
	}
	if !newLogger.IsCategoryEnabled(CategoryMoongate) {
		t.Error("Moongate category enabled state should be preserved")
	}
	if newLogger.IsCategoryEnabled(CategoryConversation) {
		t.Error("Conversation category disabled state should be preserved")
	}
}

func TestLogLevels(t *testing.T) {
	// Test log level parsing
	if ParseLogLevel("DEBUG") != DEBUG {
		t.Error("ParseLogLevel should parse DEBUG correctly")
	}
	if ParseLogLevel("INFO") != INFO {
		t.Error("ParseLogLevel should parse INFO correctly")
	}
	if ParseLogLevel("WARN") != WARN {
		t.Error("ParseLogLevel should parse WARN correctly")
	}
	if ParseLogLevel("ERROR") != ERROR {
		t.Error("ParseLogLevel should parse ERROR correctly")
	}
	if ParseLogLevel("FATAL") != FATAL {
		t.Error("ParseLogLevel should parse FATAL correctly")
	}

	// Test log level string conversion
	if DEBUG.String() != "DEBUG" {
		t.Error("DEBUG.String() should return 'DEBUG'")
	}
	if INFO.String() != "INFO" {
		t.Error("INFO.String() should return 'INFO'")
	}
}

func TestGlobalLoggerFunctions(t *testing.T) {
	// Test global logger convenience functions
	SetGlobalEnabled(true)
	if !IsGlobalEnabled() {
		t.Error("Global logger should be enabled")
	}

	SetCategoryEnabled(CategoryMoongate, true)
	if !IsCategoryEnabled(CategoryMoongate) {
		t.Error("Global logger category should be enabled")
	}

	SetStdoutEnabled(true)
	if !IsStdoutEnabled() {
		t.Error("Global logger stdout should be enabled")
	}

	// Test logging functions don't panic (basic smoke test)
	Debug(CategoryMoongate, "Test debug message")
	Info(CategoryMoongate, "Test info message")
	Warn(CategoryMoongate, "Test warn message")
	Error(CategoryMoongate, "Test error message")
	// Note: Not testing Fatal as it would terminate the test
}
