package config

// ScreenResolution pairs dimensions; the owning API specifies pixels or DIPs.
type ScreenResolution struct {
	X int
	Y int
}

// SupportedWindowResolutions preserves legacy indices and their DIP semantics.
//
//nolint:mnd // These canonical 16:9 display modes are part of the configuration contract.
func SupportedWindowResolutions() []ScreenResolution {
	return []ScreenResolution{
		{X: 1280, Y: 720},
		{X: 1366, Y: 768},
		{X: 1920, Y: 1080},
		{X: 2560, Y: 1440},
		{X: 3840, Y: 2160},
	}
}

// WindowPixelPresets returns the player-facing output-pixel sizes. This list is
// independent of the frozen legacy Resolution indices and Workbench commands.
func WindowPixelPresets() []ScreenResolution {
	legacy := SupportedWindowResolutions()
	options := make([]ScreenResolution, 0, len(legacy)+1)
	options = append(options, legacy[:2]...)
	//nolint:mnd // Intermediate 900p preset intentionally does not renumber legacy settings.
	options = append(options, ScreenResolution{X: 1600, Y: 900})
	options = append(options, legacy[2:]...)

	return options
}
