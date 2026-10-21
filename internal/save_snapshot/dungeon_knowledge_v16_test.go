package save_snapshot

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDungeonKnowledgeV16RoundTrip(t *testing.T) {
	t.Parallel()
	compiled, _ := compileCurrentNativeSaveSchema(t)
	snapshot := validSnapshot()
	// Knowledge uses lexical key order; event order remains discovery order.
	for index, key := range []string{
		DungeonKnowledgeCovetousWordOfPower, DungeonKnowledgeDeceitWordOfPower,
		DungeonKnowledgeDespiseWordOfPower, DungeonKnowledgeDestardWordOfPower,
		DungeonKnowledgeDoomWordOfPower, DungeonKnowledgeHythlothWordOfPower,
		DungeonKnowledgeShameWordOfPower, DungeonKnowledgeWrongWordOfPower,
	} {
		date := &BritannianDateSnapshot{Year: 139, Month: 4, Day: index + 1}
		entry := PlayerKnowledgeSnapshot{
			Category: KnowledgeCategoryDungeon, Key: key,
			Source:       KnowledgeSourceSnapshot{Kind: KnowledgeSourceDialogue, DisplayName: "Word keeper"},
			DiscoveredOn: date,
		}
		snapshot.Redux.PlayerKnowledge = append(snapshot.Redux.PlayerKnowledge, entry)
		snapshot.Redux.QuestJournal = append(snapshot.Redux.QuestJournal, QuestJournalEventSnapshot{
			Sequence: uint64(index + 1), Kind: QuestJournalEventKnowledgeLearned, Subject: "dungeon." + key,
			OccurredOn: date, Source: QuestJournalSourceDialogue, SourceName: entry.Source.DisplayName,
		})
	}
	for _, withEvents := range []bool{false, true} {
		t.Run(fmt.Sprintf("current/events=%v", withEvents), func(t *testing.T) {
			current := snapshot
			if !withEvents {
				current.Redux.QuestJournal = []QuestJournalEventSnapshot{}
			}
			data, err := EncodeCanonical(validEnvelope(current))
			require.NoError(t, err)
			require.NoError(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
			decoded, err := Decode(data)
			require.NoError(t, err)
			assert.Equal(t, current, decoded.Snapshot, "v16 decoding must not invent events or overwrite provenance")
		})
	}
}

func TestDungeonKnowledgeV16RejectsInvalidLoreWithSchemaParity(t *testing.T) {
	t.Parallel()
	compiled, _ := compileCurrentNativeSaveSchema(t)
	for name, mutate := range map[string]func(*PlayerKnowledgeSnapshot){
		"unknown dungeon":         func(entry *PlayerKnowledgeSnapshot) { entry.Key = "abyss.word_of_power" },
		"wrong suffix":            func(entry *PlayerKnowledgeSnapshot) { entry.Key = "doom.word-of-power" },
		"full semantic ID in key": func(entry *PlayerKnowledgeSnapshot) { entry.Key = "dungeon.doom.word_of_power" },
		"uppercase":               func(entry *PlayerKnowledgeSnapshot) { entry.Key = "Doom.word_of_power" },
		"undated":                 func(entry *PlayerKnowledgeSnapshot) { entry.DiscoveredOn = nil },
		"invalid date":            func(entry *PlayerKnowledgeSnapshot) { entry.DiscoveredOn.Day = 29 },
		"unnamed":                 func(entry *PlayerKnowledgeSnapshot) { entry.Source.DisplayName = "\u0085" },
		"migration": func(entry *PlayerKnowledgeSnapshot) {
			entry.Source = KnowledgeSourceSnapshot{Kind: KnowledgeSourceMigration, DisplayName: migratedKnowledgeDisplayName}
		},
		"experimentation": func(entry *PlayerKnowledgeSnapshot) {
			entry.Source = KnowledgeSourceSnapshot{Kind: KnowledgeSourceExperimentation, DisplayName: experimentationKnowledgeDisplayName}
		},
		"unread": func(entry *PlayerKnowledgeSnapshot) { entry.Unread = true },
		"spell formula": func(entry *PlayerKnowledgeSnapshot) {
			entry.SpellFormula = &SpellFormulaKnowledgeSnapshot{State: SpellFormulaConfirmed, ConfirmedMask: 20, CandidateMasks: []uint16{}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := validSnapshot()
			entry := PlayerKnowledgeSnapshot{
				Category: KnowledgeCategoryDungeon, Key: DungeonKnowledgeDoomWordOfPower,
				Source:       KnowledgeSourceSnapshot{Kind: KnowledgeSourceRevelation, DisplayName: "A revelation"},
				DiscoveredOn: &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5},
			}
			mutate(&entry)
			snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{entry}
			_, err := EncodeCanonical(validEnvelope(snapshot))
			assert.ErrorIs(t, err, ErrInvalidSnapshot)
			data, err := json.Marshal(validEnvelope(snapshot))
			require.NoError(t, err)
			_, err = Decode(data)
			assert.ErrorIs(t, err, ErrInvalidSnapshot)
			assert.Error(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
		})
	}
}
