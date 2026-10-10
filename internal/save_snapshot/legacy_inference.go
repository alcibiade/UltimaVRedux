package save_snapshot

import "github.com/bradhannah/UltimaVRedux/internal/references"

const migratedKnowledgeDisplayName = "Previously learned; date unknown"

// InferLegacyPlayerKnowledge reconstructs generic knowledge represented by
// prepared hidden spells or known hidden-spell recipes in a legacy save.
func InferLegacyPlayerKnowledge(inventory InventorySnapshot, progression ProgressionSnapshot) []PlayerKnowledgeSnapshot {
	definitions := hiddenSpellKnowledgeDefinitions()
	entries := make([]PlayerKnowledgeSnapshot, 0, len(definitions))
	for _, definition := range definitions {
		recipe := progression.Recipes[definition.spell]
		if inventory.Spells[definition.spell] == 0 && !recipe.Known {
			continue
		}
		entry := PlayerKnowledgeSnapshot{
			Category: KnowledgeCategorySpell,
			Key:      definition.key,
			Source: KnowledgeSourceSnapshot{
				Kind:        KnowledgeSourceMigration,
				DisplayName: migratedKnowledgeDisplayName,
			},
		}
		canonical := references.GetSpellReagents(definition.spell)
		if recipe.Known && recipe.Mask == canonical {
			entry.SpellFormula = &SpellFormulaKnowledgeSnapshot{
				State:          SpellFormulaConfirmed,
				ConfirmedMask:  canonical,
				CandidateMasks: make([]uint16, 0),
			}
		}
		entries = append(entries, entry)
	}

	return entries
}

// InferLegacyQuestJournal acknowledges permanent Shadowlord defeats already
// represented by a legacy save. Identity order is stable and dates are unknown.
func InferLegacyQuestJournal(progression ProgressionSnapshot) []QuestJournalEventSnapshot {
	subjects := [ShadowlordCount]string{
		QuestJournalSubjectFaulinei,
		QuestJournalSubjectAstaroth,
		QuestJournalSubjectNosfentor,
	}
	events := make([]QuestJournalEventSnapshot, 0, ShadowlordCount)
	for index, alive := range progression.Shadowlords.Alive {
		if alive {
			continue
		}
		events = append(events, QuestJournalEventSnapshot{
			Sequence:     uint64(len(events) + 1),
			Kind:         QuestJournalEventShadowlordDefeated,
			Subject:      subjects[index],
			Source:       QuestJournalSourceMigration,
			Acknowledged: true,
		})
	}

	return events
}

type hiddenSpellKnowledgeDefinition struct {
	key   string
	spell references.Spell
}

func hiddenSpellKnowledgeDefinitions() []hiddenSpellKnowledgeDefinition {
	return []hiddenSpellKnowledgeDefinition{
		{key: SpellKnowledgeAnYlem, spell: references.AnYlem},
		{key: SpellKnowledgeInXenMani, spell: references.InXenMani},
		{key: SpellKnowledgeRelXenBet, spell: references.RelXenBet},
	}
}
