package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCorpserMeleeExposureAndDragGates(t *testing.T) {
	tests := []struct {
		name          string
		selectSeed    func(CombatHitResult, CombatDamageResult) bool
		prepareTarget func(*party_state.PlayerCharacter, *CombatMonster)
		expectResult  CombatFeedbackResult
		expectDragged bool
		expectDead    bool
	}{
		{
			name: "miss exposes without dragging",
			selectSeed: func(hit CombatHitResult, _ CombatDamageResult) bool {
				return hit.Outcome == CombatHitMiss
			},
			expectResult: CombatFeedbackMiss,
		},
		{
			name: "graze exposes without dragging",
			prepareTarget: func(character *party_state.PlayerCharacter, corpser *CombatMonster) {
				character.SetEquippedItem(references.SlotArmour, references.ClothArmour)
				corpser.enemyRef.Damage = 1
			},
			selectSeed: func(hit CombatHitResult, damage CombatDamageResult) bool {
				return hit.Outcome != CombatHitMiss && damage.Deflected
			},
			expectResult: CombatFeedbackGraze,
		},
		{
			name: "positive nonlethal damage drags after damage",
			selectSeed: func(hit CombatHitResult, damage CombatDamageResult) bool {
				return hit.Outcome != CombatHitMiss && damage.Damage > 0
			},
			expectResult:  CombatFeedbackDamage,
			expectDragged: true,
		},
		{
			name: "lethal damage does not drag",
			selectSeed: func(hit CombatHitResult, damage CombatDamageResult) bool {
				return hit.Outcome != CombatHitMiss && damage.Damage > 0
			},
			prepareTarget: func(character *party_state.PlayerCharacter, _ *CombatMonster) {
				character.SetHp(1)
			},
			expectResult: CombatFeedbackDamage,
			expectDead:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock, corpser := setupCombatWithMonster(t, references.MonsterCorpser, references.Down, 42)
			corpserRef := *corpser.enemyRef
			corpser.enemyRef = &corpserRef
			target := findNearestPC(corpser.Position(), gs.CombatScene)
			require.True(t, target.found)
			require.Equal(t, monsterAdjacentDistance, target.distance)
			character := gs.PartyState.GetCharacter(target.identity.EntityIndex)
			if test.prepareTarget != nil {
				test.prepareTarget(character, corpser)
			}
			hpBefore := character.GetCurrentHp()
			seed := findCorpserMeleeSeed(t, gs, corpser, target.identity, test.selectSeed)
			gs.SetRandomSeed(seed)
			mock.Reset()

			require.True(t, gs.monsterMeleeAttack(corpser, 0, target))
			assert.Equal(t, []SoundEffect{SoundCombatSwing}, mock.SoundEffectsPlayed,
				"each committed enemy melee attack plays one opening cue, regardless of outcome")

			assert.True(t, corpser.IsCorpserExposed(), "exposure precedes hit resolution")
			assert.Equal(t, test.expectDead, character.IsDead())
			assert.Equal(t, test.expectDragged, gs.CombatScene.isCombatPCUnderground(target.identity))
			require.NotEmpty(t, mock.CombatEvents)
			assert.Equal(t, CombatEventAttack, mock.CombatEvents[0].Kind)
			assert.Equal(t, test.expectResult, mock.CombatEvents[0].Result)
			if test.expectDragged {
				assert.Less(t, character.GetCurrentHp(), hpBefore)
				require.Len(t, mock.CombatEvents, 2)
				assert.Equal(t, CombatEventCondition, mock.CombatEvents[1].Kind)
				assert.Equal(t, CombatConditionDraggedUnder, mock.CombatEvents[1].Condition)
				assert.Equal(t, target.identity.Slot, mock.CombatEvents[1].Target.Slot)
				assert.Equal(t, []string{character.GetNameAsString() + " dragged under!"}, mock.Messages)
			} else {
				assert.Len(t, mock.CombatEvents, 1)
				assert.Empty(t, mock.Messages)
			}
		})
	}
}

func TestHiddenCorpserDirectTargetingAndRawConeLookup(t *testing.T) {
	gs, _, corpser := setupCombatWithMonster(t, references.MonsterCorpser, references.Down, 42)
	position := corpser.Position()
	monsterAtPosition, monsterIndex := gs.CombatScene.MonsterManager.GetMonsterIndexAtPosition(position)
	require.Same(t, corpser, monsterAtPosition)

	assert.True(t, corpser.IsCorpserHidden())
	assert.Equal(t, CombatAimEmpty, gs.GetCombatAimDisposition(position))
	_, _, _, attackFound := gs.combatAttackOccupantAt(position)
	assert.False(t, attackFound)
	_, missileFound := gs.combatMissileImpactOccupantAt(position)
	assert.False(t, missileFound)
	_, spellFound := gs.combatSpellTargetAt(position, monsterAtPosition, monsterIndex)
	assert.False(t, spellFound)
	hiddenHP := corpser.CurrentHP()
	attackCombatSquare(t, gs, position)
	assert.Equal(t, hiddenHP, corpser.CurrentHP(), "Attack cannot find a hidden Corpser")
	assert.True(t, gs.CombatScene.MonsterManager.IsPositionOccupied(position),
		"the hidden base shape still blocks movement")

	seed := seedForFirstIntnValue(t, gs, 30, 1)
	gs.SetRandomSeed(seed)
	hpBefore := corpser.CurrentHP()
	assert.True(t, gs.applyCombatInFlamHurAtPosition(gs.CombatScene.GetActiveCombatantIndex(), position))
	assert.Less(t, corpser.CurrentHP(), hpBefore, "raw cone scans still affect hidden Corpsers")

	corpser.SetCorpserExposed(true)
	assert.Equal(t, CombatAimEnemy, gs.GetCombatAimDisposition(position))
	_, _, _, attackFound = gs.combatAttackOccupantAt(position)
	assert.True(t, attackFound)
	_, missileFound = gs.combatMissileImpactOccupantAt(position)
	assert.True(t, missileFound)
}

func TestUndergroundPCRemainsScheduledButIsUntargetableAndNonblocking(t *testing.T) {
	gs, _, corpser := setupCombatWithMonster(t, references.MonsterCorpser, references.Down, 42)
	identity := gs.CombatScene.GetCurrentTurnIdentity()
	require.True(t, identity.IsPartyMember)
	position := gs.CombatScene.GetCombatantPosition(identity)
	for index := range gs.CombatScene.PartyActive {
		if index != identity.EntityIndex {
			gs.CombatScene.PartyActive[index] = false
		}
	}
	gs.CombatScene.setCombatPCUnderground(identity, true)

	assert.False(t, gs.BeginCombatAttack())
	assert.Equal(t, "Not thy turn!", gs.BeginCombatSpellTargeting(CastRequest{
		Spell: references.VasFlam, CasterIdx: identity.EntityIndex,
	}).ErrorMsg)
	assert.Equal(t, "Not thy turn!", gs.BeginCombatAdjacentSpellTargeting(CastRequest{
		Spell: references.AnYlem, CasterIdx: identity.EntityIndex,
	}).ErrorMsg)
	assert.Equal(t, "Not thy turn!", gs.BeginCombatConeTargeting(CastRequest{
		Spell: references.InNoxHur, CasterIdx: identity.EntityIndex,
	}).ErrorMsg)
	entry := gs.CombatScene.TurnQueue.GetEntryByIdentity(identity)
	require.NotNil(t, entry)
	assert.True(t, entry.Combatant.IsAlive())
	assert.False(t, entry.Combatant.CanTakeTurn())
	assert.Equal(t, CombatAimEmpty, gs.GetCombatAimDisposition(position))
	_, _, _, attackFound := gs.combatAttackOccupantAt(position)
	assert.False(t, attackFound)
	_, missileFound := gs.combatMissileImpactOccupantAt(position)
	assert.False(t, missileFound)
	_, spellFound := gs.combatSpellTargetAt(position, nil, NoCombatTarget)
	assert.False(t, spellFound)
	rawIdentity, rawFound := gs.CombatScene.combatPCAtPosition(position)
	assert.True(t, rawFound, "area and field scans retain the queue entry")
	assert.Equal(t, identity, rawIdentity)
	assert.False(t, gs.CombatScene.isPartyMemberAtPosition(position, -1))
	assert.True(t, isCombatPositionValidForMonster(gs, position, corpser, 0))
	assert.False(t, findNearestPC(corpser.Position(), gs.CombatScene).found)
	assert.False(t, gs.combatPCEligibleForPossession(identity))
	partyLoyal, monsterLoyal := gs.countActiveCombatLoyalties()
	assert.Equal(t, 1, partyLoyal)
	assert.Equal(t, 1, monsterLoyal)
	assert.Equal(t, CombatNotEnded, gs.checkCombatEndConditions())

	character := gs.PartyState.GetCharacter(identity.EntityIndex)
	hpBefore := character.GetCurrentHp()
	gs.SetRandomSeed(seedForFirstIntnValue(t, gs, 30, 1))
	assert.True(t, gs.applyCombatInFlamHurAtPosition(identity.EntityIndex, position))
	assert.Less(t, character.GetCurrentHp(), hpBefore, "raw cone scans still affect underground PCs")
}

func TestUndergroundEscapeUsesStrictDexterityRollAndAlwaysSkipsTurn(t *testing.T) {
	tests := []struct {
		name              string
		roll              int
		expectUnderground bool
		expectCondition   CombatConditionResult
		expectMessages    func(string) []string
		killCorpser       bool
	}{
		{
			name:              "equal roll fails",
			roll:              10,
			expectUnderground: true,
			expectCondition:   CombatConditionUnderground,
			expectMessages:    func(string) []string { return []string{"ARGH!"} },
		},
		{
			name:              "lower roll succeeds after Corpser death",
			roll:              9,
			expectUnderground: false,
			expectCondition:   CombatConditionRegurgitated,
			expectMessages: func(name string) []string {
				return []string{"ARGH!", name + " regurgitated!"}
			},
			killCorpser: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock, corpser := setupCombatWithMonster(t, references.MonsterCorpser, references.Down, 42)
			identity := findNearestPC(corpser.Position(), gs.CombatScene).identity
			character := gs.PartyState.GetCharacter(identity.EntityIndex)
			character.Dexterity = 10
			gs.CombatScene.setCombatPCUnderground(identity, true)
			entry := gs.CombatScene.TurnQueue.GetEntryByIdentity(identity)
			require.NotNil(t, entry)
			assert.False(t, entry.Combatant.CanTakeTurn())
			if test.killCorpser {
				corpser.SetCurrentHP(0)
				gs.CombatScene.TurnQueue.MarkDead(0, false)
			}
			seed := findOriginalD30Seed(t, gs, test.roll)
			gs.SetRandomSeed(seed)
			mock.Reset()

			result := gs.processCombatTurnStart(entry)
			actualNextRoll := gs.RandomIntInRange(0, 10000)
			gs.SetRandomSeed(seed)
			gs.RollOriginalD30()
			expectedNextRoll := gs.RandomIntInRange(0, 10000)

			assert.Equal(t, combatTurnSkipped, result)
			assert.Equal(t, test.expectUnderground, gs.CombatScene.isCombatPCUnderground(identity))
			assert.Equal(t, !test.expectUnderground, entry.Combatant.CanTakeTurn())
			assert.Equal(t, expectedNextRoll, actualNextRoll, "escape must consume exactly one source RNG draw")
			assert.Equal(t, test.expectMessages(character.GetNameAsString()), mock.Messages)
			require.Len(t, mock.CombatEvents, 1)
			assert.Equal(t, test.expectCondition, mock.CombatEvents[0].Condition)
			assert.Equal(t, identity.Slot, mock.CombatEvents[0].Target.Slot)
		})
	}
}

func TestUndergroundEscapeTurnStillResolvesStandingField(t *testing.T) {
	gs, mock, corpser := setupCombatWithMonster(t, references.MonsterCorpser, references.Down, 42)
	identity := findNearestPC(corpser.Position(), gs.CombatScene).identity
	character := gs.PartyState.GetCharacter(identity.EntityIndex)
	character.Dexterity = 255
	gs.CombatScene.setCombatPCUnderground(identity, true)
	position := corpser.Position()
	gs.CombatScene.PartyPositions[identity.EntityIndex] = position
	require.True(t, gs.CombatScene.AddField(NewCombatField(CombatFieldFire, position)))
	forceNextCombatActor(gs.CombatScene.TurnQueue, identity.EntityIndex, true)
	monsterHPBefore := corpser.CurrentHP()
	randomCallsBefore := gs.GetRNG().Calls()
	mock.Reset()

	gs.AdvanceCombatTurn()

	assert.False(t, gs.CombatScene.isCombatPCUnderground(identity))
	require.Len(t, mock.CombatEvents, 2)
	assert.Equal(t, CombatEventCondition, mock.CombatEvents[0].Kind)
	assert.Equal(t, CombatConditionRegurgitated, mock.CombatEvents[0].Condition)
	assert.Equal(t, CombatEventFieldTriggered, mock.CombatEvents[1].Kind)
	assert.True(t, mock.CombatEvents[1].Target.IsPartyMember)
	assert.Equal(t, identity.EntityIndex, mock.CombatEvents[1].Target.Index)
	assert.Equal(t, position, mock.CombatEvents[1].Target.Position)
	assert.Equal(t, monsterHPBefore, corpser.CurrentHP(), "infield targets the scheduled queue entry")
	assert.Equal(t, randomCallsBefore+2, gs.GetRNG().Calls(), "escape and standing fire each consume one draw")
	assert.Len(t, gs.CombatScene.Fields(), 1, "skipped turns do not establish endaction field expiration")
}

func TestUndergroundChaosSwordControlPrecedesEscape(t *testing.T) {
	gs, mock, corpser := setupCombatWithMonster(t, references.MonsterCorpser, references.Down, 42)
	identity := findNearestPC(corpser.Position(), gs.CombatScene).identity
	gs.CombatScene.setCombatPCUnderground(identity, true)
	gs.PartyState.GetCharacter(identity.EntityIndex).SetEquippedItem(references.SlotShield, references.ChaosSword)
	entry := gs.CombatScene.TurnQueue.GetEntryByIdentity(identity)
	require.NotNil(t, entry)
	const seed = uint64(119)
	gs.SetRandomSeed(seed)
	mock.Reset()

	assert.Equal(t, combatTurnSkipped, gs.processCombatTurnStart(entry))
	actualFirstRoll := gs.RandomIntInRange(0, 10000)
	gs.SetRandomSeed(seed)
	expectedFirstRoll := gs.RandomIntInRange(0, 10000)

	assert.True(t, gs.CombatScene.isCombatPCPossessed(identity))
	assert.True(t, gs.CombatScene.isCombatPCUnderground(identity))
	assert.Equal(t, expectedFirstRoll, actualFirstRoll, "Chaos interception must bypass regurgitation RNG")
	assert.Empty(t, mock.Messages)
	assert.Empty(t, mock.CombatEvents)
}

func TestCorpserResetAndUndergroundCloneIdentity(t *testing.T) {
	gs, mock, corpser := setupCombatWithMonster(t, references.MonsterCorpser, references.Down, 42)
	entry := gs.CombatScene.TurnQueue.GetEntry(0, false)
	require.NotNil(t, entry)
	corpser.SetCorpserExposed(true)
	assert.Equal(t, combatTurnReady, gs.processCombatMonsterTurnStart(0))
	assert.True(t, corpser.IsCorpserHidden())

	corpser.SetCorpserExposed(true)
	corpser.SetAsleep(true)
	gs.SetRandomSeed(findCombatWakeSeed(t, gs, combatMonsterWakeRollMaximum, func(roll int) bool {
		return roll != combatMonsterWakeRollMaximum
	}))
	assert.Equal(t, combatTurnSkipped, gs.processCombatMonsterTurnStart(0))
	assert.True(t, corpser.IsCorpserExposed(), "sleeping turns do not run the Corpser reset")

	source := findNearestPC(corpser.Position(), gs.CombatScene).identity
	clone := gs.CombatScene.addPCCloneFromSource(
		source,
		source.EntityIndex,
		references.Position{X: corpser.Position().X + 1, Y: corpser.Position().Y},
		NewCombatPCProvider(gs.PartyState.GetCharacter(source.EntityIndex)),
	)
	require.True(t, clone.IsPartyMember)
	mock.Reset()
	gs.tryDragCombatPCUnder(corpser, clone, true, 1, false)
	assert.True(t, gs.CombatScene.isCombatPCUnderground(clone))
	assert.False(t, gs.CombatScene.isCombatPCUnderground(source))
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, clone.Slot, mock.CombatEvents[0].Target.Slot)
}

func TestUndergroundRememberedMeleeEngagerCannotInterfere(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	attacker := findNearestPC(monster.Position(), gs.CombatScene).identity
	target := CombatTurnIdentity{IsPartyMember: false, EntityIndex: 0}
	gs.rememberMeleeEngager(target, attacker)
	_, found := gs.validRememberedMeleeEngager(target)
	require.True(t, found)

	gs.CombatScene.setCombatPCUnderground(attacker, true)
	_, found = gs.validRememberedMeleeEngager(target)
	assert.False(t, found)
}

func findCorpserMeleeSeed(
	t *testing.T,
	gs *GameState,
	corpser *CombatMonster,
	target CombatTurnIdentity,
	matches func(CombatHitResult, CombatDamageResult) bool,
) uint64 {
	t.Helper()
	defense := gs.combatTargetDefense(target)
	armor := gs.combatPartyDefense(gs.PartyState.GetCharacter(target.EntityIndex))
	attackerStat := corpser.CurrentDex()
	if corpser.HasAbility(references.Bludgeons) {
		attackerStat = corpser.EnemyRef().Strength
	}
	for seed := uint64(0); seed < 10000; seed++ {
		gs.SetRandomSeed(seed)
		rng := gs.AsRandomProvider()
		hit := CombatHitCheck(attackerStat, defense.EffectiveDex, false, rng)
		damage := CombatDamageResult{}
		if hit.Outcome != CombatHitMiss {
			damage = CombatCalcDamageRaw(corpser.EnemyRef().Damage, armor, rng)
		}
		if matches(hit, damage) {
			return seed
		}
	}
	t.Fatal("no seed produced the requested Corpser melee result")

	return 0
}

func findOriginalD30Seed(t *testing.T, gs *GameState, wanted int) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		gs.SetRandomSeed(seed)
		if gs.RollOriginalD30() == wanted {
			return seed
		}
	}
	t.Fatalf("no seed produced original D30 roll %d", wanted)

	return 0
}
