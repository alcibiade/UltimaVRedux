package datetime

// SetTimeOfDay selects a preset hour.
func (d *UltimaDate) SetTimeOfDay(timeOfDay TimeOfDay) {
	d.Minute = 0

	switch timeOfDay {
	case Morning:
		d.Hour = 5
	case Noon:
		d.Hour = 12
	case Evening:
		d.Hour = 17
	case Midnight:
		d.Hour = 0
	case Dusk:
		d.Hour = 20
	}
}

// TimeOfDay identifies a convenient calendar preset.
type TimeOfDay int

// Calendar presets for test setup.
const (
	Morning TimeOfDay = iota
	Noon
	Evening
	Midnight
	Dusk
)
