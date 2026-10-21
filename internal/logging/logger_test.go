package logging

import (
	"strings"
	"testing"
)

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
	if logger.GetCategoryStates()[CategoryMoongate] {
		t.Error("Category should be disabled initially")
	}

	logger.SetCategoryEnabled(CategoryMoongate, true)
	if !logger.GetCategoryStates()[CategoryMoongate] {
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

func TestGlobalLoggerFunctions(t *testing.T) {
	// Test global logger convenience functions
	SetGlobalEnabled(true)
	if !GetGlobalLogger().IsEnabled() {
		t.Error("Global logger should be enabled")
	}

	SetCategoryEnabled(CategoryMoongate, true)
	if !GetGlobalLogger().GetCategoryStates()[CategoryMoongate] {
		t.Error("Global logger category should be enabled")
	}

	SetStdoutEnabled(true)
	if !GetGlobalLogger().IsStdoutEnabled() {
		t.Error("Global logger stdout should be enabled")
	}

	// Test logging functions don't panic (basic smoke test)
	Debugf(CategoryMoongate, "Test debug message")
	Infof(CategoryMoongate, "Test info message")
	Warnf(CategoryMoongate, "Test warn message")
	Errorf(CategoryMoongate, "Test error message")
}

func TestWarningsAreRecordedWhileLoggingIsOff(t *testing.T) {
	logger := NewLogger()
	var diagnostics strings.Builder
	logger.SetDiagnosticsWriter(&diagnostics)

	logger.Debugf(CategorySystem, "hidden")
	logger.Warnf(CategorySystem, "audio failed: %d", 7)

	written := diagnostics.String()
	if !strings.Contains(written, "WARN [SYSTEM] audio failed: 7") || strings.Contains(written, "hidden") {
		t.Fatalf("diagnostics must hold only the warning, got %q", written)
	}
	if entries := logger.EntriesAfter(0, 10).Entries; len(entries) != 1 || entries[0].Message != "audio failed: 7" {
		t.Fatalf("ring buffer must hold only the warning, got %+v", entries)
	}
}
