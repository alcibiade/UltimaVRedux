package journal

// AtlasChange is detached live feedback after a successful knowledge commit.
// It contains legitimate public identity and meaningful deltas, never positions.
type AtlasChange struct {
	EntryKey, Name, Kind string
	Discovered           bool
	Named                bool
	Surveyed             bool
	Measured             bool
}
