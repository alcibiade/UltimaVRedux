package game_state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShrineData_AllShrinesPopulated verifies all 8 shrines have valid data
func TestShrineData_AllShrinesPopulated(t *testing.T) {
	t.Parallel()

	for i := 0; i < TotalShrines; i++ {
		shrine := GetShrineData(i)
		if shrine == nil {
			t.Errorf("GetShrineData(%d) returned nil", i)
			continue
		}

		if shrine.Index != i {
			t.Errorf("Shrine %d has wrong index: %d", i, shrine.Index)
		}

		if shrine.VirtueName == "" {
			t.Errorf("Shrine %d has empty VirtueName", i)
		}

		if shrine.VirtueKey == "" {
			t.Errorf("Shrine %d has empty VirtueKey", i)
		}

		if shrine.Mantra == "" {
			t.Errorf("Shrine %d has empty Mantra", i)
		}

		if shrine.WordOfPower == "" {
			t.Errorf("Shrine %d has empty WordOfPower", i)
		}
	}
}

// TestGetShrineByPosition_AllLocations verifies each coordinate returns correct shrine
func TestGetShrineByPosition_AllLocations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		x, y      byte
		wantIndex int
	}{
		{"Honesty", 0xE9, 0x42, 0},
		{"Compassion", 0x80, 0x5C, 1},
		{"Valour", 0x24, 0xE5, 2},
		{"Justice", 0x49, 0x0B, 3},
		{"Sacrifice", 0xCD, 0x2D, 4},
		{"Honor", 0x51, 0xCF, 5},
		{"Spirituality", 0x00, 0x00, 6}, // Via moongate
		{"Humility", 0xE7, 0xD8, 7},
		{"Invalid location", 0xFF, 0xFF, -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetShrineByPosition(tc.x, tc.y)
			if got != tc.wantIndex {
				t.Errorf("GetShrineByPosition(%#x, %#x) = %d, want %d", tc.x, tc.y, got, tc.wantIndex)
			}
		})
	}
}

// TestGetShrineForWord_AllWords verifies all 8 Words of Power are recognized
func TestGetShrineForWord_AllWords(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		word      string
		wantIndex int
	}{
		{"FALLAX", 0},
		{"VILIS", 1},
		{"INOPIA", 2},
		{"MALUM", 3},
		{"AVIDUS", 4},
		{"INFAMA", 5},
		{"IGNAVUS", 6},
		{"VERAMOCOR", 7},
		// Case insensitive
		{"fallax", 0},
		{"Vilis", 1},
		{"inopia", 2},
		// With whitespace
		{"  FALLAX  ", 0},
		// Invalid
		{"INVALID", -1},
		{"", -1},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			t.Parallel()
			got := GetShrineForWordResponse(tc.word)
			if got != tc.wantIndex {
				t.Errorf("GetShrineForWordResponse(%q) = %d, want %d", tc.word, got, tc.wantIndex)
			}
		})
	}
}

// TestValidateVirtue_PartialMatches verifies partial string matching works
func TestValidateVirtue_PartialMatches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		virtueKey string
		want      bool
	}{
		{"Exact match", "hone", "hone", true},
		{"Full word contains key", "honesty", "hone", true},
		{"Prefix", "honest", "hone", true},
		{"Case insensitive", "HONESTY", "hone", true},
		{"Mixed case", "Compassion", "comp", true},
		{"With whitespace", "  honesty  ", "hone", true},
		{"Wrong virtue", "valor", "hone", false},
		{"Empty input", "", "hone", false},
		{"Partial wrong", "hon", "hone", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ValidateVirtue(tc.input, tc.virtueKey)
			if got != tc.want {
				t.Errorf("ValidateVirtue(%q, %q) = %v, want %v", tc.input, tc.virtueKey, got, tc.want)
			}
		})
	}
}

func TestValidateRestorationVirtueRequiresFullCanonicalNameInsideResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "exact", input: "Honesty", expected: true},
		{name: "contained case insensitive", input: "I meditate upon HONESTY today", expected: true},
		{name: "four letter meditation key", input: "hone", expected: false},
		{name: "prefix", input: "honest", expected: false},
		{name: "suffix fragment", input: "nesty", expected: false},
		{name: "empty", input: "", expected: false},
		{name: "wrong virtue", input: "Compassion", expected: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, ValidateRestorationVirtue(test.input, "Honesty"))
		})
	}
}

// TestValidateMantra_CaseInsensitive verifies mantra validation is case insensitive
func TestValidateMantra_CaseInsensitive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  string
		mantra string
		want   bool
	}{
		{"Exact lowercase", "ahm", "ahm", true},
		{"Exact uppercase", "AHM", "ahm", true},
		{"Mixed case", "Ahm", "ahm", true},
		{"With extra text", "my ahm mantra", "ahm", true},
		{"With whitespace", "  ahm  ", "ahm", true},
		{"Wrong mantra", "mu", "ahm", false},
		{"Empty input", "", "ahm", false},
		{"Partial mantra", "ah", "ahm", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ValidateMantra(tc.input, tc.mantra)
			if got != tc.want {
				t.Errorf("ValidateMantra(%q, %q) = %v, want %v", tc.input, tc.mantra, got, tc.want)
			}
		})
	}
}

// TestGetShrineData_ValidIndex verifies GetShrineData returns correct data
func TestGetShrineData_ValidIndex(t *testing.T) {
	t.Parallel()

	for i := 0; i < TotalShrines; i++ {
		shrine := GetShrineData(i)
		if shrine == nil {
			t.Errorf("GetShrineData(%d) returned nil", i)
			continue
		}

		if shrine.Index != i {
			t.Errorf("GetShrineData(%d).Index = %d, want %d", i, shrine.Index, i)
		}
	}

	// Invalid indices
	if GetShrineData(-1) != nil {
		t.Error("GetShrineData(-1) should return nil")
	}

	if GetShrineData(TotalShrines) != nil {
		t.Errorf("GetShrineData(%d) should return nil", TotalShrines)
	}
}

// TestShrineData_KarmaBonus verifies karma bonuses are correct
func TestShrineData_KarmaBonus(t *testing.T) {
	t.Parallel()

	for i := 0; i < TotalShrines; i++ {
		shrine := GetShrineData(i)
		karma := shrine.KarmaBonus()

		if i == 7 { // Humility
			if karma != 6 {
				t.Errorf("Humility shrine karma bonus = %d, want 6", karma)
			}
		} else {
			if karma != 3 {
				t.Errorf("Shrine %d (%s) karma bonus = %d, want 3", i, shrine.VirtueName, karma)
			}
		}
	}
}

// TestShrineData_StatBonuses verifies stat bonuses match original game data
func TestShrineData_StatBonuses(t *testing.T) {
	t.Parallel()

	expectedBonuses := []struct {
		virtue string
		str    byte
		dex    byte
		intl   byte
	}{
		{"Honesty", 0, 0, 1},
		{"Compassion", 0, 1, 0},
		{"Valour", 1, 0, 0},
		{"Justice", 0, 1, 1},
		{"Sacrifice", 1, 1, 0},
		{"Honor", 1, 0, 1},
		{"Spirituality", 1, 1, 1},
		{"Humility", 0, 0, 0},
	}

	for i, expected := range expectedBonuses {
		shrine := GetShrineData(i)
		if shrine.StrBonus != expected.str || shrine.DexBonus != expected.dex || shrine.IntBonus != expected.intl {
			t.Errorf("Shrine %s bonuses: got (STR=%d, DEX=%d, INT=%d), want (STR=%d, DEX=%d, INT=%d)",
				expected.virtue,
				shrine.StrBonus, shrine.DexBonus, shrine.IntBonus,
				expected.str, expected.dex, expected.intl)
		}
	}
}

// TestQuestStateConstants verifies quest state constants are distinct
func TestQuestStateConstants(t *testing.T) {
	t.Parallel()

	states := map[int]string{
		QuestStateNotAssigned:     "NotAssigned",
		QuestStateReadyToComplete: "ReadyToComplete",
		QuestStateOffering:        "Offering",
	}

	if len(states) != 3 {
		t.Error("Quest state constants are not distinct")
	}
}

func TestShrineDataQuestAssignmentsTeachOpposedViceAtCodex(t *testing.T) {
	t.Parallel()

	expected := []string{
		"the failing of Dishonesty!",
		"of the heart of a cruel soul!",
		"the failing of a life without Valour!",
		"the weakness of the Unjust!",
		"the failing of unwilling Sacrifice!",
		"the darkness of Dishonor!",
		"the neglect of one's Spirit!",
		"the weakness of a life consumed by Pride!",
	}
	for index, want := range expected {
		assert.Equal(t, want, GetShrineData(index).QuestText())
		assert.NotContains(t, GetShrineData(index).QuestText(), "VERAMOCOR")
		assert.NotContains(t, GetShrineData(index).AnswerText(), "VERAMOCOR")
	}
}

func TestCodexFinalRevelationContainsDoomDiscoveryFacts(t *testing.T) {
	t.Parallel()

	transcript := strings.Join(strings.Fields(strings.Join(CodexRevelationMessages[:], " ")), " ")
	assert.Contains(t, transcript, "SHAME'S EGRESS")
	assert.Contains(t, transcript, "CENTRE OF THE UNDERWORLD")
	assert.Contains(t, transcript, "GATE TO THE CORE OF THE WORLD")
	assert.Contains(t, transcript, "VERAMOCOR")
	assert.Contains(t, transcript, "ETHEREAL WARDS")
	assert.Contains(t, transcript, "STEALERS OF SOULS")
}

// TestShrineData_AnswerText verifies AnswerText() returns non-empty text
func TestShrineData_AnswerText(t *testing.T) {
	t.Parallel()

	for i := 0; i < TotalShrines; i++ {
		shrine := GetShrineData(i)
		text := shrine.AnswerText()

		if text == "" {
			t.Errorf("Shrine %d AnswerText() is empty", i)
		}
	}
}
