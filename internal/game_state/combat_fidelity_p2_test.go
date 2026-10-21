package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Original item-use rules changes the acting object's appearance or tactical sleep,
// not the identity of the PC (including a clone sharing its roster record).
// See ref:COMBAT-0005.
func TestFidelityP2PotionRetainsExactPC(t *testing.T) {
	for _, clone := range []bool{false, true} {
		for _, potion := range []references.Potion{references.Purple, references.Orange} {
			t.Run(fmt.Sprintf("potion%d/clone%t", potion, clone), func(t *testing.T) {
				gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
				require.NotNil(t, gs)
				source := gs.CombatScene.GetCurrentTurnIdentity()
				identity := source
				character := gs.PartyState.GetCharacter(source.EntityIndex)
				character.SetStatus(party_state.Good)
				if clone {
					identity = gs.CombatScene.addPCCloneFromSource(source, source.EntityIndex,
						references.Position{X: 1, Y: 1}, NewCombatPCProvider(character))
					gs.CombatScene.CurrentTurnIdentity = identity
					gs.CombatScene.currentTurnIdentitySet = true
				}
				entry := gs.CombatScene.TurnQueue.GetEntryByIdentity(identity)
				before := *character
				countdown := entry.turnCountdown
				count := gs.CombatScene.MonsterManager.Count()
				gs.PartyState.Inventory.Potions.Set(potion, 1)
				commit, _ := gs.BeginPotionUse(potion, UsageCombat)
				require.NotNil(t, commit)
				outcome, ok := gs.CommitPotionUse(commit)
				require.True(t, ok)
				gs.SetRandomSeed(42)
				gs.CommitPotionEffect(outcome, source.EntityIndex)
				assert.Equal(t, uint64(1), gs.GetRNG().Calls())
				assert.Equal(t, count, gs.CombatScene.MonsterManager.Count())
				assert.True(t, gs.CombatScene.isCombatPCActive(identity))
				assert.Same(t, entry, gs.CombatScene.TurnQueue.GetEntryByIdentity(identity))
				assert.Equal(t, turnQueueActive, entry.status)
				if potion == references.Purple {
					assert.Equal(t, before, *character)
					assert.Equal(t, countdown, entry.turnCountdown)
					assert.True(t, gs.CombatScene.IsCombatPCRat(identity))
					if clone {
						assert.False(t, gs.CombatScene.IsCombatPCRat(source))
					}
				}
				if potion == references.Orange {
					assert.True(t, gs.CombatScene.isCombatPCSleeping(identity))
					assert.Equal(t, 1, gs.combatTargetDefense(identity).EffectiveDex)
					assert.Equal(t, combatTurnSkipped, gs.processCombatPCTurnStart(identity))
					gs.setCombatPCSleeping(identity, false)
					assert.False(t, gs.CombatScene.isCombatPCSleeping(identity))
					if clone {
						assert.False(t, gs.CombatScene.isCombatPCSleeping(source))
					}
				}
			})
		}
	}
}

// Original prepared-spell rules use Intelligence saving throw for PC targets as well as monsters.
// See ref:COMBAT-0005.
func TestFidelityP2PCIntelligenceSaves(t *testing.T) {
	for _, spell := range []references.Spell{references.AnXenEx, references.RelXenBet} {
		gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
		require.NotNil(t, gs)
		caster := gs.CombatScene.GetActiveCombatantIndex()
		targetIndex := (caster + 1) % 3
		target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: targetIndex}
		gs.PartyState.GetCharacter(caster).Intelligence = 1
		gs.PartyState.GetCharacter(targetIndex).Intelligence = 255
		gs.CombatScene.setCombatPCPossessed(target, true)
		gs.SetRandomSeed(42)
		if spell == references.AnXenEx {
			assert.False(t, gs.CastCombatAnXenEx(caster, target))
		} else {
			assert.False(t, gs.CastCombatRelXenBet(caster, target))
		}
		assert.True(t, gs.CombatScene.isCombatPCActive(target))
		assert.True(t, gs.CombatScene.isCombatPCPossessed(target))
		assert.Equal(t, uint64(1), gs.GetRNG().Calls())
	}
}

func TestFidelityP2PCCloneIntelligenceSaveAndSuccess(t *testing.T) {
	for _, spell := range []references.Spell{references.AnXenEx, references.RelXenBet} {
		gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
		caster := gs.CombatScene.GetActiveCombatantIndex()
		source := CombatTurnIdentity{IsPartyMember: true, EntityIndex: (caster + 1) % 3}
		pc := gs.PartyState.GetCharacter(source.EntityIndex)
		clone := gs.CombatScene.addPCCloneFromSource(source, source.EntityIndex, references.Position{X: 1, Y: 1}, NewCombatPCProvider(pc))
		gs.CombatScene.setCombatPCPossessed(clone, true)
		pc.Intelligence = 255
		gs.PartyState.GetCharacter(caster).Intelligence = 1
		cast := gs.CastCombatAnXenEx
		if spell == references.RelXenBet {
			cast = gs.CastCombatRelXenBet
		}
		gs.SetRandomSeed(42)
		assert.False(t, cast(caster, clone))
		assert.Equal(t, uint64(1), gs.GetRNG().Calls())
		assert.True(t, gs.CombatScene.isCombatPCPossessed(clone))
		pc.Intelligence = 1
		gs.PartyState.GetCharacter(caster).Intelligence = 255
		gs.SetRandomSeed(42)
		assert.True(t, cast(caster, clone))
		assert.True(t, gs.CombatScene.isCombatPCActive(source))
		if spell == references.AnXenEx {
			assert.False(t, gs.CombatScene.isCombatPCPossessed(clone))
			assert.Equal(t, uint64(1), gs.GetRNG().Calls())
		} else {
			assert.False(t, gs.CombatScene.isCombatPCActive(clone))
			assert.Equal(t, uint64(2), gs.GetRNG().Calls(), "save then replacement DEX")
		}
	}
}

func TestFidelityP2MislabelledPurplePreservesActorAndRNG(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	var seed uint64
	for candidate := uint64(1); candidate < 10000; candidate++ {
		gs.SetRandomSeed(candidate)
		if gs.RandomIntInRange(0, potionRandomEffectRollMax-1) == 1 &&
			gs.RandomIntInRange(0, potionRandomPotionMax-1) == int(references.Purple) {
			seed = candidate
			break
		}
	}
	require.NotZero(t, seed)
	identity := gs.CombatScene.GetCurrentTurnIdentity()
	pc := gs.PartyState.GetCharacter(identity.EntityIndex)
	before := *pc
	gs.PartyState.Inventory.Potions.Set(references.Yellow, 1)
	commit, _ := gs.BeginPotionUse(references.Yellow, UsageCombat)
	outcome, ok := gs.CommitPotionUse(commit)
	require.True(t, ok)
	gs.SetRandomSeed(seed)
	assert.Equal(t, PotionEffectRat, gs.CommitPotionEffect(outcome, identity.EntityIndex))
	assert.True(t, outcome.Mislabelled)
	assert.True(t, gs.CombatScene.IsCombatPCRat(identity))
	assert.True(t, gs.CombatScene.isCombatPCActive(identity))
	assert.Equal(t, before, *pc)
	assert.Equal(t, uint64(2), gs.GetRNG().Calls())
}

func TestFidelityP2EncounterDexterity(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	require.NotNil(t, gs)
	ref := *monster.enemyRef
	monster.enemyRef = &ref
	monster.currentDex = 7
	monster.enemyRef.Dexterity = 20
	gs.SetRandomSeed(42)
	assert.Equal(t, 7, gs.combatTargetDefense(CombatTurnIdentity{EntityIndex: 0}).EffectiveDex)
	assert.Equal(t, monster.EnemyRef().Strength, gs.combatPhysicalAttackAttribute(CombatTurnIdentity{EntityIndex: 0}, references.NoEquipment))
	monster.enemyRef.EnemyAbilities = map[references.EnemyAbility]bool{}
	assert.Equal(t, 7, gs.combatPhysicalAttackAttribute(CombatTurnIdentity{EntityIndex: 0}, references.NoEquipment))
	monster.SetAsleep(true)
	assert.Equal(t, 1, gs.combatPhysicalAttackAttribute(CombatTurnIdentity{EntityIndex: 0}, references.NoEquipment))
	assert.Zero(t, gs.GetRNG().Calls())
}

func TestFidelityP2PoisonWindMonsterKillXPAndImmortality(t *testing.T) {
	for _, monsterType := range []references.MonsterType{references.MonsterOrc, references.MonsterLordBritish} {
		gs, mock, monster := setupCombatWithMonster(t, monsterType, references.Down, 42)
		caster := gs.CombatScene.GetActiveCombatantIndex()
		pc := gs.PartyState.GetCharacter(caster)
		xpBefore := pc.Exp
		monster.SetCurrentHP(1)
		monster.currentDex = 1
		gs.SetRandomSeed(42)
		mock.Reset()
		applied, onset := gs.applyCombatInNoxHurAtPosition(caster, monster.Position())
		assert.True(t, applied)
		assert.False(t, onset)
		assert.False(t, monster.IsPoisoned())
		require.NotEmpty(t, mock.CombatEvents)
		if monsterType == references.MonsterOrc {
			assert.True(t, monster.IsDead())
			assert.Equal(t, int(xpBefore)+combatCalcExperience(monster.EnemyRef().HitPoints), int(pc.Exp))
		} else {
			assert.False(t, monster.IsDead())
			assert.Equal(t, xpBefore, pc.Exp)
			assert.Equal(t, CombatFeedbackGraze, mock.CombatEvents[0].Result)
			assert.Equal(t, uint64(2), gs.GetRNG().Calls(), "immortality does not bypass save or damage draws")
		}
	}
}

func TestFidelityP2SummonIgnoresRetiredStorage(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	require.NotNil(t, gs)
	for gs.CombatScene.MonsterManager.Count() < references.MaxMonstersInCombat {
		retired := NewCombatMonster(references.MonsterOrc, monster.EnemyRef(), references.Position{}, newFixedRNG(1))
		retired.SetActive(false)
		gs.CombatScene.MonsterManager.AddMonster(retired)
	}
	gs.CombatScene.CombatMapRef = newOpenCombatConeMap()
	result := gs.CastCombatInBetXen()
	assert.Equal(t, CombatSummonSucceeded, result.Outcome)
	assert.Equal(t, 4, result.Count)
	assert.Positive(t, result.Attempts)
}

func TestFidelityP2DivisionIgnoresRetiredStorage(t *testing.T) {
	gs, _, slime := setupCombatWithMonster(t, references.MonsterSlime, references.Down, 42)
	setRepeatedMonsterDivisionForTest(t, gs, false)
	fillCombatMapForDivisionTest(gs)
	slime.SetLoyalty(CombatMonsterLoyaltyParty)
	for gs.CombatScene.MonsterManager.Count() < references.MaxMonstersInCombat {
		retired := NewCombatMonster(references.MonsterSlime, slime.EnemyRef(), references.Position{}, newFixedRNG(1))
		retired.SetActive(false)
		gs.CombatScene.MonsterManager.AddMonster(retired)
	}
	gs.tryDivideCombatMonster(slime, 0)
	require.Equal(t, references.MaxMonstersInCombat+1, gs.CombatScene.MonsterManager.Count())
	child := gs.CombatScene.MonsterManager.GetMonster(references.MaxMonstersInCombat)
	assert.Equal(t, slime.CurrentHP(), child.CurrentHP())
	assert.True(t, slime.IsPartyLoyal())
	assert.True(t, child.IsHostile(), "F16: division creates a fresh hostile monster")
	gs.tryDivideCombatMonster(slime, 0)
	gs.tryDivideCombatMonster(child, references.MaxMonstersInCombat)
	assert.Equal(t, references.MaxMonstersInCombat+1, gs.CombatScene.MonsterManager.Count())
}

func TestFidelityP2SummonReusesFledQueueSlot(t *testing.T) {
	gs, _ := newCombatSummonTestState(t)
	ref, err := gs.enemyReferenceForCombat(references.MonsterOrc)
	require.NoError(t, err)
	for range sourceCombatQueueSlots - sourceMonsterSlotStart {
		gs.CombatScene.AddMonsterToQueue(NewCombatMonster(references.MonsterOrc, ref,
			references.Position{X: 1, Y: 1}, newFixedRNG(1)))
	}
	identity := CombatTurnIdentity{EntityIndex: 0}
	gs.CombatScene.MonsterManager.GetMonster(0).SetActive(false)
	gs.CombatScene.TurnQueue.markFledIdentity(identity)
	rng := newStrictSummonRNG(t, 5, 5, 0)
	result := gs.summonCombatMonsters(references.InBetXen, references.MonsterInsect, 4, CombatMonsterLoyaltyParty, rng)
	assert.Equal(t, 1, result.Count)
	require.Len(t, result.Created, 1)
	entry := gs.CombatScene.TurnQueue.GetEntryByIdentity(CombatTurnIdentity{EntityIndex: result.Created[0].Index})
	assert.Equal(t, sourceMonsterSlotStart, entry.sourceSlot)
	assert.Equal(t, []int{16, 16, 8}, rng.requests, "placement, then one creation DEX, then queue full")
}

func TestFidelityP2TownWindScrollOnly(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	require.True(t, gs.isScrollLocationValid(references.ScrollRelHur))
	gs.PartyState.Inventory.Scrolls.Set(references.ScrollRelHur, 1)
	assert.True(t, requireUsableItem(t, gs.GetUsableItemsForContext(UsageField), references.ScrollRelHur).AvailableNow)
	assert.True(t, gs.useScroll(references.ScrollRelHur))
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur))
	assert.False(t, references.IsSpellValidInContext(references.RelHur, references.ContextTown))
	gs.CompleteWindChangeScroll(references.Up)
	assert.Zero(t, gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur))
}

func TestFidelityP2UnreadyRingExactEntry(t *testing.T) {
	for _, useClone := range []bool{false, true} {
		gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
		source := gs.CombatScene.GetCurrentTurnIdentity()
		character := gs.PartyState.GetCharacter(source.EntityIndex)
		gs.CombatScene.setCombatPCInvisible(source, true)
		clone := gs.CombatScene.addPCCloneFromSource(source, source.EntityIndex, references.Position{X: 1, Y: 1}, NewCombatPCProvider(character))
		actor, other := source, clone
		if useClone {
			actor, other = clone, source
		}
		gs.CombatScene.CurrentTurnIdentity = actor
		gs.CombatScene.currentTurnIdentitySet = true
		gs.CombatScene.turnsStarted = true
		character.SetEquippedItem(references.SlotRing, references.RingInvisibility)
		gs.PartyState.Inventory.Equipment.Set(references.RingInvisibility, 0)
		request, result := gs.BeginReadyEquipment(true)
		require.Equal(t, ReadyBeginDialog, result)
		gs.SetRandomSeed(42)
		_, err := gs.ApplyReadyEquipment(ReadyEquipmentTransaction{Request: request, CharacterIndex: source.EntityIndex,
			ExpectedOccupant: character.GetNameAsString(), Equipment: references.RingInvisibility})
		require.NoError(t, err)
		assert.False(t, gs.CombatScene.isCombatPCInvisible(actor))
		assert.True(t, gs.CombatScene.isCombatPCInvisible(other))
		assert.Equal(t, uint16(1), gs.PartyState.Inventory.Equipment.Get(references.RingInvisibility))
		assert.Zero(t, gs.GetRNG().Calls())
	}
}

func TestFidelityP2PoisonWindToxifyPCStatuses(t *testing.T) {
	for _, status := range []party_state.CharacterStatus{party_state.Good, party_state.Poisoned, party_state.Sleep} {
		gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
		target := gs.CombatScene.GetCurrentTurnIdentity()
		pc := gs.PartyState.GetCharacter(target.EntityIndex)
		pc.SetStatus(status)
		pc.SetMaxHp(100)
		pc.SetHp(100)
		pc.Dexterity = 1
		gs.SetRandomSeed(42)
		gs.RollOriginalD30()
		damage := gs.GetRNG().Intn(21)
		gs.SetRandomSeed(42)
		mock.Reset()
		applied, onset := gs.applyCombatInNoxHurAtPosition(target.EntityIndex, gs.CombatScene.GetCombatantPosition(target))
		assert.True(t, applied)
		assert.Equal(t, status == party_state.Good, onset)
		if status == party_state.Good {
			assert.Equal(t, uint16(100), pc.GetCurrentHp())
			assert.True(t, pc.IsPoisoned())
			assert.Equal(t, uint64(1), gs.GetRNG().Calls())
		} else {
			assert.Equal(t, uint16(100-damage), pc.GetCurrentHp())
			assert.Equal(t, status, pc.GetStatus())
			assert.Equal(t, uint64(2), gs.GetRNG().Calls())
		}
		require.Len(t, mock.CombatEvents, 1)
		assert.True(t, mock.CombatEvents[0].Diagnostics.Hit.Rolled)
	}
}

func TestFidelityP2ConeLoyaltyDoesNotChangeEffects(t *testing.T) {
	for _, spell := range []references.Spell{references.InZu, references.InNoxHur, references.InFlamHur, references.InVasGravCorp} {
		var previousCalls uint64
		for _, loyalty := range []CombatMonsterLoyalty{CombatMonsterLoyaltyMonster, CombatMonsterLoyaltyParty} {
			gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
			caster := gs.CombatScene.GetActiveCombatantIndex()
			gs.PartyState.GetCharacter(caster).Intelligence = 255
			monster.currentDex = 1
			monster.SetLoyalty(loyalty)
			gs.SetRandomSeed(42)
			position := monster.Position()
			switch spell {
			case references.InZu:
				assert.True(t, gs.applyCombatInZuAtPosition(position, 255))
				assert.True(t, monster.IsAsleep())
			case references.InNoxHur:
				applied, _ := gs.applyCombatInNoxHurAtPosition(caster, position)
				assert.True(t, applied)
				assert.False(t, monster.IsPoisoned())
			case references.InFlamHur:
				assert.True(t, gs.applyCombatInFlamHurAtPosition(caster, position))
			case references.InVasGravCorp:
				assert.True(t, gs.applyCombatInVasGravCorpAtPosition(caster, position))
				assert.True(t, monster.IsDead())
			}
			if loyalty == CombatMonsterLoyaltyMonster {
				previousCalls = gs.GetRNG().Calls()
			} else {
				assert.Equal(t, previousCalls, gs.GetRNG().Calls())
			}
		}
	}
}

func TestFidelityP2DirectSpellsIgnoreCosmeticWall(t *testing.T) {
	for _, spell := range []references.Spell{references.AnXenEx, references.RelXenBet, references.InQuasXen, references.VasFlam} {
		gs, _, monster := setupVasFlamTargeting(t)
		caster := gs.CombatScene.GetActiveCombatantIndex()
		pc := gs.PartyState.GetCharacter(caster)
		pc.Level, pc.Intelligence = 8, 255
		pc.SetMp(99)
		gs.CombatScene.SetPartyMemberPosition(caster, references.Position{X: 5, Y: 5})
		monster.SetPosition(references.Position{X: 5, Y: 7})
		gs.CombatScene.CombatMapRef.SetTileNumber(references.Position{X: 5, Y: 6}, indexes.StoneBrickWall)
		gs.PartyState.Inventory.Spells.Set(spell, 1)
		require.True(t, gs.BeginCombatSpellTargeting(CastRequest{Spell: spell, CasterIdx: caster}).Valid)
		gs.CombatScene.SpellTargeting.Cursor = monster.Position()
		before := gs.CombatScene.MonsterManager.Count()
		hp := monster.CurrentHP()
		result := gs.ConfirmCombatSpellTarget()
		require.Equal(t, CombatSpellTargetResolved, result.Outcome)
		assert.Equal(t, spell == references.VasFlam, result.Launch.Blocked)
		assert.Equal(t, references.MissileNone, result.Launch.Missile)
		require.True(t, gs.ResolvePendingCombatSpell())
		assert.False(t, gs.ResolvePendingCombatSpell())
		switch spell {
		case references.AnXenEx:
			assert.False(t, monster.IsHostile())
		case references.RelXenBet, references.InQuasXen:
			assert.Equal(t, before+1, gs.CombatScene.MonsterManager.Count())
		case references.VasFlam:
			assert.Equal(t, hp, monster.CurrentHP())
		}
	}
}

func TestFidelityP2EarthquakeHostilePCAndStatusDexSourceOrder(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterMimic, references.Down, 42)
	caster := gs.CombatScene.GetActiveCombatantIndex()
	target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: (caster + 1) % 3}
	gs.CombatScene.setCombatPCPossessed(target, true)
	gs.setCombatPCSleeping(target, true)
	pc := gs.PartyState.GetCharacter(target.EntityIndex)
	pc.Dexterity = 255
	pc.SetHp(100)
	monster.currentDex = 255
	gs.SetRandomSeed(42)
	mock.Reset()
	affected := gs.CastCombatInVasPorYlem(caster)
	assert.Contains(t, affected, gs.CombatScene.GetCombatantPosition(target))
	assert.Contains(t, affected, monster.Position())
	require.Len(t, mock.CombatEvents, 2)
	assert.True(t, mock.CombatEvents[0].Target.IsPartyMember)
	assert.Equal(t, target.EntityIndex, mock.CombatEvents[0].Target.Index)
	assert.False(t, mock.CombatEvents[1].Target.IsPartyMember)
	assert.Equal(t, uint64(4), gs.GetRNG().Calls())
}

func TestFidelityP2MixSkipsSleepingVictim(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	gs.PartyState.ForEachCharacter(func(_ int, pc *party_state.PlayerCharacter) { pc.SetStatus(party_state.Sleep) })
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Poisoned)
	gs.SetRandomSeed(42)
	result := gs.ExecuteBoom()
	require.NotNil(t, result)
	assert.Equal(t, 1, result.TargetIndex)
	assert.Equal(t, uint64(1), gs.GetRNG().Calls())
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Sleep)
	assert.Nil(t, gs.ExecuteBoom())
	assert.Equal(t, uint64(1), gs.GetRNG().Calls())
}

func TestFidelityP2MixSparseRosterAndInactiveVictims(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	avatar := *gs.PartyState.GetCharacter(0)
	avatar.SetStatus(party_state.Sleep)
	iolo := *gs.PartyState.GetCharacter(1)
	iolo.SetStatus(party_state.Good)
	inactive := iolo
	inactive.PartyMap = party_state.PartyMapUnrecruited
	roster := [party_state.NRosterSlots]*party_state.PlayerCharacter{0: &avatar, 2: &inactive, 3: &iolo}
	gs.PartyState = *party_state.NewPartyStateFromRoster(roster)
	gs.SetRandomSeed(42)
	result := gs.ExecuteBoom()
	require.NotNil(t, result)
	assert.Equal(t, 3, result.TargetIndex)
	assert.Equal(t, uint64(1), gs.GetRNG().Calls())
}
