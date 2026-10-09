package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestLoreKnowledgeKeysAreStableAndComplete(t *testing.T) {
	shadowlords := []struct {
		lord        references.Shadowlord
		name        string
		association string
	}{
		{lord: references.Faulinei, name: "faulinei.name", association: "faulinei.association"},
		{lord: references.Astaroth, name: "astaroth.name", association: "astaroth.association"},
		{lord: references.Nosfentor, name: "nosfentor.name", association: "nosfentor.association"},
	}
	for _, test := range shadowlords {
		name, supported := ShadowlordKnowledgeKey(test.lord, ShadowlordKnowledgeName)
		require.True(t, supported)
		assert.Equal(t, KnowledgeKey{Category: KnowledgeCategoryShadowlord, ID: test.name}, name)
		association, supported := ShadowlordKnowledgeKey(test.lord, ShadowlordKnowledgeAssociation)
		require.True(t, supported)
		assert.Equal(t, KnowledgeKey{Category: KnowledgeCategoryShadowlord, ID: test.association}, association)
	}

	mantras := []string{
		"honesty.mantra", "compassion.mantra", "valour.mantra", "justice.mantra",
		"sacrifice.mantra", "honor.mantra", "spirituality.mantra", "humility.mantra",
	}
	for index, id := range mantras {
		key, supported := ShrineMantraKnowledgeKey(index)
		require.True(t, supported)
		assert.Equal(t, KnowledgeKey{Category: KnowledgeCategoryShrine, ID: id}, key)
	}

	_, supported := ShadowlordKnowledgeKey(references.NoShadowlord, ShadowlordKnowledgeName)
	assert.False(t, supported)
	_, supported = ShadowlordKnowledgeKey(references.Faulinei, ShadowlordKnowledgeKind("unknown"))
	assert.False(t, supported)
	_, supported = ShrineMantraKnowledgeKey(TotalShrines)
	assert.False(t, supported)
}

func TestLoreDiscoveryIsIndependentIdempotentAndPreservesFirstMetadata(t *testing.T) {
	state := &GameState{
		DateTime:        datetime.UltimaDate{Year: 139, Month: 2, Day: 3},
		playerKnowledge: newPlayerKnowledge(),
		questJournal:    newQuestJournal(),
		SystemCallbacks: journalTestCallbacks(),
	}
	firstSource := DialogueKnowledgeSource("Sin'Vraal")
	require.True(t, state.DiscoverShadowlordKnowledge(
		references.Astaroth,
		ShadowlordKnowledgeAssociation,
		firstSource,
	))
	_, nameLearned := state.ShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeName)
	assert.False(t, nameLearned, "name and association knowledge remain independent")

	association, learned := state.ShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeAssociation)
	require.True(t, learned)
	assert.Equal(t, firstSource, association.Source)
	assert.Equal(t, &datetime.BritannianDate{Year: 139, Month: 2, Day: 3}, association.DiscoveredOn)
	assert.False(t, association.Unread)
	assert.Nil(t, association.SpellFormula)

	state.DateTime.Day = 4
	assert.False(t, state.DiscoverShadowlordKnowledge(
		references.Astaroth,
		ShadowlordKnowledgeAssociation,
		DialogueKnowledgeSource("Later speaker"),
	))
	retained, learned := state.ShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeAssociation)
	require.True(t, learned)
	assert.Equal(t, association, retained)

	require.True(t, state.DiscoverShrineMantraKnowledge(0, DialogueKnowledgeSource("Malifora")))
	mantra, learned := state.ShrineMantraKnowledge(0)
	require.True(t, learned)
	assert.Equal(t, DialogueKnowledgeSource("Malifora"), mantra.Source)
	assert.Equal(t, &datetime.BritannianDate{Year: 139, Month: 2, Day: 4}, mantra.DiscoveredOn)
	assert.False(t, mantra.Unread)
	assert.Nil(t, mantra.SpellFormula)

	association.DiscoveredOn.Day = 28
	detached, learned := state.ShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeAssociation)
	require.True(t, learned)
	assert.Equal(t, byte(3), detached.DiscoveredOn.Day)
	require.PanicsWithValue(t, "lore knowledge source must be dialogue, observation, or revelation", func() {
		state.DiscoverShrineMantraKnowledge(1, ExperimentationKnowledgeSource())
	})
}
