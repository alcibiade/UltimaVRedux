package game_state

import "github.com/bradhannah/UltimaVRedux/internal/journal"

// codexWordSummary is the only disclosure before the final page is acknowledged.
const codexWordSummary = "The Codex revealed VERAMOCOR to unlock the gate."

// The four clue paragraphs from CodexRevelationMessages, with source hard
// wrapping removed. The opening page-turn announcement is presentation only.
const codexRevelationText = "Beyond Shame's egress in the centre of the Underworld there is a place of darkness.\n\n" +
	"Beyond this darkness lies the gate to the core of the world.\n\n" +
	"When thou art ready thou must call forth VERAMOCOR to unlock the gate and venture past ethereal wards and stealers of souls.\n\n" +
	"That which the world hath lost awaits thy coming."

// RecordCompleteCodexRevelation enriches the existing word entry only at the
// final-page acknowledgment boundary. It preserves the first discovery's
// identity, date and source, and stamps the existing reader entry just once.
func (g *GameState) RecordCompleteCodexRevelation() bool {
	if g.CodexRevelationComplete {
		return false
	}
	g.requireQuestJournal()
	if _, known := g.WordOfPowerKnowledge("doom"); !known {
		panic("complete Codex revelation requires acknowledged word knowledge")
	}
	for index, event := range g.questJournal.events {
		if event.Kind != QuestJournalEventKnowledgeLearned || event.Subject != "dungeon.doom.word_of_power" {
			continue
		}
		g.CodexRevelationComplete = true
		g.SystemCallbacks.UI.JournalChanged(journal.ChronicleTab, index+1)

		return true
	}
	panic("complete Codex revelation requires its Chronicle entry")
}
