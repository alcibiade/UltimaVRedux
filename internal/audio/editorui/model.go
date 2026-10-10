// Package editorui implements the standalone Bubble Tea audio-library editor.
package editorui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

const (
	refreshInterval = 50 * time.Millisecond
	defaultWidth    = 100
	defaultHeight   = 36
	msPerSecond     = 1000
	decibelBase     = 10
	decibelDivisor  = 20
	keyEscape       = "esc"
	keyEnter        = "enter"
	keyDown         = "down"
	keyTab          = "tab"
	keyBackspace    = "backspace"
	keyPageUp       = "pgup"
	keyPageDown     = "pgdown"
	keyPreviousPage = "ctrl+p"
	keyNextPage     = "ctrl+n"
	keyShiftUp      = "shift+up"
	keyShiftDown    = "shift+down"
	keyCtrlEnter    = "ctrl+enter"
	keyCtrlDown     = "ctrl+down"
	keyShiftTab     = "shift+tab"
	keyCtrlA        = "ctrl+a"
	keyLeft         = "left"
	keyRight        = "right"
	keySave         = "ctrl+s"
	fieldNote       = "note"
	fieldGain       = "gain"
	fieldStart      = "start"
	fieldEnd        = "end"
	markerStart     = 1
	markerEnd       = 2
)

type loadedMsg struct {
	generation uint64
	clip       *audio.RecordingAudio
	err        error
	autoplay   bool
}
type documentMsg struct {
	document *audio.LibraryEditor
	err      error
	summary  string
	reload   bool
}
type tickMsg time.Time

type playback interface {
	Play(clip *audio.RecordingAudio, window *audio.PlaybackWindow, loop bool, gain float64) error
	SetGain(gain float64)
	Playing() bool
	Active() bool
	Pause(paused bool)
	PositionMs() int64
	SeekMs(position int64) error
	Close() error
}

type model struct {
	doc                 *audio.LibraryEditor
	preview             playback
	clip                *audio.RecordingAudio
	keys                []string
	cursor              int
	browserStart        int
	workspace           editorWorkspace
	search              string
	filterClearPending  bool
	width, height       int
	generation          uint64
	cancelLoad          context.CancelFunc
	loading, busy       bool
	status              string
	field, input        string
	licenseRecordingID  string
	inputError          string
	committingInput     bool
	statusIsError       bool
	form                *editorForm
	infoDialog          *editorInfo
	previewPaused       bool
	previewLoop         bool
	previewEndMs        int64
	picker              *filePicker
	assigning           bool
	assignmentCue       string
	fullSource, looping bool
	autoPreview         bool
	position            int64
	marker              int
	zoom                bool
	previewSliders      [previewCategoryCount]int
	referencePreview    bool
	recordingCategory   previewCategory
	volumeControls      bool
	volumeCursor        int
	confirmQuit         bool
	mix                 []*mixLayer
	mixCursor           int
	soundscapeVisible   bool
	pendingAmbient      *mixLayer
	sceneFXPreview      bool
	previewLabel        string
	now                 func() time.Time
	newVoice            func() playback
	availability        map[string]audio.RecordingAvailability
	filter              availabilityFilter
	scanGeneration      uint64
	scanQueue           []string
	cancelScan          context.CancelFunc
	workspaceState      [workspaceCount]browseState
	assignmentState     browseState
	focus               panelFocus
	detailsScroll       int
	colors              palette
	variantCue          string
	variantState        browseState
	assignmentAppend    bool
	duplicate           *clipDraft
	currentVariant      map[string]int
	nextVariant         map[string]int
	pendingCycle        string
	autoGainDraft       *audio.CalibrationSuggestion
	calibrationCache    *calibrationCache
	routing             routingEditor
}

func newModel(doc *audio.LibraryEditor) *model {
	editor := new(model)
	editor.doc, editor.preview = doc, new(audio.Audition)
	editor.width, editor.height = defaultWidth, defaultHeight
	editor.previewSliders = [previewCategoryCount]int{defaultPreviewSlider, defaultPreviewSlider, defaultAmbientSlider}
	editor.recordingCategory = previewCategoryAuto
	editor.now = time.Now
	editor.newVoice = newVoice
	editor.colors = newPalette()
	editor.currentVariant = make(map[string]int)
	editor.nextVariant = make(map[string]int)
	editor.routing.tiles = audio.TileTargets()
	editor.routing.selected = make(map[string]bool)
	editor.prepareAvailability()
	editor.status = "Enter: play selection. f: choose file. Ctrl+s: save the real manifest."
	editor.refreshKeys()

	return editor
}

//nolint:ireturn // The presentation adapter allows deterministic input tests without opening an audio device.
func newVoice() playback { return new(audio.Audition) }

// Run restores the terminal and closes every player on quit, interrupt, or error.
func Run(doc *audio.LibraryEditor) error {
	m := newModel(doc)
	defer m.close()
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return fmt.Errorf("audio editor: %w", err)
	}

	return nil
}

func (m *model) Init() tea.Cmd { return tea.Batch(m.loadSelection(), tick(), m.scanNext()) }
func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

//nolint:ireturn // Bubble Tea requires Update to return its Model interface.
func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case loadedMsg:
		m.acceptLoaded(msg)
	case documentMsg:
		m.busy = false
		m.doc = msg.document
		if msg.err == nil {
			m.statusIsError = false
			m.status = msg.summary
			selected := m.selectedID()
			m.refreshKeys()
			if msg.reload {
				return m, m.recheckFiles()
			}
			if selected != m.selectedID() {
				return m, m.loadSelection()
			}

			return m, nil
		}
		m.report(msg.err)
	case availabilityMsg:
		return m, m.acceptAvailability(msg)
	case tickMsg:
		if voice := m.previewVoice(); voice.Active() {
			m.position = voice.PositionMs()
		}
		m.updateMix()
		m.updateRoutingPreview()

		return m, tick()
	case tea.KeyMsg:
		return m, m.key(msg)
	}

	return m, nil
}

func (m *model) acceptLoaded(message loadedMsg) {
	if message.generation != m.generation {
		return
	}
	m.loading, m.clip = false, message.clip
	m.report(message.err)
	m.updateSelectedAvailability(message)
	if message.err == nil && m.clip != nil && m.picker == nil && !m.assigning && m.form == nil {
		recording := m.doc.Recording(m.recordingID())
		if recording.CalibrationMode() == audio.CalibrationAuto && m.followAutoCalibration(&recording) {
			m.report(m.doc.SetRecording(m.recordingID(), recording))
			m.refreshMixGains()
		}
	}
	if message.err == nil && message.autoplay && m.autoPreview && m.play() {
		m.advancePreviewCycle()
	}
}

func (m *model) selectedID() string {
	if len(m.keys) == 0 {
		return ""
	}

	return m.keys[m.cursor]
}

func (m *model) recordingID() string {
	if menu := m.routing.menu; m.isRecordingMenu() {
		choices := m.routingMenuChoices()
		if len(choices) > 0 && choices[menu.cursor].id != formEmptyRecording {
			return choices[menu.cursor].id
		}

		return ""
	}
	if m.form != nil && m.routing.menu == nil {
		return m.formRecordingID()
	}
	if m.routingWorkspace() || m.routing.menu != nil {
		return m.doc.Cue(m.routingCue()).PrimaryRecording()
	}
	if m.browsingCues() {
		ids := m.doc.Cue(m.selectedID()).Recordings()
		if len(ids) == 0 {
			return ""
		}

		return ids[m.currentVariant[m.selectedID()]%len(ids)]
	}

	return m.selectedID()
}

func (m *model) formRecordingID() string {
	switch {
	case m.form.cue != "":
		return m.doc.Cue(m.form.cue).PrimaryRecording()
	case m.form.kind == formNewAmbient:
		return m.form.fields[newRecordingField].value
	case m.form.kind == formSurface:
		if len(m.form.sequence) > 0 {
			return m.doc.Cue(m.form.sequence[m.form.cursor]).PrimaryRecording()
		}

		return ""
	default:
		return m.selectedID()
	}
}

func (m *model) loadSelection() tea.Cmd {
	m.statusIsError = false
	m.routing.fading = false
	m.previewPaused = false
	m.sceneFXPreview = false
	m.pendingCycle = ""
	m.fullSource = false
	path, err := m.doc.RecordingPath(m.recordingID())
	if err != nil {
		if m.cancelLoad != nil {
			m.cancelLoad()
		}
		m.generation++
		m.clip = nil
		m.loading = false
		m.report(m.preview.Close())
		m.report(err)
		if m.doc.FileAvailability(m.recordingID()).State == audio.RecordingNoFile {
			m.statusIsError = false // An intentional empty slot is not an invalid file.
		}
		if m.selectedID() == "" {
			m.status = "No matches. / changes search; Ctrl+f changes the filter."
		} else if m.routingWorkspace() && m.routingCue() == "" {
			m.status = "No sound selected. f assigns; Enter previews."
		}

		return nil
	}

	return m.loadPath(path, false)
}

func (m *model) loadPath(path string, autoplay bool) tea.Cmd {
	m.sceneFXPreview = false
	m.pendingCycle = ""
	m.autoPreview = autoplay
	if m.cancelLoad != nil {
		m.cancelLoad()
	}
	m.generation++
	m.clip, m.position = nil, 0
	m.previewPaused = false
	m.report(m.preview.Close())
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelLoad = cancel
	generation := m.generation
	m.loading = true

	return func() tea.Msg {
		defer cancel()
		clip, err := audio.LoadRecordingContext(ctx, path)

		return loadedMsg{generation: generation, clip: clip, err: err, autoplay: autoplay}
	}
}

func (m *model) gain() float64 {
	if m.form != nil && m.form.kind == formNewAmbient && m.routing.menu == nil {
		return m.newAmbientPreviewGain()
	}
	gain := m.auditionGain(m.recordingID(), m.activeCue(), m.picker == nil)
	if m.form != nil && m.form.previewLevel != nil && m.routing.menu == nil {
		return gain * *m.form.previewLevel
	}
	if m.workspace == workspaceTiles && !m.routing.footsteps && m.routing.menu == nil && m.form == nil {
		_, level := m.doc.Routing().TileAmbientSettings(m.selectedID())
		gain *= level
	}

	return gain
}

func (m *model) play() bool {
	if m.form != nil && m.routing.menu == nil {
		m.previewForm(false, false)

		return m.preview.Playing()
	}
	if (m.routingWorkspace() || m.routing.menu != nil) && m.picker == nil {
		return m.playRouting()
	}
	if m.clip == nil {
		m.status = "Choose a recording with an audio file; wait for waveform loading."

		return false
	}
	if (m.soundscapeVisible || len(m.mix) > 0) && m.picker == nil && !m.assigning && m.duplicate == nil &&
		m.categoryForPreview(m.recordingID(), m.activeCue()) == previewSFX {
		layer := m.selectedMixLayer()
		layer.slot = sfxSlot
		layer.fullSource = m.fullSource
		if m.looping {
			layer.mode = modeLoop
		}
		if m.installMixLayer(layer) {
			m.sceneFXPreview = true
			m.previewPaused, m.previewLoop = false, layer.mode == modeLoop
			m.previewLabel = m.layerName(layer)
			m.mixCursor = int(sfxSlot)
			m.status = "SFX triggered over Soundscape: " + m.layerName(layer)

			return true
		}

		return false
	}
	m.sceneFXPreview = false
	window := m.doc.Recording(m.recordingID()).PlaybackWindow
	if m.fullSource {
		window = nil
	}
	err := m.preview.Play(m.clip, window, m.looping, m.gain())
	if err == nil {
		m.status = "Playing " + m.recordingID()
		m.previewLabel = m.recordingID()
		m.previewPaused, m.previewLoop = false, m.looping
		m.previewEndMs = m.clip.DurationMs
		if window != nil {
			m.previewEndMs = window.EndMs
		}
	}
	m.report(err)

	return err == nil
}

//nolint:ireturn // The same playback adapter owns standalone and soundscape preview controls.
func (m *model) previewVoice() playback {
	if m.sceneFXPreview {
		if layer := m.layerAt(sfxSlot); layer != nil {
			return layer.voice
		}
	}

	return m.preview
}

func (m *model) pausePreview(paused bool) {
	if paused && !m.previewPaused {
		m.routing.pauseStart = m.now()
	} else if !paused && m.previewPaused && m.routing.fading {
		m.routing.fadeStart = m.routing.fadeStart.Add(m.now().Sub(m.routing.pauseStart))
	}
	m.previewPaused = paused
	if m.sceneFXPreview {
		if layer := m.layerAt(sfxSlot); layer != nil {
			layer.paused = paused
		}
	}
	m.previewVoice().Pause(paused)
}

func (m *model) report(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		m.status = err.Error()
		m.statusIsError = true
		if m.committingInput {
			m.inputError = err.Error()
		}
	}
}

func (m *model) rejectInput(message string) {
	m.status, m.inputError, m.statusIsError = message, message, true
}

func (m *model) close() {
	if m.cancelScan != nil {
		m.cancelScan()
	}
	if m.cancelLoad != nil {
		m.cancelLoad()
	}
	m.report(m.preview.Close())
	for _, layer := range m.mix {
		m.report(layer.voice.Close())
	}
}

func (m *model) save() tea.Cmd {
	m.statusIsError = false
	document := m.doc.Clone()
	m.confirmQuit = false
	m.busy, m.status = true, "Validating and saving library.json…"

	return func() tea.Msg {
		err := document.Save()

		return documentMsg{document: document, err: err,
			summary: "Saved library.json. Rebuild the game to use these changes.", reload: false}
	}
}
