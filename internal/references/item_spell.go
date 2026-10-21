package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

func newSpellItemReferences() map[Spell]InventoryItem {
	items := make(map[Spell]InventoryItem, TotalSpellCount)
	for spell := Spell(0); spell < Spell(TotalSpellCount); spell++ {
		//nolint:exhaustruct // Spell prose and names are owned by GetSpellInfo.
		items[spell] = InventoryItem{ItemSprite: indexes.ItemScroll}
	}

	return items
}

// Spell is a castable spell, numbered in the original spell order.
type Spell int

// InvalidSpell represents an unknown or invalid spell (for failed mixing attempts)
const InvalidSpell Spell = -1

// ID returns the spell number within its item type.
func (s Spell) ID() int {
	return int(s)
}

// Type returns ItemTypeSpell.
func (s Spell) Type() ItemType {
	return ItemTypeSpell
}

// Spells in original order, starting at InLor.
//
//goland:noinspection GoUnusedConst
const (
	InLor Spell = iota
	GravPor
	AnZu
	AnNox
	Mani
	AnYlem
	AnSanct
	AnXenCorp
	RelHur
	InWis
	KalXen
	InXenMani
	VasLor
	VasFlam
	InFlamGrav
	InNoxGrav
	InZuGrav
	InPor
	AnGrav
	InSanct
	InSanctGrav
	UusPor
	DesPor
	WisQuas
	InBetXen
	AnExPor
	InExPor
	VasMani
	InZu
	RelTym
	InVasPorYlem
	QuasAnWis
	InAn
	WisAnYlem
	AnXenEx
	RelXenBet
	SanctLor
	XenCorp
	InQuasXen
	InQuasWis
	InNoxHur
	InQuasCorp
	InManiCorp
	KalXenCorp
	InVasGravCorp
	InFlamHur
	VasRelPor
	AnTym
	Nox
)
