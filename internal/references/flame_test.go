package references

import (
	"testing"
)

func TestFlameOfPrinciple_String(t *testing.T) {
	tests := []struct {
		flame    FlameOfPrinciple
		expected string
	}{
		{FlameOfTruth, "Truth"},
		{FlameOfLove, "Love"},
		{FlameOfCourage, "Courage"},
		{FlameOfPrinciple(-1), "Unknown"},
		{FlameOfPrinciple(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.flame.String(); got != tt.expected {
				t.Errorf("FlameOfPrinciple.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFlameOfPrinciple_GetPrinciple(t *testing.T) {
	tests := []struct {
		flame    FlameOfPrinciple
		expected string
	}{
		{FlameOfTruth, "Flame of Truth"},
		{FlameOfLove, "Flame of Love"},
		{FlameOfCourage, "Flame of Courage"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.flame.GetPrinciple(); got != tt.expected {
				t.Errorf("FlameOfPrinciple.GetPrinciple() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFlameOfPrinciple_GetShadowlord(t *testing.T) {
	tests := []struct {
		flame    FlameOfPrinciple
		expected Shadowlord
	}{
		{FlameOfTruth, Faulinei},
		{FlameOfLove, Astaroth},
		{FlameOfCourage, Nosfentor},
	}

	for _, tt := range tests {
		t.Run(tt.flame.String(), func(t *testing.T) {
			if got := tt.flame.GetShadowlord(); got != tt.expected {
				t.Errorf("FlameOfPrinciple.GetShadowlord() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFlameOfPrinciple_GetRequiredShard(t *testing.T) {
	tests := []struct {
		flame    FlameOfPrinciple
		expected QuestItem
	}{
		{FlameOfTruth, ShardOfFalsehood},
		{FlameOfLove, ShardOfHatred},
		{FlameOfCourage, ShardOfCowardice},
	}

	for _, tt := range tests {
		t.Run(tt.flame.String(), func(t *testing.T) {
			if got := tt.flame.GetRequiredShard(); got != tt.expected {
				t.Errorf("FlameOfPrinciple.GetRequiredShard() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFlameLocations_CoordinatesFromOriginalSource(t *testing.T) {
	// Verify the three ritual locations in ref:SHADOW-0005:
	// Truth is at Lycaeum (15,9), floor 2; Love at Empath Abbey (15,3), floor 1;
	// Courage at Serpent's Hold (15,16), basement.

	tests := []struct {
		flame    FlameOfPrinciple
		location Location
		x        int
		y        int
		floor    int
	}{
		{FlameOfTruth, Lycaeum, 15, 9, 2},
		{FlameOfLove, EmpathAbbey, 15, 3, 1},
		{FlameOfCourage, SerpentsHold, 15, 16, -1}, // 0xFF = -1 (basement)
	}

	for _, tt := range tests {
		t.Run(tt.flame.String(), func(t *testing.T) {
			loc, ok := FlameLocations[tt.flame]
			if !ok {
				t.Fatalf("FlameLocations[%v] not found", tt.flame)
			}

			if loc.Location != tt.location {
				t.Errorf("Location = %v, want %v", loc.Location, tt.location)
			}
			if loc.X != tt.x {
				t.Errorf("X = %v, want %v", loc.X, tt.x)
			}
			if loc.Y != tt.y {
				t.Errorf("Y = %v, want %v", loc.Y, tt.y)
			}
			if loc.Floor != tt.floor {
				t.Errorf("Floor = %v, want %v", loc.Floor, tt.floor)
			}
		})
	}
}

func TestFlameLocations_Count(t *testing.T) {
	if len(FlameLocations) != FlameCount {
		t.Errorf("FlameLocations has %d entries, want %d", len(FlameLocations), FlameCount)
	}
}

func TestGetFlameAtLocation(t *testing.T) {
	tests := []struct {
		location Location
		expected FlameOfPrinciple
	}{
		{Lycaeum, FlameOfTruth},
		{EmpathAbbey, FlameOfLove},
		{SerpentsHold, FlameOfCourage},
		{Britain, FlameOfPrinciple(-1)},             // Not a Keep
		{Moonglow, FlameOfPrinciple(-1)},            // Not a Keep
		{BritanniaUnderworld, FlameOfPrinciple(-1)}, // Large map
	}

	for _, tt := range tests {
		t.Run(tt.location.String(), func(t *testing.T) {
			if got := GetFlameAtLocation(tt.location); got != tt.expected {
				t.Errorf("GetFlameAtLocation(%v) = %v, want %v", tt.location, got, tt.expected)
			}
		})
	}
}

func TestIsKeepLocation(t *testing.T) {
	tests := []struct {
		location Location
		expected bool
	}{
		{Lycaeum, true},
		{EmpathAbbey, true},
		{SerpentsHold, true},
		{Britain, false},
		{Moonglow, false},
		{Trinsic, false},
		{BritanniaUnderworld, false},
	}

	for _, tt := range tests {
		t.Run(tt.location.String(), func(t *testing.T) {
			if got := IsKeepLocation(tt.location); got != tt.expected {
				t.Errorf("IsKeepLocation(%v) = %v, want %v", tt.location, got, tt.expected)
			}
		})
	}
}

func TestIsAtFlamePosition(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		x        int
		y        int
		floor    int
		expected FlameOfPrinciple
		found    bool
	}{
		// Exact flame positions
		{"Truth flame", Lycaeum, 15, 9, 2, FlameOfTruth, true},
		{"Love flame", EmpathAbbey, 15, 3, 1, FlameOfLove, true},
		{"Courage flame", SerpentsHold, 15, 16, -1, FlameOfCourage, true},

		// Wrong coordinates at right location
		{"Lycaeum wrong position", Lycaeum, 10, 10, 2, -1, false},
		{"Lycaeum wrong floor", Lycaeum, 15, 9, 1, -1, false},
		{"Lycaeum wrong Y", Lycaeum, 15, 10, 2, -1, false},

		// Right coordinates wrong location
		{"Wrong location", Britain, 15, 9, 2, -1, false},

		// Non-Keep locations
		{"Britain", Britain, 5, 5, 0, -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flame, found := IsAtFlamePosition(tt.location, tt.x, tt.y, tt.floor)
			if found != tt.found {
				t.Errorf("IsAtFlamePosition() found = %v, want %v", found, tt.found)
			}
			if flame != tt.expected {
				t.Errorf("IsAtFlamePosition() flame = %v, want %v", flame, tt.expected)
			}
		})
	}
}

func TestFlame_Shadowlord_Shard_Mapping_Consistency(t *testing.T) {
	// Verify the three-way mapping is consistent:
	// Flame -> Shadowlord -> QuestItem == Flame -> GetRequiredShard

	for flame := FlameOfTruth; flame <= FlameOfCourage; flame++ {
		shadowlord := flame.GetShadowlord()
		shardFromShadowlord := shadowlord.ToQuestItem()
		shardFromFlame := flame.GetRequiredShard()

		if shardFromShadowlord != shardFromFlame {
			t.Errorf("Mapping inconsistency for %s: Shadowlord.ToQuestItem()=%v but Flame.GetRequiredShard()=%v",
				flame.String(), shardFromShadowlord, shardFromFlame)
		}
	}
}
