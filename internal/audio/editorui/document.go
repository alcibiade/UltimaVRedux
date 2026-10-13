package editorui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

func (m *model) documentOperation(reload bool, summary string, apply func(*audio.LibraryEditor) error) tea.Cmd {
	m.documentGeneration++
	m.busy = true
	document, revision, generation := m.doc.Clone(), m.doc.Revision(), m.documentGeneration

	return func() tea.Msg {
		err := apply(document)

		return documentMsg{generation: generation, baseRevision: revision,
			document: document, err: err, summary: summary, reload: reload}
	}
}

func (m *model) acceptDocument(message documentMsg) tea.Cmd {
	if !m.busy || message.generation != m.documentGeneration {
		return nil
	}
	m.busy = false
	cropCommit := m.crop != nil && m.crop.applying
	if cropCommit {
		m.crop.applying = false
	}
	if message.baseRevision != m.doc.Revision() {
		m.quitAfterOperation = false
		m.statusIsError = true
		m.status = "Library changed during the file operation; current draft retained. Reopen before saving again."

		return nil
	}
	m.doc = message.document
	if message.err != nil {
		if cropCommit {
			if errors.Is(message.err, audio.ErrRecordingSourceChanged) {
				m.rejectSourceSnapshot(m.crop.source, message.err)
			}
		}
		m.quitAfterOperation = false
		var reload tea.Cmd
		if message.reload {
			reload = m.loadSelection()
		}
		m.report(message.err)

		return reload
	}
	m.status, m.statusIsError = message.summary, false
	selected := m.selectedID()
	m.refreshKeys()
	var refresh tea.Cmd
	switch {
	case cropCommit:
		refresh = m.finishCropCommit()
	case message.reload:
		refresh = m.recheckFiles()
	case selected != m.selectedID():
		refresh = m.loadSelection()
	}
	if m.quitAfterOperation {
		m.quitAfterOperation = false
		if quit := m.quit(); quit != nil {
			return quit
		}
	}

	return refresh
}
