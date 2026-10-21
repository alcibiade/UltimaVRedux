package map_state

// Test support: fixtures and seams that tests in other packages use. The
// dead-code check treats this file as test support (workshop deadcode config).

// GetAmbientLightRadius returns the current ambient light radius.
func GetAmbientLightRadius() int {
	return ambientLightRadius
}
