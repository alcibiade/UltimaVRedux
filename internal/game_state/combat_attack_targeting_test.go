package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func setupCombatAttackTargeting(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()

	gs, mock := newBritainCombatTestState(t, 42)
	clearCombatMonstersForTest(gs.CombatScene)
	mock.Reset()

	return gs, mock
}

func TestCollectCombatAttackSources_OriginalOrderAndRanges(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotWeapon, references.Bow)
	character.SetEquippedItem(references.SlotShield, references.SpikedShield)

	sources := collectCombatAttackSources(character)
	require.Len(t, sources, 3)
	assert.Equal(t, references.SlotHelm, sources[0].Slot)
	assert.Equal(t, references.SpikedHelm, sources[0].Equipment)
	assert.Equal(t, 1, sources[0].MaxRange)
	assert.Equal(t, references.SlotWeapon, sources[1].Slot)
	assert.Equal(t, references.Bow, sources[1].Equipment)
	assert.Equal(t, 7, sources[1].MaxRange)
	assert.Equal(t, references.SlotShield, sources[2].Slot)
	assert.Equal(t, references.SpikedShield, sources[2].Equipment)
	assert.Equal(t, 1, sources[2].MaxRange)
}

func TestCollectCombatAttackSources_BareHandsFallback(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	sources := collectCombatAttackSources(character)
	require.Len(t, sources, 1)
	assert.Equal(t, references.SlotNone, sources[0].Slot)
	assert.Equal(t, references.NoEquipment, sources[0].Equipment)
	assert.Equal(t, 1, sources[0].MaxRange)
}

func TestMoveCombatAttackCursor_EnforcesIntegerRangeAndBounds(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	origin := references.Position{X: 5, Y: 5}
	gs.CombatScene.SetPartyMemberPosition(attackerIndex, origin)
	character := gs.PartyState.GetCharacter(attackerIndex)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, references.MorningStar)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	require.True(t, gs.BeginCombatAttack())

	assert.True(t, gs.MoveCombatAttackCursor(1, 1), "diagonal distance floor(sqrt(2)) is 1")
	assert.Equal(t, references.Position{X: 6, Y: 6}, gs.CombatScene.AttackTargeting.Cursor)
	assert.True(t, gs.MoveCombatAttackCursor(1, 1), "offset (2,2) has integer distance 2")
	assert.Equal(t, references.Position{X: 7, Y: 7}, gs.CombatScene.AttackTargeting.Cursor)
	assert.False(t, gs.MoveCombatAttackCursor(1, 1), "offset (3,3) exceeds range 2")
	assert.Equal(t, references.Position{X: 7, Y: 7}, gs.CombatScene.AttackTargeting.Cursor)

	gs.CombatScene.AttackTargeting.Cursor = references.Position{X: 0, Y: 0}
	assert.False(t, gs.MoveCombatAttackCursor(-1, 0))
	assert.Equal(t, references.Position{X: 0, Y: 0}, gs.CombatScene.AttackTargeting.Cursor)
}

func TestBeginCombatAttack_ReusesVisibleTargetWithinCurrentRange(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	origin := gs.CombatScene.GetPartyMemberPosition(attackerIndex)
	targetPos := references.Position{X: origin.X + 1, Y: origin.Y}
	enemyRefs := *gs.GameReferences.EnemyReferences
	monster := NewCombatMonster(references.MonsterOrc, &enemyRefs[references.MonsterOrc], targetPos, newFixedRNG(1))
	monsterIndex := gs.CombatScene.AddMonsterToQueue(monster)
	gs.CombatScene.LastAttackTargets[attackerIndex] = monsterIndex

	character := gs.PartyState.GetCharacter(attackerIndex)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	require.True(t, gs.BeginCombatAttack())
	assert.Equal(t, targetPos, gs.CombatScene.AttackTargeting.Cursor)
}

func TestConfirmCombatAttackTarget_ProcessesEachSourceAndCompletes(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	origin := gs.CombatScene.GetPartyMemberPosition(attackerIndex)
	targetPos := references.Position{X: origin.X + 1, Y: origin.Y}
	enemyRefs := *gs.GameReferences.EnemyReferences
	monster := NewCombatMonster(references.MonsterOrc, &enemyRefs[references.MonsterOrc], targetPos, newFixedRNG(1))
	monster.SetCurrentHP(100)
	gs.CombatScene.AddMonsterToQueue(monster)

	character := gs.PartyState.GetCharacter(attackerIndex)
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	require.True(t, gs.BeginCombatAttack())
	require.True(t, gs.MoveCombatAttackCursor(1, 0))

	assert.Equal(t, CombatAttackSourceResolved, gs.ConfirmCombatAttackTarget().Outcome)
	require.NotNil(t, gs.CombatScene.AttackTargeting)
	assert.Equal(t, 1, gs.CombatScene.AttackTargeting.SourceIndex)
	assert.Equal(t, targetPos, gs.CombatScene.AttackTargeting.Cursor, "next source reuses previous target")

	assert.Equal(t, CombatAttackSequenceComplete, gs.ConfirmCombatAttackTarget().Outcome)
	assert.Nil(t, gs.CombatScene.AttackTargeting)
}

func TestCancelCombatAttackSource_ContinuesThenCompletes(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	require.True(t, gs.BeginCombatAttack())

	assert.Equal(t, CombatAttackSourceCancelled, gs.CancelCombatAttackSource().Outcome)
	assert.Equal(t, 1, gs.CombatScene.AttackTargeting.SourceIndex)
	assert.Equal(t, CombatAttackSequenceComplete, gs.CancelCombatAttackSource().Outcome)
	assert.Nil(t, gs.CombatScene.AttackTargeting)
}

func TestCancelInitialCombatAttack_ClearsTargetingWithoutAdvancingSource(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	require.True(t, gs.BeginCombatAttack())

	assert.Equal(t, CombatAttackSequenceCancelled, gs.CancelInitialCombatAttack().Outcome)
	assert.Nil(t, gs.CombatScene.AttackTargeting)
}

func TestCancelInitialCombatAttack_IgnoresLaterSource(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	require.True(t, gs.BeginCombatAttack())
	require.Equal(t, CombatAttackSourceCancelled, gs.CancelCombatAttackSource().Outcome)

	assert.Equal(t, CombatAttackTargetIgnored, gs.CancelInitialCombatAttack().Outcome)
	assert.NotNil(t, gs.CombatScene.AttackTargeting)
}

func TestConfirmCombatAttackTarget_EmptyMeleeSquareReportsNothing(t *testing.T) {
	gs, mock := setupCombatAttackTargeting(t)

	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	require.True(t, gs.BeginCombatAttack())
	require.True(t, gs.MoveCombatAttackCursor(1, 0))

	assert.Equal(t, CombatAttackSequenceComplete, gs.ConfirmCombatAttackTarget().Outcome)
	assert.Contains(t, mock.Messages, "Nothing!")
}

func TestConfirmCombatAttackTarget_RangedSourceDoesNotApplyRemoteDamage(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	origin := gs.CombatScene.GetPartyMemberPosition(attackerIndex)
	targetPos := references.Position{X: origin.X + 2, Y: origin.Y}
	enemyRefs := *gs.GameReferences.EnemyReferences
	monster := NewCombatMonster(references.MonsterOrc, &enemyRefs[references.MonsterOrc], targetPos, newFixedRNG(1))
	gs.CombatScene.AddMonsterToQueue(monster)
	hpBefore := monster.CurrentHP()

	character := gs.PartyState.GetCharacter(attackerIndex)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, references.Bow)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	require.True(t, gs.BeginCombatAttack())
	require.True(t, gs.MoveCombatAttackCursor(1, 0))
	require.True(t, gs.MoveCombatAttackCursor(1, 0))

	assert.Equal(t, CombatAttackSequenceComplete, gs.ConfirmCombatAttackTarget().Outcome)
	assert.Equal(t, hpBefore, monster.CurrentHP())
}

func TestGetCombatAimDisposition_DoesNotRevealInvisibleMonsters(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	origin := gs.CombatScene.GetPartyMemberPosition(attackerIndex)
	emptyPos := references.Position{X: origin.X + 1, Y: origin.Y}
	assert.Equal(t, CombatAimEmpty, gs.GetCombatAimDisposition(emptyPos))
	assert.Equal(t, CombatAimFriendly, gs.GetCombatAimDisposition(origin))

	enemyRefs := *gs.GameReferences.EnemyReferences
	monster := NewCombatMonster(references.MonsterOrc, &enemyRefs[references.MonsterOrc], emptyPos, newFixedRNG(1))
	gs.CombatScene.AddMonsterToQueue(monster)
	assert.Equal(t, CombatAimEnemy, gs.GetCombatAimDisposition(emptyPos))

	monster.SetInvisible(true)
	assert.Equal(t, CombatAimEmpty, gs.GetCombatAimDisposition(emptyPos),
		"invisible monster must look identical to an empty square")
	gs.CombatScene.PreCombatPlayerLocation.Location = references.Doom
	assert.Equal(t, CombatAimEmpty, gs.GetCombatAimDisposition(emptyPos),
		"pre-combat location must not enable Doom targeting")
	gs.CombatScene.DungeonRoomOrigin = &DungeonRoomEncounterOrigin{Location: references.Doom}
	assert.Equal(t, CombatAimEnemy, gs.GetCombatAimDisposition(emptyPos),
		"Doom interior must expose invisible monsters to targeting")

	monster.SetInvisible(false)
	monster.SetCurrentHP(0)
	assert.Equal(t, CombatAimEmpty, gs.GetCombatAimDisposition(emptyPos))
}

func TestCombatAttackTargetingTreatsPossessedPCAsEnemy(t *testing.T) {
	gs, mock := setupCombatAttackTargeting(t)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	targetIndex := (attackerIndex + 1) % party_state.MaxActiveParty
	require.True(t, gs.CombatScene.PartyActive[targetIndex])
	origin := gs.CombatScene.GetPartyMemberPosition(attackerIndex)
	targetPosition := references.Position{X: origin.X + 1, Y: origin.Y}
	gs.CombatScene.SetPartyMemberPosition(targetIndex, targetPosition)
	gs.CombatScene.PartyPossessed[targetIndex] = true
	gs.PartyState.GetCharacter(targetIndex).SetStatus(party_state.Charmed)
	gs.PartyState.GetCharacter(targetIndex).Dexterity = 1
	mock.Reset()

	require.True(t, gs.BeginCombatAttack())
	require.True(t, gs.MoveCombatAttackCursor(1, 0))
	assert.Equal(t, CombatAimEnemy, gs.GetCombatAimDisposition(targetPosition))
	result := gs.ConfirmCombatAttackTarget()
	assert.NotEqual(t, CombatAttackTargetIgnored, result.Outcome)
	if gs.CombatScene.pendingAttack != nil {
		gs.ResolvePendingCombatAttack()
	}
	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.Contains(t, []CombatEventKind{CombatEventAttack, CombatEventRangedAttack}, event.Kind)
	assert.True(t, event.Target.IsPartyMember)
	assert.Equal(t, targetIndex, event.Target.Index)
}

func TestCombatAttackTargetingKeepsUnpossessedPCFriendly(t *testing.T) {
	gs, _ := setupCombatAttackTargeting(t)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	targetIndex := (attackerIndex + 1) % party_state.MaxActiveParty
	require.True(t, gs.CombatScene.PartyActive[targetIndex])
	origin := gs.CombatScene.GetPartyMemberPosition(attackerIndex)
	targetPosition := references.Position{X: origin.X + 1, Y: origin.Y}
	gs.CombatScene.SetPartyMemberPosition(targetIndex, targetPosition)

	require.True(t, gs.BeginCombatAttack())
	assert.Equal(t, CombatAimFriendly, gs.GetCombatAimDisposition(targetPosition))
}
