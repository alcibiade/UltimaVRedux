package references

import "testing"

func TestPositiveRangeWeaponProperties(t *testing.T) {
	tests := []struct {
		weapon       Equipment
		rangeTiles   int
		missile      MissileType
		ammo         Equipment
		interference bool
		recoverable  bool
		returns      bool
		bypasses     bool
		consumed     bool
	}{
		{Dagger, 3, MissileArrow, NoEquipment, false, true, false, false, false},
		{Sling, 4, MissileRock, NoEquipment, true, false, false, false, false},
		{FlamingOil, 4, MissileRed, NoEquipment, true, false, false, false, true},
		{Spear, 5, MissileArrow, NoEquipment, false, true, false, false, false},
		{ThrowingAxe, 4, MissileAxe, NoEquipment, false, true, false, false, false},
		{MorningStar, 2, MissileArrow, NoEquipment, false, false, false, true, false},
		{Bow, 7, MissileArrow, Arrows, true, false, false, false, false},
		{Crossbow, 8, MissileArrow, Quarrels, true, false, false, false, false},
		{Halberd, 2, MissileArrow, NoEquipment, false, false, false, true, false},
		{MagicBow, 15, MissileArrow, Arrows, true, false, false, false, false},
		{MagicAxe, 15, MissileAxe, NoEquipment, false, false, true, false, false},
	}

	tested := make(map[Equipment]bool, len(tests))
	for _, tc := range tests {
		t.Run(tc.weapon.String(), func(t *testing.T) {
			tested[tc.weapon] = true
			if got := tc.weapon.GetAttackRange(); got != tc.rangeTiles {
				t.Errorf("GetAttackRange() = %d, want %d", got, tc.rangeTiles)
			}
			if got := tc.weapon.GetMissileType(); got != tc.missile {
				t.Errorf("GetMissileType() = %d, want %d", got, tc.missile)
			}
			if got := tc.weapon.GetRequiredAmmo(); got != tc.ammo {
				t.Errorf("GetRequiredAmmo() = %v, want %v", got, tc.ammo)
			}
			if got := tc.weapon.IsSubjectToCombatInterference(); got != tc.interference {
				t.Errorf("IsSubjectToCombatInterference() = %t, want %t", got, tc.interference)
			}
			if got := tc.weapon.IsRecoverableThrownWeapon(); got != tc.recoverable {
				t.Errorf("IsRecoverableThrownWeapon() = %t, want %t", got, tc.recoverable)
			}
			if got := tc.weapon.HasAbility(AbilityReturns); got != tc.returns {
				t.Errorf("HasAbility(AbilityReturns) = %t, want %t", got, tc.returns)
			}
			if got := tc.weapon.BypassesTerrainOnSuccessfulHit(); got != tc.bypasses {
				t.Errorf("BypassesTerrainOnSuccessfulHit() = %t, want %t", got, tc.bypasses)
			}
			if got := tc.weapon.IsConsumedOnRangedAttack(); got != tc.consumed {
				t.Errorf("IsConsumedOnRangedAttack() = %t, want %t", got, tc.consumed)
			}
		})
	}

	for equipment, definition := range equipmentDefinitions {
		if definition.Range > 0 && !tested[equipment] {
			t.Errorf("positive-range weapon %v is missing from exhaustive property table", equipment)
		}
	}
}
