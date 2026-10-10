package environment

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/stretchr/testify/assert"
)

func TestConfiguredHazardDamageAboveUint16DoesNotWrapToZero(t *testing.T) {
	party := createTestPartyWithStats([]TestCharacterStats{{HitPoints: 20, Status: party_state.Good}})
	hazards := NewEnvironmentalHazards(&scriptedHazardRNG{}, &MockSystemCallbacks{})
	result := hazards.ApplyDamageToParty(party, DamageConfig{
		FixedDamage:   1 << 16,
		TargetIndices: []int{0},
	})
	assert.Equal(t, 0, int(party.GetCharacter(0).GetCurrentHp()))
	assert.True(t, party.GetCharacter(0).IsDead())
	assert.Equal(t, 1<<16, result.DamageDealt, "the original configured roll is still reported")
}
