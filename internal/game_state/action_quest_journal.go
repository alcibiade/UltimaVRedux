package game_state

// ActionQuestJournal presents a detached journal dossier without advancing gameplay.
func (g *GameState) ActionQuestJournal() {
	g.SystemCallbacks.UI.ShowQuestJournalDossier(g.CurrentQuestJournalDossier())
}
