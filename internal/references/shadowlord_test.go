package references

import (
	"testing"
)

func TestShadowlord_String(t *testing.T) {
	tests := []struct {
		shadowlord Shadowlord
		expected   string
	}{
		{Faulinei, "Faulinei"},
		{Astaroth, "Astaroth"},
		{Nosfentor, "Nosfentor"},
		{Shadowlord(-1), "Unknown"},
		{Shadowlord(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.shadowlord.String(); got != tt.expected {
				t.Errorf("Shadowlord.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShadowlord_GetAntiVirtue(t *testing.T) {
	tests := []struct {
		shadowlord Shadowlord
		expected   string
	}{
		{Faulinei, "Falsehood"},
		{Astaroth, "Hatred"},
		{Nosfentor, "Cowardice"},
		{Shadowlord(-1), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.shadowlord.String(), func(t *testing.T) {
			if got := tt.shadowlord.GetAntiVirtue(); got != tt.expected {
				t.Errorf("Shadowlord.GetAntiVirtue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShadowlord_ToQuestItem(t *testing.T) {
	tests := []struct {
		shadowlord Shadowlord
		expected   QuestItem
	}{
		{Faulinei, ShardOfFalsehood},
		{Astaroth, ShardOfHatred},
		{Nosfentor, ShardOfCowardice},
	}

	for _, tt := range tests {
		t.Run(tt.shadowlord.String(), func(t *testing.T) {
			if got := tt.shadowlord.ToQuestItem(); got != tt.expected {
				t.Errorf("Shadowlord.ToQuestItem() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShadowlordFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected Shadowlord
	}{
		{"FAULINEI", Faulinei},
		{"faulinei", Faulinei},
		{"Faulinei", Faulinei},
		{"the evil FAULINEI", Faulinei},
		{"ASTAROTH", Astaroth},
		{"astaroth", Astaroth},
		{"NOSFENTOR", Nosfentor},
		{"nosfentor", Nosfentor},
		{"unknown", Shadowlord(-1)},
		{"", Shadowlord(-1)},
		{"FAUL", Shadowlord(-1)}, // Partial match should not work
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShadowlordFromName(tt.name); got != tt.expected {
				t.Errorf("ShadowlordFromName(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestNewShadowlordState(t *testing.T) {
	state := NewShadowlordState()

	// All Shadowlords should be alive
	for i := 0; i < ShadowlordCount; i++ {
		if !state.Alive[i] {
			t.Errorf("NewShadowlordState() Shadowlord %d should be alive", i)
		}
	}

	// LastCalled should be -1 (none)
	if state.LastCalled != -1 {
		t.Errorf("NewShadowlordState() LastCalled = %v, want -1", state.LastCalled)
	}

	// Summoned should be false
	if state.Summoned {
		t.Error("NewShadowlordState() Summoned should be false")
	}
}

func TestShadowlordState_IsAlive(t *testing.T) {
	state := NewShadowlordState()

	// All should be alive initially
	if !state.IsAlive(Faulinei) {
		t.Error("Faulinei should be alive")
	}
	if !state.IsAlive(Astaroth) {
		t.Error("Astaroth should be alive")
	}
	if !state.IsAlive(Nosfentor) {
		t.Error("Nosfentor should be alive")
	}

	// Invalid Shadowlord should return false
	if state.IsAlive(Shadowlord(-1)) {
		t.Error("Invalid Shadowlord (-1) should return false")
	}
	if state.IsAlive(Shadowlord(99)) {
		t.Error("Invalid Shadowlord (99) should return false")
	}
}

func TestShadowlordState_Kill(t *testing.T) {
	state := NewShadowlordState()

	// Kill Faulinei
	state.Kill(Faulinei)
	if state.IsAlive(Faulinei) {
		t.Error("Faulinei should be dead after Kill()")
	}
	if !state.IsAlive(Astaroth) {
		t.Error("Astaroth should still be alive")
	}
	if !state.IsAlive(Nosfentor) {
		t.Error("Nosfentor should still be alive")
	}

	// Kill invalid Shadowlord should not panic
	state.Kill(Shadowlord(-1))
	state.Kill(Shadowlord(99))
}

func TestShadowlordState_Kill_ClearsSummonedState(t *testing.T) {
	state := NewShadowlordState()

	// Summon Faulinei
	state.Summon(Faulinei)
	if !state.Summoned {
		t.Error("Shadowlord should be summoned")
	}
	if state.LastCalled != Faulinei {
		t.Error("LastCalled should be Faulinei")
	}

	// Kill Faulinei - should clear summoned state
	state.Kill(Faulinei)
	if state.Summoned {
		t.Error("Summoned should be cleared after killing summoned Shadowlord")
	}
	if state.LastCalled != -1 {
		t.Errorf("LastCalled should be -1 after killing summoned Shadowlord, got %v", state.LastCalled)
	}
}

func TestShadowlordState_Summon(t *testing.T) {
	state := NewShadowlordState()

	// Summon Faulinei
	if !state.Summon(Faulinei) {
		t.Error("Summon(Faulinei) should return true")
	}
	if !state.Summoned {
		t.Error("Summoned should be true")
	}
	if state.LastCalled != Faulinei {
		t.Errorf("LastCalled = %v, want Faulinei", state.LastCalled)
	}

	// Summon another while one is summoned should fail
	if state.Summon(Astaroth) {
		t.Error("Summon(Astaroth) should return false when another is summoned")
	}
	if state.LastCalled != Faulinei {
		t.Error("LastCalled should still be Faulinei")
	}

	// Clear and try again
	state.ClearSummoned()
	if !state.Summon(Astaroth) {
		t.Error("Summon(Astaroth) should succeed after clearing")
	}
}

func TestShadowlordState_Summon_DeadShadowlord(t *testing.T) {
	state := NewShadowlordState()

	// Kill Faulinei
	state.Kill(Faulinei)

	// Try to summon dead Shadowlord
	if state.Summon(Faulinei) {
		t.Error("Summon(Faulinei) should return false when dead")
	}
	if state.Summoned {
		t.Error("Summoned should still be false")
	}
}

func TestShadowlordState_Summon_InvalidShadowlord(t *testing.T) {
	state := NewShadowlordState()

	if state.Summon(Shadowlord(-1)) {
		t.Error("Summon(-1) should return false")
	}
	if state.Summon(Shadowlord(99)) {
		t.Error("Summon(99) should return false")
	}
}

func TestShadowlordState_ClearSummoned(t *testing.T) {
	state := NewShadowlordState()
	state.Summon(Faulinei)

	state.ClearSummoned()

	if state.Summoned {
		t.Error("Summoned should be false after ClearSummoned()")
	}
	if state.LastCalled != -1 {
		t.Errorf("LastCalled = %v, want -1", state.LastCalled)
	}
}

func TestShadowlordState_Reset(t *testing.T) {
	state := NewShadowlordState()

	// Kill all and summon one
	state.Kill(Faulinei)
	state.Kill(Astaroth)
	state.Summon(Nosfentor) // This will fail since Nosfentor is alive but let's kill it first
	state.Kill(Nosfentor)

	// Reset
	state.Reset()

	// All should be alive
	for i := 0; i < ShadowlordCount; i++ {
		if !state.Alive[i] {
			t.Errorf("After Reset(), Shadowlord %d should be alive", i)
		}
	}
	if state.Summoned {
		t.Error("After Reset(), Summoned should be false")
	}
	if state.LastCalled != -1 {
		t.Errorf("After Reset(), LastCalled = %v, want -1", state.LastCalled)
	}
}

func TestShadowlordState_AllDefeated(t *testing.T) {
	state := NewShadowlordState()

	// Initially not all defeated
	if state.AllDefeated() {
		t.Error("AllDefeated() should be false initially")
	}

	// Kill one
	state.Kill(Faulinei)
	if state.AllDefeated() {
		t.Error("AllDefeated() should be false with 2 alive")
	}

	// Kill second
	state.Kill(Astaroth)
	if state.AllDefeated() {
		t.Error("AllDefeated() should be false with 1 alive")
	}

	// Kill last
	state.Kill(Nosfentor)
	if !state.AllDefeated() {
		t.Error("AllDefeated() should be true when all are dead")
	}
}

func TestShadowlordState_CountAlive(t *testing.T) {
	state := NewShadowlordState()

	if state.CountAlive() != 3 {
		t.Errorf("CountAlive() = %d, want 3", state.CountAlive())
	}

	state.Kill(Faulinei)
	if state.CountAlive() != 2 {
		t.Errorf("CountAlive() = %d, want 2", state.CountAlive())
	}

	state.Kill(Astaroth)
	if state.CountAlive() != 1 {
		t.Errorf("CountAlive() = %d, want 1", state.CountAlive())
	}

	state.Kill(Nosfentor)
	if state.CountAlive() != 0 {
		t.Errorf("CountAlive() = %d, want 0", state.CountAlive())
	}
}
