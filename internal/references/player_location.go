package references

// PlayerLocation is where the party stands: location, tile and floor.
type PlayerLocation struct {
	Location Location
	Position Position
	Floor    FloorNumber
}
