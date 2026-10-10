package main

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/conversation"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

func (g *GameScene) excerptRepository() (journal.NotebookRepository, error) {
	if g.notebookRoot == "" || g.activePlaythrough.id == "" {
		return journal.NotebookRepository{}, fmt.Errorf("%w: excerpts require a configured playthrough", journal.ErrInvalidNotebook)
	}

	return journal.NotebookRepository{Root: g.notebookRoot}, nil
}

func (g *GameScene) loadExcerpts() ([]journal.Exchange, error) {
	repository, err := g.excerptRepository()
	if err != nil {
		return nil, err
	}
	notebook, err := repository.Load(g.activePlaythrough.id)
	if err != nil {
		return nil, fmt.Errorf("load excerpts: %w", err)
	}

	return g.journalExcerptsForDisplay(notebook.Excerpts), nil
}

func (g *GameScene) keepExcerpt(exchange journal.Exchange) ([]journal.Exchange, error) {
	repository, err := g.excerptRepository()
	if err != nil {
		return nil, err
	}
	notebook, err := repository.Keep(g.activePlaythrough.id, exchange)
	if err != nil {
		logging.Error(logging.CategoryConversation, "Unable to keep conversation excerpt: %v", err)

		return nil, fmt.Errorf("keep excerpt: %w", err)
	}

	return g.journalExcerptsForDisplay(notebook.Excerpts), nil
}

func (g *GameScene) attributeKeptConversation(conversationID, speaker string) error {
	if g.notebookRoot == "" || g.activePlaythrough.id == "" {
		return nil
	}
	repository, err := g.excerptRepository()
	if err != nil {
		return err
	}
	if _, err = repository.AttributeConversation(g.activePlaythrough.id, conversationID, speaker); err != nil {
		logging.Error(logging.CategoryConversation, "Unable to attribute kept conversation excerpts: %v", err)

		return fmt.Errorf("attribute excerpts: %w", err)
	}

	return nil
}

func (g *GameScene) deleteExcerpt(id string) ([]journal.Exchange, error) {
	repository, err := g.excerptRepository()
	if err != nil {
		return nil, err
	}
	notebook, err := repository.Delete(g.activePlaythrough.id, id)
	if err != nil {
		logging.Error(logging.CategoryConversation, "Unable to delete conversation excerpt: %v", err)

		return nil, fmt.Errorf("delete excerpt: %w", err)
	}

	return g.journalExcerptsForDisplay(notebook.Excerpts), nil
}

func (d *LinearTalkDialog) beginJournalExchange(input string) {
	if d.conversationID == "" {
		// Journal records assume the speaker's name for readable attribution,
		// without requiring NAME or retained evidence from an earlier conversation.
		// Extracting the label does not execute dialogue or present unseen speech.
		d.knownSpeaker = d.engine.ExtractNPCName()
		id, err := newPlaythroughID()
		if err != nil {
			d.setExcerptNotice("Conversation recording unavailable.", false)

			return
		}
		d.conversationID = id
	}
	d.exchangeNumber++
	d.exchange = journal.Exchange{
		ID:      fmt.Sprintf("%s-%d", d.conversationID, d.exchangeNumber),
		Speaker: journal.UnknownSpeaker, Location: d.gameScene.journalLocationLabel(d.npcReference.Location.String()),
		Input: input, Passages: nil,
	}
	if d.knownSpeaker != "" {
		d.exchange.Speaker = d.knownSpeaker
	}
	if d.callbacks != nil {
		d.callbacks.knownSpeaker = d.exchange.Speaker
	}
	d.setExcerptNotice("", false)
}

func (d *LinearTalkDialog) retainPresentedExchange(segments []conversation.TextSegment, continues bool) {
	if d.exchange.ID == "" {
		return
	}
	previous := d.exchange.Clone()
	if !continues && len(d.exchange.Passages) > 0 {
		d.exchange.Passages = append(d.exchange.Passages, journal.Passage{Text: "\n", Runic: false})
	}
	for _, segment := range segments {
		if segment.Text != "" {
			d.exchange.Passages = append(d.exchange.Passages, journal.Passage{Text: segment.Text, Runic: segment.IsRunic})
		}
	}
	if err := d.exchange.Validate(); err != nil {
		d.exchange = previous
		d.setExcerptNotice("Conversation limit reached; latest text not retained.", false)

		return
	}
	d.publishPresentedExchange()
}

func (d *LinearTalkDialog) publishPresentedExchange() {
	if !d.mounted || d.closed || len(d.exchange.Passages) == 0 {
		return
	}
	if !d.conversationStarted {
		// A failed Talk or construction with no speech leaves the previous log alone.
		d.gameScene.lastConversation = journal.Conversation{Exchanges: nil, Truncated: false}
		d.conversationStarted = true
	}
	if err := d.gameScene.lastConversation.Present(d.exchange); err != nil {
		d.setExcerptNotice("Conversation limit reached; latest text not retained.", false)
	}
	if d.knownSpeaker != "" {
		for index := range d.gameScene.lastConversation.Exchanges {
			exchange := &d.gameScene.lastConversation.Exchanges[index]
			if strings.HasPrefix(exchange.ID, d.conversationID+"-") {
				exchange.Speaker = d.knownSpeaker
			}
		}
	}
}

func (g *GameScene) journalLocationLabel(location string) string {
	if g.gameState.GameReferences == nil || g.gameState.GameReferences.LocationReferences == nil {
		return location
	}
	reference := g.gameState.GameReferences.LocationReferences.GetSmallLocationReference(location)
	if reference != nil && reference.FriendlyLocationName != "" {
		return reference.FriendlyLocationName
	}

	return location
}

func (g *GameScene) journalConversationForDisplay() journal.Conversation {
	conversation := g.lastConversation.Clone()
	speaker := journal.UnknownSpeaker
	for _, exchange := range conversation.Exchanges {
		if exchange.Speaker != journal.UnknownSpeaker {
			speaker = exchange.Speaker
		}
	}
	for index := range conversation.Exchanges {
		conversation.Exchanges[index].Location = g.journalLocationLabel(conversation.Exchanges[index].Location)
		if conversation.Exchanges[index].Speaker == journal.UnknownSpeaker {
			conversation.Exchanges[index].Speaker = speaker
		}
	}

	return conversation
}

func (g *GameScene) journalExcerptsForDisplay(excerpts []journal.Exchange) []journal.Exchange {
	knownSpeakers := make(map[string]string)
	for _, exchange := range g.journalConversationForDisplay().Exchanges {
		knownSpeakers[exchange.ID] = exchange.Speaker
	}
	for index := range excerpts {
		excerpts[index].Location = g.journalLocationLabel(excerpts[index].Location)
		if excerpts[index].Speaker == journal.UnknownSpeaker && knownSpeakers[excerpts[index].ID] != "" {
			excerpts[index].Speaker = knownSpeakers[excerpts[index].ID]
		}
	}

	return excerpts
}

func (d *LinearTalkDialog) keepPresentedExchange() {
	exchange := d.exchange
	if d.historyPanel != nil {
		exchanges := d.gameScene.lastConversation.Exchanges
		if len(exchanges) == 0 {
			return
		}
		exchange = exchanges[min(d.historyExchange, len(exchanges)-1)]
	}
	if len(exchange.Passages) == 0 {
		d.setExcerptNotice("No presented speech to keep.", false)

		return
	}
	if _, err := d.gameScene.keepExcerpt(exchange); err != nil {
		d.setExcerptNotice("Unable to keep excerpt.", false)

		return
	}
	d.setExcerptNotice("Excerpt kept", true)
}
