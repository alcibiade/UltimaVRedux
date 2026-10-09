package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

func (g *GameScene) archiveExcerpt(id string, archived bool) ([]journal.Exchange, error) {
	repository, err := g.excerptRepository()
	if err != nil {
		return nil, err
	}
	notebook, err := repository.SetArchived(g.activePlaythrough.id, id, archived)
	if err != nil {
		logging.Error(logging.CategoryConversation, "Unable to change excerpt archive state: %v", err)

		return nil, fmt.Errorf("archive excerpt: %w", err)
	}

	return g.journalExcerptsForDisplay(notebook.Excerpts), nil
}
