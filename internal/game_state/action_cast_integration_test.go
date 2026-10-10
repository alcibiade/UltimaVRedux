package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestValidateSpellResourcesDistinguishesMagicAbilityFromDepletedMP(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	caster := gs.PartyState.GetCharacter(0)
	gs.PartyState.Inventory.Spells.Set(references.InLor, 1)
	request := CastRequest{Spell: references.InLor, CasterIdx: 0}

	caster.Class = party_state.Fighter
	caster.SetMp(0)
	result := gs.ValidateSpellResources(request)
	require.False(t, result.Valid)
	assert.Equal(t, SpellNoMagicAbilityMessage, result.ErrorMsg)

	caster.Class = party_state.Wizard
	result = gs.ValidateSpellResources(request)
	require.False(t, result.Valid)
	assert.Equal(t, SpellNoMPMessage, result.ErrorMsg)
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(references.InLor), "failed validation must not consume a spell")
}

func TestUnlimitedCastingResourcesBypassesResourceChecksWithoutGrantingMagicAbility(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	caster := gs.PartyState.GetCharacter(0)
	caster.Class = party_state.Wizard
	caster.SetMp(0)
	caster.Level = 8
	gs.PartyState.Inventory.Spells.Set(references.InLor, 0)
	gs.DebugOptions.UnlimitedCastingResources = true
	request := CastRequest{Spell: references.InLor, CasterIdx: 0}

	result := gs.ValidateAndConsumeSpellResources(request)

	require.True(t, result.Valid)
	assert.Zero(t, caster.GetCurrentMp(), "unlimited casting must not consume or add MP")
	assert.Zero(t, gs.PartyState.Inventory.Spells.Get(references.InLor), "unlimited casting must not consume or add prepared spells")

	caster.Class = party_state.Fighter
	result = gs.ValidateSpellResources(request)
	require.False(t, result.Valid)
	assert.Equal(t, SpellNoMagicAbilityMessage, result.ErrorMsg)
}

func TestBypassSpellLevelRestrictionsIsIndependentFromUnlimitedResources(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	caster := gs.PartyState.GetCharacter(0)
	caster.Class = party_state.Wizard
	caster.Level = 1
	caster.SetMp(8)
	gs.PartyState.Inventory.Spells.Set(references.AnTym, 1)
	request := CastRequest{Spell: references.AnTym, CasterIdx: 0}

	result := gs.ValidateSpellResources(request)
	require.False(t, result.Valid)
	assert.Equal(t, "Level too low!", result.ErrorMsg)

	gs.DebugOptions.BypassSpellLevelRestrictions = true
	result = gs.ValidateSpellResources(request)
	require.True(t, result.Valid)

	gs.PartyState.Inventory.Spells.Set(references.AnTym, 0)
	result = gs.ValidateSpellResources(request)
	require.False(t, result.Valid)
	assert.Equal(t, "None mixed!", result.ErrorMsg, "level bypass must not bypass resource requirements")
}

func TestIsImplementedSpellInContextAllowsDungeonTimeSpells(t *testing.T) {
	assert.True(t, IsImplementedSpellInContext(references.RelTym, references.ContextDungeon))
	assert.True(t, IsImplementedSpellInContext(references.AnTym, references.ContextDungeon))
}

func TestPreparedCastReportsLocationAbsorptionWithoutResourceCommitment(t *testing.T) {
	tests := []struct {
		name       string
		location   references.Location
		ownCrown   bool
		wearCrown  bool
		wantAbsorb bool
	}{
		{name: "Stonegate", location: references.Stonegate, wantAbsorb: true},
		{name: "Palace without Crown", location: references.PalaceOfBlackthorn, wantAbsorb: true},
		{name: "Palace with unworn Crown", location: references.PalaceOfBlackthorn, ownCrown: true},
		{name: "Palace with worn Crown", location: references.PalaceOfBlackthorn, ownCrown: true, wearCrown: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).
				WithLocation(test.location).
				WithSystemCallbacks().
				Build()
			caster := gs.PartyState.GetCharacter(0)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			gs.PartyState.Inventory.Spells.Set(references.InLor, 1)
			if test.ownCrown {
				gs.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
			}
			if test.wearCrown {
				gs.ToggleCrown()
			}

			result := gs.ValidateAndConsumeSpellResources(CastRequest{Spell: references.InLor, CasterIdx: 0})

			assert.Equal(t, test.wantAbsorb, result.IsAbsorb)
			assert.Equal(t, !test.wantAbsorb, result.Valid)
			if test.wantAbsorb {
				assert.Equal(t, "Absorbed!", result.ErrorMsg)
				assert.Equal(t, byte(8), caster.GetCurrentMp())
				assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(references.InLor))
			}
		})
	}
}

func TestCombatSpellAbsorptionUsesPalaceCrownOwnershipButNotStonegateOrigin(t *testing.T) {
	tests := []struct {
		name       string
		origin     references.Location
		ownCrown   bool
		wantAbsorb bool
	}{
		{name: "Stonegate origin", origin: references.Stonegate},
		{name: "Palace without Crown", origin: references.PalaceOfBlackthorn, wantAbsorb: true},
		{name: "Palace with unworn Crown", origin: references.PalaceOfBlackthorn, ownCrown: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _ := newSyntheticDungeonTriggerCombatState(t)
			gs.CombatScene.PreCombatPlayerLocation.Location = test.origin
			casterIdx := gs.CombatScene.GetActiveCombatantIndex()
			caster := gs.PartyState.GetCharacter(casterIdx)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			gs.PartyState.Inventory.Spells.Set(references.InAn, 1)
			if test.ownCrown {
				gs.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
			}

			result := gs.ValidateAndConsumeSpellResources(CastRequest{Spell: references.InAn, CasterIdx: casterIdx})

			assert.Equal(t, test.wantAbsorb, result.IsAbsorb)
			assert.Equal(t, !test.wantAbsorb, result.Valid)
			if test.wantAbsorb {
				assert.Equal(t, byte(8), caster.GetCurrentMp())
				assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(references.InAn))
			} else {
				assert.Equal(t, byte(2), caster.GetCurrentMp())
				assert.Zero(t, gs.PartyState.Inventory.Spells.Get(references.InAn))
			}
		})
	}
}

func TestActiveNegateMagicAbsorbsEveryPreparedCombatCast(t *testing.T) {
	for _, spell := range []references.Spell{references.InAn, references.InXenMani} {
		t.Run(references.GetSpellName(spell), func(t *testing.T) {
			gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
			if gs == nil {
				return
			}
			casterIdx := gs.CombatScene.GetActiveCombatantIndex()
			caster := gs.PartyState.GetCharacter(casterIdx)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			gs.PartyState.Inventory.Spells.Set(spell, 1)
			require.True(t, gs.ActivateNegateMagicEffect(10))

			result := gs.ValidateAndConsumeSpellResources(CastRequest{Spell: spell, CasterIdx: casterIdx})

			require.False(t, result.Valid)
			assert.Equal(t, "Absorbed!", result.ErrorMsg)
			assert.True(t, result.IsAbsorb)
			assert.Equal(t, byte(8), caster.GetCurrentMp())
			assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(spell))
		})
	}
}

func TestAnTymShadowlordAbsorptionPrecedesAtomicResourceCommitment(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T) *GameState
	}{
		{
			name: "town NPC",
			setup: func(t *testing.T) *GameState {
				state, _ := NewIntegrationTestBuilder(t).
					WithLocation(references.Britain).
					WithSystemCallbacks().
					Build()
				addPhysicalShadowlord(t, state, references.Faulinei, references.Position{X: 1, Y: 1})

				return state
			},
		},
		{
			name: "combat monster",
			setup: func(t *testing.T) *GameState {
				state, _, _ := setupCombatWithMonster(t, references.MonsterShadowlord, references.Down, 42)

				return state
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := test.setup(t)
			require.NotNil(t, state)
			casterIdx := 0
			if state.CombatScene != nil {
				casterIdx = state.CombatScene.GetActiveCombatantIndex()
			}
			caster := state.PartyState.GetCharacter(casterIdx)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			state.PartyState.Inventory.Spells.Set(references.AnTym, 1)

			result := state.ValidateAndConsumeSpellResources(CastRequest{Spell: references.AnTym, CasterIdx: casterIdx})

			require.False(t, result.Valid)
			assert.True(t, result.IsAbsorb)
			assert.Equal(t, "Absorbed!", result.ErrorMsg)
			assert.Equal(t, byte(8), caster.GetCurrentMp())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.AnTym))
		})
	}
}

func TestAnTymRemainsAvailableInDoomWithoutPresentShadowlord(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Doom).
		WithFloor(1).
		WithPlayerAt(1, 1).
		WithSystemCallbacks().
		Build()
	caster := state.PartyState.GetCharacter(0)
	caster.Class = party_state.Wizard
	caster.Level = 8
	caster.SetMp(8)
	state.PartyState.Inventory.Spells.Set(references.AnTym, 1)

	result := state.ValidateAndConsumeSpellResources(CastRequest{Spell: references.AnTym, CasterIdx: 0})

	require.True(t, result.Valid, result.ErrorMsg)
	assert.Zero(t, caster.GetCurrentMp())
	assert.Zero(t, state.PartyState.Inventory.Spells.Get(references.AnTym))
}

func TestVasManiCombatVictoryGatePrecedesTargetAndResources(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	caster := gs.PartyState.GetCharacter(casterIdx)
	caster.Class = party_state.Wizard
	caster.Level = 8
	caster.SetMp(8)
	caster.SubtractHp(10)
	gs.PartyState.Inventory.Spells.Set(references.VasMani, 1)
	request := CastRequest{Spell: references.VasMani, CasterIdx: casterIdx, TargetIdx: casterIdx}

	prerequisite := gs.ValidateSpellTargetingPrerequisites(request)
	committed := gs.ValidateAndConsumeSpellResources(request)

	require.False(t, prerequisite.Valid)
	assert.Equal(t, "Not until victory!", prerequisite.ErrorMsg)
	require.False(t, committed.Valid)
	assert.Equal(t, byte(8), caster.GetCurrentMp())
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(references.VasMani))

	gs.CombatScene.Victory = true
	require.True(t, gs.ValidateSpellTargetingPrerequisites(request).Valid)
	require.True(t, gs.ValidateAndConsumeSpellResources(request).Valid)
	result := gs.CastVasMani(casterIdx)
	assert.True(t, result.Applied)
	assert.True(t, caster.IsFullyHealed())
}

func TestVasManiNonCombatRemainsAvailable(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	caster := gs.PartyState.GetCharacter(0)
	caster.Class = party_state.Wizard
	caster.Level = 8
	caster.SetMp(8)
	caster.SetHp(caster.GetMaxHp())
	caster.SubtractHp(10)
	gs.PartyState.Inventory.Spells.Set(references.VasMani, 1)
	request := CastRequest{Spell: references.VasMani, CasterIdx: 0, TargetIdx: 0}

	require.True(t, gs.ValidateSpellTargetingPrerequisites(request).Valid)
	require.True(t, gs.ValidateAndConsumeSpellResources(request).Valid)
	result := gs.CastVasMani(0)

	assert.True(t, result.Applied)
	assert.True(t, caster.IsFullyHealed())
}

func TestPartySpellIneffectiveTargetsCommitResourcesBeforeSelection(t *testing.T) {
	tests := []struct {
		spell references.Spell
		setup func(*party_state.PlayerCharacter)
	}{
		{spell: references.Mani, setup: func(character *party_state.PlayerCharacter) { character.SetHp(character.GetMaxHp()) }},
		{spell: references.VasMani, setup: func(character *party_state.PlayerCharacter) { character.SetHp(character.GetMaxHp()) }},
		{spell: references.AnNox, setup: func(character *party_state.PlayerCharacter) { character.SetStatus(party_state.Good) }},
	}
	for _, test := range tests {
		t.Run(references.GetSpellName(test.spell), func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
			caster := gs.PartyState.GetCharacter(0)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			test.setup(caster)
			gs.PartyState.Inventory.Spells.Set(test.spell, 1)

			result := gs.ValidateAndConsumePartyTargetSpellResources(CastRequest{Spell: test.spell, CasterIdx: 0})

			require.True(t, result.Valid)
			spellInfo := references.GetSpellInfo(test.spell)
			assert.Equal(t, byte(8-spellInfo.Circle), caster.GetCurrentMp())
			assert.Zero(t, gs.PartyState.Inventory.Spells.Get(test.spell))
		})
	}
}

func TestFullHealthHealingSpellsStillApply(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	target := gs.PartyState.GetCharacter(0)
	target.SetHp(target.GetMaxHp())
	gs.SetRandomSeed(7)

	mani := gs.CastMani(0)
	vasMani := gs.CastVasMani(0)

	assert.True(t, mani.Applied)
	assert.Positive(t, mani.Amount)
	assert.True(t, vasMani.Applied)
	assert.Zero(t, vasMani.Amount)
	assert.True(t, target.IsFullyHealed())
	assert.Empty(t, callbacks.TransientIconCalls)
}

func TestCastManiAndAnNoxOwnDeterministicPartyMutation(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	target := gs.PartyState.GetCharacter(0)
	target.SetHp(target.GetMaxHp() - 30)
	gs.SetRandomSeed(7)

	heal := gs.CastMani(0)

	assert.True(t, heal.Applied)
	assert.Equal(t, heal.Amount, int(target.GetCurrentHp()-(target.GetMaxHp()-30)))
	target.SetStatus(party_state.Poisoned)
	assert.True(t, gs.CastAnNox(0).Applied)
	assert.True(t, target.IsGood())
	assert.Equal(t, []TransientIconKind{TransientIconHealing, TransientIconPoisonCure}, callbacks.TransientIconCalls)
}

func TestIsImplementedSpellInContextAllowsRemainingDungeonPreparedSpells(t *testing.T) {
	for _, spell := range []references.Spell{references.RelTym, references.AnTym, references.VasRelPor} {
		assert.True(t, IsImplementedSpellInContext(spell, references.ContextDungeon))
	}
}

func TestIsImplementedSpellInContextAllowsDungeonFloorSpells(t *testing.T) {
	assert.True(t, IsImplementedSpellInContext(references.UusPor, references.ContextDungeon))
	assert.True(t, IsImplementedSpellInContext(references.DesPor, references.ContextDungeon))
}

func TestIsImplementedSpellInContextAllowsSurfaceVasRelPor(t *testing.T) {
	assert.True(t, IsImplementedSpellInContext(references.VasRelPor, references.ContextOverworld))
	assert.True(t, IsImplementedSpellInContext(references.VasRelPor, references.ContextTown))
	assert.True(t, IsImplementedSpellInContext(references.VasRelPor, references.ContextDungeon))
}

func TestIsImplementedSpellInContextAllowsWisAnYlemOutsideDungeons(t *testing.T) {
	assert.True(t, IsImplementedSpellInContext(references.WisAnYlem, references.ContextOverworld))
	assert.True(t, IsImplementedSpellInContext(references.WisAnYlem, references.ContextTown))
	assert.False(t, IsImplementedSpellInContext(references.WisAnYlem, references.ContextDungeon))
}

func TestIsImplementedSpellInContextAllowsAnSanctOutsideCombat(t *testing.T) {
	assert.True(t, IsImplementedSpellInContext(references.AnYlem, references.ContextTown))
	assert.False(t, IsImplementedSpellInContext(references.AnYlem, references.ContextOverworld))
	assert.True(t, IsImplementedSpellInContext(references.AnSanct, references.ContextOverworld))
	assert.True(t, IsImplementedSpellInContext(references.AnSanct, references.ContextTown))
	assert.True(t, IsImplementedSpellInContext(references.AnSanct, references.ContextDungeon))
}

func TestIsImplementedSpellInContextAllowsInQuasWisOutsideCombat(t *testing.T) {
	assert.True(t, IsImplementedSpellInContext(references.InQuasWis, references.ContextOverworld))
	assert.True(t, IsImplementedSpellInContext(references.InQuasWis, references.ContextTown))
	assert.True(t, IsImplementedSpellInContext(references.InQuasWis, references.ContextDungeon))
	assert.False(t, IsImplementedSpellInContext(references.InQuasWis, references.ContextCombat))
}

func TestImplementedSpellContextsCatalogCoversAllSpells(t *testing.T) {
	expected := [references.TotalSpellCount]references.SpellContext{
		references.InLor:         references.ContextNonCombat,
		references.GravPor:       references.ContextCombat,
		references.AnZu:          references.ContextAll,
		references.AnNox:         references.ContextAll,
		references.Mani:          references.ContextAll,
		references.AnYlem:        references.ContextTown | references.ContextCombat,
		references.AnSanct:       references.ContextAll,
		references.AnXenCorp:     references.ContextCombat,
		references.RelHur:        references.ContextOverworld,
		references.InWis:         references.ContextOverworld,
		references.KalXen:        references.ContextCombat,
		references.InXenMani:     references.ContextAll,
		references.VasLor:        references.ContextNonCombat,
		references.VasFlam:       references.ContextCombat,
		references.InFlamGrav:    references.ContextDungeon | references.ContextCombat,
		references.InNoxGrav:     references.ContextDungeon | references.ContextCombat,
		references.InZuGrav:      references.ContextDungeon | references.ContextCombat,
		references.InPor:         references.ContextOverworld | references.ContextCombat,
		references.AnGrav:        references.ContextDungeon | references.ContextCombat,
		references.InSanct:       references.ContextAll,
		references.InSanctGrav:   references.ContextDungeon | references.ContextCombat,
		references.UusPor:        references.ContextDungeon,
		references.DesPor:        references.ContextDungeon,
		references.WisQuas:       references.ContextCombat,
		references.InBetXen:      references.ContextCombat,
		references.AnExPor:       references.ContextTown | references.ContextCombat,
		references.InExPor:       references.ContextTown | references.ContextCombat,
		references.VasMani:       references.ContextAll,
		references.InZu:          references.ContextCombat,
		references.RelTym:        references.ContextAll,
		references.InVasPorYlem:  references.ContextCombat,
		references.QuasAnWis:     references.ContextCombat,
		references.InAn:          references.ContextAll,
		references.WisAnYlem:     references.ContextOverworld | references.ContextTown,
		references.AnXenEx:       references.ContextCombat,
		references.RelXenBet:     references.ContextCombat,
		references.SanctLor:      references.ContextCombat,
		references.XenCorp:       references.ContextCombat,
		references.InQuasXen:     references.ContextCombat,
		references.InQuasWis:     references.ContextNonCombat,
		references.InNoxHur:      references.ContextCombat,
		references.InQuasCorp:    references.ContextCombat,
		references.InManiCorp:    references.ContextNonCombat,
		references.KalXenCorp:    references.ContextCombat,
		references.InVasGravCorp: references.ContextCombat,
		references.InFlamHur:     references.ContextCombat,
		references.VasRelPor:     references.ContextNonCombat,
		references.AnTym:         references.ContextAll,
	}

	for spell := references.Spell(0); spell < references.TotalSpellCount; spell++ {
		implemented := GetImplementedSpellContexts(spell)
		assert.Equal(t, expected[spell], implemented, references.GetSpellName(spell))
		assert.Zero(t, implemented&^references.GetSpellInfo(spell).Context,
			"%s cannot be implemented in an originally illegal context", references.GetSpellName(spell))
	}
	assert.Zero(t, GetImplementedSpellContexts(references.Spell(-1)))
	assert.Zero(t, GetImplementedSpellContexts(references.Spell(references.TotalSpellCount)))
}
