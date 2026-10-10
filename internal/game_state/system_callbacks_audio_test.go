package game_state

import "testing"

func TestHarpsichordSoundEffectForDigit(t *testing.T) {
	want := [...]SoundEffect{
		SoundHarpsichord0,
		SoundHarpsichord1,
		SoundHarpsichord2,
		SoundHarpsichord3,
		SoundHarpsichord4,
		SoundHarpsichord5,
		SoundHarpsichord6,
		SoundHarpsichord7,
		SoundHarpsichord8,
		SoundHarpsichord9,
	}

	for digit, wantEffect := range want {
		effect, ok := HarpsichordSoundEffectForDigit(digit)
		if !ok {
			t.Fatalf("digit %d was not recognized", digit)
		}
		if effect != wantEffect {
			t.Errorf("digit %d mapped to %v, want %v", digit, effect, wantEffect)
		}
	}

	for _, digit := range []int{-1, 10} {
		if _, ok := HarpsichordSoundEffectForDigit(digit); ok {
			t.Errorf("out-of-range digit %d was recognized", digit)
		}
	}
}
