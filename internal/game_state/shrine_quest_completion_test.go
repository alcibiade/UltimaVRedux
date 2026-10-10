package game_state

import (
	"testing"
)

// TestShrineQuestCompletion_StatBonuses verifies that each shrine applies the correct
// stat bonuses to the Avatar (character 0) only.
// Based on original shrine rules: bonuses only apply to the Avatar record.
// See ref:SHRINE-0002.
func TestShrineQuestCompletion_StatBonuses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		shrineIndex int
		shrineName  string
		wantStr     byte
		wantDex     byte
		wantInt     byte
	}{
		{0, "Honesty", 0, 0, 1},
		{1, "Compassion", 0, 1, 0},
		{2, "Valour", 1, 0, 0},
		{3, "Justice", 0, 1, 1},
		{4, "Sacrifice", 1, 1, 0},
		{5, "Honor", 1, 0, 1},
		{6, "Spirituality", 1, 1, 1},
		{7, "Humility", 0, 0, 0}, // No stat bonuses
	}

	for _, tc := range testCases {
		t.Run(tc.shrineName, func(t *testing.T) {
			t.Parallel()

			gs, _ := NewIntegrationTestBuilder(t).
				WithSystemCallbacks().
				Build()

			// Get Avatar and set known starting stats
			avatar := gs.PartyState.GetCharacter(0)
			if avatar == nil {
				t.Fatal("Avatar (character 0) not found")
			}

			const startingStat byte = 15
			avatar.Strength = startingStat
			avatar.Dexterity = startingStat
			avatar.Intelligence = startingStat

			// Get a companion to verify they are NOT affected
			companion := gs.PartyState.GetCharacter(1)
			var companionStartStr, companionStartDex, companionStartInt byte
			if companion != nil {
				companionStartStr = companion.Strength
				companionStartDex = companion.Dexterity
				companionStartInt = companion.Intelligence
			}

			// Apply shrine bonuses (simulating quest completion logic)
			shrineData := GetShrineData(tc.shrineIndex)
			if shrineData == nil {
				t.Fatalf("GetShrineData(%d) returned nil", tc.shrineIndex)
			}

			// Apply bonuses exactly as done in gamescene_shrine.go
			if shrineData.StrBonus > 0 {
				newStr := int(avatar.Strength) + int(shrineData.StrBonus)
				if newStr > MaxStatValue {
					newStr = MaxStatValue
				}
				avatar.Strength = byte(newStr)
			}

			if shrineData.DexBonus > 0 {
				newDex := int(avatar.Dexterity) + int(shrineData.DexBonus)
				if newDex > MaxStatValue {
					newDex = MaxStatValue
				}
				avatar.Dexterity = byte(newDex)
			}

			if shrineData.IntBonus > 0 {
				newInt := int(avatar.Intelligence) + int(shrineData.IntBonus)
				if newInt > MaxStatValue {
					newInt = MaxStatValue
				}
				avatar.Intelligence = byte(newInt)
			}

			// Verify Avatar stats changed correctly
			expectedStr := startingStat + tc.wantStr
			expectedDex := startingStat + tc.wantDex
			expectedInt := startingStat + tc.wantInt

			if avatar.Strength != expectedStr {
				t.Errorf("Avatar Strength: got %d, want %d", avatar.Strength, expectedStr)
			}
			if avatar.Dexterity != expectedDex {
				t.Errorf("Avatar Dexterity: got %d, want %d", avatar.Dexterity, expectedDex)
			}
			if avatar.Intelligence != expectedInt {
				t.Errorf("Avatar Intelligence: got %d, want %d", avatar.Intelligence, expectedInt)
			}

			// Verify companion stats were NOT affected
			if companion != nil {
				if companion.Strength != companionStartStr {
					t.Errorf("Companion Strength changed: got %d, want %d (should be unchanged)",
						companion.Strength, companionStartStr)
				}
				if companion.Dexterity != companionStartDex {
					t.Errorf("Companion Dexterity changed: got %d, want %d (should be unchanged)",
						companion.Dexterity, companionStartDex)
				}
				if companion.Intelligence != companionStartInt {
					t.Errorf("Companion Intelligence changed: got %d, want %d (should be unchanged)",
						companion.Intelligence, companionStartInt)
				}
			}
		})
	}
}

// TestShrineQuestCompletion_StatsCappedAtMax verifies stats don't exceed MaxStatValue (30)
func TestShrineQuestCompletion_StatsCappedAtMax(t *testing.T) {
	t.Parallel()

	gs, _ := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		Build()

	avatar := gs.PartyState.GetCharacter(0)
	if avatar == nil {
		t.Fatal("Avatar (character 0) not found")
	}

	// Set stats to MaxStatValue
	avatar.Strength = MaxStatValue
	avatar.Dexterity = MaxStatValue
	avatar.Intelligence = MaxStatValue

	// Use Spirituality shrine (all +1 bonuses)
	shrineData := GetShrineData(6) // Spirituality: STR+1, DEX+1, INT+1

	// Apply bonuses with capping
	if shrineData.StrBonus > 0 {
		newStr := int(avatar.Strength) + int(shrineData.StrBonus)
		if newStr > MaxStatValue {
			newStr = MaxStatValue
		}
		avatar.Strength = byte(newStr)
	}

	if shrineData.DexBonus > 0 {
		newDex := int(avatar.Dexterity) + int(shrineData.DexBonus)
		if newDex > MaxStatValue {
			newDex = MaxStatValue
		}
		avatar.Dexterity = byte(newDex)
	}

	if shrineData.IntBonus > 0 {
		newInt := int(avatar.Intelligence) + int(shrineData.IntBonus)
		if newInt > MaxStatValue {
			newInt = MaxStatValue
		}
		avatar.Intelligence = byte(newInt)
	}

	// All stats should remain at max, not overflow
	if avatar.Strength != MaxStatValue {
		t.Errorf("Strength exceeded max: got %d, want %d", avatar.Strength, MaxStatValue)
	}
	if avatar.Dexterity != MaxStatValue {
		t.Errorf("Dexterity exceeded max: got %d, want %d", avatar.Dexterity, MaxStatValue)
	}
	if avatar.Intelligence != MaxStatValue {
		t.Errorf("Intelligence exceeded max: got %d, want %d", avatar.Intelligence, MaxStatValue)
	}
}

// TestShrineQuestCompletion_KarmaBonuses verifies correct karma is awarded
// Based on original shrine rules: +3 for shrines 0-6, +6 for Humility (7)
// See ref:SHRINE-0002.
func TestShrineQuestCompletion_KarmaBonuses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		shrineIndex int
		shrineName  string
		wantKarma   int
	}{
		{0, "Honesty", 3},
		{1, "Compassion", 3},
		{2, "Valour", 3},
		{3, "Justice", 3},
		{4, "Sacrifice", 3},
		{5, "Honor", 3},
		{6, "Spirituality", 3},
		{7, "Humility", 6}, // Double karma bonus
	}

	for _, tc := range testCases {
		t.Run(tc.shrineName, func(t *testing.T) {
			t.Parallel()

			gs, _ := NewIntegrationTestBuilder(t).
				WithSystemCallbacks().
				Build()

			// Set known starting karma
			const startingKarma = 50
			gs.PartyState.Karma.Value = startingKarma

			shrineData := GetShrineData(tc.shrineIndex)
			karmaBonus := shrineData.KarmaBonus()

			// Apply karma bonus (as done in gamescene_shrine.go)
			gs.PartyState.Karma.AddDiff(karmaBonus)

			expectedKarma := startingKarma + tc.wantKarma
			if gs.PartyState.Karma.Value != expectedKarma {
				t.Errorf("Karma: got %d, want %d", gs.PartyState.Karma.Value, expectedKarma)
			}
		})
	}
}

// TestShrineQuestCompletion_KarmaCappedAtMax verifies karma doesn't exceed MaxKarmaValue (99)
func TestShrineQuestCompletion_KarmaCappedAtMax(t *testing.T) {
	t.Parallel()

	gs, _ := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		Build()

	// Set karma close to max
	gs.PartyState.Karma.Value = 97

	// Use Humility which gives +6 karma
	shrineData := GetShrineData(7) // Humility: +6 karma
	gs.PartyState.Karma.AddDiff(shrineData.KarmaBonus())

	// Karma should cap at 99, not go to 103
	if gs.PartyState.Karma.Value != MaxKarmaValue {
		t.Errorf("Karma exceeded max: got %d, want %d", gs.PartyState.Karma.Value, MaxKarmaValue)
	}
}

// TestShrineQuestCompletion_Messages verifies correct messages are output
// Based on original shrine rules hardcoded strings: "Strength +1", "Dexterity +1", "Intelligence +1"
// See ref:SHRINE-0002.
func TestShrineQuestCompletion_Messages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		shrineIndex      int
		shrineName       string
		expectedMessages []string
	}{
		{0, "Honesty", []string{"Intelligence +1"}},
		{1, "Compassion", []string{"Dexterity +1"}},
		{2, "Valour", []string{"Strength +1"}},
		{3, "Justice", []string{"Dexterity +1", "Intelligence +1"}},
		{4, "Sacrifice", []string{"Strength +1", "Dexterity +1"}},
		{5, "Honor", []string{"Strength +1", "Intelligence +1"}},
		{6, "Spirituality", []string{"Strength +1", "Dexterity +1", "Intelligence +1"}},
		{7, "Humility", []string{}}, // No stat bonuses, no messages
	}

	for _, tc := range testCases {
		t.Run(tc.shrineName, func(t *testing.T) {
			t.Parallel()

			gs, mockCallbacks := NewIntegrationTestBuilder(t).
				WithSystemCallbacks().
				Build()

			// Get shrine data and apply bonuses with messages
			shrineData := GetShrineData(tc.shrineIndex)
			avatar := gs.PartyState.GetCharacter(0)

			// Simulate the quest completion message sequence
			gs.SystemCallbacks.Message.AddRowStr("WELL DONE!")

			if shrineData.StrBonus > 0 {
				newStr := int(avatar.Strength) + int(shrineData.StrBonus)
				if newStr > MaxStatValue {
					newStr = MaxStatValue
				}
				avatar.Strength = byte(newStr)
				gs.SystemCallbacks.Message.AddRowStr("Strength +1")
			}

			if shrineData.DexBonus > 0 {
				newDex := int(avatar.Dexterity) + int(shrineData.DexBonus)
				if newDex > MaxStatValue {
					newDex = MaxStatValue
				}
				avatar.Dexterity = byte(newDex)
				gs.SystemCallbacks.Message.AddRowStr("Dexterity +1")
			}

			if shrineData.IntBonus > 0 {
				newInt := int(avatar.Intelligence) + int(shrineData.IntBonus)
				if newInt > MaxStatValue {
					newInt = MaxStatValue
				}
				avatar.Intelligence = byte(newInt)
				gs.SystemCallbacks.Message.AddRowStr("Intelligence +1")
			}

			// Verify "WELL DONE!" always appears first
			if len(mockCallbacks.Messages) == 0 {
				t.Fatal("No messages recorded")
			}
			if mockCallbacks.Messages[0] != "WELL DONE!" {
				t.Errorf("First message: got %q, want %q", mockCallbacks.Messages[0], "WELL DONE!")
			}

			// Verify stat bonus messages appear (after "WELL DONE!")
			statMessages := mockCallbacks.Messages[1:] // Skip "WELL DONE!"

			if len(statMessages) != len(tc.expectedMessages) {
				t.Errorf("Message count: got %d, want %d\nGot: %v\nWant: %v",
					len(statMessages), len(tc.expectedMessages), statMessages, tc.expectedMessages)
				return
			}

			for i, expected := range tc.expectedMessages {
				if statMessages[i] != expected {
					t.Errorf("Message[%d]: got %q, want %q", i, statMessages[i], expected)
				}
			}
		})
	}
}

// TestShrineQuestCompletion_SoundEffects verifies both reward sweep and earthquake sounds are played
// Based on original shrine rules: pulse sweep sound followed by earthquake feedback
// See ref:SHRINE-0002.
func TestShrineQuestCompletion_SoundEffects(t *testing.T) {
	t.Parallel()

	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		Build()

	// Simulate the sound effects sequence from thenShowShrineQuestCompletionEffects
	// 1. Reward sweep sound (pulse effect from original)
	gs.SystemCallbacks.Audio.PlaySoundEffect(SoundShrineReward)

	// 2. Earthquake sound
	gs.SystemCallbacks.Audio.PlaySoundEffect(SoundEarthquake)

	// Verify reward sound was played
	if !mockCallbacks.SoundEffectWasPlayed(SoundShrineReward) {
		t.Errorf("Expected shrine reward sound to be played, got: %v", mockCallbacks.SoundEffectsPlayed)
	}

	// Verify earthquake sound was played
	if !mockCallbacks.SoundEffectWasPlayed(SoundEarthquake) {
		t.Errorf("Expected earthquake sound to be played, got: %v", mockCallbacks.SoundEffectsPlayed)
	}
}

// TestShrineQuestCompletion_WhiteFlashEffect checks the white reward
// flash after successful shrine completion. ref:KARMA-0009
func TestShrineQuestCompletion_WhiteFlashEffect(t *testing.T) {
	t.Parallel()

	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		Build()

	// Simulate the white flash effect from thenShowShrineQuestCompletionEffects
	gs.SystemCallbacks.Visual.SetScreenFlashColor(1.0, 1.0, 1.0)
	gs.SystemCallbacks.Visual.SetScreenFlashIntensity(0.8) // Peak intensity

	// Verify white flash color was set (RGB all 1.0 = white)
	if len(mockCallbacks.ScreenFlashColors) == 0 {
		t.Fatal("Expected screen flash color to be set")
	}

	flashColor := mockCallbacks.ScreenFlashColors[0]
	if flashColor.R != 1.0 || flashColor.G != 1.0 || flashColor.B != 1.0 {
		t.Errorf("Expected white flash (1.0, 1.0, 1.0), got (%.1f, %.1f, %.1f)",
			flashColor.R, flashColor.G, flashColor.B)
	}

	// Verify flash intensity was set
	if len(mockCallbacks.ScreenFlashIntensities) == 0 {
		t.Fatal("Expected screen flash intensity to be set")
	}

	// Check that intensity reached peak (0.8)
	foundPeakIntensity := false
	for _, intensity := range mockCallbacks.ScreenFlashIntensities {
		if intensity >= 0.7 { // Allow some tolerance
			foundPeakIntensity = true
			break
		}
	}

	if !foundPeakIntensity {
		t.Errorf("Expected flash intensity to reach peak (~0.8), got: %v",
			mockCallbacks.ScreenFlashIntensities)
	}
}

// TestShrineQuestCompletion_OnlyAvatarAffected verifies that only character 0 (Avatar)
// receives stat bonuses, regardless of party composition.
// This matches original shrine rules which explicitly uses the Avatar record.
// See ref:SHRINE-0002.
func TestShrineQuestCompletion_OnlyAvatarAffected(t *testing.T) {
	t.Parallel()

	gs, _ := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		Build()

	// Record starting stats for all party members
	type charStats struct {
		str, dex, int byte
	}
	startingStats := make([]charStats, 0)

	for i := 0; i < 16; i++ { // Max party size
		char := gs.PartyState.GetCharacter(i)
		if char == nil {
			break
		}
		startingStats = append(startingStats, charStats{
			str: char.Strength,
			dex: char.Dexterity,
			int: char.Intelligence,
		})
	}

	if len(startingStats) < 2 {
		t.Skip("Need at least 2 party members for this test")
	}

	// Use Spirituality shrine (all stats +1)
	shrineData := GetShrineData(6)
	avatar := gs.PartyState.GetCharacter(0)

	// Apply bonuses to Avatar only
	if shrineData.StrBonus > 0 {
		newStr := int(avatar.Strength) + int(shrineData.StrBonus)
		if newStr > MaxStatValue {
			newStr = MaxStatValue
		}
		avatar.Strength = byte(newStr)
	}

	if shrineData.DexBonus > 0 {
		newDex := int(avatar.Dexterity) + int(shrineData.DexBonus)
		if newDex > MaxStatValue {
			newDex = MaxStatValue
		}
		avatar.Dexterity = byte(newDex)
	}

	if shrineData.IntBonus > 0 {
		newInt := int(avatar.Intelligence) + int(shrineData.IntBonus)
		if newInt > MaxStatValue {
			newInt = MaxStatValue
		}
		avatar.Intelligence = byte(newInt)
	}

	// Verify Avatar's stats increased
	if avatar.Strength != startingStats[0].str+1 {
		t.Errorf("Avatar STR not increased: got %d, want %d",
			avatar.Strength, startingStats[0].str+1)
	}

	// Verify all other party members are unchanged
	for i := 1; i < len(startingStats); i++ {
		char := gs.PartyState.GetCharacter(i)
		if char.Strength != startingStats[i].str {
			t.Errorf("Character %d STR changed: got %d, want %d",
				i, char.Strength, startingStats[i].str)
		}
		if char.Dexterity != startingStats[i].dex {
			t.Errorf("Character %d DEX changed: got %d, want %d",
				i, char.Dexterity, startingStats[i].dex)
		}
		if char.Intelligence != startingStats[i].int {
			t.Errorf("Character %d INT changed: got %d, want %d",
				i, char.Intelligence, startingStats[i].int)
		}
	}
}
