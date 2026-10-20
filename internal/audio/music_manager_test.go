package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAuthoredLocationMusic(t *testing.T) {
	tests := []struct {
		location      references.Location
		expectedTrack string
		description   string
	}{
		{references.Britain, "town", "Britain should play town music"},
		{references.BritanniaUnderworld, "overworld", "Overworld should play exploration music"},
		{references.LordBritishsCastle, "rule", "Lord British's castle should play Rule Britannia"},
		{references.PalaceOfBlackthorn, "blackthorn", "Blackthorn's palace should play Lord Blackthorn theme"},
		{references.Deceit, "dungeon", "Dungeons should play Halls of Doom"},
		{references.BuccaneersDen, "overworld", "Sailing music is no longer tied to pirate locations"},
		{references.Moonglow, "ambient", "Mage city should play mystical music"},
		{references.Paws, "tarantella", "Villages should play village music"},
		{references.CombatRestingShrine, "victory", "Combat shrine should play victory music"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			track := productionLibrary().Routing.musicFor(test.location, references.FloorNumber(references.OVERWORLD))
			assert.Equal(t, test.expectedTrack, track, test.description)
		})
	}
}

// TestMusicManager_GetMusicForContext tests context-based music mapping
func TestMusicManager_GetMusicForContext(t *testing.T) {
	audioManager, _, _ := gameAudioFixture(t)
	musicManager := audioManager.policy

	tests := []struct {
		context       string
		expectedTrack string
		description   string
	}{
		{"title", "title", "Title context should play Ultima Theme"},
		{"intro", "title", "Intro context should play Ultima Theme"},
		{"menu", "title", "Menu context should play Ultima Theme"},
		{"victory", "victory", "Victory context should play Fanfare for the Virtuous"},
		{"reunion", "reunion", "Reunion context should play Joyous Reunion"},
		{"underworld", "worlds", "Underworld context should play Worlds Below"},
		{"combat", "combat", "Combat context should play Engagement and Malee"},
		{"unknown", "overworld", "Unknown context should default to overworld"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			track := musicManager.GetMusicForContext(test.context)
			assert.Equal(t, test.expectedTrack, track, test.description)
		})
	}
}

func TestMusicContextLookupPreservesDefaultAndExplicitSilence(t *testing.T) {
	defaultCue, combatCue := "music/track/default", "music/track/battle"
	assignments := map[string]*string{
		routingDefault: &defaultCue, "context/combat": &combatCue, "context/victory": nil,
	}
	assert.Equal(t, "battle", musicForContext(assignments, "combat"))
	assert.Empty(t, musicForContext(assignments, "victory"))
	assert.Equal(t, "default", musicForContext(assignments, "unknown"))
	assert.Empty(t, musicForContext(nil, "combat"))
}

// TestMusicManager_LocationMappingCompleteness tests that all major locations have mappings
func TestMusicManager_LocationMappingCompleteness(t *testing.T) {
	// Test that all major location types have appropriate mappings
	allMappings := locationMusicMappings()

	// Verify we have mappings for all major locations
	assert.Greater(t, len(allMappings), 30, "Should have mappings for most locations")

	// Verify specific important locations
	importantLocations := []references.Location{
		references.Britain,
		references.LordBritishsCastle,
		references.PalaceOfBlackthorn,
		references.Deceit,
		references.BuccaneersDen,
		references.Moonglow,
	}

	for _, location := range importantLocations {
		track, exists := allMappings[location]
		assert.True(t, exists, "Location %v should have music mapping", location)
		assert.NotEmpty(t, track, "Location %v should have non-empty track", location)
	}
}

// TestMusicManager_StateManagement tests music manager state operations
func TestMusicManager_StateManagement(t *testing.T) {
	audioManager, _, _ := gameAudioFixture(t)
	musicManager := audioManager.policy

	// Test initial state
	assert.True(t, musicManager.IsEnabled(), "Music manager should be enabled by default")
	assert.Equal(t, "", musicManager.GetCurrentTrack(), "Should start with no track")
	assert.Equal(t, references.EmptyLocation, musicManager.GetCurrentLocation(), "Should start with empty location")

	// Test disable/enable
	musicManager.SetEnabled(false)
	assert.False(t, musicManager.IsEnabled(), "Music manager should be disabled")

	musicManager.SetEnabled(true)
	assert.True(t, musicManager.IsEnabled(), "Music manager should be enabled again")
}
