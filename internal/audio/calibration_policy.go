package audio

const familyScore = "score"
const familyInstrument = "instrument"
const familyNature = "nature"
const familyFire = "fire"

// DefaultCalibrationTarget retains the game's family-specific monitoring policy.
func DefaultCalibrationTarget(family string) float64 {
	switch family {
	case familyScore, familyInstrument:
		return -23
	case catalogAmbience, familyNature, familyFire:
		return -26
	default:
		return -20
	}
}
