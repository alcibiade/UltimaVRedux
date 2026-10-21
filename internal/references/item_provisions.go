package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// Provision is a counted party supply such as torches.
type Provision int

// ID returns the provision number within its item type.
func (p Provision) ID() int {
	return int(p)
}

// Type returns ItemTypeProvision.
func (p Provision) Type() ItemType {
	return ItemTypeProvision
}

// NoProvision stands for no provision at all.
const NoProvision Provision = -1

// Provisions in original order, starting at Torches.
const (
	Torches   Provision = 0
	Gem       Provision = 1
	Key       Provision = 2
	SkullKeys Provision = 3
	Food      Provision = 4
	Gold      Provision = 5 // not really a provision - but needed for descriptions
)

// GetSpriteIndex returns the sprite used to show the provision.
func (p Provision) GetSpriteIndex() indexes.SpriteIndex {
	switch p {
	case Food:
		return indexes.ItemFood
	case Gold:
		return indexes.ItemMoney
	case Key, SkullKeys:
		return indexes.ItemKey
	case Gem:
		return indexes.ItemGem
	case Torches:
		return indexes.ItemTorch
	case NoProvision:
	}

	return indexes.AvatarKeyIndex
}
