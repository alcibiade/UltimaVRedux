package color

// Matches returns true if the given RGB values exactly match this EGA color.
func (c EGAColor) Matches(r, g, b uint8) bool {
	return c.R == r && c.G == g && c.B == b
}
