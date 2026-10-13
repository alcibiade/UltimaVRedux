package editorui

import (
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
	if message.baseRevision != m.doc.Revision() {
		m.statusIsError = true
		m.status = "Library changed during the file operation; current draft retained. Reopen before saving again."

		return nil
	}
	m.doc = message.document
	if message.err != nil {
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
	if message.reload {
		return m.recheckFiles()
	}
	if selected != m.selectedID() {
		return m.loadSelection()
	}

	return nil
}
