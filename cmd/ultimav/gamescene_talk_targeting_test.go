package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestFindNPCInDirectionUsesOriginalTalkRange(t *testing.T) {
	directions := []references.Direction{
		references.Up,
		references.Down,
		references.Left,
		references.Right,
	}
	for _, direction := range directions {
		t.Run(direction.GetDirectionCompassName()+" adjacent", func(t *testing.T) {
			fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
			origin := references.Position{X: 16, Y: 16}
			target := talkTestStep(t, origin, direction)
			prepareTalkTargetFixture(t, fixture, origin, target)

			position, unit := fixture.state.FindNPCInDirection(origin, direction, 2)

			require.Equal(t, &target, position)
			require.Same(t, fixture.npc, unit)
		})
		t.Run(direction.GetDirectionCompassName()+" across table", func(t *testing.T) {
			fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
			origin := references.Position{X: 16, Y: 16}
			intermediate := talkTestStep(t, origin, direction)
			target := talkTestStep(t, intermediate, direction)
			prepareTalkTargetFixture(t, fixture, origin, target)
			fixture.state.GetLayeredMapByCurrentLocation().SetTileByLayer(
				map_state.MapOverrideLayer, &intermediate, indexes.TableMiddle)

			position, unit := fixture.state.FindNPCInDirection(origin, direction, 2)

			require.Equal(t, &target, position)
			require.Same(t, fixture.npc, unit)
		})
		t.Run(direction.GetDirectionCompassName()+" across floor", func(t *testing.T) {
			fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
			origin := references.Position{X: 16, Y: 16}
			intermediate := talkTestStep(t, origin, direction)
			target := talkTestStep(t, intermediate, direction)
			prepareTalkTargetFixture(t, fixture, origin, target)
			fixture.state.GetLayeredMapByCurrentLocation().SetTileByLayer(
				map_state.MapOverrideLayer, &intermediate, indexes.BrickFloor)

			position, unit := fixture.state.FindNPCInDirection(origin, direction, 2)

			require.Nil(t, position)
			require.Nil(t, unit)
		})
	}
}

func TestFindNPCInDirectionFiltersInvalidTargets(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*merchantTalkFixture)
	}{
		{
			name: "different floor",
			prepare: func(fixture *merchantTalkFixture) {
				fixture.npc.SetFloor(fixture.state.MapState.PlayerLocation.Floor + 1)
			},
		},
		{
			name: "invisible",
			prepare: func(fixture *merchantTalkFixture) {
				fixture.npc.SetVisible(false)
			},
		},
		{
			name: "hidden quest item",
			prepare: func(fixture *merchantTalkFixture) {
				fixture.npc.NPCReference.SetKeyIndex(indexes.Crown)
			},
		},
		{
			name: "empty unit",
			prepare: func(fixture *merchantTalkFixture) {
				fixture.npc.NPCReference.Schedule.X[0] = 0
				fixture.npc.NPCReference.Schedule.Y[0] = 0
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
			origin := references.Position{X: 16, Y: 16}
			target := talkTestStep(t, origin, references.Right)
			prepareTalkTargetFixture(t, fixture, origin, target)
			test.prepare(fixture)

			position, unit := fixture.state.FindNPCInDirection(origin, references.Right, 2)

			require.Nil(t, position)
			require.Nil(t, unit)
		})
	}
}

func TestFindNPCInDirectionAdjacentNormalNPCWinsOverFartherMerchant(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
	origin := references.Position{X: 16, Y: 16}
	adjacent := talkTestStep(t, origin, references.Right)
	farther := talkTestStep(t, adjacent, references.Right)
	prepareTalkTargetFixture(t, fixture, origin, farther)
	normalReference := fixture.npc.NPCReference
	normalReference.DialogNumber = 1
	normalNPC := map_units.NewNPCFriendly(normalReference, 30)
	normalNPC.SetPos(adjacent)
	normalNPC.SetFloor(references.MainFloor)
	normalNPC.SetVisible(true)
	fixture.state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{normalNPC, fixture.npc})

	position, unit := fixture.state.FindNPCInDirection(origin, references.Right, 2)

	require.Equal(t, &adjacent, position)
	require.Same(t, normalNPC, unit)
}

func TestProcessNPCAutoTalkClearsInvalidPendingState(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*merchantTalkFixture)
	}{
		{
			name: "not adjacent",
			prepare: func(fixture *merchantTalkFixture) {
				fixture.state.MapState.PlayerLocation.Position = references.Position{X: 16, Y: 16}
				fixture.npc.SetPos(references.Position{X: 18, Y: 16})
			},
		},
		{
			name: "different floor",
			prepare: func(fixture *merchantTalkFixture) {
				fixture.npc.SetFloor(fixture.state.MapState.PlayerLocation.Floor + 1)
			},
		},
		{
			name: "invisible",
			prepare: func(fixture *merchantTalkFixture) {
				fixture.npc.SetVisible(false)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
			fixture.state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{fixture.npc})
			fixture.npc.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionTalk
			test.prepare(fixture)

			require.False(t, fixture.scene.processNPCAutoTalk())
			require.Equal(t, map_units.PendingNPCInteractionNone, fixture.npc.MapUnitDetails().PendingInteraction)
			require.False(t, fixture.state.EventQueueManager.HasActiveWorkflows())
		})
	}
}

func TestBlockedMovementPreservesOriginalAutoTalkLifecycle(t *testing.T) {
	tests := []struct {
		name         string
		ai           references.AiType
		wantAutoTalk bool
	}{
		{name: "fixed NPC only blocks", ai: references.Fixed, wantAutoTalk: false},
		{name: "TALK3 merchant initiates after turn", ai: references.Talk3, wantAutoTalk: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
			if !test.wantAutoTalk {
				fixture.npc.NPCReference.DialogNumber = 1
			}
			fixture.state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{fixture.npc})
			for slot := range fixture.npc.NPCReference.Schedule.Ai {
				fixture.npc.NPCReference.Schedule.Ai[slot] = byte(test.ai)
			}
			fixture.direction = placePlayerNextToMerchant(fixture.state, fixture.npc)
			fixture.callbacks.Reset()
			fixture.scene.output = text.NewOutput(nil, 1, 10, 80)
			fixture.scene.autoOpenDoorState = &AutoOpenDoorState{}
			fixture.scene.moongateVisualSystem = NewMoongateVisualSystem(fixture.state)
			origin := fixture.state.MapState.PlayerLocation.Position
			fixture.scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool {
				return key == ebiten.KeyRight
			})

			result := fixture.scene.handleMovement("East", ebiten.KeyRight)
			fixture.scene.completeSurfaceMovement(result)

			require.Equal(t, surfaceCommandRejected, result.State)
			require.Equal(t, origin, fixture.state.MapState.PlayerLocation.Position)
			require.Equal(t, []int{game_state.DefaultSmallMapMinutesPerTurn}, fixture.callbacks.TimeAdvanced())
			require.True(t, fixture.scene.keyboard.IsWaitingForRelease())
			wantInteraction := map_units.PendingNPCInteractionNone
			if test.wantAutoTalk {
				wantInteraction = map_units.PendingNPCInteractionTalk
			}
			require.Equal(t, wantInteraction, fixture.npc.MapUnitDetails().PendingInteraction)
			require.Equal(t, test.wantAutoTalk, fixture.scene.processNPCAutoTalk())
			if !test.wantAutoTalk {
				require.False(t, fixture.state.EventQueueManager.HasActiveWorkflows())
			}
		})
	}
}

func prepareTalkTargetFixture(
	t *testing.T,
	fixture *merchantTalkFixture,
	origin references.Position,
	target references.Position,
) {
	t.Helper()
	fixture.state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{fixture.npc})
	fixture.state.MapState.PlayerLocation.Position = origin
	fixture.state.MapState.PlayerLocation.Floor = references.MainFloor
	fixture.npc.SetPos(target)
	fixture.npc.SetFloor(references.MainFloor)
	fixture.npc.SetVisible(true)
	fixture.state.GetLayeredMapByCurrentLocation().SetTileByLayer(
		map_state.MapOverrideLayer, &target, indexes.BrickFloor)
}

func talkTestStep(t *testing.T, position references.Position, direction references.Direction) references.Position {
	t.Helper()
	next := direction.GetNewPositionInDirection(&position)
	require.NotNil(t, next)

	return *next
}
