package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCampingLocation_Constant verifies CampingLocation constant exists with correct value
func TestCampingLocation_Constant(t *testing.T) {
	// CampingLocation should be defined as 0xFE (254)
	assert.Equal(t, Location(0xFE), CampingLocation, "CampingLocation should be 0xFE")
}

// TestCampingLocation_GetMapType verifies CampingLocation returns CombatMapType
func TestCampingLocation_GetMapType(t *testing.T) {
	mapType := CampingLocation.GetMapType()
	assert.Equal(t, CombatMapType, mapType, "CampingLocation should return CombatMapType")
}

// TestCampingLocation_NotEmpty verifies CampingLocation is not EmptyLocation
func TestCampingLocation_NotEmpty(t *testing.T) {
	assert.NotEqual(t, EmptyLocation, CampingLocation, "CampingLocation should not be EmptyLocation (0xFF)")
}

// TestCampingLocation_Distinct verifies CampingLocation is distinct from other locations
func TestCampingLocation_Distinct(t *testing.T) {
	// Verify it doesn't collide with existing locations
	existingLocations := []Location{
		BritanniaUnderworld, Moonglow, Britain, Jhelom, Yew, Minoc, Trinsic,
		SkaraBrae, NewMagincia, LordBritishsCastle, CombatRestingShrine,
	}

	for _, loc := range existingLocations {
		assert.NotEqual(t, loc, CampingLocation,
			"CampingLocation should be distinct from %v", loc)
	}
}
