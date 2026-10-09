package references

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestNPCReferenceSmallMapCombatMonsterTypeCoversEveryEncodedShape(t *testing.T) {
	t.Parallel()
	for sprite := 256; sprite < 512; sprite++ {
		t.Run(fmt.Sprint(sprite), func(t *testing.T) {
			t.Parallel()
			reference := NPCReference{}
			reference.SetKeyIndex(indexes.SpriteIndex(sprite))
			got, ok := reference.SmallMapCombatMonsterType()
			wantOK := sprite >= 320 && (sprite < 436 || sprite >= 440) && (sprite < 488 || sprite >= 496)
			want := MonsterNone
			if wantOK {
				want = MonsterType((sprite - 320) / 4)
			}
			if got != want || ok != wantOK {
				t.Fatalf("SmallMapCombatMonsterType() = (%v, %v), want (%v, %v)", got, ok, want, wantOK)
			}
		})
	}
	var missing *NPCReference
	if monster, ok := missing.SmallMapCombatMonsterType(); monster != MonsterNone || ok {
		t.Fatal("nil reference must not be a combat target")
	}
}

// TestNPCReference_IsBeggar verifies that IsBeggar correctly identifies beggar NPCs.
// Beggar NPCs have sprite indices 364-367 (Begger1-Begger4).
// The Beggar family occupies raw tiles 0x6c..0x6f; ref:ENEMY-0001.
func TestNPCReference_IsBeggar(t *testing.T) {
	tests := []struct {
		name      string
		spriteIdx indexes.SpriteIndex
		expected  bool
	}{
		// Beggar sprites should return true
		{"Begger1 NPC is beggar", indexes.Begger1, true},
		{"Begger2 NPC is beggar", indexes.Begger2, true},
		{"Begger3 NPC is beggar", indexes.Begger3, true},
		{"Begger4 NPC is beggar", indexes.Begger4, true},
		{"Beggar_KeyIndex NPC is beggar", indexes.Beggar_KeyIndex, true},

		// Other NPC types should return false
		{"Guard NPC is not beggar", indexes.Guard_KeyIndex, false},
		{"Avatar NPC is not beggar", indexes.AvatarKeyIndex, false},
		{"TownsPerson NPC is not beggar", indexes.TownsPerson_KeyIndex, false},
		{"Wizard NPC is not beggar", indexes.WizardKeyIndex, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an NPCReference with the test sprite index
			npcRef := &NPCReference{}
			npcRef.SetKeyIndex(tt.spriteIdx)

			result := npcRef.IsBeggar()
			if result != tt.expected {
				t.Errorf("NPCReference with sprite %d: IsBeggar() = %v, want %v",
					tt.spriteIdx, result, tt.expected)
			}
		})
	}
}

// TestNPCReference_IsBeggar_VsGetSpriteIndex verifies IsBeggar consistency with GetSpriteIndex.
// This ensures the sprite index round-trip through SetKeyIndex/GetSpriteIndex works correctly.
func TestNPCReference_IsBeggar_VsGetSpriteIndex(t *testing.T) {
	// Test that setting a beggar sprite and getting it back works correctly
	npcRef := &NPCReference{}
	npcRef.SetKeyIndex(indexes.Begger1)

	// GetSpriteIndex should return a sprite that IsBeggar() recognizes
	spriteIdx := npcRef.GetSpriteIndex()
	if !spriteIdx.IsBeggar() {
		t.Errorf("GetSpriteIndex() returned %d which is not detected as beggar by SpriteIndex.IsBeggar()",
			spriteIdx)
	}

	// NPCReference.IsBeggar() should match SpriteIndex.IsBeggar()
	if npcRef.IsBeggar() != spriteIdx.IsBeggar() {
		t.Errorf("NPCReference.IsBeggar() = %v but SpriteIndex.IsBeggar() = %v - inconsistent",
			npcRef.IsBeggar(), spriteIdx.IsBeggar())
	}
}

func TestNPCReferenceDaemonShapeTakesPrecedenceOverGuardDialog(t *testing.T) {
	reference := NPCReference{Location: Windemere, DialogNumber: byte(Guard)}
	reference.SetKeyIndex(indexes.Daemon_KeyIndex)
	if reference.GetNPCType() != NoStatedNpc {
		t.Fatal("daemon shape must take precedence over a Guard-like dialog value")
	}
}
