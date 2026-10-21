package references

// Scroll is one of the eight spell scrolls, in save-data order.
type Scroll int

// ID returns the scroll number within its item type.
func (s Scroll) ID() int {
	return int(s)
}

// Type returns ItemTypeScroll.
func (s Scroll) Type() ItemType {
	return ItemTypeScroll
}

// Scrolls in save-data order, starting at ScrollVasLor.
//
//goland:noinspection GoUnusedConst
const (
	ScrollVasLor     Scroll = 0
	ScrollRelHur     Scroll = 1
	ScrollInSanct    Scroll = 2
	ScrollInAn       Scroll = 3
	ScrollInQuasWis  Scroll = 4
	ScrollKalXenCorp Scroll = 5
	ScrollInManiCorp Scroll = 6
	ScrollAnTym      Scroll = 7
)

// NumberOfScrolls is the number of authentic scroll inventory identities.
const NumberOfScrolls = 8

// ScrollCatalog returns every scroll identity in save-data order.
func ScrollCatalog() []Scroll {
	scrolls := make([]Scroll, 0, NumberOfScrolls)
	for scroll := ScrollVasLor; scroll < NumberOfScrolls; scroll++ {
		scrolls = append(scrolls, scroll)
	}

	return scrolls
}
