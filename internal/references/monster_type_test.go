package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonsterType_Constants(t *testing.T) {
	// Verify all 48 monster types have correct indices
	testCases := []struct {
		name     string
		constant MonsterType
		expected int
	}{
		{"MonsterMage", MonsterMage, 0},
		{"MonsterBard", MonsterBard, 1},
		{"MonsterFighter", MonsterFighter, 2},
		{"MonsterAvatar", MonsterAvatar, 3},
		{"MonsterVillager", MonsterVillager, 4},
		{"MonsterGuard", MonsterGuard, 12},
		{"MonsterSeaHorse", MonsterSeaHorse, 16},
		{"MonsterSquid", MonsterSquid, 17},
		{"MonsterSeaSerpent", MonsterSeaSerpent, 18},
		{"MonsterShark", MonsterShark, 19},
		{"MonsterRat", MonsterRat, 20},
		{"MonsterBat", MonsterBat, 21},
		{"MonsterOrc", MonsterOrc, 32},
		{"MonsterSkeleton", MonsterSkeleton, 33},
		{"MonsterDragon", MonsterDragon, 39},
		{"MonsterTroll", MonsterTroll, 41},
		{"MonsterShadowlord", MonsterShadowlord, 47},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.constant.Index())
		})
	}
}

func TestMonsterType_IsValid(t *testing.T) {
	assert.True(t, MonsterOrc.IsValid())
	assert.True(t, MonsterDragon.IsValid())
	assert.True(t, MonsterShadowlord.IsValid())
	assert.False(t, MonsterNone.IsValid())
	assert.False(t, MonsterType(-1).IsValid())
	assert.False(t, MonsterType(48).IsValid())
}

func TestMonsterType_IsEnvironmentalHazard(t *testing.T) {
	assert.True(t, MonsterForceField.IsEnvironmentalHazard())
	assert.True(t, MonsterWhirlpool.IsEnvironmentalHazard())
	assert.False(t, MonsterOrc.IsEnvironmentalHazard())
	assert.False(t, MonsterDragon.IsEnvironmentalHazard())
}

func TestMonsterType_IsWaterMonster(t *testing.T) {
	assert.True(t, MonsterSeaHorse.IsWaterMonster())
	assert.True(t, MonsterSquid.IsWaterMonster())
	assert.True(t, MonsterSeaSerpent.IsWaterMonster())
	assert.True(t, MonsterShark.IsWaterMonster())
	assert.False(t, MonsterOrc.IsWaterMonster())
	assert.False(t, MonsterDragon.IsWaterMonster())
	assert.False(t, MonsterWhirlpool.IsWaterMonster())
}

func TestMonsterType_IsUndead(t *testing.T) {
	assert.True(t, MonsterGhost.IsUndead())
	assert.True(t, MonsterSkeleton.IsUndead())
	assert.True(t, MonsterDevil.IsUndead())
	assert.True(t, MonsterShadowlord.IsUndead())
	assert.False(t, MonsterOrc.IsUndead())
	assert.False(t, MonsterDragon.IsUndead())
}

func TestMonsterType_IsFixedGroup(t *testing.T) {
	// Guards appear in groups of 8
	assert.True(t, MonsterGuard.IsFixedGroup())
	assert.True(t, MonsterGuard.IsGuard())

	// Swarms appear in groups of 16
	assert.True(t, MonsterBat.IsSwarm())
	assert.True(t, MonsterSlime.IsSwarm())
	assert.True(t, MonsterMongbat.IsSwarm())

	// Solo monsters appear alone
	assert.True(t, MonsterAvatar.IsSolo())
	assert.True(t, MonsterDragon.IsSolo())
	assert.True(t, MonsterShadowlord.IsSolo())
	assert.True(t, MonsterMimic.IsSolo())

	// Regular monsters have random group sizes
	assert.False(t, MonsterOrc.IsFixedGroup())
	assert.False(t, MonsterTroll.IsFixedGroup())
}

func TestMonsterType_Name(t *testing.T) {
	assert.Equal(t, "Orc", MonsterOrc.Name())
	assert.Equal(t, "Dragon", MonsterDragon.Name())
	assert.Equal(t, "Daemon", MonsterDevil.Name())
	assert.Equal(t, "None", MonsterNone.Name())
}

func TestMonsterType_String(t *testing.T) {
	assert.Equal(t, "Orc", MonsterOrc.String())
	assert.Equal(t, "Dragon", MonsterDragon.String())
}

func TestTotalMonsterTypes(t *testing.T) {
	assert.Equal(t, 48, TotalMonsterTypes)
}

func TestMonsterTypeNames_Completeness(t *testing.T) {
	// Verify all valid monster types have names
	for i := 0; i < TotalMonsterTypes; i++ {
		mt := MonsterType(i)
		require.Contains(t, monsterTypeNames, mt, "Missing name for MonsterType %d", i)
		assert.NotEmpty(t, mt.Name(), "Empty name for MonsterType %d", i)
	}
}
