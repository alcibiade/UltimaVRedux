package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func newPhysicalSailingInterruptionFixture(t *testing.T, wind environment.WindDirection) automaticSailingWorkflowFixture {
	t.Helper()
	fixture := newAutomaticSailingWorkflowFixture(t, wind)
	initializeDungeonUpdateTestScene(fixture.scene, fixture.state)
	fixture.scene.now = func() time.Time { return time.UnixMilli(fixture.state.ElapsedMs) }
	fixture.scene.lastEffectsUpdate = fixture.scene.now()
	fixture.scene.keyboard.SetElapsedMillisecondsSource(fixture.scene.getElapsedMs)
	fixture.state.SystemCallbacks.Visual.BlockAllInput = fixture.scene.blockAllInput
	fixture.state.CurrentNPCAIController.RemoveAllEnemies()
	fixture.state.DateTime.Hour, fixture.state.DateTime.Minute = 12, 0
	origin := fixture.state.MapState.PlayerLocation.Position
	for x := origin.X - 4; x <= origin.X+4; x++ {
		for y := origin.Y - 4; y <= origin.Y+4; y++ {
			position := references.Position{X: x, Y: y}
			fixture.state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water2)
		}
	}
	t.Cleanup(func() {
		clear(fixture.held)
		fixture.state.EventQueueManager.Shutdown()
	})

	return fixture
}

func TestAutomaticSailingShortYBetweenProductionInputPhases(t *testing.T) {
	for _, test := range []struct {
		name      string
		wind      environment.WindDirection
		direction references.Direction
		phases    int
	}{
		{name: "before first phase", wind: environment.North, direction: references.Right},
		{name: "after movement", wind: environment.North, direction: references.Right, phases: 1},
		{name: "headwind wait", wind: environment.North, direction: references.Up, phases: 1},
		{name: "tailwind wait", wind: environment.North, direction: references.Down, phases: 1},
		{name: "calm wait", wind: environment.Calm, direction: references.Up, phases: 2},
	} {
		for _, smooth := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/smooth%v", test.name, smooth), func(t *testing.T) {
				fixture := newPhysicalSailingInterruptionFixture(t, test.wind)
				enableAutomaticSailingCamera(fixture.scene, smooth)
				fixture.scene.largeMapInputHandler(automaticSailingDirectionKey(test.direction))
				fixture.advancePhases(test.phases)
				beforePosition := fixture.state.MapState.PlayerLocation
				beforeMinutes := fixture.elapsedMinutes()
				fixture.held[ebiten.KeyY] = true
				fixture.state.ElapsedMs += 10
				require.NoError(t, fixture.scene.Update(nil))
				require.False(t, fixture.state.IsSailing(), "physical Y must stop before the next cadence")
				require.Equal(t, map_units.SailFurled, fixture.state.PartyVehicle.GetVehicleDetails().SailState)
				assert.Equal(t, test.direction, fixture.state.PartyVehicle.GetVehicleDetails().CurrentDirection())
				assert.Equal(t, beforeMinutes+game_state.DefaultLargeMapMinutesPerTurn, fixture.elapsedMinutes(),
					"only ordinary Y completion is charged")
				delete(fixture.held, ebiten.KeyY)
				fixture.state.ElapsedMs++
				require.NoError(t, fixture.scene.Update(nil))
				beforeTime, beforeRNG := fixture.state.DateTime, fixture.state.GetRNG().Calls()
				fixture.advancePhases(5)
				fixture.state.ElapsedMs += 5000 // A delayed frame must not catch up sailing steps.
				fixture.state.EventQueueManager.Update()
				fixture.scene.updateSurfaceCamera()
				assert.Equal(t, beforePosition, fixture.state.MapState.PlayerLocation)
				assert.Equal(t, beforeTime, fixture.state.DateTime)
				assert.Equal(t, beforeRNG, fixture.state.GetRNG().Calls())
				assert.False(t, fixture.state.EventQueueManager.HasWorkflow(automaticSailingWorkflowName))
				assert.Zero(t, fixture.scene.surfaceCamera)
				assert.False(t, fixture.scene.inputBlocked)
				// A different direction first turns a frigate without moving it.
				// Use the preserved heading to verify fresh production rowing here.
				key := automaticSailingDirectionKey(test.direction)
				assert.False(t, fixture.scene.keyboard.IsSuppressedUntilRelease(key))
				fixture.held[key] = true
				fixture.state.ElapsedMs++
				require.NoError(t, fixture.scene.Update(nil))
				wantPosition := test.direction.GetNewPositionInDirection(&beforePosition.Position).
					GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
				assert.Equal(t, *wantPosition, fixture.state.MapState.PlayerLocation.Position, "fresh manual rowing remains usable")
				assert.Contains(t, fixture.callbacks.CommandPrompts, "Row "+test.direction.GetDirectionCompassName())
				assert.False(t, fixture.state.IsSailing())
				assert.False(t, fixture.state.EventQueueManager.HasWorkflow(automaticSailingWorkflowName))
			})
		}
	}
}

func TestFurledFrigateFreshDirectionTurnsThenRows(t *testing.T) {
	fixture := newPhysicalSailingInterruptionFixture(t, environment.North)
	fixture.scene.largeMapInputHandler(ebiten.KeyUp)
	fixture.advancePhases(1)
	fixture.held[ebiten.KeyY] = true
	fixture.state.ElapsedMs += 10
	require.NoError(t, fixture.scene.Update(nil))
	delete(fixture.held, ebiten.KeyY)
	fixture.state.ElapsedMs++
	require.NoError(t, fixture.scene.Update(nil))
	fixture.advancePhases(2)
	require.False(t, fixture.state.EventQueueManager.HasWorkflow(automaticSailingWorkflowName))
	position := fixture.state.MapState.PlayerLocation.Position
	fixture.held[ebiten.KeyRight] = true
	fixture.state.ElapsedMs++
	require.NoError(t, fixture.scene.Update(nil))
	assert.Equal(t, position, fixture.state.MapState.PlayerLocation.Position, "first eastward press changes heading only")
	assert.Equal(t, references.Right, fixture.state.PartyVehicle.GetVehicleDetails().CurrentDirection())
	assert.False(t, fixture.state.IsSailing())
	delete(fixture.held, ebiten.KeyRight)
	fixture.state.ElapsedMs++
	require.NoError(t, fixture.scene.Update(nil))
	fixture.held[ebiten.KeyRight] = true
	fixture.state.ElapsedMs++
	require.NoError(t, fixture.scene.Update(nil))
	assert.Equal(t, *references.Right.GetNewPositionInDirection(&position), fixture.state.MapState.PlayerLocation.Position)
	assert.False(t, fixture.state.IsSailing())
}

func TestAutomaticSailingPendingPhaseRevalidatesVehicleAndContext(t *testing.T) {
	for _, invalidation := range []string{"sails", "skiff", "town"} {
		t.Run(invalidation, func(t *testing.T) {
			fixture := newAutomaticSailingWorkflowFixture(t, environment.Calm)
			fixture.scene.largeMapInputHandler(ebiten.KeyUp)
			fixture.advancePhases(1)
			switch invalidation {
			case "sails":
				fixture.state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailFurled)
			case "skiff":
				fixture.state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
					references.SkiffVehicle, fixture.state.MapState.PlayerLocation.Position, 0)
			case "town":
				fixture.state.MapState.PlayerLocation.Location = references.Britain
			}
			position, date := fixture.state.MapState.PlayerLocation, fixture.state.DateTime
			rng := fixture.state.GetRNG().Calls()
			fixture.advancePhases(10)
			assert.False(t, fixture.state.IsSailing())
			assert.False(t, fixture.state.EventQueueManager.HasWorkflow(automaticSailingWorkflowName))
			assert.Equal(t, position, fixture.state.MapState.PlayerLocation)
			assert.Equal(t, date, fixture.state.DateTime)
			assert.Equal(t, rng, fixture.state.GetRNG().Calls())
		})
	}
}

func TestRangedInterruptionRequiresFreshDirectionAndLeavesFireFurlAvailable(t *testing.T) {
	for _, heldArrow := range []ebiten.Key{ebiten.KeyRight, ebiten.KeyNumpad6} {
		t.Run(heldArrow.String(), func(t *testing.T) {
			fixture := newPhysicalSailingInterruptionFixture(t, environment.North)
			enableAutomaticSailingCamera(fixture.scene, true)
			fixture.scene.largeMapInputHandler(ebiten.KeyRight)
			fixture.advancePhases(1)
			fixture.held[heldArrow] = true
			position := fixture.state.MapState.PlayerLocation
			fixture.state.ResolveRangedSourceHit()
			fixture.state.ElapsedMs++
			require.NoError(t, fixture.scene.Update(nil))
			require.True(t, fixture.scene.keyboard.IsSuppressedUntilRelease(heldArrow))
			fixture.scene.largeMapInputHandler(primaryMovementKey(heldArrow))
			assert.False(t, fixture.state.IsSailing(), "direct direction routing must also reject the interrupted physical hold")
			assert.Equal(t, position, fixture.state.MapState.PlayerLocation)
			fixture.advancePhases(4)
			date := fixture.state.DateTime
			for range 4 {
				fixture.state.ElapsedMs += 100
				require.NoError(t, fixture.scene.Update(nil))
			}
			assert.False(t, fixture.state.IsSailing())
			assert.Equal(t, position, fixture.state.MapState.PlayerLocation)
			assert.Equal(t, date, fixture.state.DateTime)
			assert.Zero(t, fixture.scene.surfaceCamera)
			assert.Equal(t, map_units.SailRaised, fixture.state.PartyVehicle.GetVehicleDetails().SailState)

			// A suppressed arrow must not starve the ordinary Fire command.
			fixture.held[ebiten.KeyF] = true
			fixture.state.ElapsedMs++
			require.NoError(t, fixture.scene.Update(nil))
			assert.Equal(t, FireDirectionInput, fixture.scene.secondaryKeyState)
			clear(fixture.held)
			fixture.state.ElapsedMs++
			require.NoError(t, fixture.scene.Update(nil))
			fixture.held[ebiten.KeyUp] = true // Broadside to the preserved eastward heading.
			fixture.state.ElapsedMs++
			require.NoError(t, fixture.scene.Update(nil))
			delete(fixture.held, ebiten.KeyUp)
			for range 30 {
				fixture.state.ElapsedMs += 100
				require.NoError(t, fixture.scene.Update(nil))
			}
			assert.Equal(t, PrimaryInput, fixture.scene.secondaryKeyState)
			assert.False(t, fixture.state.EventQueueManager.HasActiveWorkflows())
			fixture.held[heldArrow] = true
			fixture.state.ElapsedMs++
			require.NoError(t, fixture.scene.Update(nil))
			assert.True(t, fixture.state.IsSailing(), "release plus fresh direction resumes without hoisting")
			delete(fixture.held, heldArrow)
			fixture.state.ElapsedMs++
			require.NoError(t, fixture.scene.Update(nil))
			fixture.held[ebiten.KeyY] = true
			fixture.state.ElapsedMs++
			require.NoError(t, fixture.scene.Update(nil))
			assert.False(t, fixture.state.IsSailing())
			assert.Equal(t, map_units.SailFurled, fixture.state.PartyVehicle.GetVehicleDetails().SailState)
		})
	}
}

func TestAutomaticSailingRangedSinkingNeverMovesFallback(t *testing.T) {
	for _, direction := range []references.Direction{references.Right, references.Up} {
		for _, fallback := range []references.VehicleType{references.SkiffVehicle, references.CarpetVehicle, references.NoPartyVehicle} {
			t.Run(direction.GetDirectionCompassName()+"/"+fallback.GetLowerCaseName(), func(t *testing.T) {
				fixture := newPhysicalSailingInterruptionFixture(t, environment.North)
				state := fixture.state
				origin := state.MapState.PlayerLocation.Position
				for x := origin.X - 5; x <= origin.X+5; x++ {
					for y := origin.Y - 5; y <= origin.Y+5; y++ {
						position := references.Position{X: x, Y: y}
						state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)
					}
				}
				for index := range state.PartyState.GetCharacterCount() {
					state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapAtInn(references.Britain)
				}
				member := state.PartyState.GetCharacter(0)
				member.PartyMap = party_state.PartyMapActive
				member.SetMaxHp(100)
				member.SetHp(100)
				member.SetStatus(party_state.Good)
				details := state.PartyVehicle.GetVehicleDetails()
				details.SetHullHP(1)
				details.SetSkiffQuantity(0)
				state.PartyState.Inventory.QuestItems.Set(references.Carpet, 0)
				if fallback == references.SkiffVehicle {
					details.SetSkiffQuantity(1)
				} else if fallback == references.CarpetVehicle {
					state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
				}
				// Right commits movement then attacks; Up runs a headwind wait then attacks.
				piratePosition := references.Position{X: origin.X + 3, Y: origin.Y}
				tile := state.GameReferences.TileReferences.GetTile(indexes.PirateShipDown)
				pirate := map_units.NewEnemyNPC(references.NewPirateShipEnemyReference(tile), 31)
				pirate.SetPos(piratePosition)
				pirate.SetFloor(state.MapState.PlayerLocation.Floor)
				pirate.SetVisible(true)
				require.True(t, state.CurrentNPCAIController.GetNpcs().AddEnemy(pirate))
				fixture.scene.largeMapInputHandler(automaticSailingDirectionKey(direction))
				fixture.advancePhases(1)
				require.False(t, state.IsSailing())
				require.Equal(t, fallback, state.PartyVehicle.GetVehicleDetails().VehicleType)
				position, date, rng := state.MapState.PlayerLocation, state.DateTime, state.GetRNG().Calls()
				for range 30 {
					state.ElapsedMs += 1000
					state.EventQueueManager.Update()
				}
				assert.Equal(t, position, state.MapState.PlayerLocation)
				assert.Equal(t, date, state.DateTime)
				assert.Equal(t, rng, state.GetRNG().Calls())
				assert.False(t, state.EventQueueManager.HasActiveWorkflows())
				assert.False(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
				assert.Zero(t, fixture.scene.surfaceCamera)
				assert.False(t, fixture.scene.inputBlocked)
				assert.Equal(t, fallback == references.NoPartyVehicle, state.PartyState.IsAllPartyDead())
				if fallback != references.NoPartyVehicle {
					state.CurrentNPCAIController.RemoveAllEnemies()
					require.NotPanics(t, func() { fixture.scene.largeMapInputHandler(ebiten.KeyDown) })
					assert.NotEqual(t, position, state.MapState.PlayerLocation, "fallback accepts fresh manual movement")
					assert.False(t, state.IsSailing())
					for range 20 {
						state.ElapsedMs += 1000
						state.EventQueueManager.Update()
					}
					assert.False(t, state.EventQueueManager.HasActiveWorkflows())
					assert.False(t, fixture.scene.inputBlocked)
				}
			})
		}
	}
}
