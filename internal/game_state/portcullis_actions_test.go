package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortcullisPushAndPullActions(t *testing.T) {
	for _, test := range []struct {
		name     string
		hour     byte
		pull     bool
		occupied bool
		want     bool
	}{
		{name: "push before opening", hour: 4},
		{name: "push at opening", hour: 5, want: true},
		{name: "push before closing", hour: 19, want: true},
		{name: "push at closing", hour: 20},
		{name: "occupied open push destination", hour: 5, occupied: true},
		{name: "pull into raised gate", hour: 5, pull: true, want: true},
		{name: "pull into lowered gate", hour: 20, pull: true},
		{name: "occupied open pull destination", hour: 5, pull: true, occupied: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
				WithPlayerAt(12, 22).WithRandomSeed(525).WithSystemCallbacks().Build()
			gate := references.Position{X: 14, Y: 22}
			arch := gate.Up()
			layer := state.GetCurrentLayeredMap()
			require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.SpriteIndexByLayer(map_state.MapLayer, gate))
			require.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), layer.SpriteIndexByLayer(map_state.MapLayer, arch))
			state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
			if test.pull {
				state.SetPlayerPosition(gate)
				far := gate.Right().Right()
				layer.SetTileByLayer(map_state.MapOverrideLayer, &far, indexes.StoneBrickWall)
			} else {
				// No pull fallback: a blocked forward push must actually reject.
				layer.SetTileByLayer(map_state.MapLayer, &state.MapState.PlayerLocation.Position, indexes.MetalFloor)
			}
			object := state.MapState.PlayerLocation.Position.Right()
			layer.SetTileByLayer(map_state.MapLayer, &object, indexes.ChairBackForward)
			if test.occupied {
				state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{portcullisActionNPC(gate)})
			}
			state.DateTime.Hour, state.DateTime.Minute = test.hour, 0
			before, date := state.MapState.PlayerLocation, state.DateTime
			callbacks.Reset()

			assert.Equal(t, test.want, state.ActionPushSmallMap(references.Right))

			assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), layer.SpriteIndexByLayer(map_state.MapLayer, arch))
			if test.want {
				assert.Equal(t, object, state.MapState.PlayerLocation.Position)
				assert.True(t, layer.GetTileTopMapOnlyTile(&gate).IsChair())
				message := "Pushed!"
				if test.pull {
					message = "Pulled!"
				}
				assert.Equal(t, []string{message}, callbacks.Messages)
				assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
			} else {
				assert.Equal(t, before, state.MapState.PlayerLocation)
				assert.Equal(t, date, state.DateTime)
				assert.Empty(t, callbacks.TimeAdvanced())
				assert.Equal(t, []string{"Won't budge"}, callbacks.Messages)
				assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.SpriteIndexByLayer(map_state.MapLayer, gate), "rejection cannot swap away the gate's base floor")
				assert.Equal(t, indexes.SpriteIndex(indexes.ChairBackForward), layer.SpriteIndexByLayer(map_state.MapLayer, object))
				state.DateTime.Hour = 20
				assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), state.GetEffectiveTile(&gate))
			}
		})
	}
}

func TestPortcullisOpenActionUsesTerrainAndChestAuthority(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(14, 23).WithRandomSeed(525).WithSystemCallbacks().Build()
	gate := references.Position{X: 14, Y: 22}
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	for _, hour := range []byte{4, 5, 19, 20} {
		state.DateTime.Hour, state.DateTime.Minute = hour, 0
		before, calls := state.DateTime, state.GetRNG().Calls()
		callbacks.Reset()
		assert.False(t, state.ActionOpenSmallMap(references.Up))
		want := "Too heavy!"
		if hour >= 5 && hour < 20 {
			want = "Nothing to open!"
		}
		assert.Equal(t, []string{want}, callbacks.Messages)
		assert.Equal(t, before, state.DateTime)
		assert.Equal(t, calls, state.GetRNG().Calls())
		assert.False(t, state.MapState.HasTransientDoor())
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, gate))
	}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &gate, indexes.BrickFloor)
	callbacks.Reset()
	assert.False(t, state.ActionOpenSmallMap(references.Up))
	assert.Equal(t, []string{"Nothing to open!"}, callbacks.Messages, "explicit overrides win over the scheduled gate")
	state.GetCurrentLayeredMap().UnSetTileByLayer(map_state.MapOverrideLayer, &gate)
	chest := map_units.NewNPCFriendlySurfaceChest(references.SerpentsHold, gate, 0, 30)
	chest.SetPos(gate)
	chest.SetVisible(true)
	chest.UnlockSurfaceChest()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{chest})
	actor := state.PartyState.GetCharacter(0)
	actor.PartyMap = party_state.PartyMapActive
	actor.SetStatus(party_state.Good)
	callbacks.Reset()
	require.True(t, state.ActionOpenSmallMapWithCharacter(references.Up, actor))
	assert.Nil(t, state.surfaceChestAt(&gate), "a real chest on the square still takes priority")
	assert.NotContains(t, callbacks.Messages, "Too heavy!")
	assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), state.GetEffectiveTile(&gate))
}

func TestPortcullisTalkTargetUsesScheduledTerrainAndAdjacentOccupants(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(13, 22).WithSystemCallbacks().Build()
	gate, farther := references.Position{X: 14, Y: 22}, references.Position{X: 15, Y: 22}
	target := portcullisActionNPC(farther)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{target})
	for _, hour := range []byte{4, 5, 19, 20} {
		state.DateTime.Hour = hour
		// This is the production target operation invoked by startTalkWorkflow.
		position, unit := state.FindNPCInDirection(state.MapState.PlayerLocation.Position, references.Right, 2)
		if hour >= 5 && hour < 20 {
			assert.Nil(t, position, "ordinary floor does not extend the original one-square Talk range")
			assert.Nil(t, unit)
		} else {
			assert.Equal(t, &farther, position)
			assert.Same(t, target, unit)
		}
	}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &gate, indexes.BrickFloor)
	position, unit := state.FindNPCInDirection(state.MapState.PlayerLocation.Position, references.Right, 2)
	assert.Nil(t, position)
	assert.Nil(t, unit)
	state.GetCurrentLayeredMap().UnSetTileByLayer(map_state.MapOverrideLayer, &gate)
	adjacent := portcullisActionNPC(gate)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{adjacent, target})
	for _, hour := range []byte{5, 20} {
		state.DateTime.Hour = hour
		position, unit = state.FindNPCInDirection(state.MapState.PlayerLocation.Position, references.Right, 2)
		assert.Equal(t, &gate, position)
		assert.Same(t, adjacent, unit, "an adjacent occupant wins before terrain-based reach")
	}
}

func TestPortcullisLookActionDescribesTerrainWithoutHidingOccupants(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(14, 23).WithSystemCallbacks().Build()
	gate := references.Position{X: 14, Y: 22}
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	state.GetCurrentLayeredMap().UnSetTileByLayer(map_state.MapUnitLayer, &gate)
	for _, test := range []struct {
		hour byte
		unit indexes.SpriteIndex
		want indexes.SpriteIndex
	}{
		{5, indexes.NoSprites, indexes.BrickFloor},
		{20, indexes.NoSprites, indexes.Portcullis},
		{20, indexes.TownsPersonKeyIndex, indexes.TownsPersonKeyIndex},
	} {
		state.DateTime.Hour, state.DateTime.Minute = test.hour, 0
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapUnitLayer, &gate, test.unit)
		callbacks.Reset()
		require.True(t, state.ActionLookSmallMap(references.Up))
		assert.Contains(t, callbacks.Messages, "Look-Thou dost see "+state.GameReferences.LookReferences.GetTileLookDescription(test.want))
		assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	}
}

func portcullisActionNPC(position references.Position) *map_units.NPCFriendly {
	reference := references.NPCReference{Location: references.SerpentsHold, Schedule: references.CreateNPCScheduledFixedOneLocation(position, 0)}
	reference.SetKeyIndex(indexes.TownsPersonKeyIndex)
	npc := map_units.NewNPCFriendly(reference, 0)
	npc.SetPos(position)
	npc.SetVisible(true)
	return npc
}
