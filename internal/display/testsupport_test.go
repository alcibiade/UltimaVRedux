package display

import (
	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// SetWindowSize retains the legacy non-fullscreen setter behavior.
func (dm *DisplayManager) SetWindowSize(width, height int) {
	if dm.backend.IsFullscreen() {
		return
	}
	if err := dm.RequestMode(config.ModeWindowed, width, height); err != nil {
		logging.Warnf(logging.CategoryUI, "Window size request rejected: %v", err)
	}
}
