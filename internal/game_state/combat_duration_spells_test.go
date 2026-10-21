package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCastCombatRelTymActivatesOriginalQuicknessDuration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	assert.True(t, gs.CastCombatRelTym())
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectQuickness))
	assert.Equal(t, relTymDurationTurns, gs.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, "Quickness", gs.GetDurationEffectStatus().Name)
}

func TestCastCombatSanctLorHidesCasterFromMonsterTargeting(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	for pcIdx := range party_state.MaxActiveParty {
		gs.CombatScene.PartyActive[pcIdx] = pcIdx == casterIdx
	}

	require.True(t, gs.CastCombatSanctLor(casterIdx))
	assert.True(t, gs.CombatScene.IsPartyMemberInvisible(casterIdx))
	assert.Equal(t, NoTargetPC, findNearestPC(monster.Position(), gs.CombatScene).pcIndex)
}

func TestCombatDurationSpellsPassResourceValidation(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	caster := gs.PartyState.GetCharacter(casterIdx)
	caster.Level = 8
	caster.SetMp(30)

	for _, spell := range []references.Spell{references.RelTym, references.SanctLor} {
		gs.PartyState.Inventory.Spells.Set(spell, 1)
		result := gs.ValidateSpellResources(CastRequest{Spell: spell, CasterIdx: casterIdx})
		assert.True(t, result.Valid, references.GetSpellName(spell))
	}
}

func TestDurationSpellRecastRefreshesWithoutResettingQuicknessPhase(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	require.True(t, gs.ActivateQuicknessEffect(3))
	gs.quicknessWorldActorSkip = true

	require.True(t, gs.ActivateQuicknessEffect(relTymDurationTurns))

	assert.Equal(t, DurationEffectQuickness, gs.ActiveDurationEffect.EffectType)
	assert.Equal(t, relTymDurationTurns, gs.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, gs.quicknessWorldActorSkip)

	require.True(t, gs.ActivateProtectionEffect(2))
	require.True(t, gs.ActivateProtectionEffect(inSanctDurationTurns))
	assert.Equal(t, DurationEffectProtection, gs.ActiveDurationEffect.EffectType)
	assert.Equal(t, inSanctDurationTurns, gs.ActiveDurationEffect.TurnsRemaining)
}
