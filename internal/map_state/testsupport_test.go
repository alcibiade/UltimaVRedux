package map_state

// GetCurrentAngle returns the current rotation angle (0-15) or UninitializedAngle.
// Useful for debugging and testing.
func (ls *LighthouseState) GetCurrentAngle() int {
	return ls.currentAngle
}
