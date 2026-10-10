package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAITimeIntegration_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	controller, ok := gs.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	require.True(t, ok)
	gs.DateTime.Hour = 8
	gs.DateTime.Minute = 0
	controller.PopulateMapFirstLoad()

	npcs := controller.GetNpcs()
	if npcs == nil || len(*npcs) == 0 {
		t.Skip("No NPCs with schedules to test")
	}
	morningPositions := make(map[int]references.Position)
	for i, npc := range *npcs {
		if npc != nil && !npc.IsEmptyMapUnit() {
			morningPositions[i] = npc.Pos()
		}
	}

	gs.DateTime.Hour = 20
	controller.PopulateMapFirstLoad()

	positionsChanged := 0
	for i, npc := range *npcs {
		if npc == nil || npc.IsEmptyMapUnit() {
			continue
		}
		if morningPosition, exists := morningPositions[i]; exists && npc.Pos() != morningPosition {
			positionsChanged++
		}
	}
	assert.Positive(t, positionsChanged, "NPC schedules should change positions between morning and evening")
}
