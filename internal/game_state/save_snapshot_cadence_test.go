package game_state

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSaveSnapshotCadenceNativeContinuationAcrossPassAndTransitions(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld} {
		for _, phases := range []int{1, 2, 3} {
			for _, vehicle := range []references.VehicleType{references.HorseVehicle, references.CarpetVehicle} {
				t.Run(fmt.Sprintf("%s/phases=%d/vehicle=%d", location, phases, vehicle), func(t *testing.T) {
					state, _ := NewIntegrationTestBuilder(t).WithLocation(location).WithFloor(0).
						WithPlayerAt(15, 15).WithSystemCallbacks().Build()
					state.DateTime = datetime.UltimaDate{Year: 139, Month: 4, Day: 12, Hour: 8}
					state.lastProcessedHour = 8
					state.initializeMoongateStage()
					state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
					for _, world := range []references.World{references.OVERWORLD, references.UNDERWORLD} {
						state.LargeMapNPCAIController[world].GetNpcs().ReplaceWith(nil)
					}
					state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &state.MapState.PlayerLocation.Position, indexes.Grass)
					state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(vehicle, state.MapState.PlayerLocation.Position, 0)
					state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 30}
					// Seed the dormant context independently to catch cross-wired hydration.
					if location == references.Britain {
						state.outdoorMountedActorSkip = true
					} else {
						state.townQuicknessActorSkip = true
					}
					for range phases {
						state.shouldProcessNonCombatWorldActors()
					}
					state.SetRandomSeed(123)
					state.RollDice(17)
					snapshot, err := state.CaptureSaveSnapshot()
					require.NoError(t, err)
					assert.Equal(t, state.townMountedActorSkip, snapshot.Effects.TownMountedActorSkip)
					assert.Equal(t, state.townQuicknessActorSkip, snapshot.Effects.TownQuicknessActorSkip)
					assert.Equal(t, state.outdoorMountedActorSkip, snapshot.Effects.OutdoorMountedActorSkip)
					assert.Equal(t, state.quicknessWorldActorSkip, snapshot.Effects.QuicknessWorldActorSkip)
					data, err := save_snapshot.EncodeCanonical(save_snapshot.Envelope{
						Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
						Metadata: save_snapshot.Metadata{Name: "Cadence continuation", CreatedAt: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)},
						Snapshot: snapshot,
					})
					require.NoError(t, err)
					decoded, err := save_snapshot.Decode(data)
					require.NoError(t, err)
					hydrated, err := NewGameStateFromSaveSnapshot(decoded.Snapshot, state.GameConfig, state.GameReferences,
						xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
					require.NoError(t, err)
					hydrated.SystemCallbacks = NewMockSystemCallbacks(t).ToSystemCallbacks()
					recaptured, err := hydrated.CaptureSaveSnapshot()
					require.NoError(t, err)
					require.Equal(t, snapshot, recaptured, "hydrate/recapture preserves every phase after initialization")
					for _, current := range []*GameState{state, hydrated} {
						current.DebugOptions.MonsterGen = true
						if location == references.Britain {
							// Inject an observable actor draw without introducing encounter workflows.
							current.CurrentNPCAIController = &observingWorldController{observe: func() { current.RollDice(17) }}
						}
					}
					for _, pass := range []bool{true, false, false, true, false, false} {
						for _, current := range []*GameState{state, hydrated} {
							if pass {
								current.CompleteSurfacePassCycle(location.GetMapType())
							} else {
								current.CompleteSurfaceCommandCycle(location.GetMapType())
							}
						}
						left, leftErr := state.CaptureSaveSnapshot()
						right, rightErr := hydrated.CaptureSaveSnapshot()
						require.NoError(t, leftErr)
						require.NoError(t, rightErr)
						assert.Equal(t, left, right, "next command/pass must preserve authoritative state and RNG position")
					}
					assert.Greater(t, state.GetRNG().Calls(), snapshot.Random.SourceCalls, "fixture must reach observable actor RNG")
					for _, current := range []*GameState{state, hydrated} {
						outdoorMounted, outdoorQuick := current.outdoorMountedActorSkip, current.quicknessWorldActorSkip
						current.TransitionToLocation(references.BritanniaUnderworld, 0, references.Position{X: 100, Y: 100}, TransitionFullLocation, false)
						current.TransitionToLocation(references.Britain, 0, references.Position{X: 15, Y: 15}, TransitionFullLocation, false)
						assert.False(t, current.townMountedActorSkip)
						assert.False(t, current.townQuicknessActorSkip)
						assert.Equal(t, outdoorMounted, current.outdoorMountedActorSkip)
						assert.Equal(t, outdoorQuick, current.quicknessWorldActorSkip)
					}
					assert.Equal(t, state.GetRNG().Calls(), hydrated.GetRNG().Calls())
					assert.Equal(t, state.RandomIntInRange(0, 10000), hydrated.RandomIntInRange(0, 10000))
				})
			}
		}
	}
}

func TestSaveSnapshotCadenceOriginalImportStartsAllPhasesFalse(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	assert.False(t, state.townMountedActorSkip)
	assert.False(t, state.townQuicknessActorSkip)
	assert.False(t, state.outdoorMountedActorSkip)
	assert.False(t, state.quicknessWorldActorSkip)
}

func TestSaveSnapshotCadenceHistoricalQuicknessPreservesNextReachedGate(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld} {
		for _, oldSkipNext := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/skip=%v", location, oldSkipNext), func(t *testing.T) {
				state, _ := NewIntegrationTestBuilder(t).WithLocation(location).WithFloor(0).
					WithPlayerAt(15, 15).WithSystemCallbacks().Build()
				state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 30}
				snapshot, err := state.CaptureSaveSnapshot()
				require.NoError(t, err)
				envelope := save_snapshot.Envelope{
					Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
					Metadata: save_snapshot.Metadata{Name: "Historical cadence", CreatedAt: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)},
					Snapshot: snapshot,
				}
				data, err := save_snapshot.EncodeCanonical(envelope)
				require.NoError(t, err)
				var document map[string]any
				require.NoError(t, json.Unmarshal(data, &document))
				wire := document["snapshot"].(map[string]any)
				wire["version"] = 14
				delete(wire, "excerpts")
				removeSnapshotV24Fields(wire)
				delete(wire, "openDoor")
				delete(wire, "presentation")
				delete(wire, "lastConversation")
				delete(wire, "dungeonJournal")
				delete(wire, "moonstoneJournal")
				delete(wire, "atlasLandmarks")
				if units, ok := wire["mapUnits"].([]any); ok {
					for _, value := range units {
						delete(value.(map[string]any), "surfaceChestAssessment")
					}
				}
				for _, entry := range wire["redux"].(map[string]any)["questJournal"].([]any) {
					delete(entry.(map[string]any), "sourceName")
				}
				effects := wire["effects"].(map[string]any)
				delete(effects, "townMountedActorSkip")
				delete(effects, "townQuicknessActorSkip")
				delete(effects, "outdoorMountedActorSkip")
				effects["quicknessWorldActorSkip"] = oldSkipNext
				data, err = json.Marshal(document)
				require.NoError(t, err)
				decoded, err := save_snapshot.Decode(data)
				require.NoError(t, err)
				hydrated, err := NewGameStateFromSaveSnapshot(decoded.Snapshot, state.GameConfig, state.GameReferences,
					xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
				require.NoError(t, err)
				assert.Equal(t, !oldSkipNext, hydrated.shouldProcessNonCombatWorldActorsForCommand(true))
				assert.Equal(t, oldSkipNext, hydrated.shouldProcessNonCombatWorldActorsForCommand(true))
				assert.Equal(t, snapshot.Random.SourceCalls, hydrated.GetRNG().Calls())
			})
		}
	}
}

func TestSaveSnapshotMoongateContractionWaitsForStableRegrowth(t *testing.T) {
	for _, success := range []bool{false, true} {
		t.Run(fmt.Sprint(success), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithPlayerAt(100, 100).WithSystemCallbacks().Build()
			state.DateTime.Hour, state.DateTime.Minute = 21, 0
			state.initializeMoongateStage()
			require.NoError(t, state.ValidateSaveSnapshotEligibility())
			beforeDate, beforeCalls := state.DateTime, state.GetRNG().Calls()
			state.ApplyMoongateTravelEffects(success)
			assert.Equal(t, beforeDate, state.DateTime)
			assert.Equal(t, beforeCalls, state.GetRNG().Calls())
			err := state.ValidateSaveSnapshotEligibility()
			require.ErrorIs(t, err, ErrSaveSnapshotNotRepresentable)
			assert.Contains(t, err.Error(), "moongate")
			state.UpdateMoongateTransition()
			advanceMoongateWorkflow(state, 18)
			require.Equal(t, MoongateStageFull, state.MoongateStage())
			require.NoError(t, state.ValidateSaveSnapshotEligibility())
			state.ProcessWindThroughElapsed(state.ElapsedMs)
			snapshot, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			hydrated, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
				xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
			require.NoError(t, err)
			assert.Equal(t, MoongateStageFull, hydrated.MoongateStage())
			assert.Equal(t, beforeDate, hydrated.DateTime)
		})
	}
}
