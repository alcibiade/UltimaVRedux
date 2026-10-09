package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbilitiesFromAdmodesUsesOriginalHighByteMasks(t *testing.T) {
	daemon := abilitiesFromAdmodes(0x8461)

	assert.True(t, daemon[RangedMagic])
	assert.True(t, daemon[GatesInDaemon])
	assert.True(t, daemon[PossessCharm])
	assert.True(t, daemon[Undead])
	assert.True(t, daemon[NoCorpse])
	assert.False(t, daemon[Poison], "0x0400 is GATES_IN, not poison")
	assert.False(t, daemon[Teleport], "daemon admodes does not include TELEPORTS")
}

func TestAbilitiesFromAdmodesKeepsDistinctOriginalHighByteAbilities(t *testing.T) {
	abilities := abilitiesFromAdmodes(0x3f00)

	assert.True(t, abilities[Teleport])
	assert.True(t, abilities[DisappearsOnDeath])
	assert.True(t, abilities[Invisibility])
	assert.True(t, abilities[GatesInDaemon])
	assert.True(t, abilities[Poison])
	assert.True(t, abilities[InfectWithPlague])
}
