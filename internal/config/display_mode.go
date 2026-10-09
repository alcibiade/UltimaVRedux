package config

import (
	"errors"
	"fmt"
)

// Minimum decorated window dimensions supported by player settings.
const (
	MinimumWindowWidth  = 1280
	MinimumWindowHeight = 720
)

// Mode identifies the persisted desktop window mode without a graphics dependency.
type Mode string

// Supported persisted window modes.
const (
	ModeWindowed   Mode = "windowed"
	ModeMaximized  Mode = "maximized"
	ModeFullscreen Mode = "fullscreen"
)

// ErrInvalidMode identifies an unsupported desktop display mode.
var ErrInvalidMode = errors.New("invalid display mode")

// ParseMode validates a persisted mode without changing its identity.
func ParseMode(value string) (Mode, error) {
	mode := Mode(value)
	if !mode.IsValid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidMode, value)
	}

	return mode, nil
}

// IsValid reports whether the mode is supported.
func (m Mode) IsValid() bool {
	switch m {
	case ModeWindowed, ModeMaximized, ModeFullscreen:
		return true
	default:
		return false
	}
}

func (m Mode) String() string { return string(m) }
