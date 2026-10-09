package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type combatPushFixture struct {
	state     *GameState
	callbacks *MockSystemCallbacks
	actor     references.Position
	object    references.Position
	forward   references.Position
	identity  CombatTurnIdentity
}

func TestActionPushCombatMapExactSourceTileMatrix(t *testing.T) {
	objects := []indexes.SpriteIndex{
		indexes.Plant,
		indexes.ChairBackForward, indexes.ChairBackLeft, indexes.ChairBackBack, indexes.ChairBackRight,
		indexes.EndTable, indexes.Barrel, indexes.Vanity, indexes.WaterJugTable,
		indexes.Dresser, indexes.CoffeeTable, indexes.Box,
		indexes.CannonUp, indexes.CannonDown, indexes.CannonRight, indexes.CannonLeft,
	}
	require.Len(t, objects, 16)

	for _, object := range objects {
		t.Run(fmt.Sprintf("0x%02x", object), func(t *testing.T) {
			fixture := newCombatPushFixture(t, object, references.Right)
			legalFloor := indexes.SpriteIndex(indexes.BrickFloor)
			if object >= indexes.CannonUp && object <= indexes.CannonLeft {
				legalFloor = indexes.SpriteIndex(indexes.MetalFloor)
			}
			fixture.state.setCombatTerrain(fixture.forward, legalFloor)
			fixture.callbacks.Reset()

			require.True(t, fixture.state.ActionPushCombatMap(references.Right))

			assertCombatPushTerrain(t, fixture.state, fixture.object, legalFloor)
			assertCombatPushTerrain(t, fixture.state, fixture.forward, originalPushFacing(object, references.Right))
			assert.Equal(t, fixture.object, fixture.state.CombatScene.GetCombatantPosition(fixture.identity))
			assert.Equal(t, []string{"Pushed!"}, fixture.callbacks.Messages)
			assertCombatPushMovementEvent(t, fixture.callbacks, fixture.identity, fixture.actor, fixture.object)
		})
	}
}

func TestActionPushCombatMapRejectsReduxOnlyObjectAndExactWrongFloors(t *testing.T) {
	tests := []struct {
		name         string
		object       indexes.SpriteIndex
		actorFloor   indexes.SpriteIndex
		forwardFloor indexes.SpriteIndex
		wantMessage  string
	}{
		{name: "Redux table is unsupported", object: indexes.TableMiddle, actorFloor: indexes.BrickFloor, forwardFloor: indexes.BrickFloor, wantMessage: "Won't budge!"},
		{name: "ordinary object rejects metal", object: indexes.Barrel, actorFloor: indexes.Grass, forwardFloor: indexes.MetalFloor, wantMessage: "Won't budge"},
		{name: "cannon rejects brick", object: indexes.CannonUp, actorFloor: indexes.Grass, forwardFloor: indexes.BrickFloor, wantMessage: "Won't budge"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCombatPushFixture(t, test.object, references.Right)
			fixture.state.setCombatTerrain(fixture.actor, test.actorFloor)
			fixture.state.setCombatTerrain(fixture.forward, test.forwardFloor)
			fixture.callbacks.Reset()

			assert.False(t, fixture.state.ActionPushCombatMap(references.Right))

			assertCombatPushTerrain(t, fixture.state, fixture.object, test.object)
			assert.Equal(t, fixture.actor, fixture.state.CombatScene.GetCombatantPosition(fixture.identity))
			assert.Equal(t, []string{test.wantMessage}, fixture.callbacks.Messages)
			assert.Empty(t, fixture.callbacks.CombatEvents)
		})
	}
}

func TestActionPushCombatMapPushPullAndNumericRotation(t *testing.T) {
	for _, test := range []struct {
		name      string
		object    indexes.SpriteIndex
		direction references.Direction
		floor     indexes.SpriteIndex
	}{
		{name: "chair", object: indexes.ChairBackLeft, direction: references.Down, floor: indexes.BrickFloor},
		{name: "cannon", object: indexes.CannonLeft, direction: references.Left, floor: indexes.MetalFloor},
	} {
		t.Run(test.name+" push", func(t *testing.T) {
			fixture := newCombatPushFixture(t, test.object, test.direction)
			fixture.state.setCombatTerrain(fixture.forward, test.floor)
			fixture.callbacks.Reset()

			require.True(t, fixture.state.ActionPushCombatMap(test.direction))
			assertCombatPushTerrain(t, fixture.state, fixture.forward, originalPushFacing(test.object, test.direction))
			assert.Equal(t, []string{"Pushed!"}, fixture.callbacks.Messages)
		})

		t.Run(test.name+" pull", func(t *testing.T) {
			fixture := newCombatPushFixture(t, test.object, test.direction)
			fixture.state.setCombatTerrain(fixture.actor, test.floor)
			fixture.state.setCombatTerrain(fixture.forward, indexes.StoneBrickWall)
			fixture.callbacks.Reset()

			require.True(t, fixture.state.ActionPushCombatMap(test.direction))
			assertCombatPushTerrain(t, fixture.state, fixture.actor, originalPushFacing(test.object, test.direction.GetOppositeDirection()))
			assertCombatPushTerrain(t, fixture.state, fixture.object, test.floor)
			assert.Equal(t, []string{"Pulled!"}, fixture.callbacks.Messages)
		})
	}
}

func TestActionPushCombatMapUsesExactPCCloneAndControlledMonsterIdentity(t *testing.T) {
	t.Run("PC clone", func(t *testing.T) {
		fixture := newCombatPushFixture(t, indexes.Barrel, references.Right)
		source := fixture.identity
		fixture.state.CombatScene.SetPartyMemberPosition(source.EntityIndex, references.Position{X: 1, Y: 1})
		clone := fixture.state.CombatScene.addPCCloneFromSource(
			source,
			source.EntityIndex,
			fixture.actor,
			NewCombatPCProvider(fixture.state.PartyState.GetCharacter(source.EntityIndex)),
		)
		require.NotZero(t, clone.Slot)
		fixture.state.CombatScene.CurrentTurnIdentity = clone
		fixture.identity = clone
		fixture.callbacks.Reset()

		require.True(t, fixture.state.ActionPushCombatMap(references.Right))

		assert.Equal(t, fixture.object, fixture.state.CombatScene.GetCombatantPosition(clone))
		assert.Equal(t, references.Position{X: 1, Y: 1}, fixture.state.CombatScene.GetPartyMemberPosition(source.EntityIndex))
		assertCombatPushMovementEvent(t, fixture.callbacks, clone, fixture.actor, fixture.object)
	})

	t.Run("controlled monster", func(t *testing.T) {
		fixture := newCombatPushFixture(t, indexes.Barrel, references.Right)
		fixture.state.CombatScene.SetPartyMemberPosition(fixture.identity.EntityIndex, references.Position{X: 1, Y: 1})
		monster, monsterIndex := addCombatPushMonster(fixture.state, fixture.actor, CombatMonsterLoyaltyParty)
		authorizeControlledMonster(fixture.state, monsterIndex)
		fixture.identity = CombatTurnIdentity{EntityIndex: monsterIndex}
		fixture.callbacks.Reset()

		require.True(t, fixture.state.ActionPushCombatMap(references.Right))

		assert.Equal(t, fixture.object, monster.Position())
		assertCombatPushMovementEvent(t, fixture.callbacks, fixture.identity, fixture.actor, fixture.object)
	})
}

func TestActionPushCombatMapDynamicOccupancy(t *testing.T) {
	occupants := []struct {
		name string
		add  func(*combatPushFixture, references.Position)
	}{
		{name: "ordinary PC", add: func(f *combatPushFixture, position references.Position) {
			f.state.CombatScene.PartyActive[1] = true
			f.state.CombatScene.SetPartyMemberPosition(1, position)
		}},
		{name: "PC clone", add: func(f *combatPushFixture, position references.Position) {
			clone := f.state.CombatScene.addPCCloneFromSource(
				f.identity, f.identity.EntityIndex, position,
				NewCombatPCProvider(f.state.PartyState.GetCharacter(f.identity.EntityIndex)),
			)
			require.NotZero(t, clone.Slot)
		}},
		{name: "monster", add: func(f *combatPushFixture, position references.Position) {
			addCombatPushMonster(f.state, position, CombatMonsterLoyaltyMonster)
		}},
		{name: "walkable combat object", add: func(f *combatPushFixture, position references.Position) {
			f.state.CombatScene.AddObject(NewCombatCorpse(position))
		}},
		{name: "non-protection field", add: func(f *combatPushFixture, position references.Position) {
			require.True(t, f.state.CombatScene.AddField(NewCombatField(CombatFieldPoison, position)))
		}},
	}

	for _, occupant := range occupants {
		t.Run(occupant.name+" rejects adjacent object square", func(t *testing.T) {
			fixture := newCombatPushFixture(t, indexes.Barrel, references.Right)
			occupant.add(fixture, fixture.object)
			fixture.callbacks.Reset()

			assert.False(t, fixture.state.ActionPushCombatMap(references.Right))
			assert.Equal(t, []string{"Won't budge!"}, fixture.callbacks.Messages)
			assertCombatPushTerrain(t, fixture.state, fixture.object, indexes.Barrel)
			assert.Empty(t, fixture.callbacks.CombatEvents)
		})

		t.Run(occupant.name+" blocks forward and selects pull", func(t *testing.T) {
			fixture := newCombatPushFixture(t, indexes.Barrel, references.Right)
			occupant.add(fixture, fixture.forward)
			fixture.callbacks.Reset()

			require.True(t, fixture.state.ActionPushCombatMap(references.Right))
			assert.Equal(t, []string{"Pulled!"}, fixture.callbacks.Messages)
			assertCombatPushTerrain(t, fixture.state, fixture.actor, indexes.Barrel)
			assert.Equal(t, fixture.object, fixture.state.CombatScene.GetCombatantPosition(fixture.identity))
		})
	}
}

func TestActionPushCombatMapTrapPrecedesOrdinaryHandlingForEveryControlledIdentity(t *testing.T) {
	for _, actorType := range []string{"PC", "PC clone", "controlled monster"} {
		t.Run(actorType, func(t *testing.T) {
			actor := references.Position{X: 2, Y: 2}
			trigger := references.Position{X: 3, Y: 2}
			state, callbacks := newSingleTriggerCombatState(t, trigger)
			identity := state.CombatScene.GetCurrentTurnIdentity()
			state.CombatScene.SetPartyMemberPosition(identity.EntityIndex, actor)
			state.setCombatTerrain(trigger, indexes.Barrel)

			switch actorType {
			case "PC":
			case "PC clone":
				state.CombatScene.SetPartyMemberPosition(identity.EntityIndex, references.Position{X: 1, Y: 1})
				identity = state.CombatScene.addPCCloneFromSource(
					identity, identity.EntityIndex, actor,
					NewCombatPCProvider(state.PartyState.GetCharacter(identity.EntityIndex)),
				)
				state.CombatScene.CurrentTurnIdentity = identity
			case "controlled monster":
				state.CombatScene.SetPartyMemberPosition(identity.EntityIndex, references.Position{X: 1, Y: 1})
				_, monsterIndex := addCombatPushMonster(state, actor, CombatMonsterLoyaltyParty)
				authorizeControlledMonster(state, monsterIndex)
				identity = CombatTurnIdentity{EntityIndex: monsterIndex}
			default:
				t.Fatal("unknown actor type")
			}
			callbacks.Reset()

			require.True(t, state.ActionPushCombatMap(references.Right))

			assertDungeonTriggerConsumed(t, state, 0)
			assert.Equal(t, actor, state.CombatScene.GetCombatantPosition(identity))
			assertCombatPushTerrain(t, state, trigger, indexes.Barrel)
			assert.Empty(t, callbacks.Messages)
			assert.Empty(t, callbacks.CombatEvents)
		})
	}
}

func TestActionPushCombatMapDoesNotTouchExplorationAuthorityOrRNG(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		WithRandomSeed(295).
		Build()
	doorPosition, doorDirection, doorPlayerPosition := findSmallMapDoorForCombatReturnTest(t, state)
	state.MapState.PlayerLocation.Position = doorPlayerPosition
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(doorDirection))
	require.True(t, state.MapState.HasTransientDoor())
	controllerBefore := state.CurrentNPCAIController
	smallMap := state.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, state.MapState.PlayerLocation.Floor)
	smallDoorBefore := smallMap.SpriteIndexByLayer(map_state.MapOverrideLayer, doorPosition)
	actor := references.Position{X: 4, Y: 5}
	object := references.Position{X: 5, Y: 5}
	forward := references.Position{X: 6, Y: 5}
	smallMap.SetTileByLayer(map_state.MapLayer, &actor, indexes.BrickFloor)
	smallMap.SetTileByLayer(map_state.MapLayer, &object, indexes.Barrel)
	smallMap.SetTileByLayer(map_state.MapLayer, &forward, indexes.BrickFloor)
	require.NoError(t, state.TransitionToCombatMap(
		int(references.BrickCombatMap), references.BritanniaCombatTerritory, CombatEntryNorth,
	))
	fixture := configureCombatPushFixture(t, state, callbacks, indexes.Barrel, references.Right)
	worldBefore := state.CombatScene.PreCombatPlayerLocation
	cameraBefore := state.MapState.PlayerLocation
	dateBefore := state.DateTime
	elapsedBefore := state.ElapsedMs
	rngBefore := state.GetRNG().Calls()
	callbacks.Reset()

	require.True(t, state.ActionPushCombatMap(references.Right))

	assert.Equal(t, worldBefore, state.CombatScene.PreCombatPlayerLocation)
	assert.Equal(t, cameraBefore, state.MapState.PlayerLocation)
	assert.Same(t, controllerBefore, state.CurrentNPCAIController)
	assert.Equal(t, smallDoorBefore, smallMap.SpriteIndexByLayer(map_state.MapOverrideLayer, doorPosition))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), smallMap.SpriteIndexByLayer(map_state.MapLayer, actor))
	assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), smallMap.SpriteIndexByLayer(map_state.MapLayer, object))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), smallMap.SpriteIndexByLayer(map_state.MapLayer, forward))
	assert.True(t, state.MapState.HasTransientDoor())
	assert.Equal(t, dateBefore, state.DateTime)
	assert.Equal(t, elapsedBefore, state.ElapsedMs)
	assert.Equal(t, rngBefore, state.GetRNG().Calls())
	assert.Equal(t, []string{"Pushed!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Empty(t, callbacks.SoundEffectsPlayed)
	assert.Zero(t, callbacks.FlowCallCount)
	assert.Zero(t, callbacks.DisplayUpdates)
	assert.Zero(t, callbacks.VisualCallCount)
	assert.Zero(t, callbacks.TalkCallCount)
	assertCombatPushTerrain(t, fixture.state, fixture.object, indexes.BrickFloor)
	assertCombatPushTerrain(t, fixture.state, fixture.forward, indexes.Barrel)
	assertCombatPushMovementEvent(t, callbacks, fixture.identity, fixture.actor, fixture.object)

	assert.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
	assert.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied), "combat Push must not decrement the small-map door timer")
	assert.True(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
}

func newCombatPushFixture(
	t *testing.T,
	object indexes.SpriteIndex,
	direction references.Direction,
) *combatPushFixture {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(295).
		Build()
	require.NoError(t, state.TransitionToCombatMap(
		int(references.BrickCombatMap), references.BritanniaCombatTerritory, CombatEntryNorth,
	))

	return configureCombatPushFixture(t, state, callbacks, object, direction)
}

func configureCombatPushFixture(
	t *testing.T,
	state *GameState,
	callbacks *MockSystemCallbacks,
	object indexes.SpriteIndex,
	direction references.Direction,
) *combatPushFixture {
	t.Helper()
	identity := state.CombatScene.GetCurrentTurnIdentity()
	actor := references.Position{X: 4, Y: 5}
	objectPosition := direction.GetNewPositionInDirection(&actor)
	require.NotNil(t, objectPosition)
	forwardPosition := direction.GetNewPositionInDirection(objectPosition)
	require.NotNil(t, forwardPosition)
	for index := range state.CombatScene.PartyActive {
		state.CombatScene.PartyActive[index] = index == identity.EntityIndex
	}
	clearCombatMonstersForTest(state.CombatScene)
	state.CombatScene.SetPartyMemberPosition(identity.EntityIndex, actor)
	state.setCombatTerrain(actor, indexes.BrickFloor)
	state.setCombatTerrain(*objectPosition, object)
	state.setCombatTerrain(*forwardPosition, indexes.BrickFloor)
	callbacks.Reset()

	return &combatPushFixture{
		state: state, callbacks: callbacks, actor: actor, object: *objectPosition,
		forward: *forwardPosition, identity: identity,
	}
}

func addCombatPushMonster(
	state *GameState,
	position references.Position,
	loyalty CombatMonsterLoyalty,
) (*CombatMonster, int) {
	reference := (*state.GameReferences.EnemyReferences)[references.MonsterOrc]
	monster := NewCombatMonster(references.MonsterOrc, &reference, position, newFixedRNG(1))
	monster.SetLoyalty(loyalty)

	return monster, state.CombatScene.AddMonsterToQueue(monster)
}

func assertCombatPushTerrain(
	t *testing.T,
	state *GameState,
	position references.Position,
	want indexes.SpriteIndex,
) {
	t.Helper()
	assert.Equal(t, want, state.CombatScene.CombatMapRef.GetTileNumber(position))
	assert.Equal(t, want, state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, position))
}

func assertCombatPushMovementEvent(
	t *testing.T,
	callbacks *MockSystemCallbacks,
	identity CombatTurnIdentity,
	from references.Position,
	to references.Position,
) {
	t.Helper()
	require.Len(t, callbacks.CombatEvents, 1)
	event := callbacks.CombatEvents[0]
	assert.Equal(t, CombatEventMove, event.Kind)
	assert.Equal(t, identity.IsPartyMember, event.Attacker.IsPartyMember)
	assert.Equal(t, identity.EntityIndex, event.Attacker.Index)
	assert.Equal(t, identity.Slot, event.Attacker.Slot)
	assert.Equal(t, to, event.Attacker.Position)
	assert.Equal(t, from, event.From)
	assert.Equal(t, to, event.To)
}
