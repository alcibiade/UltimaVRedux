package ai

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/helpers"
)

func TestActivatedAuthoredStonegateShadowlordsBypassTerrainNotOccupancy(t *testing.T) {
	gameRefs := helpers.LoadTestGameReferences(t, helpers.LoadTestConfiguration(t))
	refs := gameRefs.NPCReferences.GetNPCReferencesByLocation(references.Stonegate)
	for slot := 1; slot <= 3; slot++ {
		for _, terrain := range []indexes.SpriteIndex{indexes.StoneBrickWall, indexes.LockedDoor} {
			for _, occupied := range []bool{false, true} {
				t.Run(fmt.Sprintf("slot%d/terrain%d/occupied%t", slot, terrain, occupied), func(t *testing.T) {
					rng := &scriptedOriginalScheduleRNG{t: t, values: []int{63}}
					start := references.Position{X: 10, Y: 10}
					controller, civilian, layeredMap := newOriginalScheduleTestController(t, rng, references.Fixed, start)
					npc := map_units.NewNPCFriendly((*refs)[slot], slot)
					monster, capable := npc.NPCReference.SmallMapCombatMonsterType()
					require.True(t, capable)
					require.Equal(t, references.MonsterShadowlord, monster)
					npc.SetPos(start)
					npc.SetFloor(0)
					npc.SetVisible(true)
					controller.mapUnits = map_units.MapUnits{npc}
					controller.mapState.PlayerLocation.Position = references.Position{X: 14, Y: 10}
					destination := start.Right()
					layeredMap.SetTileByLayer(map_state.MapLayer, &destination, terrain)
					require.False(t, controller.npcTerrainAt(destination).IsNPCTerrainPassable())
					assert.False(t, controller.canOriginalNPCMoveTo(civilian, destination, npcMovementApproach), "ordinary actors still obey terrain")
					if occupied {
						blockerRef := references.NPCReference{Schedule: references.CreateNPCScheduledFixedOneLocation(destination, 0)}
						blockerRef.SetKeyIndex(indexes.TownsPersonKeyIndex)
						blocker := map_units.NewNPCFriendly(blockerRef, 4)
						blocker.PlaceAtScheduleSlot(0)
						controller.mapUnits = append(controller.mapUnits, blocker)
					}

					controller.SetAttackAvatar()
					controller.AdvanceNextTurnCalcAndMoveNPCs()

					want := destination
					if occupied {
						want = start
					}
					assert.Equal(t, want, npc.Pos())
					assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
					assert.Equal(t, []originalScheduleRNGCall{{0, 63}}, rng.calls)
					assert.False(t, controller.canOriginalNPCMoveTo(npc, references.Position{X: -1, Y: 10}, npcMovementApproach))
					assert.False(t, controller.canOriginalNPCMoveTo(npc, controller.mapState.PlayerLocation.Position, npcMovementApproach))
				})
			}
		}
	}
}
