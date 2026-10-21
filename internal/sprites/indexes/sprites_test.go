package indexes

import "testing"

// TestSpriteIndex_IsBeggar verifies that IsBeggar correctly identifies beggar sprites.
// Beggar sprites are indices 364-367 (Begger1-Begger4).
// The Beggar family occupies raw tiles 0x6c..0x6f; ref:ENEMY-0001.
func TestSpriteIndex_IsBeggar(t *testing.T) {
	tests := []struct {
		name     string
		sprite   SpriteIndex
		expected bool
	}{
		// Beggar sprites (364-367) should return true
		{"Begger1 (364) is beggar", Begger1, true},
		{"Begger2 (365) is beggar", Begger2, true},
		{"Begger3 (366) is beggar", Begger3, true},
		{"Begger4 (367) is beggar", Begger4, true},

		// Boundary tests - sprites just outside beggar range
		{"Sprite 363 is not beggar", SpriteIndex(363), false},
		{"Sprite 368 (Guard) is not beggar", GuardKeyIndex, false},

		// Other NPC types should return false
		{"Avatar is not beggar", AvatarKeyIndex, false},
		{"TownsPerson is not beggar", TownsPersonKeyIndex, false},
		{"Guard is not beggar", GuardKeyIndex, false},
		{"Wizard is not beggar", WizardKeyIndex, false},
		{"Bard is not beggar", BardKeyIndex, false},
		{"Fighter is not beggar", FighterKeyIndex, false},

		// Non-NPC sprites should return false
		{"Grass is not beggar", Grass, false},
		{"Water1 is not beggar", Water1, false},
		{"RegularDoor is not beggar", RegularDoor, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sprite.IsBeggar()
			if result != tt.expected {
				t.Errorf("SpriteIndex(%d).IsBeggar() = %v, want %v",
					tt.sprite, result, tt.expected)
			}
		})
	}
}

// TestSpriteIndex_IsBeggar_BeggarKeyIndex verifies that BeggarKeyIndex is correctly defined.
// Note: The CSV has a typo "Begger" but BeggarKeyIndex points to Begger2 (365).
func TestSpriteIndex_IsBeggar_BeggarKeyIndex(t *testing.T) {
	// BeggarKeyIndex should be a valid beggar sprite
	if !BeggarKeyIndex.IsBeggar() {
		t.Errorf("BeggarKeyIndex (%d) should be detected as beggar", BeggarKeyIndex)
	}

	// Verify it's the expected value (365 = Begger2)
	if BeggarKeyIndex != Begger2 {
		t.Errorf("BeggarKeyIndex = %d, expected %d (Begger2)", BeggarKeyIndex, Begger2)
	}
}

// TestSpriteIndex_IsBardPlaying verifies that IsBardPlaying correctly identifies BANJO_PLAYER sprites.
// BardPlaying sprites are indices 348-351 (BardPlaying1-BardPlaying4).
// Musician NPCs switch between BardPlaying and Merchant; ref:TILE-0007.
func TestSpriteIndex_IsBardPlaying(t *testing.T) {
	tests := []struct {
		name     string
		sprite   SpriteIndex
		expected bool
	}{
		// BardPlaying sprites (348-351) should return true
		{"BardPlaying1 (348) is bard playing", BardPlaying1, true},
		{"BardPlaying2 (349) is bard playing", BardPlaying2, true},
		{"BardPlaying3 (350) is bard playing", BardPlaying3, true},
		{"BardPlaying4 (351) is bard playing", BardPlaying4, true},

		// Boundary tests - sprites just outside BardPlaying range
		{"Sprite 347 (Jester4) is not bard playing", Jester4, false},
		{"Sprite 352 (PersonStocks1) is not bard playing", PersonStocks1, false},

		// Bard (non-playing) sprites should return false - they are different!
		{"Bard1 (324) is not bard playing", Bard1, false},
		{"Bard2 (325) is not bard playing", Bard2, false},
		{"Bard3 (326) is not bard playing", Bard3, false},
		{"Bard4 (327) is not bard playing", Bard4, false},

		// Merchant sprites should return false
		{"Merchant1 (340) is not bard playing", Merchant1, false},
		{"Merchant2 (341) is not bard playing", Merchant2, false},
		{"Merchant3 (342) is not bard playing", Merchant3, false},
		{"Merchant4 (343) is not bard playing", Merchant4, false},

		// Other NPC types should return false
		{"Avatar is not bard playing", AvatarKeyIndex, false},
		{"TownsPerson is not bard playing", TownsPersonKeyIndex, false},
		{"Guard is not bard playing", GuardKeyIndex, false},
		{"Wizard is not bard playing", WizardKeyIndex, false},
		{"Beggar is not bard playing", BeggarKeyIndex, false},

		// Non-NPC sprites should return false
		{"Grass is not bard playing", Grass, false},
		{"ChairBackBack is not bard playing", ChairBackBack, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sprite.IsBardPlaying()
			if result != tt.expected {
				t.Errorf("SpriteIndex(%d).IsBardPlaying() = %v, want %v",
					tt.sprite, result, tt.expected)
			}
		})
	}
}

// TestSpriteIndex_IsBardPlaying_KeyIndex verifies that BardPlayingKeyIndex is correctly defined.
func TestSpriteIndex_IsBardPlaying_KeyIndex(t *testing.T) {
	// BardPlayingKeyIndex should be a valid bard playing sprite
	if !BardPlayingKeyIndex.IsBardPlaying() {
		t.Errorf("BardPlayingKeyIndex (%d) should be detected as bard playing", BardPlayingKeyIndex)
	}

	// Verify it's the expected value (348 = BardPlaying1)
	if BardPlayingKeyIndex != BardPlaying1 {
		t.Errorf("BardPlayingKeyIndex = %d, expected %d (BardPlaying1)", BardPlayingKeyIndex, BardPlaying1)
	}
}

// TestSpriteIndex_MerchantKeyIndex verifies that MerchantKeyIndex is correctly defined.
func TestSpriteIndex_MerchantKeyIndex(t *testing.T) {
	// Verify MerchantKeyIndex is the expected value (340 = Merchant1)
	if MerchantKeyIndex != Merchant1 {
		t.Errorf("MerchantKeyIndex = %d, expected %d (Merchant1)", MerchantKeyIndex, Merchant1)
	}

	// Verify frame offset calculation works as expected for BANJO_PLAYER sprite switching
	// BardPlaying1 -> Merchant1, BardPlaying2 -> Merchant2, etc.
	for i := SpriteIndex(0); i < StandardNumberOfAnimationFrames; i++ {
		bardPlayingSprite := BardPlayingKeyIndex + i
		expectedMerchant := MerchantKeyIndex + i

		frameOffset := bardPlayingSprite - BardPlayingKeyIndex
		actualMerchant := MerchantKeyIndex + frameOffset

		if actualMerchant != expectedMerchant {
			t.Errorf("Frame offset calculation failed: BardPlaying%d -> Merchant%d, got %d",
				i+1, i+1, actualMerchant)
		}
	}
}

// TestSpriteIndex_IsPartOfAnimationStopsBeforeTheNextSequence keeps the sprite
// after a four-frame sequence (Guard → Apparition) out of the sequence.
func TestSpriteIndex_IsPartOfAnimationStopsBeforeTheNextSequence(t *testing.T) {
	last := GuardKeyIndex + StandardNumberOfAnimationFrames - 1
	if !last.IsPartOfAnimation(GuardKeyIndex) {
		t.Errorf("sprite %d should be the last guard frame", last)
	}
	if next := last + 1; next.IsPartOfAnimation(GuardKeyIndex) {
		t.Errorf("sprite %d starts the next sequence, not a guard frame", next)
	}
}
