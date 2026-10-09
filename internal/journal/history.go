package journal

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// History limits apply to canonical UTF-8 bytes, including LF separators.
const (
	MaxHistoryEntries   = 64
	MaxHistoryLineBytes = 4096
	MaxHistoryBytes     = 65536
	HistoryShortened    = "[text shortened]"
)

// ValidateHistory rejects malformed wire values rather than silently repairing a checkpoint.
//
//nolint:cyclop // Keep all bounded wire canonicality checks together for schema parity.
func ValidateHistory(history string) error {
	if len(history) > MaxHistoryBytes || !utf8.ValidString(history) {
		return fmt.Errorf("%w: history exceeds byte limit or is invalid UTF-8", ErrInvalidPresentation)
	}
	if history == "" {
		return nil
	}
	if strings.HasPrefix(history, "\n") || strings.HasSuffix(history, "\n") || strings.Contains(history, "\n\n\n") {
		return fmt.Errorf("%w: history contains noncanonical blank padding", ErrInvalidPresentation)
	}
	entries := 0
	for line := range strings.SplitSeq(history, "\n") {
		if line == "" {
			continue
		}
		entries++
		if len(line) > MaxHistoryLineBytes || strings.TrimSpace(line) == "" {
			return fmt.Errorf("%w: history contains an oversized or whitespace-only line", ErrInvalidPresentation)
		}
		for _, char := range line {
			if unicode.IsControl(char) || char == '\u2028' || char == '\u2029' {
				return fmt.Errorf("%w: history contains a forbidden control or line separator", ErrInvalidPresentation)
			}
		}
	}
	if entries > MaxHistoryEntries {
		return fmt.Errorf("%w: history exceeds entry limit", ErrInvalidPresentation)
	}

	return nil
}
