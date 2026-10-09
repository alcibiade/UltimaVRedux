package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// testDebugPrintFoundStatus is a test helper that prints detailed status of all found search locations
// (moved from authentic_search_location_manager.go to keep test-only code in test files)
//
//nolint:unused // Test helper used by multiple search test files
func testDebugPrintFoundStatus(aslm *AuthenticSearchLocationManager) string {
	allLocations := references.GetAllSearchLocations()
	result := fmt.Sprintf("=== SEARCH LOCATION DEBUG STATUS ===\n")
	result += fmt.Sprintf("Total locations: %d\n", len(allLocations))

	foundByMap := aslm.GetFoundLocationsByMap()
	totalFound := aslm.GetFoundCount()

	result += fmt.Sprintf("Total found: %d/%d (%.1f%%)\n",
		totalFound, len(allLocations), float64(totalFound)/float64(len(allLocations))*100)

	// Group by map for readability
	locationsByMap := make(map[references.Location][]references.SearchLocationReference)
	for _, loc := range allLocations {
		mapLoc := references.Location(loc.MapNumber)
		locationsByMap[mapLoc] = append(locationsByMap[mapLoc], loc)
	}

	for mapLoc, locations := range locationsByMap {
		foundInMap := 0
		if foundList, ok := foundByMap[mapLoc]; ok {
			foundInMap = len(foundList)
		}

		result += fmt.Sprintf("\nMap %d (%v): %d/%d found\n",
			mapLoc, mapLoc, foundInMap, len(locations))

		for _, loc := range locations {
			status := "🔍"
			if aslm.isAlreadyFound(loc.Index) {
				status = "✅"
			}

			specialNote := ""
			switch loc.Index {
			case 13:
				specialNote = " [KEYS]"
			case 14:
				specialNote = " [SKULL_KEYS]"
			case 15:
				specialNote = " [GLASS_SWORDS]"
			}

			result += fmt.Sprintf("  %s [%d] (%d,%d) L%d: Class%d payload%d%s\n",
				status, loc.Index, loc.X, loc.Y, loc.Level,
				loc.ObjectClass, loc.Payload, specialNote)
		}
	}

	return result
}
