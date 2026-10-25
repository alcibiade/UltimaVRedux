package audio

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestVolumeModifier(t *testing.T) {
	tests := []struct {
		name           string
		effect         game_state.SoundEffect
		expectedVolume float64
		description    string
	}{
		{
			name:           "Normal volume door",
			effect:         game_state.SoundOpenDoor,
			expectedVolume: 1.0,
			description:    "Door sounds should have normal volume",
		},
		{
			name:           "Quieter cannon fire",
			effect:         game_state.SoundCannonFire,
			expectedVolume: 0.8,
			description:    "Cannon fire should be slightly quieter",
		},
		{
			name:           "Much quieter broadside",
			effect:         game_state.SoundBroadside,
			expectedVolume: 0.7,
			description:    "Broadsides should be much quieter",
		},
		{
			name:           "Quiet stone footsteps",
			effect:         game_state.SoundWalkStone1,
			expectedVolume: 0.25,
			description:    "Stone footsteps should be quieter since they play frequently",
		},
		{
			name:           "Very quiet metal footsteps",
			effect:         game_state.SoundWalkMetal1,
			expectedVolume: 0.35,
			description:    "Metal footsteps should be very quiet since they're sharp and frequent",
		},
		{
			name:           "Subtle ambient hum",
			effect:         game_state.SoundAmbientHum,
			expectedVolume: 0.3,
			description:    "Ambient sounds should be very subtle",
		},
		{
			name:           "Very quiet earthquake",
			effect:         game_state.SoundEarthquake,
			expectedVolume: 0.6,
			description:    "Earthquakes should be much quieter as they're very dramatic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, exists := GetSoundEffectMetadata(tt.effect)
			if !exists {
				t.Fatalf("Metadata not found for effect %v", tt.effect)
			}

			if metadata.VolumeModifier != tt.expectedVolume {
				t.Errorf("Expected VolumeModifier %f for %s, got %f. %s",
					tt.expectedVolume, metadata.Name, metadata.VolumeModifier, tt.description)
			}

			t.Logf("✅ %s: %s has VolumeModifier %.1f - %s",
				tt.name, metadata.Name, metadata.VolumeModifier, tt.description)
		})
	}
}

func TestAllSoundEffectsHaveVolumeModifier(t *testing.T) {
	// Ensure all sound effects in the metadata map have a VolumeModifier set
	for effect, metadata := range soundEffectMetadataMap() {
		if metadata.VolumeModifier <= 0 {
			t.Errorf("Sound effect %s (ID: %d) has invalid VolumeModifier: %f (should be > 0)",
				metadata.Name, effect, metadata.VolumeModifier)
		}

		// Log the volume modifier for documentation
		t.Logf("📊 %s: VolumeModifier=%.1f", metadata.Name, metadata.VolumeModifier)
	}
}

func TestShardRitualEarthquakeUsesEarthquakeMetadata(t *testing.T) {
	earthquake, exists := GetSoundEffectMetadata(game_state.SoundEarthquake)
	if !exists {
		t.Fatal("ordinary earthquake metadata not found")
	}
	ritual, exists := GetSoundEffectMetadata(game_state.SoundShardRitualEarthquake)
	if !exists {
		t.Fatal("Shard ritual earthquake metadata not found")
	}

	if ritual != earthquake {
		t.Errorf("Shard ritual metadata = %+v, want earthquake metadata %+v", ritual, earthquake)
	}
}

func TestEnvironmentalHazardSoundMetadataAndAssets(t *testing.T) {
	manager, _, _ := gameAudioLibraryFixture(t, defaultRuntimeLibrary())

	tests := []struct {
		effect          game_state.SoundEffect
		name            string
		file            string
		wantPlaceholder bool
	}{
		{game_state.SoundFireBurn, "Fire Burn", "222558__raremess__cardboard-burning-1", false},
		{game_state.SoundFallingMinor, "Falling Minor", "falling_minor_01", false},
		{game_state.SoundFallingSevere, "Falling Severe", "placeholder_falling_severe", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata, exists := GetSoundEffectMetadata(test.effect)
			if !exists {
				t.Fatalf("metadata not found for %s", test.name)
			}
			if metadata.Name != test.name || metadata.RecordingID == "" {
				t.Fatalf("metadata = %#v", metadata)
			}
			if isPlaceholderSound(test.effect) != test.wantPlaceholder {
				t.Fatalf("placeholder classification for %s = %t, want %t",
					test.name, isPlaceholderSound(test.effect), test.wantPlaceholder)
			}
		})
	}

	for _, test := range tests {
		before := len(manager.sfx.Snapshot())
		if err := manager.PlaySoundEffect(test.effect); err != nil {
			t.Fatal(err)
		}
		after := len(manager.sfx.Snapshot())
		if test.wantPlaceholder && after != before || !test.wantPlaceholder && after != before+1 {
			t.Fatalf("hazard %s: before=%d after=%d", test.name, before, after)
		}
	}
}

func TestHarpsichordSoundMetadataAndAssets(t *testing.T) {
	manager, _, _ := gameAudioLibraryFixture(t, defaultRuntimeLibrary())

	seen := make(map[game_state.SoundEffect]struct{}, 10)
	for digit := 0; digit <= 9; digit++ {
		effect, ok := game_state.HarpsichordSoundEffectForDigit(digit)
		if !ok {
			t.Fatalf("digit %d has no sound effect", digit)
		}
		if _, duplicate := seen[effect]; duplicate {
			t.Fatalf("digit %d reuses sound effect %d", digit, effect)
		}
		seen[effect] = struct{}{}

		metadata, exists := GetSoundEffectMetadata(effect)
		if !exists {
			t.Fatalf("digit %d metadata is missing", digit)
		}
		if metadata.Name != fmt.Sprintf("Harpsichord %d", digit) || metadata.RecordingID == "" {
			t.Fatalf("digit %d metadata = %#v", digit, metadata)
		}

		filePath := assetRecordingPath(productionLibrary(), metadata.RecordingID)
		fileData, err := soundFiles.ReadFile(filePath)
		if err != nil {
			t.Fatalf("read digit %d WAV: %v", digit, err)
		}
		const minimumAudiblePeak = 12_000
		if peak := pcm16WAVPeak(fileData); peak < minimumAudiblePeak {
			t.Fatalf("digit %d WAV peak = %d, want at least %d", digit, peak, minimumAudiblePeak)
		}
		if err := manager.PlaySoundEffect(effect); err != nil {
			t.Fatalf("digit %d: %v", digit, err)
		}
		if len(manager.sfx.Snapshot()) != 1 {
			t.Fatalf("digit %d did not create a voice", digit)
		}
		manager.StopSound(game_state.SoundStopRequest{Effect: effect})
	}
}

func pcm16WAVPeak(data []byte) int {
	const pcmWAVHeaderBytes = 44
	peak := 0
	for offset := pcmWAVHeaderBytes; offset+1 < len(data); offset += 2 {
		sample := int32(int16(binary.LittleEndian.Uint16(data[offset : offset+2])))
		if sample < 0 {
			sample = -sample
		}
		peak = max(peak, int(sample))
	}

	return peak
}
