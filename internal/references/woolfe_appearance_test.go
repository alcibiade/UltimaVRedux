package references

import (
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestWoolfeAuthoredAppearance(t *testing.T) {
	directory := originaldata.Directory(t, "TOWNE.NPC")
	npcs, err := getNPCsFromFile(filepath.Join(directory, "TOWNE.NPC"), 0)
	require.NoError(t, err)
	// NPC loading reads separate shape and dialogue arrays (ref:ENEMY-0003) from each
	// 576-byte town record. Trinsic slot 3 is Woolfe (TOWNE.TLK dialogue 31).
	woolfe := npcs[int(Trinsic-1)*npcsPerTown+3]
	require.Equal(t, byte(31), woolfe.DialogNumber)
	require.Equal(t, NPCType(0x48), woolfe.NpcType)
	require.Equal(t, indexes.FighterKeyIndex, woolfe.GetSpriteIndex())
	require.Equal(t, NoStatedNpc, woolfe.GetNPCType(), "dialogue 31 is not the synthetic Vehicle type")
	behaviour := woolfe.Schedule.GetIndividualNPCBehaviourByUltimaDate(datetime.UltimaDate{Hour: 14, Minute: 49})
	require.Equal(t, Position{X: 25, Y: 7}, behaviour.Position)
	require.Equal(t, FloorNumber(0), behaviour.Floor)
}

func TestDialogue31DoesNotOverrideAuthoredNPCShape(t *testing.T) {
	for _, sprite := range []indexes.SpriteIndex{indexes.FighterKeyIndex, indexes.MageKeyIndex, indexes.TownsPersonKeyIndex} {
		npc := NPCReference{DialogNumber: 31}
		npc.SetKeyIndex(sprite)
		require.Equal(t, NoStatedNpc, npc.GetNPCType())
		require.Equal(t, sprite, npc.GetSpriteIndex())
	}
	for _, sprite := range []indexes.SpriteIndex{indexes.HorseLeft, indexes.Carpet2, indexes.SkiffRight, indexes.ShipNoSailsRight} {
		npc := NPCReference{DialogNumber: 31}
		npc.SetKeyIndex(sprite)
		require.Equal(t, Vehicle, npc.GetNPCType(), "actual vehicles remain shape-driven")
	}
	npc := NewNPCReferenceForVehicle(HorseVehicle, Position{X: 4, Y: 5}, 0)
	require.Equal(t, Vehicle, npc.GetNPCType(), "synthetic vehicle marker remains supported")
}
