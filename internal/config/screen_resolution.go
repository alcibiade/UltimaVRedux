package config

type ScreenResolution struct {
	X int
	Y int
}

// SupportedWindowResolutions returns a detached copy of the supported display sizes.
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
