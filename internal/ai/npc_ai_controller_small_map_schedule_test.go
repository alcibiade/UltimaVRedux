package ai

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type originalScheduleRNGCall struct {
	minimum int
	maximum int
}

type scriptedOriginalScheduleRNG struct {
	t      *testing.T
	values []int
	calls  []originalScheduleRNGCall
}

func (r *scriptedOriginalScheduleRNG) RandomIntInRange(minimum, maximum int) int {
	r.t.Helper()
	r.calls = append(r.calls, originalScheduleRNGCall{minimum: minimum, maximum: maximum})
	require.NotEmpty(r.t, r.values, "unexpected original schedule RNG call")
	value := r.values[0]
	r.values = r.values[1:]
	require.GreaterOrEqual(r.t, value, minimum)
	require.LessOrEqual(r.t, value, maximum)

	return value
}

func (r *scriptedOriginalScheduleRNG) OneInXOdds(int) bool {
	r.t.Fatal("original schedule actions must not use OneInXOdds")

	return false
}

func (r *scriptedOriginalScheduleRNG) ShuffleDirections([]references.Position) {
	r.t.Fatal("original schedule actions must not shuffle directions")
}

func TestOriginalWanderActionsUseGateDirectionAndManhattanAnchor(t *testing.T) {
	t.Run("Wander3 only attempts selected direction", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Wander3, references.Position{X: 10, Y: 10})
		selected := references.Position{X: 11, Y: 10}
		controller.mapState.PlayerLocation.Position = selected

		controller.randomStepOriginal(npc, npc.Pos(), 3)
		assert.Equal(t, references.Position{X: 10, Y: 10}, npc.Pos())
		assert.Equal(t, []originalScheduleRNGCall{{0, 255}, {0, 64}}, rng.calls)
	})

	t.Run("Wander3 rejects a fourth Manhattan step", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Wander3, references.Position{X: 13, Y: 10})

		controller.randomStepOriginal(npc, references.Position{X: 10, Y: 10}, 3)
		assert.Equal(t, references.Position{X: 13, Y: 10}, npc.Pos())
	})

	t.Run("Wander has no anchor limit", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.BigWander, references.Position{X: 13, Y: 10})

		controller.randomStepOriginal(npc, references.Position{X: 10, Y: 10}, 0)
		assert.Equal(t, references.Position{X: 14, Y: 10}, npc.Pos())
	})
}

func TestOriginalScheduleActionsUseSourceRangesAndRNG(t *testing.T) {
	t.Run("Flee3 escapes while adjacent", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t, values: []int{1, 0}}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.ChildRunAway, references.Position{X: 10, Y: 10})
		controller.mapState.PlayerLocation.Position = references.Position{X: 11, Y: 10}
		controller.attractOriginal(npc, references.ChildRunAway)
		assert.Equal(t, references.Position{X: 9, Y: 10}, npc.Pos())
		assert.Equal(t, []originalScheduleRNGCall{{0, 1}, {0, 1}}, rng.calls)
	})
	t.Run("TalkAlways approaches outside anchor range then talks", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t, values: []int{63}}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.TalkAlways, references.Position{X: 10, Y: 10})
		controller.mapState.PlayerLocation.Position = references.Position{X: 12, Y: 10}
		npc.NPCReference.DialogNumber = 1
		behavior := references.IndividualNPCBehaviour{Ai: references.TalkAlways, Position: references.Position{X: 1, Y: 1}}
		controller.performOriginalScheduleAction(npc, behavior)
		assert.Equal(t, references.Position{X: 11, Y: 10}, npc.Pos())
		controller.performOriginalScheduleAction(npc, behavior)
		assert.Equal(t, []originalScheduleRNGCall{{0, 63}}, rng.calls)
		assert.Equal(t, map_units.PendingNPCInteractionTalk, npc.MapUnitDetails().PendingInteraction)
		assert.True(t, references.AiType(references.TalkAlways).IsValid())
	})
	t.Run("Flee3 chooses among farther directions in source order", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t, values: []int{1, 0}}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.ChildRunAway, references.Position{X: 10, Y: 10})
		controller.mapState.PlayerLocation.Position = references.Position{X: 12, Y: 10}

		controller.attractOriginal(npc, references.ChildRunAway)
		assert.Equal(t, references.Position{X: 9, Y: 10}, npc.Pos())
		assert.Equal(t, []originalScheduleRNGCall{{0, 1}, {0, 1}}, rng.calls)
	})

	t.Run("Talk3 measures Avatar distance from its schedule anchor", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Talk3, references.Position{X: 15, Y: 10})
		controller.mapState.PlayerLocation.Position = references.Position{X: 13, Y: 10}

		controller.performOriginalScheduleAction(npc, references.IndividualNPCBehaviour{
			Ai: references.Talk3, Position: references.Position{X: 10, Y: 10}, Floor: 0,
		})

		assert.Equal(t, references.Position{X: 14, Y: 10}, npc.Pos())
		assert.Empty(t, rng.calls)
	})

	t.Run("Attack3 stops at Manhattan distance four", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.ExtortOrAttackOrFollow, references.Position{X: 10, Y: 10})
		controller.mapState.PlayerLocation.Position = references.Position{X: 12, Y: 12}

		controller.performOriginalScheduleAction(npc, references.IndividualNPCBehaviour{Ai: references.ExtortOrAttackOrFollow})

		assert.Equal(t, references.Position{X: 10, Y: 10}, npc.Pos())
		assert.Empty(t, rng.calls)
	})

	t.Run("Attack variation may choose a non-greedy legal direction", func(t *testing.T) {
		rng := &scriptedOriginalScheduleRNG{t: t, values: []int{15, 16, 16}}
		controller, npc, _ := newOriginalScheduleTestController(t, rng, references.DrudgeWorthThing, references.Position{X: 10, Y: 10})
		controller.mapState.PlayerLocation.Position = references.Position{X: 12, Y: 10}

		controller.attractOriginal(npc, references.DrudgeWorthThing)
		assert.Equal(t, references.Position{X: 10, Y: 9}, npc.Pos())
		assert.Equal(t, []originalScheduleRNGCall{{0, 63}, {0, 63}, {0, 63}}, rng.calls)
	})
}

func TestOriginalAttackVariationKeepsPreferredDirectionExcluded(t *testing.T) {
	for preferredIndex, direction := range originalNPCDirections {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("preferred%d/replace%t", preferredIndex, replace), func(t *testing.T) {
				roll := 63
				if replace {
					roll = 0
				}
				rng := &scriptedOriginalScheduleRNG{t: t, values: []int{0, roll, roll, 197}}
				current := references.Position{X: 10, Y: 10}
				controller, npc, _ := newOriginalScheduleTestController(t, rng, references.DrudgeWorthThing, current)
				preferred := references.Position{X: current.X + direction.X, Y: current.Y + direction.Y}
				wantIndex := 0
				if preferredIndex == 0 {
					wantIndex = 1
				}
				if replace {
					wantIndex = 3
					if preferredIndex == 3 {
						wantIndex = 2
					}
				}
				wantDirection := originalNPCDirections[wantIndex]
				want := references.Position{X: current.X + wantDirection.X, Y: current.Y + wantDirection.Y}

				chosen := controller.originalAttackVariation(npc, current, &preferred)

				require.NotNil(t, chosen)
				assert.Equal(t, want, *chosen)
				assert.Equal(t, []originalScheduleRNGCall{{0, 63}, {0, 63}, {0, 63}}, rng.calls,
					"one gate, no draw for the first alternative, then two replacement draws")
				assert.Equal(t, 197, rng.RandomIntInRange(0, 255), "preferred direction must not consume a replacement draw")
				assert.Empty(t, rng.values)
			})
		}
	}
}

func TestSmallMapAttackActionsDispatchByCombatCapability(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		npcSlot  int
		sprite   indexes.SpriteIndex
		want     map_units.PendingNPCInteraction
	}{
		{
			name:     "Iolo hut rat",
			location: references.IolosHut,
			npcSlot:  1,
			sprite:   indexes.RatKeyIndex,
			want:     map_units.PendingNPCInteractionHostileCombat,
		},
		{
			name:     "Skara Brae bat",
			location: references.SkaraBrae,
			npcSlot:  8,
			sprite:   indexes.BatKeyIndex,
			want:     map_units.PendingNPCInteractionHostileCombat,
		},
		{
			name:     "rat outside visit activation locations",
			location: references.Yew,
			npcSlot:  1,
			sprite:   indexes.RatKeyIndex,
			want:     map_units.PendingNPCInteractionHostileCombat,
		},
		{name: "Stonegate bat", location: references.Stonegate, npcSlot: 9, sprite: indexes.BatKeyIndex + 1, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "Windemere rat", location: references.Windemere, npcSlot: 10, sprite: indexes.RatKeyIndex + 2, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "ordinary human", location: references.Britain, npcSlot: 2, sprite: indexes.TownsPersonKeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "Yew gargoyle", location: references.Yew, npcSlot: 3, sprite: indexes.StoneGargoyleKeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "Palace gargoyle", location: references.PalaceOfBlackthorn, npcSlot: 4, sprite: indexes.StoneGargoyleKeyIndex + 3, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "activated Balinor", location: references.Windemere, npcSlot: 5, sprite: indexes.DaemonKeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "guard", location: references.Britain, npcSlot: 6, sprite: indexes.GuardKeyIndex + 1, want: map_units.PendingNPCInteractionGuardArrest},
		{name: "quest object", location: references.Britain, npcSlot: 7, sprite: indexes.Chest, want: map_units.PendingNPCInteractionNone},
	}

	for _, test := range tests {
		for _, action := range []references.AiType{references.ExtortOrAttackOrFollow, references.DrudgeWorthThing} {
			for _, override := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/action%d/override%t", test.name, action, override), func(t *testing.T) {
					rng := &scriptedOriginalScheduleRNG{t: t}
					controller, npc, _ := newOriginalScheduleTestController(t, rng, action, references.Position{X: 10, Y: 10})
					npc.NPCReference.Location = test.location
					npc.NPCReference.SetKeyIndex(test.sprite)
					npc.MapUnitDetails().NPCNum = test.npcSlot
					if override {
						npc.NPCReference.Schedule.Ai[0] = byte(references.Fixed)
						npc.MapUnitDetails().SetOverriddenAiType(action)
					}
					controller.mapState.PlayerLocation.Position = references.Position{X: 11, Y: 10}

					controller.AdvanceNextTurnCalcAndMoveNPCs()

					assert.Equal(t, test.want, npc.MapUnitDetails().PendingInteraction)
					assert.Equal(t, references.Position{X: 10, Y: 10}, npc.Pos())
					assert.Empty(t, rng.calls)
				})
			}
		}
	}
}

func TestArrivedWandererDoesNotPathBackToAnchor(t *testing.T) {
	rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
	controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Wander3, references.Position{X: 10, Y: 10})
	npc.SetPos(references.Position{X: 11, Y: 10})

	controller.calculateNextNPCPosition(npc)

	assert.Equal(t, references.Position{X: 12, Y: 10}, npc.Pos())
	assert.Equal(t, []originalScheduleRNGCall{{0, 255}, {0, 64}}, rng.calls)
}

func newOriginalScheduleTestController(
	t *testing.T,
	rng RNGProvider,
	action references.AiType,
	position references.Position,
) (*NPCAIControllerSmallMap, *map_units.NPCFriendly, *map_state.LayeredMap) {
	t.Helper()
	tileRefs := references.NewTileReferences()
	emptyMap := &references.LargeMapReference{}
	layeredMaps := map_state.NewLayeredMaps(tileRefs, emptyMap, emptyMap, 11, 11)
	state := &map_state.MapState{ //nolint:exhaustruct // Focused source-action fixture.
		PlayerLocation: references.PlayerLocation{
			Location: references.BritanniaUnderworld,
			Position: references.Position{X: 30, Y: 30},
			Floor:    0,
		},
		LayeredMaps: *layeredMaps,
	}
	schedule := references.CreateNPCScheduledFixedOneLocation(position, 0)
	schedule.Ai[0] = byte(action)
	reference := references.NPCReference{Schedule: schedule}
	reference.SetKeyIndex(indexes.TownsPersonKeyIndex)
	npc := map_units.NewNPCFriendly(reference, 0)
	npc.PlaceAtScheduleSlot(0)
	controller := &NPCAIControllerSmallMap{ //nolint:exhaustruct // Focused source-action fixture.
		tileRefs:    tileRefs,
		dateTime:    &datetime.UltimaDate{},
		mapState:    state,
		mapUnits:    map_units.MapUnits{npc},
		rngProvider: rng,
	}
	layeredMap := state.GetLayeredMapByCurrentLocation()
	for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			position := references.Position{X: x, Y: y}
			layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
		}
	}

	return controller, npc, layeredMap
}
