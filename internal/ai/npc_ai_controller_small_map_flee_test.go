package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSetFleeFromAvatarAppliesOriginalEligibility(t *testing.T) {
	controller := &NPCAIControllerSmallMap{}
	tests := []struct {
		name         string
		npc          *map_units.NPCFriendly
		wantAccepted bool
	}{
		{name: "nil NPC", npc: nil, wantAccepted: false},
		{
			name:         "shape below NPC range",
			npc:          smallMapFleeTestNPC(indexes.LordBritishMirror1, 1, [4]byte{1}),
			wantAccepted: false,
		},
		{
			name:         "shape at wanderer boundary",
			npc:          smallMapFleeTestNPC(indexes.Apparation1, 1, [4]byte{1}),
			wantAccepted: false,
		},
		{
			name:         "static regular talk state",
			npc:          smallMapFleeTestNPC(indexes.TownsPerson1, 1, [4]byte{}),
			wantAccepted: false,
		},
		{
			name:         "fourth schedule time is movable",
			npc:          smallMapFleeTestNPC(indexes.TownsPerson1, 1, [4]byte{0, 0, 0, 1}),
			wantAccepted: true,
		},
		{
			name:         "original special talk state permits static NPC",
			npc:          smallMapFleeTestNPC(indexes.TownsPerson1, 0xfe, [4]byte{}),
			wantAccepted: true,
		},
		{
			name:         "movable merchant shape is eligible",
			npc:          smallMapFleeTestNPC(indexes.Merchant1, byte(references.DialogWeaponDealer), [4]byte{1}),
			wantAccepted: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			accepted := controller.SetFleeFromAvatar(test.npc)

			assert.Equal(t, test.wantAccepted, accepted)
			if test.npc == nil {
				return
			}
			if test.wantAccepted {
				assert.Equal(t, references.AiType(references.FleeFromAvatar), test.npc.MapUnitDetails().GetOverriddenAiType())
			} else {
				assert.Equal(t, references.Unset, test.npc.MapUnitDetails().GetOverriddenAiType())
			}
		})
	}
}

func smallMapFleeTestNPC(
	sprite indexes.SpriteIndex,
	dialogNumber byte,
	times [4]byte,
) *map_units.NPCFriendly {
	reference := references.NPCReference{
		DialogNumber: dialogNumber,
		Schedule: references.NPCSchedule{
			X:    [3]byte{1, 1, 1},
			Y:    [3]byte{1, 1, 1},
			Time: times,
		},
		NpcType: references.NoStatedNpc,
	}
	reference.SetKeyIndex(sprite)

	return map_units.NewNPCFriendly(reference, 1)
}
