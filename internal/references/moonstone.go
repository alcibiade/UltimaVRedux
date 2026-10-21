package references

// InitialMoonstoneData represents the unchanging starting positions of moonstones
// Initial buried moonstone sites; ref:MOON-0001.
type InitialMoonstoneData struct {
	X     uint8 // X coordinate (0-255)
	Y     uint8 // Y coordinate (0-255)
	Map   uint8 // Map number (0=overworld, 1-32=towns)
	Level uint8 // Z level/dungeon level (0=surface)
}

// InitialMoonstoneLocations contains the 8 original moonstone starting positions
// This is unchanging reference data - actual moonstone state is managed in game_state package
var InitialMoonstoneLocations = []InitialMoonstoneData{
	{X: 0xe0, Y: 0x85, Map: 0, Level: 0}, // Moonstone 0: (224, 133)
	{X: 0x60, Y: 0x66, Map: 0, Level: 0}, // Moonstone 1: (96, 102)
	{X: 0x26, Y: 0xe0, Map: 0, Level: 0}, // Moonstone 2: (38, 224)
	{X: 0x32, Y: 0x25, Map: 0, Level: 0}, // Moonstone 3: (50, 37)
	{X: 0xa6, Y: 0x13, Map: 0, Level: 0}, // Moonstone 4: (166, 19)
	{X: 0x68, Y: 0xc2, Map: 0, Level: 0}, // Moonstone 5: (104, 194)
	{X: 0x17, Y: 0x7e, Map: 0, Level: 0}, // Moonstone 6: (23, 126)
	{X: 0xbb, Y: 0xa7, Map: 0, Level: 0}, // Moonstone 7: (187, 167)
}
