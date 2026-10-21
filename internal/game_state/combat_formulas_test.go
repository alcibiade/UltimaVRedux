// Tests for combat formula helpers use controlled random inputs to verify
// hit thresholds, damage, modifiers and effective attributes. ref:COMBAT-0004
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ─── Mock RNG ───────────────────────────────────────────────────────────────

// sequenceRNG returns values from a preset sequence, cycling if exhausted.
// This lets tests control exactly what "random" values the formulas see.
type sequenceRNG struct {
	values []int
	index  int
}

func (r *sequenceRNG) Intn(n int) int {
	v := r.values[r.index%len(r.values)]
	r.index++

	return v % n
}

func newSequenceRNG(values ...int) *sequenceRNG {
	return &sequenceRNG{values: values}
}

// ─── CombatHitCheck ─────────────────────────────────────────────────────────
// Validates hit check from original combat damage rules.
// Formula: tohit = (30 + defenderDex - attackerStat) / 2
//          hit if transformed d30 >= tohit   (transformed d30 returns 1-30)
// See ref:RNG-0001.

func TestCombatHitCheck_NeverMissWeapons(t *testing.T) {
	// Glass, Jeweled and Chaos Swords bypass the ordinary hit test
	// regardless of attacker or defender attributes. ref:EQUIP-0012
	rng := newSequenceRNG(0) // would be a miss roll, but doesn't matter
	result := CombatHitCheck(1, 30, true, rng)
	assert.Equal(t, CombatHitNeverMiss, result.Outcome,
		"never-miss weapons should hit even with worst attacker vs best defender")
	assert.Equal(t, 1, result.AttackerStat)
	assert.Equal(t, 30, result.DefenderEffectiveDex)
	assert.Equal(t, 29, result.Threshold)
	assert.Zero(t, result.D30Roll)
	assert.False(t, result.Rolled)
	assert.True(t, result.AutomaticHit)
	assert.Zero(t, rng.index, "automatic hits must not consume RNG")
}

func TestCombatHitCheck_EqualDex(t *testing.T) {
	// Equal dex: tohit = (30 + 15 - 15) / 2 = 15
	// Source value 30 maps to 15, which equals the threshold and hits.
	rng := newSequenceRNG(30)
	result := CombatHitCheck(15, 15, false, rng)
	assert.Equal(t, CombatHitNormal, result.Outcome,
		"roll exactly equal to tohit should be a hit")

	// Source value 28 maps to 14, one below the threshold.
	rng = newSequenceRNG(28)
	result = CombatHitCheck(15, 15, false, rng)
	assert.Equal(t, CombatHitMiss, result.Outcome,
		"roll one below tohit should be a miss")
	assert.Equal(t, 15, result.AttackerStat)
	assert.Equal(t, 15, result.DefenderEffectiveDex)
	assert.Equal(t, 15, result.Threshold)
	assert.Equal(t, 14, result.D30Roll)
	assert.True(t, result.Rolled)
	assert.False(t, result.AutomaticHit)
	assert.Equal(t, 1, rng.index, "a normal hit check must consume exactly one RNG value")
}

func TestCombatHitCheck_HighAttackerDex(t *testing.T) {
	// Attacker dex 25 vs defender dex 10: tohit = (30 + 10 - 25) / 2 = 7
	// Source value 14 maps to 7.
	rng := newSequenceRNG(14)
	result := CombatHitCheck(25, 10, false, rng)
	assert.Equal(t, CombatHitNormal, result.Outcome,
		"high attacker dex should make hits easier")

	// Even a low roll of 7 hits — roll of 6 misses
	rng = newSequenceRNG(12)
	result = CombatHitCheck(25, 10, false, rng)
	assert.Equal(t, CombatHitMiss, result.Outcome)
}

func TestCombatHitCheck_HighDefenderDex(t *testing.T) {
	// Attacker dex 10 vs defender dex 25: tohit = (30 + 25 - 10) / 2 = 22
	// Source value 44 maps to 22.
	rng := newSequenceRNG(44)
	result := CombatHitCheck(10, 25, false, rng)
	assert.Equal(t, CombatHitNormal, result.Outcome,
		"high defender dex makes hits harder but not impossible")

	// Source value 42 maps to 21.
	rng = newSequenceRNG(42)
	result = CombatHitCheck(10, 25, false, rng)
	assert.Equal(t, CombatHitMiss, result.Outcome)
}

func TestCombatHitCheck_MaxRollAlwaysHits(t *testing.T) {
	// The sole source value 60 maps to 30 and should always hit.
	// Even worst case: attacker 1, defender 30 → tohit = (30+30-1)/2 = 29
	// 30 >= 29 → hit
	rng := newSequenceRNG(60)
	result := CombatHitCheck(1, 30, false, rng)
	assert.Equal(t, CombatHitNormal, result.Outcome,
		"maximum roll should always hit")
}

func TestCombatHitCheck_SleepingTarget(t *testing.T) {
	// Sleeping target has effective dex 1 (from CombatGetEffectiveDex).
	// Attacker dex 15 vs defender dex 1: tohit = (30 + 1 - 15) / 2 = 8
	// Source value 16 maps to 8.
	rng := newSequenceRNG(16)
	effectiveDex := CombatGetEffectiveDex(20, true)
	result := CombatHitCheck(15, effectiveDex, false, rng)
	assert.Equal(t, CombatHitNormal, result.Outcome,
		"sleeping targets should be very easy to hit")
}

// ─── CombatCalcDamage ───────────────────────────────────────────────────────
// Validates damage calculation from original combat damage rules.
// See ref:EQUIP-0013.

func TestCombatCalcDamage_GlassSword(t *testing.T) {
	// Glass Sword: instant kill (99), weapon shatters, ignores armor.
	rng := newSequenceRNG(0)
	result := CombatCalcDamage(references.GlassSword, 10, rng)
	assert.Equal(t, 99, result.Damage)
	assert.True(t, result.InstantKill, "glass sword should instant kill")
	assert.True(t, result.WeaponShattered, "glass sword should shatter")
	assert.False(t, result.Deflected)
	assert.Equal(t, 99, result.AttackMax)
	assert.Equal(t, 99, result.AttackRoll)
	assert.False(t, result.AttackRolled)
	assert.Equal(t, 10, result.ArmorValue)
	assert.Zero(t, result.ArmorRoll)
	assert.False(t, result.ArmorRolled)
	assert.Equal(t, 99, result.PreArmorDamage)
	assert.Equal(t, 99, result.PostArmorDamage)
	assert.Zero(t, rng.index, "glass sword damage must not consume RNG")
}

func TestCombatCalcDamage_JeweledSword(t *testing.T) {
	for _, test := range []struct {
		name          string
		armor         int
		wantArmorRoll int
		wantCalls     int
	}{
		{name: "zero armor", armor: 0, wantArmorRoll: 0, wantCalls: 0},
		{name: "positive armor", armor: 5, wantArmorRoll: 4, wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := newSequenceRNG(3)
			result := CombatCalcDamage(references.JeweledSword, test.armor, rng)

			assert.Zero(t, result.Damage)
			assert.True(t, result.Deflected, "jeweled sword damage should be treated as deflected")
			assert.False(t, result.InstantKill)
			assert.False(t, result.WeaponShattered)
			assert.Equal(t, 1, result.AttackMax)
			assert.Zero(t, result.AttackRoll)
			assert.False(t, result.AttackRolled)
			assert.Equal(t, test.wantArmorRoll, result.ArmorRoll)
			assert.Equal(t, test.wantCalls == 1, result.ArmorRolled)
			assert.Zero(t, result.PreArmorDamage)
			assert.Equal(t, -test.wantArmorRoll, result.PostArmorDamage)
			assert.Equal(t, test.wantCalls, rng.index)
		})
	}
}

func TestCombatCalcDamage_BareHands(t *testing.T) {
	// Bare hands: weaponAttack forced to 1, no randomization.
	// Against 0 armor: damage = 1.
	rng := newSequenceRNG(0)
	result := CombatCalcDamage(references.NoEquipment, 0, rng)
	assert.Equal(t, 1, result.Damage, "bare hands should deal 1 damage against no armor")
	assert.False(t, result.Deflected)
	assert.Equal(t, 1, result.AttackMax)
	assert.Equal(t, 1, result.AttackRoll)
	assert.False(t, result.AttackRolled)
	assert.Zero(t, result.ArmorValue)
	assert.False(t, result.ArmorRolled)
	assert.Equal(t, 1, result.PreArmorDamage)
	assert.Equal(t, 1, result.PostArmorDamage)
	assert.Zero(t, rng.index, "unarmored bare hands must not consume RNG")

	// Against armor 2: damage = 1 - random(1,2).
	// rng.Intn(2)=0, +1=1 → damage = 1 - 1 = 0 → deflected
	rng = newSequenceRNG(0)
	result = CombatCalcDamage(references.NoEquipment, 2, rng)
	assert.Equal(t, 0, result.Damage, "bare hands vs armor should deflect")
	assert.True(t, result.Deflected)
	assert.Equal(t, 1, result.AttackRoll)
	assert.False(t, result.AttackRolled)
	assert.Equal(t, 2, result.ArmorValue)
	assert.Equal(t, 1, result.ArmorRoll)
	assert.True(t, result.ArmorRolled)
	assert.Equal(t, 1, result.PreArmorDamage)
	assert.Zero(t, result.PostArmorDamage)
	assert.Equal(t, 1, rng.index, "armored bare hands must only roll armor")
}

func TestCombatCalcDamage_NormalWeapon(t *testing.T) {
	// Short Sword has attack rating 12 and no defending armor here.
	// The controlled provider returns its highest result, producing maximum
	// damage 12 without armor reduction. ref:EQUIP-0013
	rng := newSequenceRNG(11)
	result := CombatCalcDamage(references.ShortSword, 0, rng)
	assert.Equal(t, 12, result.Damage, "max weapon roll with no armor should give full damage")
	assert.False(t, result.Deflected)
	assert.False(t, result.InstantKill)

	// Min weapon roll: rng.Intn(12)=0, +1=1 → damage = 1
	rng = newSequenceRNG(0)
	result = CombatCalcDamage(references.ShortSword, 0, rng)
	assert.Equal(t, 1, result.Damage, "min weapon roll should give 1 damage")
}

func TestCombatCalcDamage_ArmorReduction(t *testing.T) {
	// Weapon attack 15 (Mace), armor 5.
	// rng calls: first Intn(15)=14 → weap=15, then Intn(5)=0 → armor reduction=1
	// damage = 15 - 1 = 14
	rng := newSequenceRNG(14, 0)
	result := CombatCalcDamage(references.Mace, 5, rng)
	assert.Equal(t, 14, result.Damage)
	assert.Equal(t, 15, result.AttackMax)
	assert.Equal(t, 15, result.AttackRoll)
	assert.True(t, result.AttackRolled)
	assert.Equal(t, 5, result.ArmorValue)
	assert.Equal(t, 1, result.ArmorRoll)
	assert.True(t, result.ArmorRolled)
	assert.Equal(t, 15, result.PreArmorDamage)
	assert.Equal(t, 14, result.PostArmorDamage)
	assert.Equal(t, 2, rng.index, "normal damage must consume one attack and one armor roll")

	// Max armor reduction: Intn(15)=14 → weap=15, Intn(5)=4 → reduction=5
	// damage = 15 - 5 = 10
	rng = newSequenceRNG(14, 4)
	result = CombatCalcDamage(references.Mace, 5, rng)
	assert.Equal(t, 10, result.Damage)
}

func TestCombatCalcDamage_DeflectedByArmor(t *testing.T) {
	// Weapon attack 6 (Dagger), armor 8.
	// Intn(6)=0 → weap=1, then Intn(8)=7 → reduction=8
	// damage = 1 - 8 = -7 → clamped to 0, deflected
	rng := newSequenceRNG(0, 7)
	result := CombatCalcDamage(references.Dagger, 8, rng)
	assert.Equal(t, 0, result.Damage)
	assert.True(t, result.Deflected, "low weapon roll vs heavy armor should deflect")
	assert.Equal(t, 1, result.PreArmorDamage)
	assert.Equal(t, -7, result.PostArmorDamage)
	assert.Equal(t, 2, rng.index)
}

func TestCombatCalcDamage_LowAttackValue(t *testing.T) {
	// Test that weapons with low attack values still get randomized if > 1.
	// Sling has attack 6, so it > 1 and gets randomized.
	// rng.Intn(6)=3, +1=4 → damage = 4
	rng := newSequenceRNG(3)
	result := CombatCalcDamage(references.Sling, 0, rng)
	assert.Equal(t, 4, result.Damage,
		"weapon attack of 6 should be randomized to 4")
	assert.False(t, result.Deflected)
}

func TestCombatCalcDamage_ChaosSwordInstantKill(t *testing.T) {
	// Chaos Sword's attack value 99 bypasses attack and armor RNG.
	rng := newSequenceRNG(0)
	result := CombatCalcDamage(references.ChaosSword, 10, rng)
	assert.Equal(t, 99, result.Damage)
	assert.True(t, result.InstantKill)
	assert.False(t, result.WeaponShattered,
		"chaos sword should NOT shatter")
	assert.Equal(t, 99, result.AttackRoll)
	assert.False(t, result.AttackRolled)
	assert.Zero(t, result.ArmorRoll)
	assert.False(t, result.ArmorRolled)
	assert.Zero(t, rng.index)
}

func TestCombatCalcDamageRaw_UsesFixedMonsterRatingWithoutAttackRoll(t *testing.T) {
	t.Run("unarmored", func(t *testing.T) {
		rng := newSequenceRNG(8)

		result := CombatCalcDamageRaw(10, 0, rng)

		assert.Equal(t, 10, result.AttackMax)
		assert.Equal(t, 10, result.AttackRoll)
		assert.False(t, result.AttackRolled)
		assert.Equal(t, 10, result.PreArmorDamage)
		assert.Equal(t, 10, result.PostArmorDamage)
		assert.Equal(t, 10, result.Damage)
		assert.Zero(t, rng.index, "unarmored monster damage must consume no RNG")
		assert.Equal(t, 8, rng.Intn(10), "the next RNG value must remain untouched")
	})

	t.Run("armored", func(t *testing.T) {
		rng := newSequenceRNG(2, 8)

		result := CombatCalcDamageRaw(10, 4, rng)

		assert.Equal(t, 10, result.AttackRoll)
		assert.False(t, result.AttackRolled)
		assert.Equal(t, 4, result.ArmorValue)
		assert.Equal(t, 3, result.ArmorRoll)
		assert.True(t, result.ArmorRolled)
		assert.Equal(t, 10, result.PreArmorDamage)
		assert.Equal(t, 7, result.PostArmorDamage)
		assert.Equal(t, 7, result.Damage)
		assert.False(t, result.Deflected)
		assert.Equal(t, 1, rng.index, "armor must be the only monster damage RNG draw")
		assert.Equal(t, 8, rng.Intn(10), "the next RNG value must follow the armor draw")
	})
}

func TestCombatCalcDamageRaw_Rating99BypassesArmorAndRequestsInstantKill(t *testing.T) {
	rng := newSequenceRNG(3)

	result := CombatCalcDamageRaw(99, 20, rng)

	assert.Equal(t, 99, result.Damage)
	assert.True(t, result.InstantKill)
	assert.False(t, result.AttackRolled)
	assert.False(t, result.ArmorRolled)
	assert.Zero(t, rng.index)
}

func TestCombatCalcDamageRaw_Final99FromArmorDoesNotRequestInstantKill(t *testing.T) {
	rng := newSequenceRNG(0)

	result := CombatCalcDamageRaw(100, 1, rng)

	assert.Equal(t, 99, result.Damage)
	assert.False(t, result.InstantKill)
	assert.True(t, result.ArmorRolled)
	assert.Equal(t, 1, rng.index)
}

// ─── CombatApplyDamageModifiers ─────────────────────────────────────────────
// Validates undead/immortal parts of damage application from original combat damage rules.
// See ref:EQUIP-0013.

func TestCombatApplyDamageModifiers_Normal(t *testing.T) {
	assert.Equal(t, 10, CombatApplyDamageModifiers(10, false, false, false),
		"normal target should take full damage")
}

func TestCombatApplyDamageModifiers_Undead_NonMagic(t *testing.T) {
	// Undead take half damage from non-magic weapons (integer division).
	assert.Equal(t, 5, CombatApplyDamageModifiers(10, true, false, false))
	assert.Equal(t, 0, CombatApplyDamageModifiers(1, true, false, false),
		"1 damage halved to 0 by integer division")
	assert.Equal(t, 4, CombatApplyDamageModifiers(9, true, false, false),
		"odd damage rounds down")
}

func TestCombatApplyDamageModifiers_Undead_MagicWeapon(t *testing.T) {
	// Magic weapons deal full damage to undead.
	assert.Equal(t, 10, CombatApplyDamageModifiers(10, true, false, true),
		"magic weapon should deal full damage to undead")
}

func TestCombatApplyDamageModifiers_Immortal(t *testing.T) {
	// Immortal targets take 0 damage regardless of weapon type.
	assert.Equal(t, 0, CombatApplyDamageModifiers(10, false, true, false))
	assert.Equal(t, 0, CombatApplyDamageModifiers(10, false, true, true),
		"immortal takes 0 even from magic weapons")
}

func TestCombatApplyDamageModifiers_ImmortalAndUndead(t *testing.T) {
	// If somehow both flags are set, immortal takes priority (0 damage).
	assert.Equal(t, 0, CombatApplyDamageModifiers(10, true, true, false))
}

// combatCalcExperience tests the maximum-HP-derived kill award. ref:PROGRESS-0002

func TestCombatCalcExperience(t *testing.T) {
	tests := []struct {
		name     string
		maxHP    int
		expected int
	}{
		{"low HP monster", 4, 2},     // 4/4 + 1 = 2
		{"orc-level HP", 20, 6},      // 20/4 + 1 = 6
		{"dragon-level HP", 100, 26}, // 100/4 + 1 = 26
		{"minimum HP", 1, 1},         // 1/4 + 1 = 1 (integer division)
		{"zero HP edge case", 0, 1},  // 0/4 + 1 = 1
		{"odd HP rounding", 7, 2},    // 7/4 + 1 = 2
		{"high HP monster", 255, 64}, // 255/4 + 1 = 64 (max u8 HP)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, combatCalcExperience(tt.maxHP))
		})
	}
}

// ─── CombatGetEffectiveDex ──────────────────────────────────────────────────
// Validates effective Dexterity from original combat damage rules.
// See ref:EQUIP-0013.

func TestCombatGetEffectiveDex_Normal(t *testing.T) {
	assert.Equal(t, 20, CombatGetEffectiveDex(20, false),
		"normal combatant returns base dex")
}

func TestCombatGetEffectiveDex_Asleep(t *testing.T) {
	assert.Equal(t, 1, CombatGetEffectiveDex(20, true),
		"asleep combatant returns 1")
}

// ─── IsMagicWeapon ──────────────────────────────────────────────────────────
// Validates which weapons deal full damage to undead.

func TestIsMagicWeapon_MagicWeapons(t *testing.T) {
	magicWeapons := []references.Equipment{
		references.ChaosSword,
		references.MagicBow,
		references.SilverSword,
		references.MagicAxe,
		references.GlassSword,
		references.JeweledSword,
		references.MysticSword,
	}

	for _, w := range magicWeapons {
		assert.True(t, IsMagicWeapon(w), "expected %d to be a magic weapon", w)
	}
}

func TestIsMagicWeapon_NonMagicWeapons(t *testing.T) {
	nonMagic := []references.Equipment{
		references.ShortSword,
		references.Mace,
		references.Club,
		references.Dagger,
		references.NoEquipment,
	}

	for _, w := range nonMagic {
		assert.False(t, IsMagicWeapon(w), "expected %d to NOT be a magic weapon", w)
	}
}

// IsBludgeoning tests the equipment attack-category interpretation. ref:EQUIP-0002

func TestIsBludgeoning_BludgeoningWeapons(t *testing.T) {
	bludgeoning := []references.Equipment{
		references.SpikedHelm,
		references.SpikedShield,
		references.Club,
		references.Mace,
		references.TwoHHammer,
	}

	for _, w := range bludgeoning {
		assert.True(t, w.IsBludgeoning(), "expected %d to be bludgeoning", w)
	}
}

func TestIsBludgeoning_NonBludgeoningWeapons(t *testing.T) {
	nonBludgeoning := []references.Equipment{
		references.ShortSword,
		references.Dagger,
		references.GlassSword,
		references.MysticSword,
		references.MagicAxe,
		references.NoEquipment,
	}

	for _, w := range nonBludgeoning {
		assert.False(t, w.IsBludgeoning(), "expected %d to NOT be bludgeoning", w)
	}
}

// ─── Combined Formula Integration ───────────────────────────────────────────
// Tests that exercise the full hit→damage→modifier pipeline as a unit.

func TestCombatFormulas_FullPipeline_NormalHitWithArmor(t *testing.T) {
	// Scenario: PC with DEX 20 attacks monster with DEX 12, armor 3.
	// Weapon: Short Sword (attack 12), non-magic, non-bludgeoning.
	rng := newSequenceRNG(
		40, // original d30 source value maps to 20; threshold is 11, so the attack hits
		7,  // weapon roll: Intn(12)=7, +1=8
		1,  // armor roll: Intn(3)=1, +1=2
	)

	hitResult := CombatHitCheck(20, 12, false, rng)
	assert.Equal(t, CombatHitNormal, hitResult.Outcome)

	dmgResult := CombatCalcDamage(references.ShortSword, 3, rng)
	assert.Equal(t, 6, dmgResult.Damage, "8 weapon - 2 armor = 6")
	assert.False(t, dmgResult.Deflected)

	finalDamage := CombatApplyDamageModifiers(dmgResult.Damage, false, false, false)
	assert.Equal(t, 6, finalDamage)
}

func TestCombatFormulas_FullPipeline_MissAgainstFastTarget(t *testing.T) {
	// Scenario: Slow PC (DEX 8) attacks fast monster (DEX 28).
	// tohit = (30 + 28 - 8) / 2 = 25. Need roll >= 25.
	rng := newSequenceRNG(
		42, // original d30 source value maps to 21, below the threshold of 25
	)

	hitResult := CombatHitCheck(8, 28, false, rng)
	assert.Equal(t, CombatHitMiss, hitResult.Outcome, "slow attacker should miss fast target")
	assert.Equal(t, 21, hitResult.D30Roll)
	assert.Equal(t, 1, rng.index, "a miss must consume only the hit roll")
}

func TestCombatFormulas_FullPipeline_UndeadWithMagicWeapon(t *testing.T) {
	// Silver Sword (attack 12, magic) vs undead skeleton, armor 2.
	rng := newSequenceRNG(
		50, // original d30 source value maps to 25, so the attack hits
		9,  // weapon roll: Intn(12)=9, +1=10
		0,  // armor roll: Intn(2)=0, +1=1
	)

	hitResult := CombatHitCheck(18, 10, false, rng)
	assert.Equal(t, CombatHitNormal, hitResult.Outcome)

	dmgResult := CombatCalcDamage(references.SilverSword, 2, rng)
	assert.Equal(t, 9, dmgResult.Damage, "10 weapon - 1 armor = 9")

	finalDamage := CombatApplyDamageModifiers(dmgResult.Damage, true, false, true)
	assert.Equal(t, 9, finalDamage, "magic weapon should deal full damage to undead")
}

func TestCombatFormulas_FullPipeline_UndeadWithNormalWeapon(t *testing.T) {
	// Short Sword (non-magic) vs undead, same damage scenario.
	finalDamage := CombatApplyDamageModifiers(9, true, false, false)
	assert.Equal(t, 4, finalDamage, "non-magic weapon should deal half (9/2=4) to undead")
}

func TestCombatFormulas_FullPipeline_GlassSwordKill(t *testing.T) {
	// Glass Sword always hits (never miss) and instant kills.
	rng := newSequenceRNG(0) // doesn't matter

	hitResult := CombatHitCheck(10, 30, true, rng)
	assert.Equal(t, CombatHitNeverMiss, hitResult.Outcome)

	dmgResult := CombatCalcDamage(references.GlassSword, 15, rng)
	assert.Equal(t, 99, dmgResult.Damage)
	assert.True(t, dmgResult.InstantKill)
	assert.True(t, dmgResult.WeaponShattered)

	// XP from a high-HP dragon kill
	xp := combatCalcExperience(100)
	assert.Equal(t, 26, xp)
}
