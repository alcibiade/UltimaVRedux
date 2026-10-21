package references

// Moonstone is one of the eight moonstones, numbered by moon phase.
type Moonstone int

// ID returns the moonstone number within its item type.
func (m Moonstone) ID() int {
	return int(m)
}

// Type returns ItemTypeMoonstone.
func (m Moonstone) Type() ItemType {
	return ItemTypeMoonstone
}

// Moonstones, in moon phase order starting at NewMoon.
//
//goland:noinspection GoUnusedConst
const (
	NewMoon        Moonstone = 0
	CrescentWaxing Moonstone = 1
	FirstQuarter   Moonstone = 2
	GibbousWaxing  Moonstone = 3
	FullMoon       Moonstone = 4
	GibbousWaning  Moonstone = 5
	LastQuarter    Moonstone = 6
	CrescentWaning Moonstone = 7
)

// NumberOfMoonstones is the number of authentic moonstone inventory identities.
const NumberOfMoonstones = 8

// MoonstoneCatalog returns every moonstone identity in save-data order.
func MoonstoneCatalog() []Moonstone {
	moonstones := make([]Moonstone, 0, NumberOfMoonstones)
	for moonstone := NewMoon; moonstone < NumberOfMoonstones; moonstone++ {
		moonstones = append(moonstones, moonstone)
	}

	return moonstones
}
