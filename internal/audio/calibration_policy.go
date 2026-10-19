package audio

import (
	"fmt"
	"math"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
)

// Persisted calibration ownership modes remain compatible with the SDK.
const (
	CalibrationAuto   = authoring.CalibrationAuto
	CalibrationManual = authoring.CalibrationManual
)
const familyScore = "score"
const familyInstrument = "instrument"
const familyNature = "nature"
const familyFire = "fire"

func (recording libraryRecording) CalibrationMode() string {
	if recording.CalibrationGainDB == nil {
		return "uncalibrated"
	}
	if recording.CalibrationPolicy == CalibrationAuto {
		return CalibrationAuto
	}

	return CalibrationManual
}

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

func validateCalibrationTarget(target *float64) error {
	if target != nil && (math.IsNaN(*target) || math.IsInf(*target, 0) || *target < -60 || *target > -6) {
		return fmt.Errorf("%w: calibration target must be between -60 and -6 dB", ErrAudioLibrary)
	}

	return nil
}

func (recording libraryRecording) validateCalibration() error {
	if value := recording.CalibrationGainDB; value != nil &&
		(math.IsNaN(*value) || math.IsInf(*value, 0) || *value < -48 || *value > 12) {
		return fmt.Errorf("%w: calibration gain must be in [-48,12] dB", ErrAudioLibrary)
	}
	if recording.CalibrationPolicy != "" && recording.CalibrationPolicy != CalibrationAuto &&
		recording.CalibrationPolicy != CalibrationManual {
		return fmt.Errorf("%w: invalid calibration mode", ErrAudioLibrary)
	}
	if recording.CalibrationPolicy != "" && recording.CalibrationGainDB == nil {
		return fmt.Errorf("%w: calibration mode requires a gain", ErrAudioLibrary)
	}

	return validateCalibrationTarget(recording.CalibrationTargetDB)
}
