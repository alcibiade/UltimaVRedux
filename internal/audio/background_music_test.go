package audio

import (
	"io"
	"testing"
)

func TestMusicVolumeModifier(t *testing.T) {
	tests := []struct {
		name           string
		trackName      string
		expectedVolume float64
		description    string
	}{
		{
			name:           "Normal volume town music",
			trackName:      "town",
			expectedVolume: 1.0,
			description:    "Town music should have normal volume",
		},
		{
			name:           "Quieter combat music",
			trackName:      "combat",
			expectedVolume: 0.8,
			description:    "Combat music should be quieter as it can be intense",
		},
		{
			name:           "Subtle ambient music",
			trackName:      "ambient",
			expectedVolume: 0.7,
			description:    "Ambient music should be quieter for atmosphere",
		},
		{
			name:           "Quieter victory fanfare",
			trackName:      "victory",
			expectedVolume: 0.8,
			description:    "Victory fanfares should be quieter as they can be loud",
		},
		{
			name:           "Normal volume overworld",
			trackName:      "overworld",
			expectedVolume: 1.0,
			description:    "Overworld exploration music should have normal volume",
		},
		{
			name:           "Quieter dungeon atmosphere",
			trackName:      "dungeon",
			expectedVolume: 0.9,
			description:    "Dungeon music should be slightly quieter for atmosphere",
		},
		{
			name:           "Quieter underground music",
			trackName:      "worlds",
			expectedVolume: 0.8,
			description:    "Underground music should be quieter for atmosphere",
		},
		{
			name:           "Slightly quieter royal theme",
			trackName:      "rule",
			expectedVolume: 0.9,
			description:    "Rule Britannia should be slightly quieter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, exists := GetMusicTrackMetadata(tt.trackName)
			if !exists {
				t.Fatalf("Metadata not found for track %s", tt.trackName)
			}

			if metadata.VolumeModifier != tt.expectedVolume {
				t.Errorf("Expected VolumeModifier %f for %s (%s), got %f. %s",
					tt.expectedVolume, tt.trackName, metadata.Name, metadata.VolumeModifier, tt.description)
			}

			t.Logf("✅ %s: %s has VolumeModifier %.1f - %s",
				tt.name, metadata.Name, metadata.VolumeModifier, tt.description)
		})
	}
}

func TestMusicVolumeModifierIntegration(t *testing.T) {
	// Test that VolumeModifier is properly integrated with the music system
	config, err := NewAudioConfiguration()
	if err != nil {
		t.Fatalf("Failed to create audio configuration: %v", err)
	}
	config.SetMusicVolume(1.0)

	// Test relative volume differences between different track types
	var normalVolume, quietVolume, subtleVolume float64

	// Test with normal volume track (town)
	if metadata, exists := GetMusicTrackMetadata("town"); exists {
		baseVolume := config.GetEffectiveMusicVolume()
		normalVolume = baseVolume * metadata.VolumeModifier
		t.Logf("✅ Normal volume town music: base=%.3f * modifier=%.1f = effective=%.3f",
			baseVolume, metadata.VolumeModifier, normalVolume)
	}

	// Test with quiet track (combat)
	if metadata, exists := GetMusicTrackMetadata("combat"); exists {
		baseVolume := config.GetEffectiveMusicVolume()
		quietVolume = baseVolume * metadata.VolumeModifier
		t.Logf("✅ Quiet combat music: base=%.3f * modifier=%.1f = effective=%.3f",
			baseVolume, metadata.VolumeModifier, quietVolume)
	}

	// Test with subtle track (ambient)
	if metadata, exists := GetMusicTrackMetadata("ambient"); exists {
		baseVolume := config.GetEffectiveMusicVolume()
		subtleVolume = baseVolume * metadata.VolumeModifier
		t.Logf("✅ Subtle ambient music: base=%.3f * modifier=%.1f = effective=%.3f",
			baseVolume, metadata.VolumeModifier, subtleVolume)
	}

	// Verify that the relative volumes are properly ordered
	if quietVolume >= normalVolume {
		t.Errorf("Combat music (%.3f) should be quieter than town music (%.3f)", quietVolume, normalVolume)
	}
	if subtleVolume >= quietVolume {
		t.Errorf("Ambient music (%.3f) should be quieter than combat music (%.3f)", subtleVolume, quietVolume)
	}

	t.Logf("✅ Volume ordering verified: Normal=%.3f > Combat=%.3f > Ambient=%.3f",
		normalVolume, quietVolume, subtleVolume)
}

func TestAllMusicTracksHaveVolumeModifier(t *testing.T) {
	// Ensure all music tracks in the metadata map have a VolumeModifier set
	for trackName, metadata := range musicTrackMetadataMap() {
		if metadata.VolumeModifier <= 0 {
			t.Errorf("Music track %s (%s) has invalid VolumeModifier: %f (should be > 0)",
				trackName, metadata.Name, metadata.VolumeModifier)
		}

		// Log the volume modifier for documentation
		t.Logf("🎵 %s (%s): VolumeModifier=%.1f - %s",
			trackName, metadata.Name, metadata.VolumeModifier, metadata.Description)
	}
}

func TestMusicTrackMetadataConsistency(t *testing.T) {
	// Verify that all tracks have consistent metadata
	for trackName, metadata := range musicTrackMetadataMap() {
		// Check that all required fields are populated
		if metadata.Name == "" {
			t.Errorf("Track %s has empty Name field", trackName)
		}
		if metadata.RecordingID == "" {
			t.Errorf("Track %s has empty RecordingID field", trackName)
		}
		if metadata.Description == "" {
			t.Errorf("Track %s has empty Description field", trackName)
		}

		// Check that VolumeModifier is in reasonable range
		if metadata.VolumeModifier < 0.1 || metadata.VolumeModifier > 2.0 {
			t.Errorf("Track %s has VolumeModifier %f outside reasonable range [0.1, 2.0]",
				trackName, metadata.VolumeModifier)
		}

		// Verify FilePath follows expected pattern
		expectedPrefix := "assets/music/"
		filePath := productionLibrary().recordingPath(metadata.RecordingID)
		if filePath != "" && (len(filePath) <= len(expectedPrefix) || filePath[:len(expectedPrefix)] != expectedPrefix) {
			t.Errorf("Track %s has unexpected FilePath format: %s (should start with %s)",
				trackName, filePath, expectedPrefix)
		}
	}
}

func TestMusicTrackStreamEndsAtRecordedDuration(t *testing.T) {
	manager, err := NewAudioManager()
	if err != nil {
		t.Fatalf("create audio manager: %v", err)
	}
	defer manager.Close()

	if err := manager.backgroundMusic.LoadAssets(); err != nil {
		t.Fatalf("load music assets: %v", err)
	}
	asset := manager.backgroundMusic.musicAssets["town"]
	if asset == nil {
		t.Fatal("town music asset was not loaded")
	}

	stream, err := asset.StreamFactory()
	if err != nil {
		t.Fatal(err)
	}
	maxBytes := (asset.Duration + millisecondsPerSecond) * int64(manager.sharedContext.SampleRate()) * decodedPCMBytesPerFrame / millisecondsPerSecond
	if _, err := io.ReadFull(stream, make([]byte, maxBytes)); err == nil {
		t.Error("town music did not end within one second of its recorded duration")
	}
}

func TestGetMusicTrackMetadataFunction(t *testing.T) {
	// Test the GetMusicTrackMetadata function
	tests := []struct {
		trackName   string
		shouldExist bool
	}{
		{"title", true},
		{"town", true},
		{"combat", true},
		{"overworld", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.trackName, func(t *testing.T) {
			metadata, exists := GetMusicTrackMetadata(tt.trackName)
			if exists != tt.shouldExist {
				t.Errorf("GetMusicTrackMetadata(%s): expected exists=%t, got exists=%t",
					tt.trackName, tt.shouldExist, exists)
			}

			if tt.shouldExist {
				if metadata.Name == "" {
					t.Errorf("Track %s returned empty metadata", tt.trackName)
				}
				t.Logf("✅ Track %s: %s (%.1fx volume)", tt.trackName, metadata.Name, metadata.VolumeModifier)
			}
		})
	}
}
