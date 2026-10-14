package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestGazerDeathLeavesCorpseAndCreatesSameSquareInsect(t *testing.T) {
	gs, mock, gazer := setupCombatWithMonster(t, references.MonsterGazer, references.Down, 169)
	if gs == nil {
		return
	}
	position := gazer.Position()
	gazerID := gazer.CombatantID()
	gazer.enemyRef.TreasureNumber = combatTreasureDiceSides
	gazer.SetCurrentHP(1)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	experienceBefore := attacker.Exp
	randomCallsBefore := gs.GetRNG().Calls()

	gs.applyCombatDamageToMonster(
		gazer, 0, 1, false, attacker, attackerIndex, CombatDiagnostics{}, //nolint:exhaustruct
	)

	assert.True(t, gazer.IsDead())
	assert.Equal(t, randomCallsBefore+1, gs.GetRNG().Calls(),
		"Gazer death consumes only the Insect DEX roll, not treasure RNG")
	assert.Equal(t, gazerID, gs.CombatScene.MonsterManager.GetMonster(0).CombatantID())
	assert.Equal(t, experienceBefore+uint16(CombatCalcExperience(gazer.MaxHP())), attacker.Exp)
	require.Equal(t, 2, gs.CombatScene.MonsterManager.Count())
	insect := gs.CombatScene.MonsterManager.GetMonster(1)
	require.NotNil(t, insect)
	assert.Equal(t, references.MonsterInsect, insect.MonsterType())
	assert.Equal(t, position, insect.Position())
	assert.True(t, insect.IsHostile())
	assert.NotEqual(t, gazerID, insect.CombatantID())
	assert.Equal(t, insect.MaxHP(), insect.CurrentHP())
	insectEntry := gs.CombatScene.TurnQueue.GetEntry(1, false)
	require.NotNil(t, insectEntry)
	assert.Equal(t, turnQueueActive, insectEntry.status)
	corpse, found := gs.CombatScene.ObjectAt(position)
	require.True(t, found)
	assert.True(t, corpse.IsCorpse())
	assert.Equal(t, 1, gs.CombatScene.ObjectCount(), "special death must bypass ordinary treasure")
	assert.Equal(t, CombatNotEnded, gs.checkCombatEndConditions())
	require.Len(t, mock.CombatEvents, 1)
	assert.True(t, mock.CombatEvents[0].Fatal)
	assert.Equal(t, 0, mock.CombatEvents[0].Target.Index,
		"delayed fatal feedback must retain the dead Gazer's stable manager index")
	deadEntry := gs.CombatScene.TurnQueue.GetEntry(0, false)
	require.NotNil(t, deadEntry)
	assert.False(t, deadEntry.sourceSlotReleased)
	_, found = gs.CombatScene.takeObjectAt(position, CombatObject.IsCorpse)
	require.True(t, found)
	assert.False(t, deadEntry.sourceSlotReleased, "consuming a death object must retain its dead combatq slot")
}

func TestGazerDeathPastHistoricalCapacityCreatesInsect(t *testing.T) {
	gs, _, gazer := setupCombatWithMonster(t, references.MonsterGazer, references.Down, 169)
	for gs.CombatScene.MonsterManager.Count() < references.MaxMonstersInCombat {
		retired := NewCombatMonster(references.MonsterOrc,
			mustCombatEnemyReference(t, gs, references.MonsterOrc), references.Position{}, newFixedRNG(0))
		retired.SetActive(false)
		gs.CombatScene.MonsterManager.AddMonster(retired)
	}
	gazer.SetCurrentHP(1)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	randomCallsBefore := gs.GetRNG().Calls()

	// Original combat damage rules call combatant allocation; original combat initialization checks free slots.
	// See ref:COMBAT-0009.
	gs.applyCombatDamageToMonster(gazer, 0, 1, false,
		gs.PartyState.GetCharacter(attackerIndex), attackerIndex, CombatDiagnostics{})

	require.Equal(t, references.MaxMonstersInCombat+1, gs.CombatScene.MonsterManager.Count())
	insect := gs.CombatScene.MonsterManager.GetMonster(references.MaxMonstersInCombat)
	assert.Equal(t, references.MonsterInsect, insect.MonsterType())
	assert.Equal(t, gazer.Position(), insect.Position())
	assert.True(t, insect.IsHostile())
	assert.Equal(t, randomCallsBefore+1, gs.GetRNG().Calls(), "creation DEX only; no placement or treasure rolls")
	corpse, found := gs.CombatScene.ObjectAt(gazer.Position())
	require.True(t, found)
	assert.True(t, corpse.IsCorpse())
}

func TestGazerDeathAtCombatCapacityLeavesOnlyCorpse(t *testing.T) {
	gs, _, gazer := setupCombatWithMonster(t, references.MonsterGazer, references.Down, 169)
	if gs == nil {
		return
	}
	for gs.CombatScene.TurnQueue.hasFreeSourceSlot(sourceMonsterSlotStart) {
		monster := NewCombatMonster(
			references.MonsterOrc,
			mustCombatEnemyReference(t, gs, references.MonsterOrc),
			references.Position{},
			newFixedRNG(0),
		)
		gs.CombatScene.AddMonsterToQueue(monster)
	}
	gazer.SetCurrentHP(0)
	randomCallsBefore := gs.GetRNG().Calls()

	handled := gs.resolveSpecialCombatMonsterDeath(gazer, 0)

	assert.True(t, handled)
	assert.Equal(t, sourceCombatQueueSlots-sourceMonsterSlotStart, gs.CombatScene.MonsterManager.Count())
	assert.Equal(t, randomCallsBefore, gs.GetRNG().Calls(), "full queue rejects before creation DEX")
	corpse, found := gs.CombatScene.ObjectAt(gazer.Position())
	require.True(t, found)
	assert.True(t, corpse.IsCorpse())
}

func TestGargoyleDeathReplacesUnderlyingCombatTerrain(t *testing.T) {
	gs, _, gargoyle := setupCombatWithMonster(t, references.MonsterGargoyle, references.Down, 169)
	if gs == nil {
		return
	}
	position := gargoyle.Position()
	gargoyle.enemyRef.TreasureNumber = combatTreasureDiceSides
	gargoyle.SetCurrentHP(1)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	randomCallsBefore := gs.GetRNG().Calls()

	gs.applyCombatDamageToMonster(
		gargoyle, 0, 1, false, attacker, attackerIndex, CombatDiagnostics{}, //nolint:exhaustruct
	)

	assert.True(t, gargoyle.IsDead())
	assert.Equal(t, randomCallsBefore, gs.GetRNG().Calls(),
		"Gargoyle death must not consume treasure RNG")
	assert.Equal(t, indexes.SpriteIndex(indexes.SmallRockWall), gs.CombatScene.CombatMapRef.GetTileNumber(position))
	layeredTile := gs.GetCurrentLayeredMap().GetTileByLayer(map_state.MapLayer, &position)
	require.NotNil(t, layeredTile)
	assert.Equal(t, indexes.SpriteIndex(indexes.SmallRockWall), layeredTile.Index)
	assert.False(t, layeredTile.IsWalkingPassable())
	assert.Zero(t, gs.CombatScene.ObjectCount())
	entry := gs.CombatScene.TurnQueue.GetEntry(0, false)
	require.NotNil(t, entry)
	assert.Equal(t, turnQueueDead, entry.status)
}

func TestDisappearingDeathRemovesBeforeRecoveryAndCreditsExperienceAfterward(t *testing.T) {
	gs, mock, shadowlord, shadowlordIndex := setupHostileSpecialMonsterTest(t, references.MonsterShadowlord)
	if gs == nil {
		return
	}
	addHostileOrcForShadowlordRecoveryTest(gs)
	shadowlordRef := *shadowlord.EnemyRef()
	shadowlordRef.TreasureNumber = combatTreasureDiceSides
	shadowlord.enemyRef = &shadowlordRef
	shadowlord.SetCurrentHP(1)
	position := shadowlord.Position()
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	experienceBefore := attacker.Exp
	victim := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	require.True(t, gs.CombatScene.acquireCombatPossession(victim, shadowlord.CombatantID()))
	attacker.SetEquippedItem(references.SlotWeapon, references.ChaosSword)
	entry := gs.CombatScene.TurnQueue.GetEntry(shadowlordIndex, false)
	require.NotNil(t, entry)
	randomCallsBefore := gs.GetRNG().Calls()
	eventExperience := make([]uint16, 0, 3)
	releasedBeforeRecovery := false
	publishConclusion := gs.SystemCallbacks.Combat.PublishConclusion
	gs.SystemCallbacks.Combat = NewCombatCallbacks(func(event CombatEvent) {
		mock.CombatEvents = append(mock.CombatEvents, event)
		eventExperience = append(eventExperience, attacker.Exp)
		if event.Kind == CombatEventPossessionRelease {
			releasedBeforeRecovery = entry.sourceSlotReleased
		}
	}, publishConclusion)
	mock.Reset()

	gs.applyCombatDamageToMonster(
		shadowlord, shadowlordIndex, 1, false, attacker, attackerIndex, CombatDiagnostics{}, //nolint:exhaustruct
	)

	assert.True(t, shadowlord.IsDead())
	assert.Equal(t, randomCallsBefore, gs.GetRNG().Calls(), "disappearance must bypass both treasure rolls")
	assert.Zero(t, gs.CombatScene.ObjectCount())
	assert.True(t, entry.sourceSlotReleased)
	assert.True(t, releasedBeforeRecovery, "the disappearing controller is removed before attributed release")
	assert.False(t, gs.CombatScene.isCombatPCPossessed(victim))
	assert.False(t, gs.CombatScene.isCombatPCSleeping(victim))
	assert.Equal(t, references.ChaosSword, attacker.GetEquippedItem(references.SlotWeapon))
	assert.Equal(t, experienceBefore+uint16(CombatCalcExperience(shadowlord.MaxHP())), attacker.Exp)
	assert.Equal(t, []uint16{experienceBefore, experienceBefore, experienceBefore}, eventExperience,
		"XP credit must follow fatal feedback, disappearance, and recovery")
	require.Len(t, mock.CombatEvents, 3)
	assert.Equal(t, []CombatEventKind{CombatEventAttack, CombatEventVanished, CombatEventPossessionRelease},
		combatEventKinds(mock.CombatEvents))
	vanished := mock.CombatEvents[1].Vanished
	assert.Equal(t, shadowlord.CombatantID(), vanished.CombatantID)
	assert.Equal(t, shadowlordIndex, vanished.Index)
	assert.Equal(t, references.MonsterShadowlord, vanished.MonsterType)
	assert.Equal(t, position, vanished.Position)

	gs.applyCombatDamageToMonster(
		shadowlord, shadowlordIndex, 1, false, attacker, attackerIndex, CombatDiagnostics{}, //nolint:exhaustruct
	)
	assert.Equal(t, 1, countCombatEvents(mock.CombatEvents, CombatEventVanished))
	assert.Equal(t, experienceBefore+uint16(CombatCalcExperience(shadowlord.MaxHP())), attacker.Exp)
}

func TestDisappearingDeathUsesAbilityRatherThanMonsterType(t *testing.T) {
	gs, mock, daemon, daemonIndex := setupHostileDaemonTest(t)
	if gs == nil {
		return
	}
	daemonRef := *daemon.EnemyRef()
	daemonRef.EnemyAbilities = map[references.EnemyAbility]bool{
		references.DisappearsOnDeath: true,
	}
	daemonRef.TreasureNumber = combatTreasureDiceSides
	daemon.enemyRef = &daemonRef
	daemon.SetCurrentHP(0)
	randomCallsBefore := gs.GetRNG().Calls()
	mock.Reset()

	gs.finalizeCombatMonsterDeathWithPolicy(daemon, daemonIndex, nil, NoTargetPC, false, true)

	assert.Equal(t, randomCallsBefore, gs.GetRNG().Calls())
	assert.Zero(t, gs.CombatScene.ObjectCount())
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatEventVanished, mock.CombatEvents[0].Kind)
	assert.Equal(t, references.MonsterDevil, mock.CombatEvents[0].Vanished.MonsterType)
}

func TestOrdinaryNoCorpseDeathDoesNotUseDisappearancePath(t *testing.T) {
	gs, mock, daemon, daemonIndex := setupHostileDaemonTest(t)
	if gs == nil {
		return
	}
	require.True(t, daemon.HasAbility(references.NoCorpse))
	require.False(t, daemon.HasAbility(references.DisappearsOnDeath))
	daemon.SetCurrentHP(0)
	victim := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	gs.CombatScene.setCombatPCPossessed(victim, true)
	randomCallsBefore := gs.GetRNG().Calls()
	mock.Reset()

	gs.finalizeCombatMonsterDeathWithPolicy(daemon, daemonIndex, nil, NoTargetPC, false, true)

	assert.Equal(t, randomCallsBefore, gs.GetRNG().Calls())
	assert.Zero(t, gs.CombatScene.ObjectCount())
	assert.True(t, gs.CombatScene.isCombatPCPossessed(victim))
	assert.Empty(t, mock.CombatEvents)
}

func TestImmortalDisappearingMonsterDoesNotReachDeathHandling(t *testing.T) {
	gs, mock, blackthorn := setupCombatWithMonster(t, references.MonsterBlackthorn, references.Down, 42)
	if gs == nil {
		return
	}
	require.True(t, blackthorn.HasAbility(references.Immortal))
	require.True(t, blackthorn.HasAbility(references.DisappearsOnDeath))
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	hpBefore := blackthorn.CurrentHP()
	experienceBefore := attacker.Exp
	mock.Reset()

	gs.applyCombatDamageToMonster(
		blackthorn, 0, 0, true, attacker, attackerIndex,
		CombatDiagnostics{TargetImmortal: true}, //nolint:exhaustruct // Exercises forced-kill suppression only.
	)

	assert.Equal(t, hpBefore, blackthorn.CurrentHP())
	assert.Equal(t, experienceBefore, attacker.Exp)
	assert.Zero(t, countCombatEvents(mock.CombatEvents, CombatEventVanished))
}

func countCombatEvents(events []CombatEvent, kind CombatEventKind) int {
	count := 0
	for _, event := range events {
		if event.Kind == kind {
			count++
		}
	}

	return count
}

func TestDebugDeathPolicyDoesNotResolveSpecialDeaths(t *testing.T) {
	tests := []references.MonsterType{references.MonsterGazer, references.MonsterGargoyle}
	for _, monsterType := range tests {
		t.Run(monsterType.Name(), func(t *testing.T) {
			gs, _, monster := setupCombatWithMonster(t, monsterType, references.Down, 169)
			if gs == nil {
				return
			}
			position := monster.Position()
			originalTerrain := gs.CombatScene.CombatMapRef.GetTileNumber(position)
			monster.SetCurrentHP(0)

			gs.finalizeCombatMonsterDeathWithPolicy(monster, 0, nil, NoTargetPC, false, false)

			assert.Equal(t, 1, gs.CombatScene.MonsterManager.Count())
			assert.Zero(t, gs.CombatScene.ObjectCount())
			assert.Equal(t, originalTerrain, gs.CombatScene.CombatMapRef.GetTileNumber(position))
		})
	}
}
