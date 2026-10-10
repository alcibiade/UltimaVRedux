package references

// Friends system helpers for monster encounter generation.
// Eligible early group members may be replaced by companion candidates; ref:ENEMY-0003.

// ShouldUseFriend determines if a friend should replace the primary monster.
// Companion selection follows ref:ENEMY-0003:
//   - Only for first quarter of monsters (index < numMonsters/4 + 1)
//   - An inclusive 0..8 draw of zero selects the companion (1/9 under uniform draws)
//
// Returns true if a friend type should be used instead of primary.
//
//nolint:mnd // Original companion threshold and inclusive 0..8 range; ref:ENEMY-0003.
func ShouldUseFriend(monsterIndex int, totalMonsters int, rng RandomProvider) bool {
	// Only first quarter of monsters can have friends
	threshold := totalMonsters/4 + 1
	if monsterIndex >= threshold {
		return false
	}

	// The companion draw uses the inclusive range 0..8; ref:ENEMY-0003.
	return rng.Intn(9) == 0
}

// GetFriendType returns the friend monster type for a given monster.
// The friends data is loaded from DATA.OVL and stored in EnemyReference.Friend.
//
// Returns the friend's MonsterType, or the original monster type if no friend exists.
func (e *EnemyReference) GetFriendType() MonsterType {
	if e.Friend == nil {
		return e.MonsterType
	}

	return e.Friend.MonsterType
}

// GetFriendType returns the friend monster type for a given monster by MonsterType.
// This is a convenience method for use with EnemyReferences slice.
//
// Returns the friend's MonsterType, or MonsterNone if the index is invalid.
func (refs EnemyReferences) GetFriendType(monsterType MonsterType) MonsterType {
	if !monsterType.IsValid() {
		return MonsterNone
	}

	if int(monsterType) >= len(refs) {
		return MonsterNone
	}

	enemyRef := refs[monsterType]
	if enemyRef.Friend == nil {
		return monsterType
	}

	return enemyRef.Friend.MonsterType
}

// FriendReplacement represents the result of a friend replacement check.
type FriendReplacement struct {
	// Original monster type
	Original MonsterType
	// Replacement monster type (may be same as Original if no replacement)
	Replacement MonsterType
	// Whether a friend replacement occurred
	WasReplaced bool
}

// CheckFriendReplacement performs the full friend replacement logic.
// Returns a FriendReplacement with the result.
func CheckFriendReplacement(
	primaryMonster MonsterType,
	monsterIndex int,
	totalMonsters int,
	refs EnemyReferences,
	rng RandomProvider,
) FriendReplacement {
	result := FriendReplacement{
		Original:    primaryMonster,
		Replacement: primaryMonster,
		WasReplaced: false,
	}

	if !ShouldUseFriend(monsterIndex, totalMonsters, rng) {
		return result
	}

	friendType := refs.GetFriendType(primaryMonster)
	if friendType != primaryMonster && friendType.IsValid() {
		result.Replacement = friendType
		result.WasReplaced = true
	}

	return result
}
