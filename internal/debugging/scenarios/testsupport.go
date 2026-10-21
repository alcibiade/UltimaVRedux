package scenarios

// Test support: fixtures and seams that tests in other packages use. The
// dead-code check treats this file as test support (workshop deadcode config).

// CurrentBaseline returns a current-state baseline.
func CurrentBaseline() Baseline {
	return Baseline{Type: BaselineCurrent}
}

// InitBaseline returns an INIT.GAM baseline.
func InitBaseline() Baseline {
	return Baseline{Type: BaselineInit}
}

// SnapshotBaseline returns a named save-snapshot baseline.
func SnapshotBaseline(reference string) Baseline {
	return Baseline{Type: BaselineSnapshot, Reference: reference}
}
