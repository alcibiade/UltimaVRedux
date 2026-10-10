package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestOriginalCombatMissileSamplingByType(t *testing.T) {
	combatMap := references.NewCombatMapReference(0, references.BritanniaCombatTerritory)
	raw := &[references.XCombatMapTiles][references.YCombatMapTiles]byte{}
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			raw[x][y] = byte(indexes.BrickFloor)
		}
	}
	wall := references.Position{X: 5, Y: 5}
	raw[wall.X][wall.Y] = byte(indexes.StoneBrickWall)
	combatMap.SetRawData(raw)
	from := references.Position{X: 2, Y: 5}
	aim := references.Position{X: 6, Y: 5}

	tests := []struct {
		name        string
		missile     references.MissileType
		wantImpact  references.Position
		wantReached bool
	}{
		{name: "Arrow starts four pixels into the path", missile: references.MissileArrow, wantImpact: wall},
		{name: "Rock advances eight pixels", missile: references.MissileRock, wantImpact: wall},
		{name: "Default advances thirteen pixels", missile: references.MissileBlue, wantImpact: aim, wantReached: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			launch := traceCombatProjectileLaunch(combatMap.GetTileNumber, from, aim, test.missile)

			assert.Equal(t, test.wantImpact, launch.Impact)
			assert.Equal(t, test.wantReached, !launch.Blocked)
		})
	}
}

func TestEveryCombatCrosshairSpellRejectsExactCasterCoordinateWithoutCommitment(t *testing.T) {
	spells := []references.Spell{
		references.GravPor, references.VasFlam, references.XenCorp,
		references.InFlamGrav, references.InNoxGrav, references.InZuGrav, references.InSanctGrav,
		references.AnXenEx, references.RelXenBet, references.InQuasXen,
	}
	for _, spell := range spells {
		t.Run(references.GetSpellName(spell), func(t *testing.T) {
			state, _ := newSyntheticDungeonTriggerCombatState(t)
			caster, character := prepareSyntheticCombatSpell(t, state, spell)
			mpBefore := character.GetCurrentMp()
			callsBefore := state.GetRNG().Calls()

			result := state.ConfirmCombatSpellTarget()

			assert.Equal(t, CombatSpellTargetInvalid, result.Outcome)
			assert.NotNil(t, state.CombatScene.SpellTargeting)
			assert.Nil(t, state.CombatScene.pendingSpell)
			assert.Equal(t, mpBefore, character.GetCurrentMp())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(spell))
			assert.Equal(t, callsBefore, state.GetRNG().Calls())
			assert.Equal(t, caster, state.CombatScene.GetCurrentTurnIdentity())
		})
	}
}

func TestCombatDamageProjectileAdmitsEmptyEndpointsWithoutTriggerDisclosure(t *testing.T) {
	target := references.Position{X: 6, Y: 5}
	tests := []struct {
		name        string
		withTrigger bool
	}{
		{name: "ordinary empty tile"},
		{name: "empty dungeon trigger", withTrigger: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var state *GameState
			if test.withTrigger {
				state, _ = newSingleTriggerCombatState(t, target)
			} else {
				state, _ = newSyntheticDungeonTriggerCombatState(t)
			}
			_, character := prepareSyntheticCombatSpell(t, state, references.GravPor)
			state.CombatScene.SpellTargeting.Cursor = target

			result := state.ConfirmCombatSpellTarget()

			require.Equal(t, CombatSpellTargetResolved, result.Outcome)
			require.NotNil(t, state.CombatScene.pendingSpell)
			assert.False(t, state.CombatScene.pendingSpell.targetSet)
			assert.Equal(t, byte(29), character.GetCurrentMp())
			assert.Zero(t, state.PartyState.Inventory.Spells.Get(references.GravPor))
			require.True(t, state.ResolvePendingCombatSpell())
			if test.withTrigger {
				assertDungeonTriggerConsumed(t, state, 0)
			}
		})
	}
}

func TestCombatDamageProjectileTargetsPossessedCloneButRejectsFriendlyClone(t *testing.T) {
	state, callbacks := newSyntheticDungeonTriggerCombatState(t)
	caster, character := prepareSyntheticCombatSpell(t, state, references.VasFlam)
	clonePosition := references.Position{X: 5, Y: 5}
	clone := state.CombatScene.addPCCloneFromSource(
		caster, caster.EntityIndex, clonePosition, NewCombatPCProvider(character),
	)
	state.CombatScene.SpellTargeting.Cursor = clonePosition
	mpBefore := character.GetCurrentMp()

	assert.Equal(t, CombatSpellTargetInvalid, state.ConfirmCombatSpellTarget().Outcome)
	assert.Equal(t, mpBefore, character.GetCurrentMp())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.VasFlam))

	state.CombatScene.setCombatPCPossessed(clone, true)
	result := state.ConfirmCombatSpellTarget()
	require.Equal(t, CombatSpellTargetResolved, result.Outcome)
	require.True(t, state.CombatScene.pendingSpell.targetSet)
	assert.Equal(t, clone, state.CombatScene.pendingSpell.target)
	callbacks.Reset()
	require.True(t, state.ResolvePendingCombatSpell())

	require.Len(t, callbacks.CombatEvents, 1)
	event := callbacks.CombatEvents[0]
	assert.True(t, event.Target.IsPartyMember)
	assert.Equal(t, clone.EntityIndex, event.Target.Index)
	assert.Equal(t, clone.Slot, event.Target.Slot)
}

func TestXenCorpCommitRollsSaveBeforeScatterAndTriggersOnlyReachedEndpoint(t *testing.T) {
	aimed := references.Position{X: 5, Y: 5}
	scattered := references.Position{X: 6, Y: 5}
	state, callbacks := newSyntheticDungeonTriggerCombatState(t,
		indexedDungeonTrigger{index: 0, trigger: references.DungeonCombatMapTrigger{
			ReplacementTile: indexes.BrickFloor,
			Trigger:         aimed,
			Destinations:    [2]references.Position{{X: 0xff, Y: 0xff}, {X: 0xff, Y: 0xff}},
		}},
		indexedDungeonTrigger{index: 1, trigger: references.DungeonCombatMapTrigger{
			ReplacementTile: indexes.BrickFloor,
			Trigger:         scattered,
			Destinations:    [2]references.Position{{X: 0xff, Y: 0xff}, {X: 0xff, Y: 0xff}},
		}},
	)
	_, caster := prepareSyntheticCombatSpell(t, state, references.XenCorp)
	caster.Intelligence = 1
	target, targetIndex := addSyntheticTriggerMonster(
		state, aimed, CombatMonsterLoyaltyMonster, references.MissileNone,
	)
	collateral, collateralIndex := addSyntheticTriggerMonster(
		state, scattered, CombatMonsterLoyaltyMonster, references.MissileNone,
	)
	state.CombatScene.SpellTargeting.Cursor = aimed
	targetHP := target.CurrentHP()
	collateralHP := collateral.CurrentHP()
	rng := newSequenceRNG(0, 2, 1)

	result := state.confirmCombatSpellTargetWithRNG(rng)

	require.Equal(t, CombatSpellTargetResolved, result.Outcome)
	assert.Equal(t, 3, rng.index, "save draw must precede the two scatter-coordinate draws")
	require.NotNil(t, state.CombatScene.pendingSpell)
	assert.Equal(t, CombatHitMiss, state.CombatScene.pendingSpell.hit.Outcome)
	assert.Equal(t, CombatTurnIdentity{EntityIndex: targetIndex}, state.CombatScene.pendingSpell.target)
	assert.Equal(t, scattered, result.Launch.Aim)
	callbacks.Reset()
	require.True(t, state.ResolvePendingCombatSpell())

	assertDungeonTriggerUnconsumed(t, state, 0, aimed)
	assertDungeonTriggerConsumed(t, state, 1)
	assert.Equal(t, targetHP, target.CurrentHP())
	assert.Equal(t, collateralHP, collateral.CurrentHP(), "a failed spell cannot damage scatter collateral")
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, CombatFeedbackMiss, callbacks.CombatEvents[0].Result)
	assert.Equal(t, targetIndex, callbacks.CombatEvents[0].Target.Index)
	assert.NotEqual(t, collateralIndex, callbacks.CombatEvents[0].Target.Index)
}

func TestXenCorpBlockedLaunchStillRollsCommittedSave(t *testing.T) {
	state, callbacks := newSyntheticDungeonTriggerCombatState(t)
	_, caster := prepareSyntheticCombatSpell(t, state, references.XenCorp)
	caster.Intelligence = 255
	targetPosition := references.Position{X: 6, Y: 5}
	target, _ := addVulnerableProjectileTarget(state, targetPosition)
	state.CombatScene.SpellTargeting.Cursor = targetPosition
	setBlockedCombatMissileMap(state, references.Position{X: 4, Y: 5})
	hpBefore := target.CurrentHP()
	rng := newSequenceRNG(0)

	result := state.confirmCombatSpellTargetWithRNG(rng)

	require.Equal(t, CombatSpellTargetResolved, result.Outcome)
	assert.Equal(t, 1, rng.index)
	assert.True(t, result.Launch.Blocked)
	assert.True(t, state.CombatScene.pendingSpell.hit.Rolled)
	callbacks.Reset()
	require.True(t, state.ResolvePendingCombatSpell())
	assert.Equal(t, hpBefore, target.CurrentHP())
	require.Len(t, callbacks.CombatEvents, 1)
	assert.True(t, callbacks.CombatEvents[0].Diagnostics.ProjectileBlocked)
	assert.True(t, callbacks.CombatEvents[0].Diagnostics.Hit.Rolled)
}

func TestXenCorpRetainsMonsterZeroAsExplicitTarget(t *testing.T) {
	state, callbacks := newSyntheticDungeonTriggerCombatState(t)
	_, caster := prepareSyntheticCombatSpell(t, state, references.XenCorp)
	caster.Intelligence = 255
	targetPosition := references.Position{X: 5, Y: 5}
	target, targetIndex := addVulnerableProjectileTarget(state, targetPosition)
	require.Zero(t, targetIndex)
	state.CombatScene.SpellTargeting.Cursor = targetPosition
	rng := newSequenceRNG(0)

	result := state.confirmCombatSpellTargetWithRNG(rng)

	require.Equal(t, CombatSpellTargetResolved, result.Outcome)
	require.NotNil(t, state.CombatScene.pendingSpell)
	assert.True(t, state.CombatScene.pendingSpell.targetSet)
	assert.Equal(t, CombatTurnIdentity{EntityIndex: 0}, state.CombatScene.pendingSpell.target)
	callbacks.Reset()
	require.True(t, state.ResolvePendingCombatSpell())
	assert.True(t, target.IsDead())
	require.NotEmpty(t, callbacks.CombatEvents)
	assert.Equal(t, 0, callbacks.CombatEvents[0].Target.Index)
}

func TestXenCorpDamagesExactPossessedCloneWithoutExperienceAward(t *testing.T) {
	state, callbacks := newSyntheticDungeonTriggerCombatState(t)
	caster, character := prepareSyntheticCombatSpell(t, state, references.XenCorp)
	casterPosition := state.CombatScene.GetCombatantPosition(caster)
	clonePosition := references.Position{X: casterPosition.X + 3, Y: casterPosition.Y}
	clone := state.CombatScene.addPCCloneFromSource(
		caster, caster.EntityIndex, clonePosition, NewCombatPCProvider(character),
	)
	state.CombatScene.setCombatPCPossessed(clone, true)
	character.Intelligence = 255
	experienceBefore := character.Exp
	state.CombatScene.SpellTargeting.Cursor = clonePosition
	rng := newSequenceRNG(60)

	result := state.confirmCombatSpellTargetWithRNG(rng)
	require.Equal(t, CombatSpellTargetResolved, result.Outcome)
	callbacks.Reset()
	require.True(t, state.ResolvePendingCombatSpell())

	assert.True(t, character.IsDead())
	assert.Equal(t, experienceBefore, character.Exp)
	require.Len(t, callbacks.CombatEvents, 1)
	assert.True(t, callbacks.CombatEvents[0].Target.IsPartyMember)
	assert.Equal(t, clone.Slot, callbacks.CombatEvents[0].Target.Slot)
	assert.True(t, callbacks.CombatEvents[0].Fatal)
}

func prepareSyntheticCombatSpell(
	t *testing.T,
	state *GameState,
	spell references.Spell,
) (CombatTurnIdentity, *party_state.PlayerCharacter) {
	t.Helper()

	caster := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	state.CombatScene.CurrentTurnIdentity = caster
	state.CombatScene.currentTurnIdentitySet = true
	state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 2, Y: 5})
	character := state.PartyState.GetCharacter(0)
	character.Class = party_state.Wizard
	character.Level = 8
	character.Intelligence = 30
	character.SetMp(30)
	character.SetHp(100)
	character.SetMaxHp(100)
	character.SetStatus(party_state.Good)
	state.PartyState.Inventory.Spells.Set(spell, 1)
	setOpenCombatMissileMap(state)
	require.True(t, state.BeginCombatSpellTargeting(CastRequest{Spell: spell, CasterIdx: 0}).Valid)

	return caster, character
}

func addVulnerableProjectileTarget(
	state *GameState,
	position references.Position,
) (*CombatMonster, int) {
	reference := (*state.GameReferences.EnemyReferences)[references.MonsterOrc]
	reference.EnemyAbilities = map[references.EnemyAbility]bool{}
	reference.HitPoints = 100
	monster := NewCombatMonster(references.MonsterOrc, &reference, position, newFixedRNG(1))
	monster.SetLoyalty(CombatMonsterLoyaltyMonster)
	monster.SetCurrentHP(100)

	return monster, state.CombatScene.AddMonsterToQueue(monster)
}
