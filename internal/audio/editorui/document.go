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
		m.status = "Library changed during the file operation; current draft retained. File outcome kept; review Details."
		m.report(m.doc.MergePublication(message.document))

		return nil
	}
	publicationChanged := message.document.LastPublication().Sequence != m.doc.LastPublication().Sequence
	m.doc = message.document
	if message.err != nil {
		return m.failedDocument(message, cropCommit, publicationChanged)
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
	if publicationChanged {
		if !m.statusIsError {
			m.status = message.summary
		}
		m.appendPublicationReceipt()
	}

	return refresh
}

func (m *model) failedDocument(message documentMsg, cropCommit, publicationChanged bool) tea.Cmd {
	if cropCommit && errors.Is(message.err, audio.ErrRecordingSourceChanged) {
		m.rejectSourceSnapshot(m.crop.source, message.err)
	}
	m.quitAfterOperation = false
	var reload tea.Cmd
	if message.reload {
		reload = m.loadSelection()
	}
	m.report(message.err)
	if publicationChanged {
		m.appendPublicationReceipt()
	}

	return reload
}

func (m *model) appendPublicationReceipt() {
	m.status += publicationReceiptText(m.doc.LastPublication(), m.doc.NeedsSync())
}

func publicationReceiptText(receipt audio.PublicationOutcome, needsSync bool) string {
	text := ""
	if receipt.Import && receipt.Created {
		text += " File retained: " + receipt.Path
		if !receipt.Assigned {
			text += " (not assigned to the requested recording; retry import)."
		}
	}
	if needsSync {
		text += " Durability is unconfirmed; Ctrl+s retries the flush."
	} else if !receipt.DirectorySyncSupported && receipt.State == audio.PublicationComplete {
		text += " Directory durability is limited by this platform."
	}

	return text
}
