package editorui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

type fakeVoice struct {
	active, playing bool
	position        int64
	gain            float64
	window          *audio.PlaybackWindow
	loop            bool
	starts, closes  int
}

func (v *fakeVoice) Play(_ *audio.RecordingAudio, window *audio.PlaybackWindow, loop bool, gain float64) error {
	v.active, v.playing, v.loop, v.gain = true, true, loop, gain
	v.window = window
	v.starts++
	return nil
}
func (v *fakeVoice) SetGain(gain float64)        { v.gain = gain }
func (v *fakeVoice) Active() bool                { return v.active }
func (v *fakeVoice) Playing() bool               { return v.playing }
func (v *fakeVoice) Pause(paused bool)           { v.playing = !paused }
func (v *fakeVoice) PositionMs() int64           { return v.position }
func (v *fakeVoice) SeekMs(position int64) error { v.position = position; return nil }
func (v *fakeVoice) Close() error                { v.active = false; v.playing = false; v.closes++; return nil }

func fixture(t *testing.T) *model {
	t.Helper()
	data, err := os.ReadFile("../assets/library.json")
	require.NoError(t, err)
	var library struct {
		SchemaVersion int                        `json:"schemaVersion"`
		Recordings    map[string]audio.Recording `json:"recordings"`
		Cues          map[string]audio.Cue       `json:"cues"`
		Taxonomy      audio.Taxonomy             `json:"taxonomy"`
		Routing       audio.Routing              `json:"routing"`
	}
	require.NoError(t, json.Unmarshal(data, &library))
	for id, cue := range library.Cues {
		library.Cues[id] = cue.WithRecordings([]string{cue.PrimaryRecording()})
	}
	for id, recording := range library.Recordings {
		recording.File, recording.PlaybackWindow, recording.CalibrationGainDB = nil, nil, nil
		recording.CalibrationPolicy, recording.CalibrationTargetDB = "", nil
		library.Recordings[id] = recording
	}
	path := "sounds/heal.wav"
	for _, id := range []string{"healing_01", "healing_02"} {
		recording := library.Recordings[id]
		recording.File = &path
		library.Recordings[id] = recording
	}
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "assets", "sounds"), 0o755))
	wav, err := os.ReadFile("../assets/sounds/magic/healing_01.wav")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "assets", path), wav, 0o644))
	data, err = json.MarshalIndent(library, "", "  ")
	require.NoError(t, err)
	manifest := filepath.Join(root, "assets", "library.json")
	require.NoError(t, os.WriteFile(manifest, data, 0o644))
	doc, err := audio.OpenLibraryEditor(manifest)
	require.NoError(t, err)
	model := newModel(doc)
	model.preview = &fakeVoice{}
	model.newVoice = func() playback { return &fakeVoice{} }
	model.selectID("healing_01")
	finish(model, model.loadSelection())
	require.NotNil(t, model.clip)
	t.Cleanup(model.close)
	return model
}

func finish(model *model, command tea.Cmd) {
	if command != nil {
		model.Update(command())
	}
}

func fillSoundscape(t *testing.T, model *model) {
	t.Helper()
	focus := model.focus
	model.clearSoundscape()
	for slot := ambientOne; slot < soundscapeSlotCount; slot++ {
		layer := model.selectedMixLayer()
		require.NotNil(t, layer)
		layer.slot = slot
		require.True(t, model.installMixLayer(layer))
	}
	model.soundscapeVisible = true
	model.focus = focus
}

func TestLiveEditingWaveformSaveAndUndo(t *testing.T) {
	model := fixture(t)
	voice := model.preview.(*fakeVoice)
	model.browserKey("enter")
	require.True(t, voice.Playing())
	initialGain := voice.gain
	model.browserKey("-")
	require.Equal(t, 1, voice.starts, "live level changes must not restart a sound")
	require.Less(t, voice.gain, initialGain)
	model.position = 40
	model.browserKey("i")
	model.position = 260
	model.browserKey("o")
	require.Equal(t, &audio.PlaybackWindow{StartMs: 40, EndMs: 260}, model.doc.Recording("healing_01").PlaybackWindow)
	model.browserKey("[")
	model.browserKey("right")
	require.Equal(t, int64(50), model.window().StartMs)
	require.Contains(t, model.View(), "UNSAVED")
	require.Contains(t, model.View(), "Start 50")
	starts := voice.starts
	finish(model, model.save())
	require.False(t, model.doc.Dirty())
	require.Equal(t, starts, voice.starts, "Save must not restart preview")
	reopened, err := audio.OpenLibraryEditor(model.doc.Path())
	require.NoError(t, err)
	require.Equal(t, model.doc.Recording("healing_01"), reopened.Recording("healing_01"))
	finish(model, model.history(false))
	require.Equal(t, int64(40), model.window().StartMs)
	require.True(t, model.doc.Dirty())
	finish(model, model.history(true))
	require.False(t, model.doc.Dirty())
}

func TestMomentCandidateAuditionDoesNotAssignUntilConfirmed(t *testing.T) {
	model := fixture(t)
	model.workspace = workspaceMoments
	model.refreshKeys()
	model.selectID("sfx/SoundHeal")
	original := model.doc.Cue("sfx/SoundHeal")
	finish(model, model.chooseFileOrAssignment())
	model.selectID("healing_02")
	finish(model, model.loadSelection())
	model.play()
	require.Equal(t, original, model.doc.Cue("sfx/SoundHeal"))
	finish(model, model.acceptAssignment())
	require.Equal(t, "healing_02", model.doc.Cue("sfx/SoundHeal").RecordingID)
	require.True(t, model.doc.Dirty())
	require.False(t, model.assigning)
}

func TestStaleDecodeAndStopAllNeverStartUnexpectedAudio(t *testing.T) {
	model := fixture(t)
	voice := model.preview.(*fakeVoice)
	model.autoPreview = true
	oldGeneration := model.generation
	model.generation++
	model.Update(loadedMsg{generation: oldGeneration, clip: model.clip, autoplay: true})
	require.Zero(t, voice.starts)
	model.picker = &filePicker{}
	model.key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model.Update(loadedMsg{generation: model.generation, clip: model.clip, autoplay: true})
	require.Zero(t, voice.starts, "X also cancels a pending file-picker preview")
}

func TestMixRepeatIsBoundedAndNeverSaved(t *testing.T) {
	model := fixture(t)
	now := time.Unix(100, 0)
	model.now = func() time.Time { return now }
	model.addMix()
	require.Len(t, model.mix, 1)
	layer := model.mix[0]
	voice := layer.voice.(*fakeVoice)
	layer.mode = modeRepeat
	now = now.Add(repeatInterval)
	model.updateMix()
	require.Equal(t, 2, voice.starts)
	require.False(t, voice.loop, "repeat is a finite restart, not looping PCM")
	model.workspace = workspaceMoments
	model.refreshKeys()
	require.True(t, voice.playing, "workspace changes keep the comparison mix")
	require.False(t, model.doc.Dirty())
	model.stopAll()
	require.False(t, voice.playing)
	require.Len(t, model.mix, 1, "Stop all retains the assigned slots for restarting")
	require.True(t, layer.stopped)
}

func TestQuitProtectsUnsavedDocumentAndResizeFits(t *testing.T) {
	model := fixture(t)
	model.adjustGain(-1)
	require.Nil(t, model.quit())
	require.True(t, model.confirmQuit)
	command := model.key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	require.NotNil(t, command)
	require.IsType(t, tea.QuitMsg{}, command())
	model.width = 80
	model.height = 24
	view := model.View()
	require.LessOrEqual(t, len(strings.Split(view, "\n")), 24)
	fillSoundscape(t, model)
	view = model.View()
	require.LessOrEqual(t, len(strings.Split(view, "\n")), 24)
	for _, line := range strings.Split(view, "\n") {
		require.LessOrEqual(t, ansi.StringWidth(line), 80)
	}
	model.width = 40
	require.Contains(t, model.View(), "enlarge terminal")
}
