package editorui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

// Measures key dispatch separately from device playback. Each iteration moves
// a marker out and back; setup and source analysis are outside the measurement.
func BenchmarkCropArrowFeedback(b *testing.B) {
	for _, auto := range []bool{false, true} {
		name := "manual"
		if auto {
			name = "auto"
		}
		b.Run(name, func(b *testing.B) {
			m := fixture(b)
			recording := m.doc.Recording(m.recordingID())
			recording.PlaybackWindow = &audio.PlaybackWindow{StartMs: 100, EndMs: 700}
			require.NoError(b, m.doc.SetRecording(m.recordingID(), recording))
			if auto {
				m.beginAutoCalibration()
				finish(m, m.autoCalibrationKey(keyEnter))
			}
			m.key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
			b.ReportAllocs()
			for b.Loop() {
				m.Update(tea.KeyMsg{Type: tea.KeyRight})
				m.Update(tea.KeyMsg{Type: tea.KeyLeft})
			}
		})
	}
}

func benchmarkSource(b *testing.B, path, family string) *model {
	b.Helper()
	m := fixture(b)
	data, err := os.ReadFile(path)
	require.NoError(b, err)
	file := "sounds/benchmark" + filepath.Ext(path)
	require.NoError(b, os.WriteFile(filepath.Join(filepath.Dir(m.doc.Path()), file), data, 0o644))
	recording := m.doc.Recording(m.recordingID())
	recording.File = &file
	require.NoError(b, m.doc.SetRecording(m.recordingID(), recording))
	require.NoError(b, m.doc.SetRecordingFamily(m.recordingID(), family))
	finish(m, m.loadSelection())
	require.NotNil(b, m.clip)
	m.beginAutoCalibration()
	require.NotNil(b, m.autoGainDraft)
	finish(m, m.autoCalibrationKey(keyEnter))
	start := m.clip.DurationMs() / 3
	m.setWindow(&audio.PlaybackWindow{StartMs: start, EndMs: min(start+5000, m.clip.DurationMs())})
	b.Logf("source duration %.1fs; %d encoded bytes; backend playback excluded", float64(m.clip.DurationMs())/1000, len(data))
	return m
}

// Measures user-visible state construction as well as key dispatch. Device
// buffering/listening latency is separate; decoder construction is measured below.
func BenchmarkCropRenderedFeedback(b *testing.B) {
	for _, source := range []struct{ name, path, family string }{
		{"wav", "../assets/sounds/magic/healing_01.wav", "magic"},
		{"mp3", "../assets/music/ultima_theme.mp3", "score"},
		{"ogg", "../assets/sounds/ambience/forest_nature_01.ogg", "ambience"},
	} {
		b.Run(source.name, func(b *testing.B) {
			m := benchmarkSource(b, source.path, source.family)
			var samples []int64
			var bytes int
			b.ReportAllocs()
			for b.Loop() {
				start := time.Now()
				key := tea.KeyRight
				if len(samples)%2 != 0 {
					key = tea.KeyLeft
				}
				m.Update(tea.KeyMsg{Type: key})
				bytes += len(m.View())
				samples = append(samples, time.Since(start).Nanoseconds())
			}
			slices.Sort(samples)
			b.ReportMetric(float64(samples[(len(samples)-1)*95/100])/1e6, "p95-ms")
			b.ReportMetric(float64(bytes)/float64(len(samples)), "render-bytes/op")
		})
	}
}

func BenchmarkCropCompressedPreviewPreparation(b *testing.B) {
	for _, source := range []struct{ name, path, family string }{
		{"mp3", "../assets/music/ultima_theme.mp3", "score"},
		{"ogg", "../assets/sounds/ambience/forest_nature_01.ogg", "ambience"},
	} {
		b.Run(source.name, func(b *testing.B) {
			m := benchmarkSource(b, source.path, source.family)
			b.ReportAllocs()
			for b.Loop() {
				preview, err := m.crop.draft.Preview(m.crop.window)
				require.NoError(b, err)
				_, err = m.clip.Stream(preview.Window, false)
				require.NoError(b, err)
			}
		})
	}
}
