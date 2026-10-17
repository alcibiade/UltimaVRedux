package main

import "github.com/bradhannah/UltimaVRedux/internal/journal"

func (g *GameScene) archiveExcerpt(id string, archived bool) ([]journal.Exchange, error) {
	g.excerpts.SetArchived(id, archived)

	return g.loadExcerpts()
}
