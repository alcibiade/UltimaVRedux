package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/display"
)

const (
	configurationDisplayModeRow = iota
	configurationDisplaySizeRow
	configurationDisplayRowCount

	configurationDisplayTransitionTimeout = display.TransitionTimeoutUpdates
	configurationDisplayConfirmationTime  = 720
	configurationTicksPerSecond           = 60
	unknownDisplayLabel                   = "Unknown"
)

type configurationWindowSizeChoice struct {
	config.ScreenResolution
	available bool
	custom    bool
}

func (d *ConfigurationDialog) updateDisplayWorkflow() bool {
	switch d.displayPhase {
	case configurationDisplayEditing:
		d.syncCurrentDraggedWindowSize()

		return false
	case configurationDisplayApplying:
		return d.updateDisplayApplying()
	case configurationDisplayConfirming:
		return d.updateDisplayConfirming()
	case configurationDisplayReverting:
		return d.updateDisplayReverting()
	default:
		panic("unsupported Configuration display phase")
	}
}

func (d *ConfigurationDialog) updateDisplayApplying() bool {
	d.displayTransitionTicks++
	if !d.displayManager.IsTransitionPending() {
		if d.displayStateObserved(
			d.stagedDisplay.Mode,
			d.stagedDisplay.WindowWidth,
			d.stagedDisplay.WindowHeight,
		) {
			d.displayPhase = configurationDisplayConfirming
			d.displayConfirmationTicks = configurationDisplayConfirmationTime
			d.keepDisplaySettings = true
			d.keyboard.PrepareForTransition(0)

			return true
		}
		d.saveError = "Display change was not applied. Previous settings restored."
		d.beginDisplayRollback(false)

		return true
	}
	if d.displayTransitionTicks >= configurationDisplayTransitionTimeout {
		d.saveError = "Display change timed out. Previous settings restored."
		d.beginDisplayRollback(false)
	}

	return true
}

func (d *ConfigurationDialog) updateDisplayConfirming() bool {
	d.displayConfirmationTicks--
	if d.displayConfirmationTicks <= 0 {
		d.beginDisplayRollback(true)

		return true
	}
	d.updateDisplayConfirmationInput()

	return true
}

func (d *ConfigurationDialog) updateDisplayReverting() bool {
	d.displayTransitionTicks++
	if !d.displayManager.IsTransitionPending() {
		if d.displayStateObserved(d.rollbackMode, d.rollbackWidth, d.rollbackHeight) {
			d.displayPhase = configurationDisplayEditing
			d.displayTransitionTicks = 0
			d.displayRollbackRetry = false
			if d.resetDisplayAfterRollback {
				d.stagedDisplay.Mode = d.rollbackMode
				d.stagedDisplay.WindowWidth = d.rollbackWidth
				d.stagedDisplay.WindowHeight = d.rollbackHeight
				d.displaySizeDirty = false
			}
			d.keyboard.PrepareForTransition(0)

			return true
		}
		d.displayPhase = configurationDisplayEditing
		d.displayRollbackRetry = true
		d.saveError = "Could not restore the previous display settings."

		return true
	}
	if d.displayTransitionTicks >= configurationDisplayTransitionTimeout {
		d.displayPhase = configurationDisplayEditing
		d.displayRollbackRetry = true
		d.saveError = "Could not restore the previous display settings."
	}

	return true
}

func (d *ConfigurationDialog) displayStateObserved(mode config.Mode, width, height int) bool {
	if d.displayManager.Mode() != mode {
		return false
	}
	if mode != config.ModeWindowed {
		return true
	}
	actualWidth, actualHeight := d.displayManager.LastNormalWindowSize()

	return actualWidth == width && actualHeight == height
}

func (d *ConfigurationDialog) beginDisplayApply() {
	d.displayRollbackRetry = false
	d.rollbackMode = d.displayManager.Mode()
	d.rollbackWidth, d.rollbackHeight = d.displayManager.LastNormalWindowSize()
	if err := d.displayManager.RequestMode(
		d.stagedDisplay.Mode,
		d.stagedDisplay.WindowWidth,
		d.stagedDisplay.WindowHeight,
	); err != nil {
		d.saveError = fmt.Sprintf("Could not apply display settings: %v", err)

		return
	}
	d.saveError = ""
	d.displayPhase = configurationDisplayApplying
	d.displayTransitionTicks = 0
	d.keyboard.PrepareForTransition(0)
}

func (d *ConfigurationDialog) beginDisplayRollback(resetStaged bool) {
	d.resetDisplayAfterRollback = resetStaged
	if err := d.displayManager.RequestMode(d.rollbackMode, d.rollbackWidth, d.rollbackHeight); err != nil {
		d.displayPhase = configurationDisplayEditing
		d.displayRollbackRetry = true
		d.saveError = fmt.Sprintf("Could not restore display settings: %v", err)

		return
	}
	d.displayPhase = configurationDisplayReverting
	d.displayTransitionTicks = 0
	d.keyboard.PrepareForTransition(0)
}

func (d *ConfigurationDialog) updateDisplayConfirmationInput() {
	boundKeys := []ebiten.Key{ebiten.KeyEscape, ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeyLeft, ebiten.KeyRight}
	boundKey := d.keyboard.GetBoundKeyPressed(&boundKeys)
	if boundKey == nil || !d.keyboard.TryToRegisterKeyPress(*boundKey) {
		return
	}

	//nolint:exhaustive // The lookup above restricts input to confirmation controls.
	switch *boundKey {
	case ebiten.KeyEscape:
		d.beginDisplayRollback(true)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if !d.keepDisplaySettings {
			d.beginDisplayRollback(true)

			return
		}
		if !d.persistConfiguration() {
			d.beginDisplayRollback(false)
		}
	case ebiten.KeyLeft, ebiten.KeyRight:
		d.keepDisplaySettings = !d.keepDisplaySettings
	default:
		return
	}
}

func (d *ConfigurationDialog) updateDisplayTab() {
	boundKeys := []ebiten.Key{
		ebiten.KeyEscape,
		ebiten.KeyEnter,
		ebiten.KeyNumpadEnter,
		ebiten.KeyLeft,
		ebiten.KeyRight,
		ebiten.KeyTab,
	}
	boundKey := d.keyboard.GetBoundKeyPressed(&boundKeys)
	if boundKey == nil || !d.keyboard.TryToRegisterKeyPress(*boundKey) {
		return
	}

	//nolint:exhaustive // The lookup above restricts input to Display controls.
	switch *boundKey {
	case ebiten.KeyEscape:
		d.onCancelClicked()
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		d.onOKClicked()
	case ebiten.KeyTab:
		d.cycleConfigurationTab()
	case ebiten.KeyLeft:
		d.adjustFocusedDisplaySetting(-1)
	case ebiten.KeyRight:
		d.adjustFocusedDisplaySetting(1)
	default:
		return
	}
}

func (d *ConfigurationDialog) adjustFocusedDisplaySetting(delta int) {
	d.displayRollbackRetry = false
	switch d.focusedDisplayIndex {
	case configurationDisplayModeRow:
		modes := [...]config.Mode{config.ModeWindowed, config.ModeMaximized, config.ModeFullscreen}
		current := 0
		for index, mode := range modes {
			if mode == d.stagedDisplay.Mode {
				current = index

				break
			}
		}
		current = (current + delta + len(modes)) % len(modes)
		d.stagedDisplay.Mode = modes[current]
		if d.stagedDisplay.Mode != config.ModeWindowed {
			d.focusedDisplayIndex = configurationDisplayModeRow
		}
	case configurationDisplaySizeRow:
		if d.stagedDisplay.Mode != config.ModeWindowed {
			return
		}
		choices := d.availableWindowSizes()
		if len(choices) == 0 {
			return
		}
		current := 0
		for index, choice := range choices {
			if choice.X == d.stagedDisplay.WindowWidth && choice.Y == d.stagedDisplay.WindowHeight {
				current = index

				break
			}
		}
		current = (current + delta + len(choices)) % len(choices)
		d.stagedDisplay.WindowWidth = choices[current].X
		d.stagedDisplay.WindowHeight = choices[current].Y
		d.displaySizeDirty = true
	default:
		panic("unsupported Display setting index")
	}
	d.saveError = ""
}

func (d *ConfigurationDialog) syncCurrentDraggedWindowSize() {
	if d.displaySizeDirty || d.displayManager.Mode() != config.ModeWindowed ||
		d.displayManager.IsTransitionPending() {
		return
	}
	width, height := d.displayManager.PersistableWindowSize()
	if width < config.MinimumWindowWidth || height < config.MinimumWindowHeight {
		return
	}
	d.stagedDisplay.WindowWidth = width
	d.stagedDisplay.WindowHeight = height
}

func (d *ConfigurationDialog) availableWindowSizes() []config.ScreenResolution {
	allChoices := d.windowSizeChoices()
	choices := make([]config.ScreenResolution, 0, len(allChoices))
	for _, choice := range allChoices {
		if choice.available {
			choices = append(choices, choice.ScreenResolution)
		}
	}

	return choices
}

func (d *ConfigurationDialog) windowSizeChoices() []configurationWindowSizeChoice {
	monitorWidth, monitorHeight := d.displayManager.MonitorSize()
	choices := make([]configurationWindowSizeChoice, 0, len(d.ultimaConfig.GetWindowResolutionOptions())+1)
	custom := config.ScreenResolution{X: d.stagedDisplay.WindowWidth, Y: d.stagedDisplay.WindowHeight}
	customIsPreset := false
	for _, resolution := range d.ultimaConfig.GetWindowResolutionOptions() {
		if resolution == custom {
			customIsPreset = true
		}
		choices = append(choices, configurationWindowSizeChoice{
			ScreenResolution: resolution,
			available:        resolution.X <= monitorWidth && resolution.Y <= monitorHeight,
			custom:           false,
		})
	}
	if !customIsPreset && custom.X >= config.MinimumWindowWidth && custom.Y >= config.MinimumWindowHeight {
		choices = append(choices, configurationWindowSizeChoice{
			ScreenResolution: custom,
			available:        custom.X <= monitorWidth && custom.Y <= monitorHeight,
			custom:           true,
		})
	}

	return choices
}

func displayModeLabel(mode config.Mode) string {
	switch mode {
	case config.ModeWindowed:
		return "Windowed"
	case config.ModeMaximized:
		return "Maximized Window"
	case config.ModeFullscreen:
		return "Borderless Fullscreen"
	default:
		return unknownDisplayLabel
	}
}

func (d *ConfigurationDialog) displaySizeLabel() string {
	if d.stagedDisplay.Mode == config.ModeMaximized || d.stagedDisplay.Mode == config.ModeFullscreen {
		width, height := d.displayManager.MonitorSize()

		return fmt.Sprintf("%d x %d", width, height)
	}
	label := fmt.Sprintf("%d x %d", d.stagedDisplay.WindowWidth, d.stagedDisplay.WindowHeight)
	for _, resolution := range d.ultimaConfig.GetWindowResolutionOptions() {
		if resolution.X == d.stagedDisplay.WindowWidth && resolution.Y == d.stagedDisplay.WindowHeight {
			return label
		}
	}

	return "Custom: " + label
}
