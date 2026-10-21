package character_creation

// Valid reports whether the value identifies one of the eight virtues.
func (virtue Virtue) Valid() bool { return virtue >= Honesty && virtue < VirtueCount }
