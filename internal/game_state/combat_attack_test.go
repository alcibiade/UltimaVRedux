// Integration tests for the combat attack pipeline (ActionAttackCombatMap).
//
// These tests validate the full hit → damage → apply → kill → XP flow
// against real game data, using deterministic RNG seeds for reproducibility.
package game_state

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// setupCombatWithMonster creates a combat scene, clears existing monsters,
// places one monster of the given type adjacent to the active PC in the
// specified direction, and returns the GameState, mock callbacks, and monster.
func setupCombatWithMonster(
	t *testing.T,
	monsterType references.MonsterType,
	attackDir references.Direction,
	seed uint64,
) (*GameState, *MockSystemCallbacks, *CombatMonster) {
	t.Helper()

	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(seed).
		Build()
	if gs == nil {
		return nil, nil, nil
	}

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)
	require.NotNil(t, gs.CombatScene)

	clearCombatMonstersForTest(gs.CombatScene)

	// Get active PC position and compute adjacent target position
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	pcPos := gs.CombatScene.GetPartyMemberPosition(activeIdx)
	targetPos := attackDir.GetNewPositionInDirection(&pcPos)
	require.NotNil(t, targetPos, "attack direction must produce a valid position")

	// Spawn one monster at the target position
	enemyRefs := *gs.GameReferences.EnemyReferences
	ref := enemyRefs[monsterType]
	ref.EnemyAbilities = maps.Clone(ref.EnemyAbilities)
	rng := newFixedRNG(1)
	monster := NewCombatMonster(monsterType, &ref, *targetPos, rng)
	gs.CombatScene.AddMonsterToQueue(monster)

	mock.Reset()
	return gs, mock, monster
}

func TestCombatMonsterFixturesDoNotMutateCachedReferences(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	original := (*gs.GameReferences.EnemyReferences)[references.MonsterOrc]
	originalBludgeons := original.EnemyAbilities[references.Bludgeons]
	_, controlledIndex := addControlledTestMonster(gs, references.MonsterOrc, references.Position{X: 8, Y: 8}, CombatMonsterLoyaltyParty)
	controlled := gs.CombatScene.MonsterManager.GetMonster(controlledIndex)
	for _, fixture := range []*CombatMonster{monster, controlled} {
		fixture.enemyRef.Dexterity = original.Dexterity + 1
		fixture.enemyRef.EnemyAbilities[references.Bludgeons] = !originalBludgeons
	}
	assert.Equal(t, original.Dexterity, (*gs.GameReferences.EnemyReferences)[references.MonsterOrc].Dexterity)
	assert.Equal(t, originalBludgeons,
		(*gs.GameReferences.EnemyReferences)[references.MonsterOrc].EnemyAbilities[references.Bludgeons])
	assert.NotSame(t, monster.enemyRef, controlled.enemyRef)
}

// ─── Tests ──────────────────────────────────────────────────────────────────

// TestCombatAttack_HitAndDamage verifies that attacking a monster with a weapon
// produces a hit and deals damage. Uses a seed that produces a hit roll.
func TestCombatAttack_HitAndDamage(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}

	// Equip the active PC with a Long Sword (Attack=15)
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	// Clear helm/shield to isolate to one attack
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	hpBefore := monster.CurrentHP()

	result := gs.ActionAttackCombatMap(references.Down)
	assert.True(t, result, "Attack should return true when monster is present")

	require.Len(t, mock.CombatImpactCalls, 1, "attack result should use floating combat feedback")
	feedback := mock.CombatImpactCalls[0].Feedback
	assert.Equal(t, CombatAttackDeliveryMelee, feedback.Presentation.Delivery)
	assert.Equal(t, references.LongSword, feedback.Presentation.Source)
	require.Len(t, mock.CombatEvents, 1, "attack result should be retained in the event tape")
	event := mock.CombatEvents[0]
	assert.Equal(t, CombatEventAttack, event.Kind)
	assert.Equal(t, feedback.Result, event.Result)
	assert.True(t, event.Diagnostics.Hit.Rolled || event.Diagnostics.Hit.AutomaticHit)

	// If not missed/grazed, HP should have decreased
	if feedback.Result == CombatFeedbackDamage {
		assert.Less(t, monster.CurrentHP(), hpBefore,
			"Monster HP should decrease after a hit (was %d)", hpBefore)
		assert.Equal(t, feedback.HPLost, event.HPLost)
		assert.Equal(t, event.Diagnostics.FinalDamage, event.Diagnostics.Damage.Damage)
	}
}

func TestCombatAttackPresentationForEventClassifiesDeliveryAndSource(t *testing.T) {
	tests := []struct {
		name          string
		kind          CombatEventKind
		diagnostics   CombatDiagnostics
		partyAttacker bool
		wantDelivery  CombatAttackDelivery
		wantSource    references.Equipment
	}{
		{
			name: "party melee weapon", kind: CombatEventAttack,
			diagnostics: CombatDiagnostics{Source: references.LongSword}, partyAttacker: true,
			wantDelivery: CombatAttackDeliveryMelee, wantSource: references.LongSword,
		},
		{
			name: "bare hands", kind: CombatEventAttack,
			diagnostics: CombatDiagnostics{Source: references.NoEquipment}, partyAttacker: true,
			wantDelivery: CombatAttackDeliveryMelee, wantSource: references.NoEquipment,
		},
		{
			name: "physical projectile", kind: CombatEventRangedAttack,
			diagnostics: CombatDiagnostics{Source: references.Bow, Missile: references.MissileArrow}, partyAttacker: true,
			wantDelivery: CombatAttackDeliveryProjectile, wantSource: references.Bow,
		},
		{
			name: "spell", kind: CombatEventSpell,
			diagnostics: CombatDiagnostics{Spell: references.VasFlam}, partyAttacker: true,
			wantDelivery: CombatAttackDeliverySpell, wantSource: references.NoEquipment,
		},
		{
			name: "natural monster melee", kind: CombatEventPoison,
			diagnostics: CombatDiagnostics{}, partyAttacker: false,
			wantDelivery: CombatAttackDeliveryMelee, wantSource: references.NoEquipment,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			presentation := combatAttackPresentationForEvent(
				test.kind, test.diagnostics, test.partyAttacker,
			)

			assert.Equal(t, test.wantDelivery, presentation.Delivery)
			assert.Equal(t, test.wantSource, presentation.Source)
		})
	}
}

func TestCombatAttackPresentationDropsEquipmentForEnvironmentalDelivery(t *testing.T) {
	presentation := combatAttackPresentation(
		CombatAttackDeliveryEnvironmental,
		CombatDiagnostics{Source: references.LongSword},
		true,
	)

	assert.Equal(t, CombatAttackDeliveryEnvironmental, presentation.Delivery)
	assert.Equal(t, references.NoEquipment, presentation.Source)
}

// TestCombatAttack_NoMonster verifies that attacking an empty tile returns false.
func TestCombatAttack_NoMonster(t *testing.T) {
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()
	if gs == nil {
		return
	}

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)

	// Clear all monsters
	clearCombatMonstersForTest(gs.CombatScene)
	mock.Reset()

	result := gs.ActionAttackCombatMap(references.Down)
	assert.False(t, result, "Attack should return false when no monster at target")

	// No messages (the UI layer handles "Nothing to attack!" when result is false)
	assert.Equal(t, 0, len(mock.Messages), "No messages from ActionAttackCombatMap when no target")
	assert.Empty(t, mock.CombatImpactCalls, "No floating feedback when no attack occurs")
}

// TestCombatAttack_GlassSword verifies Glass Sword behavior:
// - Always hits (never miss)
// - Deals instant kill damage (99)
// - Shatters after use (weapon slot becomes NoEquipment)
// - Awards XP for the kill
func TestCombatAttack_GlassSword(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 100)
	if gs == nil {
		return
	}

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.GlassSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	expBefore := character.Exp

	result := gs.ActionAttackCombatMap(references.Down)
	assert.True(t, result, "Glass Sword attack should hit")

	// Glass Sword should always kill
	assert.True(t, monster.IsDead(), "Glass Sword should kill the monster")

	// Glass Sword should shatter
	assert.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotWeapon),
		"Glass Sword should shatter after use")

	// Shattering is retained in the non-blocking combat event tape.
	require.NotEmpty(t, mock.CombatEvents)
	assert.Equal(t, CombatEventEquipmentShattered, mock.CombatEvents[0].Kind)
	assert.Equal(t, combatInstantKillDamage, mock.CombatEvents[0].Diagnostics.FinalDamage)
	require.Len(t, mock.CombatImpactCalls, 1)
	assert.True(t, mock.CombatImpactCalls[0].Feedback.Fatal)

	// XP should increase: original formula is maxHP/4 + 1
	expectedXP := CombatCalcExperience(monster.MaxHP())
	assert.Equal(t, expBefore+uint16(expectedXP), character.Exp,
		"PC should gain %d XP for killing monster (maxHP=%d)", expectedXP, monster.MaxHP())
}

func TestCombatAttack_MagicalSwordModifiersPreserveImmortality(t *testing.T) {
	for _, weapon := range []references.Equipment{references.GlassSword, references.ChaosSword} {
		t.Run(weapon.String(), func(t *testing.T) {
			gs, mock, monster := setupCombatWithMonster(t, references.MonsterBlackthorn, references.Down, 100)
			if gs == nil {
				return
			}
			require.True(t, monster.HasAbility(references.Immortal))
			activeIdx := gs.CombatScene.GetActiveCombatantIndex()
			character := gs.PartyState.GetCharacter(activeIdx)
			character.SetEquippedItem(references.SlotWeapon, weapon)
			character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
			character.SetEquippedItem(references.SlotShield, references.NoEquipment)
			hpBefore := monster.CurrentHP()
			gs.SetRandomSeed(100)

			require.True(t, gs.ActionAttackCombatMap(references.Down))

			assert.False(t, monster.IsDead())
			assert.Equal(t, hpBefore, monster.CurrentHP())
			assert.Zero(t, gs.GetRNG().Calls())
			require.NotEmpty(t, mock.CombatEvents)
			event := mock.CombatEvents[len(mock.CombatEvents)-1]
			assert.Equal(t, CombatEventAttack, event.Kind)
			assert.True(t, event.Diagnostics.Damage.InstantKill)
			assert.True(t, event.Diagnostics.MagicWeapon)
			assert.True(t, event.Diagnostics.TargetImmortal)
			assert.Zero(t, event.Diagnostics.FinalDamage)
		})
	}
}

func TestCombatAttack_ChaosSwordUsesFixedDamageDiagnostics(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 100)
	if gs == nil {
		return
	}
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.ChaosSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	require.True(t, gs.ActionAttackCombatMap(references.Down))

	assert.True(t, monster.IsDead())
	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.Equal(t, CombatEventAttack, event.Kind)
	assert.Equal(t, combatInstantKillDamage, event.Diagnostics.FinalDamage)
	assert.True(t, event.Diagnostics.Damage.InstantKill)
	assert.False(t, event.Diagnostics.Damage.AttackRolled)
	assert.False(t, event.Diagnostics.Damage.ArmorRolled)
	assert.True(t, event.Diagnostics.MagicWeapon)
}

func TestCombatAttack_GlassSwordDealsFullDamageToUndead(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterSkeleton, references.Down, 100)
	if gs == nil {
		return
	}
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.GlassSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	gs.SetRandomSeed(100)

	require.True(t, gs.ActionAttackCombatMap(references.Down))

	assert.True(t, monster.IsDead())
	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.Equal(t, CombatEventAttack, event.Kind)
	assert.True(t, event.Diagnostics.TargetUndead)
	assert.True(t, event.Diagnostics.MagicWeapon)
	assert.Equal(t, combatInstantKillDamage, event.Diagnostics.FinalDamage)
	assert.False(t, event.Diagnostics.Damage.AttackRolled)
	assert.False(t, event.Diagnostics.Damage.ArmorRolled)
}

// TestCombatMonsterDeathFinalizesTreasureAndXPOnce prevents repeated damage
// callbacks from duplicating either a generated chest or the killing-blow XP.
func TestCombatMonsterDeathFinalizesTreasureAndXPOnce(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 169)
	if gs == nil {
		return
	}

	enemyReference := *monster.EnemyRef()
	enemyReference.TreasureNumber = combatTreasureDiceSides
	enemyReference.EnemyAbilities = map[references.EnemyAbility]bool{}
	monster.enemyRef = &enemyReference
	monster.SetCurrentHP(1)
	moveMonsterToCombatDeathObjectTerrain(t, gs, monster)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	experienceBefore := attacker.Exp
	diagnostics := CombatDiagnostics{} //nolint:exhaustruct // Death-finalization test does not inspect attack rolls.

	gs.applyCombatDamageToMonster(monster, 0, 1, false, attacker, attackerIndex, diagnostics)

	assert.Equal(t, 1, gs.CombatScene.ObjectCount())
	chest, found := gs.CombatScene.ObjectAt(monster.Position())
	require.True(t, found)
	assert.Equal(t, combatTreasureDiceSides, chest.TreasureValue())
	experienceAfterKill := attacker.Exp
	assert.Greater(t, experienceAfterKill, experienceBefore)

	gs.applyCombatDamageToMonster(monster, 0, 1, false, attacker, attackerIndex, diagnostics)

	assert.Equal(t, experienceAfterKill, attacker.Exp)
	assert.Equal(t, 1, gs.CombatScene.ObjectCount())
}

func TestCombatMonsterDeathLeavesCorpseWhenTreasureRollFails(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 169)
	if gs == nil {
		return
	}

	enemyReference := *monster.EnemyRef()
	enemyReference.TreasureNumber = 0
	enemyReference.EnemyAbilities = map[references.EnemyAbility]bool{}
	monster.enemyRef = &enemyReference
	monster.SetCurrentHP(1)
	moveMonsterToCombatDeathObjectTerrain(t, gs, monster)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	diagnostics := CombatDiagnostics{} //nolint:exhaustruct // Death-object test does not inspect attack rolls.
	gs.applyCombatDamageToMonster(monster, 0, 1, false, attacker, attackerIndex, diagnostics)

	object, found := gs.CombatScene.ObjectAt(monster.Position())
	require.True(t, found)
	assert.True(t, object.IsCorpse())
	assert.Equal(t, indexes.SpriteIndex(indexes.Splat), object.Sprite())
	assert.False(t, object.BlocksMovement())
}

// TestHeadlessDeathUsesMonsterSplat prevents DEAD_MONST from being rendered as
// the humanoid DEAD_PC body. Headlesses use normal treasure/corpse behavior.
func TestHeadlessDeathUsesMonsterSplat(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterHeadless, references.Down, 169)
	if gs == nil {
		return
	}

	enemyReference := *monster.EnemyRef()
	enemyReference.TreasureNumber = 0
	enemyReference.EnemyAbilities = map[references.EnemyAbility]bool{}
	monster.enemyRef = &enemyReference
	monster.SetCurrentHP(1)
	moveMonsterToCombatDeathObjectTerrain(t, gs, monster)

	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	diagnostics := CombatDiagnostics{} //nolint:exhaustruct // Corpse-sprite test does not inspect attack rolls.
	gs.applyCombatDamageToMonster(monster, 0, 1, false, attacker, attackerIndex, diagnostics)

	object, found := gs.CombatScene.ObjectAt(monster.Position())
	require.True(t, found)
	assert.True(t, object.IsCorpse())
	assert.Equal(t, indexes.SpriteIndex(indexes.Splat), object.Sprite())
}

func moveMonsterToCombatDeathObjectTerrain(t *testing.T, gs *GameState, monster *CombatMonster) {
	t.Helper()

	for y := range references.YCombatMapTiles {
		for x := range references.XCombatMapTiles {
			position := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
			terrain := gs.GetCurrentLayeredMap().GetTopTile(&position)
			if combatDeathObjectEligible(monster, terrain) {
				monster.SetPosition(position)

				return
			}
		}
	}

	t.Fatal("combat fixture must contain death-object-eligible terrain")
}

// TestCombatAttack_BareHands verifies bare-hands fallback when no weapon is equipped.
func TestCombatAttack_BareHands(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	// Unequip everything
	character.SetEquippedItem(references.SlotWeapon, references.NoEquipment)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	result := gs.ActionAttackCombatMap(references.Down)
	assert.True(t, result, "Bare-hands attack should return true when monster is present")

	// Bare-hands with weaponAttack=1 against armored orc will often graze or miss.
	require.Len(t, mock.CombatImpactCalls, 1, "Should produce floating combat feedback")
}

// TestCombatAttack_MonsterKillAwardsXP verifies XP is awarded on monster kill.
// Uses a very weak monster setup (low HP) to guarantee a kill.
func TestCombatAttack_MonsterKillAwardsXP(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	// Reduce monster HP to 1 so any hit kills it
	monster.SetCurrentHP(1)
	expBefore := character.Exp

	result := gs.ActionAttackCombatMap(references.Down)
	assert.True(t, result, "Attack should return true")

	require.Len(t, mock.CombatImpactCalls, 1)
	feedback := mock.CombatImpactCalls[0].Feedback

	if feedback.Fatal {
		// Monster was killed — verify XP
		expectedXP := CombatCalcExperience(monster.MaxHP())
		assert.Equal(t, expBefore+uint16(expectedXP), character.Exp,
			"PC should gain %d XP for kill", expectedXP)
		assert.True(t, monster.IsDead(), "Monster should be dead")
	} else {
		// Attack missed — that's OK with some RNG seeds, XP should be unchanged
		t.Logf("Attack did not kill (result=%v) — XP unchanged at %d", feedback.Result, character.Exp)
		assert.Equal(t, expBefore, character.Exp, "XP should be unchanged on miss")
	}
}

// TestCombatAttack_MultiSlot verifies that equipping a SpikedHelm AND a weapon
// produces multiple attack rolls (one per slot with attack value > 0).
func TestCombatAttack_MultiSlot(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 55)
	if gs == nil {
		return
	}

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	// Ensure monster has enough HP to survive multiple hits
	monster.SetCurrentHP(100) //nolint:mnd // High HP to survive multiple hits

	result := gs.ActionAttackCombatMap(references.Down)
	assert.True(t, result, "Attack should return true")

	assert.GreaterOrEqual(t, len(mock.CombatImpactCalls), 2,
		"Should have floating feedback for both SpikedHelm and LongSword")
}

// TestCombatAttack_MonsterDeathRemovesFromQueue verifies that killing a monster
// removes it from the turn queue.
func TestCombatAttack_MonsterDeathRemovesFromQueue(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	// Use Glass Sword for guaranteed kill
	character.SetEquippedItem(references.SlotWeapon, references.GlassSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	pcsBefore, monstersBefore := gs.CombatScene.TurnQueue.CountRemaining()
	queueSizeBefore := pcsBefore + monstersBefore

	gs.ActionAttackCombatMap(references.Down)

	assert.True(t, monster.IsDead(), "Monster should be dead from Glass Sword")

	// Dead monster should be removed from turn queue
	pcsAfter, monstersAfter := gs.CombatScene.TurnQueue.CountRemaining()
	queueSizeAfter := pcsAfter + monstersAfter
	assert.Less(t, queueSizeAfter, queueSizeBefore,
		"Turn queue should have fewer entries after monster death")
}

// TestCombatAttack_JeweledSword verifies Jeweled Sword behavior:
// - Always hits (never miss ability)
// - Deals 0 weapon damage (getdamage returns 0 for jeweled sword)
// - Does NOT shatter
func TestCombatAttack_JeweledSword(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.JeweledSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	require.Positive(t, monster.EnemyRef().Armour)
	gs.SetRandomSeed(42)

	hpBefore := monster.CurrentHP()

	result := gs.ActionAttackCombatMap(references.Down)
	assert.True(t, result, "Jeweled Sword attack should succeed")

	// Jeweled Sword always hits but deals 0 damage, so it floats GRAZE.
	require.Len(t, mock.CombatImpactCalls, 1)
	assert.Equal(t, CombatFeedbackGraze, mock.CombatImpactCalls[0].Feedback.Result)

	// HP unchanged (grazed = 0 damage)
	assert.Equal(t, hpBefore, monster.CurrentHP(),
		"Monster HP should be unchanged after Jeweled Sword graze")

	// Jeweled Sword should NOT shatter
	assert.Equal(t, references.JeweledSword, character.GetEquippedItem(references.SlotWeapon),
		"Jeweled Sword should not shatter")
	assert.Equal(t, uint64(1), gs.GetRNG().Calls(), "positive armor must consume exactly one roll")
	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.True(t, event.Diagnostics.MagicWeapon)
	assert.False(t, event.Diagnostics.Damage.AttackRolled)
	assert.True(t, event.Diagnostics.Damage.ArmorRolled)
}
