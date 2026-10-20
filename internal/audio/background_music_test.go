package audio

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthoredMusicGains(t *testing.T) {
	for track, gain := range map[string]float64{
		"town": 1, "combat": 0.8, "ambient": 0.7, "victory": 0.8,
		"overworld": 1, "dungeon": 0.9, "worlds": 0.8, "rule": 0.9,
	} {
		t.Run(track, func(t *testing.T) {
			metadata, exists := musicTrackMetadataMap()[track]
			require.True(t, exists)
			require.Equal(t, gain, metadata.VolumeModifier)
		})
	}
}

func TestMusicTrackMetadataConsistency(t *testing.T) {
	for track, metadata := range musicTrackMetadataMap() {
		t.Run(track, func(t *testing.T) {
			require.NotEmpty(t, metadata.Name)
			require.NotEmpty(t, metadata.RecordingID)
			require.NotEmpty(t, metadata.Description)
			require.GreaterOrEqual(t, metadata.VolumeModifier, 0.1)
			require.LessOrEqual(t, metadata.VolumeModifier, 2.0)
			if file := productionLibrary().recordingPath(metadata.RecordingID); file != "" {
				relative, music := strings.CutPrefix(file, "assets/music/")
				require.True(t, music)
				require.NotEmpty(t, relative)
			}
		})
	}
}
