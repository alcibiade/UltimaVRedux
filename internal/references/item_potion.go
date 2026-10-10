package references

type Potion int

func (p Potion) ID() int {
	return int(p)
}

func (p Potion) Type() ItemType {
	return ItemTypePotion
}

//goland:noinspection GoUnusedConst
const (
	Blue   Potion = 0
	Yellow Potion = 1
	Red    Potion = 2
	Green  Potion = 3
	Orange Potion = 4
	Purple Potion = 5
	Black  Potion = 6
	White  Potion = 7
)

// NumberOfPotions is the number of authentic potion inventory identities.
const NumberOfPotions = 8

// PotionCatalog returns every potion identity in save-data order.
func PotionCatalog() []Potion {
	potions := make([]Potion, 0, NumberOfPotions)
	for potion := Blue; potion < NumberOfPotions; potion++ {
		potions = append(potions, potion)
	}

	return potions
}
