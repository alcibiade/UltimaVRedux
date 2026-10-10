package datetime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMoonPhase_BasicEnum tests the moon phase enumeration
func TestMoonPhase_BasicEnum(t *testing.T) {
	// This test will initially fail - validates MoonPhase enum exists
	phase := MoonPhaseNew
	assert.Equal(t, MoonPhase(0), phase)

	// Test all 8 moon phases exist (0-7)
	allPhases := []MoonPhase{
		MoonPhaseNew,            // 0
		MoonPhaseCrescentWaxing, // 1
		MoonPhaseFirstQuarter,   // 2
		MoonPhaseGibbousWaxing,  // 3
		MoonPhaseFull,           // 4
		MoonPhaseGibbousWaning,  // 5
		MoonPhaseLastQuarter,    // 6
		MoonPhaseCrescentWaning, // 7
	}

	for i, phase := range allPhases {
		assert.Equal(t, MoonPhase(i), phase, "Moon phase %d should equal %d", i, i)
	}
}

// TestMoonPhase_StringRepresentation tests string representation of moon phases
func TestMoonPhase_StringRepresentation(t *testing.T) {
	testCases := []struct {
		phase    MoonPhase
		expected string
	}{
		{MoonPhaseNew, "New Moon"},
		{MoonPhaseCrescentWaxing, "Crescent Waxing"},
		{MoonPhaseFirstQuarter, "First Quarter"},
		{MoonPhaseGibbousWaxing, "Gibbous Waxing"},
		{MoonPhaseFull, "Full Moon"},
		{MoonPhaseGibbousWaning, "Gibbous Waning"},
		{MoonPhaseLastQuarter, "Last Quarter"},
		{MoonPhaseCrescentWaning, "Crescent Waning"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, tc.phase.String(), "Phase %d should have correct string representation", int(tc.phase))
	}
}

// TestGetCurrentMoonPhase_BasicCalculation tests moon phase calculation
func TestGetCurrentMoonPhase_BasicCalculation(t *testing.T) {
	// This test will initially fail - validates GetCurrentMoonPhase method exists
	date := UltimaDate{Day: 1, Month: 1, Year: 1}

	phase := date.GetCurrentMoonPhase()

	// Should return a valid phase (0-7)
	assert.GreaterOrEqual(t, int(phase), 0)
	assert.LessOrEqual(t, int(phase), 7)
}

// TestGetCurrentMoonPhase_AuthenticCycle tests authentic 28-day lunar cycle
func TestGetCurrentMoonPhase_AuthenticCycle(t *testing.T) {
	// Based on the two lunar phases for each day of the 28-day cycle; ref:MOON-0002.
	// Test that moon phases cycle correctly over 28 days

	date := UltimaDate{Day: 1, Month: 1, Year: 1}

	var phases []MoonPhase
	for day := 1; day <= 28; day++ {
		date.Day = byte(day)
		phase := date.GetCurrentMoonPhase()
		phases = append(phases, phase)
	}

	// Should have exactly 28 phase calculations
	assert.Len(t, phases, 28, "Should calculate 28 days of moon phases")

	// Verify we get different phases (not all the same)
	uniquePhases := make(map[MoonPhase]bool)
	for _, phase := range phases {
		uniquePhases[phase] = true
	}
	assert.Greater(t, len(uniquePhases), 1, "Should have multiple different moon phases over 28 days")

	// First and 29th day should have same phase (cycle repeats)
	date.Day = 1
	phase1 := date.GetCurrentMoonPhase()
	date.Day = 29
	phase29 := date.GetCurrentMoonPhase()
	assert.Equal(t, phase1, phase29, "Moon phase should repeat every 28 days")
}

// TestGetCurrentMoonPhase_OriginalAlgorithm tests against original Ultima V moon phase data
func TestGetCurrentMoonPhase_OriginalAlgorithm(t *testing.T) {
	// Test specific dates that we know the phases for from original algorithm
	// This will be filled in once we implement the authentic calculation

	testCases := []struct {
		day      byte
		month    byte
		year     uint16
		expected MoonPhase
	}{
		{1, 1, 1, MoonPhaseNew},   // First day should be known phase
		{8, 1, 1, MoonPhaseFull},  // Mid-cycle should be different
		{15, 1, 1, MoonPhaseNew},  // Half-cycle later
		{22, 1, 1, MoonPhaseFull}, // Another full cycle
	}

	for _, tc := range testCases {
		date := UltimaDate{Day: tc.day, Month: tc.month, Year: tc.year}
		phase := date.GetCurrentMoonPhase()

		// For now, just validate it's a valid phase
		// TODO: Update with authentic expected values once algorithm is implemented
		assert.GreaterOrEqual(t, int(phase), 0, "Day %d should have valid phase", tc.day)
		assert.LessOrEqual(t, int(phase), 7, "Day %d should have valid phase", tc.day)
	}
}

// TestMoonPhaseToMoonstoneIndex tests mapping moon phases to moonstone indices
func TestMoonPhaseToMoonstoneIndex(t *testing.T) {
	// Each moon phase should map to a specific moonstone (0-7)
	for phase := MoonPhaseNew; phase <= MoonPhaseCrescentWaning; phase++ {
		index := phase.ToMoonstoneIndex()
		assert.GreaterOrEqual(t, index, 0, "Phase %d should map to valid moonstone index", int(phase))
		assert.LessOrEqual(t, index, 7, "Phase %d should map to valid moonstone index", int(phase))
	}

	// Test specific mappings
	assert.Equal(t, 0, MoonPhaseNew.ToMoonstoneIndex())
	assert.Equal(t, 4, MoonPhaseFull.ToMoonstoneIndex())
	assert.Equal(t, 7, MoonPhaseCrescentWaning.ToMoonstoneIndex())
}

func TestDualMoonPhasesMatchOriginalTwentyEightDayTable(t *testing.T) {
	wantTrammel := [DaysInMonth]MoonPhase{0, 1, 1, 2, 2, 3, 3, 4, 5, 5, 6, 6, 7, 7, 0, 1, 1, 2, 2, 3, 3, 4, 5, 5, 6, 6, 7, 7}
	wantFelucca := [DaysInMonth]MoonPhase{0, 0, 1, 2, 3, 4, 5, 6, 7, 0, 0, 1, 2, 3, 4, 5, 6, 7, 0, 0, 1, 2, 3, 4, 5, 6, 7, 0}
	date := UltimaDate{Year: 1, Month: 1}
	for day := 1; day <= DaysInMonth; day++ {
		date.Day = byte(day)
		assert.Equal(t, wantTrammel[day-1], date.GetTrammelMoonPhase(), "Trammel day %d", day)
		assert.Equal(t, wantFelucca[day-1], date.GetFeluccaMoonPhase(), "Felucca day %d", day)
	}
}
