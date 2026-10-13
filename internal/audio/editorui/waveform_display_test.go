package editorui

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

func TestFooterPacksAcrossWidthWithoutDroppingTrimHints(t *testing.T) {
	model := fixture(t)
	model.focus = focusWave
	model.clip = &audio.RecordingAudio{DurationMs: 3000}
	model.width = 80
	narrow := model.footerLines()
	model.width = 180
	wide := model.footerLines()
	require.Less(t, len(wide), len(narrow))
	require.LessOrEqual(t, len(wide), 2, "wide terminals should not keep fixed short shortcut rows")
	for _, rows := range [][]string{narrow, wide} {
		plain := ansi.Strip(strings.Join(rows, "\n"))
		for _, hint := range []string{"[i/o] Mark trim", "[s/e] Exact trim", "[?] Help", "[Ctrl+s] Save", "[q] Quit"} {
			require.Contains(t, plain, hint)
		}
	}
	model.beginInput(fieldStart, "0")
	require.Equal(t, focusWave, model.focus)
	require.Contains(t, ansi.Strip(strings.Join(model.footerLines(), "\n")), "420=0.420s; sets the draft only")
	model.field = ""
	model.beginInput(fieldEnd, "3000")
	require.Contains(t, ansi.Strip(strings.Join(model.footerLines(), "\n")), "2350=2.350s; sets the draft only")
}

func TestWaveformUsesMoreHeightWhileKeepingOtherPanesUsable(t *testing.T) {
	model := fixture(t)
	model.width, model.height = 160, 60
	require.GreaterOrEqual(t, model.layout().wave, 20)
	model.width, model.height, model.focus = 60, 24, focusWave
	fillSoundscape(t, model)
	assertFullScreen(t, model)
	require.GreaterOrEqual(t, model.listHeight(), 1)
	require.Greater(t, model.layout().wave, panelEdges+waveLabelRows, "retain actual waveform rows on small screens")
}

func TestLogWaveformMakesQuietAudioVisibleAndPreservesDigitalSilence(t *testing.T) {
	const height = 8
	require.Zero(t, waveformDisplayUnits(0, height))
	require.Equal(t, "─", waveformCell(0, height-1, height))
	last := 0
	for _, peak := range []float64{1.0 / 32768, 0.001, 0.01, 0.1, 1} {
		units := waveformDisplayUnits(peak, height)
		require.Greater(t, units, 0)
		require.GreaterOrEqual(t, units, last)
		require.NotEqual(t, "─", waveformCell(units, height-1, height))
		last = units
	}
	require.GreaterOrEqual(t, waveformDisplayUnits(0.001, height), waveformCellSteps,
		"-60 dB audio should occupy more than a barely visible fractional cell")
	require.Equal(t, height*waveformCellSteps, waveformDisplayUnits(1, height))
	require.Zero(t, waveformDisplayUnits(math.NaN(), height))
	model := fixture(t)
	peaks := []float64{0, 0.001, 0.01, 0}
	model.clip = &audio.RecordingAudio{DurationMs: 40, Peaks: peaks}
	voice := model.preview.(*fakeVoice)
	beforeGain := voice.gain
	model.drawWaveform(audio.PlaybackWindow{StartMs: 0, EndMs: 40}, 0, 0, 40, 40, height)
	require.Equal(t, []float64{0, 0.001, 0.01, 0}, peaks)
	require.Equal(t, beforeGain, voice.gain, "display scaling must never change playback gain")
	require.False(t, model.doc.Dirty())
}
