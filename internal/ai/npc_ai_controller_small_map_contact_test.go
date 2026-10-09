package ai

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSmallMapContactClassifiesAllShadowlordAndGuardFrames(t *testing.T) {
	for _, key := range []indexes.SpriteIndex{indexes.ShadowLord_KeyIndex, indexes.Guard_KeyIndex} {
		for frame := range indexes.StandardNumberOfAnimationFrames {
			t.Run(fmt.Sprintf("shape%d/frame%d", key, frame), func(t *testing.T) {
				rng := &scriptedOriginalScheduleRNG{t: t}
				controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Fixed, references.Position{X: 10, Y: 10})
				npc.NPCReference.SetKeyIndex(key + indexes.SpriteIndex(frame))
				controller.mapState.PlayerLocation.Position = references.Position{X: 11, Y: 10}

				controller.SetAttackAvatar()
				controller.AdvanceNextTurnCalcAndMoveNPCs()

				want := map_units.PendingNPCInteractionHostileCombat
				if key == indexes.Guard_KeyIndex {
					want = map_units.PendingNPCInteractionGuardArrest
				}
				assert.Equal(t, want, npc.MapUnitDetails().PendingInteraction)
				assert.Empty(t, rng.calls)

				npc.MapUnitDetails().SetAIOverride(references.DrudgeWorthThing, references.AIOverrideProvenanceAutomaticAstaroth)
				controller.AdvanceNextTurnCalcAndMoveNPCs()
				assert.Equal(t, map_units.PendingNPCInteractionTalk, npc.MapUnitDetails().PendingInteraction,
					"Astaroth marker precedes shape-based arrest or combat")
				assert.Empty(t, rng.calls)
			})
		}
	}
}

func TestSmallMapAttackActionsRequireEligibleContact(t *testing.T) {
	tests := []struct {
		name    string
		avatar  references.Position
		floor   references.FloorNumber
		visible bool
	}{
		{name: "diagonal", avatar: references.Position{X: 11, Y: 11}, visible: true},
		{name: "other floor", avatar: references.Position{X: 11, Y: 10}, floor: 1, visible: true},
		{name: "invisible", avatar: references.Position{X: 11, Y: 10}},
	}
	for _, test := range tests {
		for _, action := range []references.AiType{references.ExtortOrAttackOrFollow, references.DrudgeWorthThing} {
			for _, override := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/action%d/override%t", test.name, action, override), func(t *testing.T) {
					rng := &scriptedOriginalScheduleRNG{t: t, values: []int{63}}
					controller, npc, _ := newOriginalScheduleTestController(t, rng, action, references.Position{X: 10, Y: 10})
					if override {
						npc.MapUnitDetails().SetOverriddenAiType(action)
					}
					npc.SetVisible(test.visible)
					npc.SetFloor(test.floor)
					npc.NPCReference.Schedule.Floor[0] = byte(test.floor)
					controller.mapState.PlayerLocation.Position = test.avatar
					npc.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionHostileCombat

					controller.AdvanceNextTurnCalcAndMoveNPCs()

					assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
					if !test.visible || npc.Floor() != controller.mapState.PlayerLocation.Floor {
						assert.Equal(t, references.Position{X: 10, Y: 10}, npc.Pos())
						assert.Empty(t, rng.calls)
					}
				})
			}
		}
	}
}

func TestSmallMapSetAttackAvatarUsesUnlimitedPursuit(t *testing.T) {
	for _, sprite := range []indexes.SpriteIndex{indexes.Guard_KeyIndex, indexes.Daemon_KeyIndex} {
		t.Run(fmt.Sprint(sprite), func(t *testing.T) {
			rng := &scriptedOriginalScheduleRNG{t: t, values: []int{63}}
			controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Fixed, references.Position{X: 10, Y: 10})
			npc.NPCReference.SetKeyIndex(sprite)
			npc.MapUnitDetails().SetCurrentPath([]references.Position{{X: 9, Y: 10}})
			npc.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionTalk
			controller.mapState.PlayerLocation.Position = references.Position{X: 14, Y: 10}

			controller.SetAttackAvatar()

			assert.Equal(t, references.AiType(references.DrudgeWorthThing), npc.MapUnitDetails().GetOverriddenAiType())
			assert.Equal(t, references.AIOverrideProvenanceGuardHostility, npc.MapUnitDetails().AIOverrideProvenance())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
			controller.AdvanceNextTurnCalcAndMoveNPCs()
			assert.Equal(t, references.Position{X: 11, Y: 10}, npc.Pos())
			assert.Equal(t, []originalScheduleRNGCall{{0, 63}}, rng.calls)
		})
	}
}

func TestSmallMapAttackOverridesUseOriginalRangeAndVariation(t *testing.T) {
	tests := []struct {
		name   string
		action references.AiType
		avatar references.Position
		rolls  []int
		want   references.Position
	}{
		{name: "action6 pursues within three", action: references.ExtortOrAttackOrFollow, avatar: references.Position{X: 13, Y: 10}, want: references.Position{X: 11, Y: 10}},
		{name: "action6 stops at four", action: references.ExtortOrAttackOrFollow, avatar: references.Position{X: 12, Y: 12}, want: references.Position{X: 10, Y: 10}},
		{name: "action7 pursues beyond eight", action: references.DrudgeWorthThing, avatar: references.Position{X: 20, Y: 10}, rolls: []int{63}, want: references.Position{X: 11, Y: 10}},
		{name: "action7 can deviate away", action: references.DrudgeWorthThing, avatar: references.Position{X: 20, Y: 10}, rolls: []int{15, 16, 16}, want: references.Position{X: 10, Y: 9}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedOriginalScheduleRNG{t: t, values: test.rolls}
			controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Fixed, references.Position{X: 10, Y: 10})
			npc.MapUnitDetails().SetOverriddenAiType(test.action)
			controller.mapState.PlayerLocation.Position = test.avatar

			controller.AdvanceNextTurnCalcAndMoveNPCs()

			assert.Equal(t, test.want, npc.Pos())
			assert.Empty(t, rng.values)
			assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
		})
	}
}

func TestSmallMapAttackContactWaitsForFollowingSourcePhase(t *testing.T) {
	for _, action := range []references.AiType{references.ExtortOrAttackOrFollow, references.DrudgeWorthThing} {
		for _, override := range []bool{false, true} {
			t.Run(fmt.Sprintf("action%d/override%t", action, override), func(t *testing.T) {
				rng := &scriptedOriginalScheduleRNG{t: t, values: []int{63}}
				controller, npc, _ := newOriginalScheduleTestController(t, rng, action, references.Position{X: 10, Y: 10})
				if override {
					npc.MapUnitDetails().SetOverriddenAiType(action)
				}
				controller.mapState.PlayerLocation.Position = references.Position{X: 12, Y: 10}

				controller.AdvanceNextTurnCalcAndMoveNPCs()
				assert.Equal(t, references.Position{X: 11, Y: 10}, npc.Pos())
				assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
				callsAfterMovement := len(rng.calls)
				controller.AdvanceNextTurnCalcAndMoveNPCs()
				assert.Equal(t, references.Position{X: 11, Y: 10}, npc.Pos())
				assert.Equal(t, map_units.PendingNPCInteractionHostileCombat, npc.MapUnitDetails().PendingInteraction)
				assert.Len(t, rng.calls, callsAfterMovement)
			})
		}
	}
}

func TestSmallMapCombatShapeDoesNotSupplyHostileIntent(t *testing.T) {
	for _, sprite := range []indexes.SpriteIndex{indexes.Troll_KeyIndex, indexes.Daemon_KeyIndex} {
		for _, action := range []references.AiType{references.Fixed, references.Talk3} {
			t.Run(fmt.Sprintf("shape%d/action%d", sprite, action), func(t *testing.T) {
				rng := &scriptedOriginalScheduleRNG{t: t}
				controller, npc, _ := newOriginalScheduleTestController(t, rng, action, references.Position{X: 10, Y: 10})
				npc.NPCReference.SetKeyIndex(sprite)
				npc.NPCReference.DialogNumber = 1
				controller.mapState.PlayerLocation.Position = references.Position{X: 11, Y: 10}
				_, capable := npc.NPCReference.SmallMapCombatMonsterType()
				require.True(t, capable)

				controller.AdvanceNextTurnCalcAndMoveNPCs()

				want := map_units.PendingNPCInteractionNone
				if action == references.Talk3 {
					want = map_units.PendingNPCInteractionTalk
				}
				assert.Equal(t, want, npc.MapUnitDetails().PendingInteraction)

				npc.NPCReference.Schedule.Ai[0] = byte(references.DrudgeWorthThing)
				npc.MapUnitDetails().SetOverriddenAiType(references.Fixed)
				controller.AdvanceNextTurnCalcAndMoveNPCs()
				assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
			})
		}
	}
}

func TestSmallMapAttackContactLastAuthoredSlotWins(t *testing.T) {
	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, guard, _ := newOriginalScheduleTestController(t, rng, references.ExtortOrAttackOrFollow, references.Position{X: 10, Y: 10})
	controller.mapState.PlayerLocation.Position = references.Position{X: 11, Y: 10}
	guard.NPCReference.SetKeyIndex(indexes.Guard_KeyIndex)
	guard.MapUnitDetails().NPCNum = 5
	attacker := map_units.NewNPCFriendly(guard.NPCReference, 2)
	attacker.NPCReference.SetKeyIndex(indexes.StoneGargoyle_KeyIndex)
	attacker.PlaceAtScheduleSlot(0)
	attacker.SetPos(references.Position{X: 12, Y: 10})
	attacker.MapUnitDetails().SetOverriddenAiType(references.DrudgeWorthThing)
	controller.mapUnits = map_units.MapUnits{guard, attacker}

	controller.AdvanceNextTurnCalcAndMoveNPCs()
	assert.Equal(t, map_units.PendingNPCInteractionGuardArrest, guard.MapUnitDetails().PendingInteraction)
	assert.Equal(t, map_units.PendingNPCInteractionNone, attacker.MapUnitDetails().PendingInteraction)

	attacker.MapUnitDetails().NPCNum = 6
	controller.AdvanceNextTurnCalcAndMoveNPCs()
	assert.Equal(t, map_units.PendingNPCInteractionNone, guard.MapUnitDetails().PendingInteraction)
	assert.Equal(t, map_units.PendingNPCInteractionHostileCombat, attacker.MapUnitDetails().PendingInteraction)
}
