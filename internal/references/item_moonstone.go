package references

type Moonstone int

func (m Moonstone) ID() int {
	return int(m)
}

func (m Moonstone) Type() ItemType {
	return ItemTypeMoonstone
}

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
