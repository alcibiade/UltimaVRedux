package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortcullisCollisionDetection_Integration(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(14, 23).WithSystemCallbacks().Build()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	arch, gate := references.Position{X: 14, Y: 21}, references.Position{X: 14, Y: 22}
	source := state.GetCurrentSmallLocationReference()
	require.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), source.GetTileNumber(arch, 0))
	require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), source.GetTileNumber(gate, 0))
	for _, hour := range []byte{4, 19, 23} {
		state.DateTime.Hour, state.DateTime.Minute = hour, 59
		before := state.DateTime
		state.AdvanceGameplayMinutes(1)
		require.NotEqual(t, before, state.DateTime)
		open := state.DateTime.Hour >= 5 && state.DateTime.Hour < 20
		want := indexes.SpriteIndex(indexes.Portcullis)
		if open {
			want = indexes.BrickFloor
		}
		assert.Equal(t, want, state.GetEffectiveTile(&gate))
		assert.Equal(t, want, state.GetBonusTerrainTile(&gate))
		assert.Equal(t, want, state.GetCurrentLayeredMap().GetSmallMapTerrainTile(&gate, state.DateTime.Hour).Index)
		assert.Equal(t, open, state.IsPassable(&gate))
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), state.GetEffectiveTile(&arch))
	}
	for _, hour := range []byte{4, 5, 19, 20} {
		state.DateTime.Hour = hour // The same direct edit used by the debug clock.
		want := state.GetEffectiveTile(&gate)
		state.reloadCurrentSmallMapTerrain()
		assert.Equal(t, want, state.GetEffectiveTile(&gate))
		state.TransitionToLocation(references.SerpentsHold, 1, state.MapState.PlayerLocation.Position, TransitionFloorOnly, false)
		state.TransitionToLocation(references.SerpentsHold, 0, state.MapState.PlayerLocation.Position, TransitionFloorOnly, false)
		assert.Equal(t, want, state.GetEffectiveTile(&gate), "floor reconstruction never applies the gate twice")
	}
}

func TestPortcullisCollisionWithObjects_Integration(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(14, 23).WithSystemCallbacks().Build()
	gate := references.Position{X: 14, Y: 22}
	units := state.CurrentNPCAIController.GetNpcs()
	npcReference := references.NPCReference{Location: references.SerpentsHold, Schedule: references.CreateNPCScheduledFixedOneLocation(gate, 0)}
	npcReference.SetKeyIndex(indexes.TownsPerson_KeyIndex)
	occupant := map_units.NewNPCFriendly(npcReference, 0)
	units.ReplaceWith(nil)
	state.DateTime.Hour = 19
	require.True(t, state.IsPassable(&gate))
	occupant.SetPos(gate)
	occupant.SetFloor(0)
	units.ReplaceWith([]map_units.MapUnit{occupant})
	before := *occupant.MapUnitDetails()
	assert.False(t, state.IsPassable(&gate), "an actual NPC blocks an open gate")
	state.DateTime.Hour = 20
	assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), state.GetEffectiveTile(&gate))
	assert.False(t, state.IsPassable(&gate))
	assert.Equal(t, before, *occupant.MapUnitDetails(), "closing never crushes or moves the NPC")
	units.ReplaceWith(nil)
	state.SetPlayerPosition(gate)
	state.DateTime.Hour = 5
	state.DateTime.Hour = 20
	assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), state.GetEffectiveTile(&gate))
	assert.Equal(t, gate, state.MapState.PlayerLocation.Position, "closing under the Avatar leaves them in place")
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &gate, indexes.BrickFloor)
	assert.True(t, state.IsPassable(&gate), "explicit override policy is shared")
}

func TestPortcullisOrdinaryActorsObserveBoundary(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(14, 23).WithSystemCallbacks().Build()
	gate := references.Position{X: 14, Y: 22}
	for _, hour := range []byte{4, 19, 23} {
		state.DateTime.Hour, state.DateTime.Minute = hour, 59
		controller := &observingWorldController{observe: func() {
			want := indexes.SpriteIndex(indexes.Portcullis)
			if state.DateTime.Hour == 5 {
				want = indexes.BrickFloor
			}
			assert.Equal(t, want, state.GetEffectiveTile(&gate))
			assert.Equal(t, want, state.GetCurrentLayeredMap().GetSmallMapTerrainTile(&gate, state.DateTime.Hour).Index)
			assert.Zero(t, state.DateTime.Minute)
		}}
		state.CurrentNPCAIController = controller
		state.CompleteSurfaceCommandCycle(references.SmallMapType)
		assert.Equal(t, 1, controller.turns, "terrain is updated before the actual town actor phase")
	}
}
