package editorui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

type pickerEntry struct {
	name, path string
	directory  bool
}
type filePicker struct {
	directory string
	entries   []pickerEntry
	cursor    int
	start     int
}

func (m *model) openPicker() tea.Cmd {
	m.focus = focusBrowser
	m.picker = &filePicker{directory: filepath.Dir(m.doc.Path()), entries: nil, cursor: 0, start: 0}
	m.readDirectory(m.picker.directory)
	m.status = "Enter imports a file or opens a directory; p previews; / enters a path; Esc cancels."

	return nil
}

func (m *model) readDirectory(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		m.report(fmt.Errorf("browse audio files: %w", err))

		return
	}
	rows := []pickerEntry{{name: "..", path: filepath.Dir(path), directory: true}}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() && audio.DetectAudioFormat(entry.Name()) == audio.AudioFormatUnknown {
			continue
		}
		rows = append(rows, pickerEntry{
			name: entry.Name(), path: filepath.Join(path, entry.Name()), directory: entry.IsDir(),
		})
	}
	slices.SortStableFunc(rows[1:], func(left, right pickerEntry) int {
		if left.directory != right.directory {
			if left.directory {
				return -1
			}

			return 1
		}

		return strings.Compare(left.name, right.name)
	})
	m.picker = &filePicker{directory: path, entries: rows, cursor: 0, start: 0}
}

func (m *model) pickerKey(key tea.KeyMsg) tea.Cmd {
	picker := m.picker
	switch key.String() {
	case keyEscape:
		m.picker = nil

		return m.loadSelection()
	case "up":
		picker.cursor = max(0, picker.cursor-1)
	case keyDown:
		picker.cursor = min(len(picker.entries)-1, picker.cursor+1)
	case keyPageUp, keyShiftUp, keyPreviousPage:
		picker.start = listViewportStart(picker.start, picker.cursor, len(picker.entries), m.listHeight()) - m.listHeight()
		picker.cursor = max(0, picker.cursor-m.listHeight())
	case keyPageDown, keyShiftDown, keyNextPage:
		picker.start = listViewportStart(picker.start, picker.cursor, len(picker.entries), m.listHeight()) + m.listHeight()
		picker.cursor = max(0, min(len(picker.entries)-1, picker.cursor+m.listHeight()))
	case keyBackspace:
		m.readDirectory(filepath.Dir(picker.directory))
	case "/":
		m.field, m.input = "path", picker.directory+string(filepath.Separator)
	case keyEnter:
		if len(picker.entries) == 0 {
			return nil
		}
		entry := picker.entries[picker.cursor]
		if entry.directory {
			m.readDirectory(entry.path)
		} else {
			return m.importFile(entry.path)
		}
	case "p":
		if len(picker.entries) > 0 && !picker.entries[picker.cursor].directory {
			m.fullSource = true

			return m.loadPath(picker.entries[picker.cursor].path, true)
		}
	case " ":
		if m.preview.Active() && (m.preview.Playing() || m.previewPaused) {
			m.togglePreview()
		} else if len(picker.entries) > 0 && !picker.entries[picker.cursor].directory {
			m.fullSource = true

			return m.loadPath(picker.entries[picker.cursor].path, true)
		}
	}
	picker.start = listViewportStart(picker.start, picker.cursor, len(picker.entries), m.listHeight())

	return nil
}

func (m *model) enterPickerPath(path string) tea.Cmd {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			m.report(err)

			return nil
		}
		path = filepath.Join(home, path[2:])
	}
	info, err := os.Stat(path)
	if err != nil {
		m.report(err)

		return nil
	}
	if info.IsDir() {
		m.readDirectory(path)

		return nil
	}

	return m.importFile(path)
}

func (m *model) importFile(path string) tea.Cmd {
	document, recordingID := m.doc.Clone(), m.recordingID()
	m.busy, m.picker = true, nil
	m.status = "Validating and importing recording…"
	m.clearSoundscape()

	return func() tea.Msg {
		err := document.ImportRecording(recordingID, path)

		return documentMsg{document: document, err: err,
			summary: "File assigned; old trim/calibration cleared. Ctrl+s saves; u undoes assignment.", reload: true}
	}
}
