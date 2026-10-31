package game_state

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// REDUX DEVIATION: CRAFT-0011: the original has no player storage. A storage chest the party
// built on its own deed map keeps goods across visits, and opening it costs no karma; see
// ref:CRAFT-0011.

// REDUX DEVIATION: CRAFT-0018: the original has no player storage for arms, potions or scrolls.
// A storage chest also keeps them, never taking a readied item; see ref:CRAFT-0018.

// StorageChestProvisions are the provisions a storage chest keeps besides gold, in the order of
// StorageChest.Provisions (references.Provision values 0 to 4).
//
//nolint:gochecknoglobals // Read-only save-order table.
var StorageChestProvisions = [...]references.Provision{
	references.Torches, references.Gem, references.Key, references.SkullKeys, references.Food,
}

// NumberOfChestProvisions is the number of provision slots in a storage chest.
const NumberOfChestProvisions = len(StorageChestProvisions)

// StorageChest is the contents of one storage chest the party built. Every slot is capped like
// the party stack it comes from.
type StorageChest struct {
	Materials  [references.NumberOfMaterials]uint16
	Tools      [references.NumberOfTools]uint16
	Pieces     [references.NumberOfBuildPieces]uint16
	Provisions [NumberOfChestProvisions]uint16
	Gold       uint16
	// Equipment, Potions and Scrolls hold original items (ref:CRAFT-0018).
	Equipment [references.NumberOfEquipment]uint16
	Potions   [references.NumberOfPotions]uint16
	Scrolls   [references.NumberOfScrolls]uint16
}

// IsEmpty reports whether the chest holds nothing.
func (c *StorageChest) IsEmpty() bool {
	return *c == StorageChest{}
}

// StorageChestCategory is a group of goods moved together into or out of a storage chest.
type StorageChestCategory int

// Storage chest categories, in menu order.
const (
	StorageChestMaterials StorageChestCategory = iota
	StorageChestTools
	StorageChestPieces
	StorageChestGold
	StorageChestProvisionsCategory
	// StorageChestEquipment moves weapons, armour, shields, ammunition, rings and amulets, but
	// never a readied item (ref:CRAFT-0018).
	StorageChestEquipment
	StorageChestPotions
	StorageChestScrolls
)

// StorageChestCategories returns every category in menu order.
func StorageChestCategories() []StorageChestCategory {
	return []StorageChestCategory{
		StorageChestMaterials, StorageChestTools, StorageChestPieces, StorageChestGold, StorageChestProvisionsCategory,
		StorageChestEquipment, StorageChestPotions, StorageChestScrolls,
	}
}

// Name returns the menu label of the category.
func (c StorageChestCategory) Name() string {
	switch c {
	case StorageChestMaterials:
		return "Materials"
	case StorageChestTools:
		return "Tools"
	case StorageChestPieces:
		return "Pieces"
	case StorageChestGold:
		return "Gold"
	case StorageChestProvisionsCategory:
		return "Food, torches, gems and keys"
	case StorageChestEquipment:
		return "Arms and armour"
	case StorageChestPotions:
		return "Potions"
	case StorageChestScrolls:
		return "Scrolls"
	default:
		return "Goods"
	}
}

// storageChestSlot pairs one party stack with the chest counter that stores it. Keep is the part
// of the party stack that never goes into the chest: readied equipment.
type storageChestSlot struct {
	party party_state.ItemQuantity
	chest *uint16
	limit uint16
	keep  uint16
}

func (g *GameState) storageChestSlots(chest *StorageChest, category StorageChestCategory) []storageChestSlot {
	inventory := &g.PartyState.Inventory
	switch category {
	case StorageChestMaterials:
		slots := make([]storageChestSlot, 0, references.NumberOfMaterials)
		for _, material := range references.MaterialCatalog() {
			slots = append(slots, storageChestSlot{party: *inventory.Materials.GetQuantity(material),
				chest: &chest.Materials[material], limit: party_state.ItemQuantityLargeMax})
		}

		return slots
	case StorageChestTools:
		slots := make([]storageChestSlot, 0, references.NumberOfTools)
		for _, tool := range references.ToolCatalog() {
			slots = append(slots, storageChestSlot{party: *inventory.Tools.GetQuantity(tool),
				chest: &chest.Tools[tool], limit: party_state.ItemQuantitySmallMax})
		}

		return slots
	case StorageChestPieces:
		slots := make([]storageChestSlot, 0, references.NumberOfBuildPieces)
		for _, piece := range references.BuildPieceCatalog() {
			slots = append(slots, storageChestSlot{party: *inventory.Pieces.GetQuantity(piece.ID),
				chest: &chest.Pieces[piece.ID], limit: party_state.ItemQuantitySmallMax})
		}

		return slots
	case StorageChestGold:
		stack, limit := inventory.ProvisionStack(references.Gold)

		return []storageChestSlot{{party: stack, chest: &chest.Gold, limit: limit}}
	case StorageChestProvisionsCategory:
		slots := make([]storageChestSlot, 0, NumberOfChestProvisions)
		for index, provision := range StorageChestProvisions {
			stack, limit := inventory.ProvisionStack(provision)
			slots = append(slots, storageChestSlot{party: stack, chest: &chest.Provisions[index], limit: limit})
		}

		return slots
	case StorageChestEquipment:
		slots := make([]storageChestSlot, 0, references.NumberOfEquipment)
		for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
			slots = append(slots, storageChestSlot{party: *inventory.Equipment.GetQuantity(equipment),
				chest: &chest.Equipment[equipment], limit: party_state.ItemQuantitySmallMax,
				keep: g.readiedArmsQuantity(equipment)})
		}

		return slots
	case StorageChestPotions:
		slots := make([]storageChestSlot, 0, references.NumberOfPotions)
		for _, potion := range references.PotionCatalog() {
			slots = append(slots, storageChestSlot{party: *inventory.Potions.GetQuantity(potion),
				chest: &chest.Potions[potion], limit: party_state.ItemQuantitySmallMax})
		}

		return slots
	case StorageChestScrolls:
		slots := make([]storageChestSlot, 0, references.NumberOfScrolls)
		for _, scroll := range references.ScrollCatalog() {
			slots = append(slots, storageChestSlot{party: *inventory.Scrolls.GetQuantity(scroll),
				chest: &chest.Scrolls[scroll], limit: party_state.ItemQuantityLargeMax})
		}

		return slots
	default:
		panic(fmt.Sprintf("unknown storage chest category %d", category))
	}
}

// StorageChestInDirection returns the square of the party's storage chest next to it in the
// direction, on its own deed map with the option on.
func (g *GameState) StorageChestInDirection(direction references.Direction) (references.Position, bool) {
	if !g.IsCraftingEnabled() || !g.IsOnDeedMap() {
		return references.Position{}, false
	}
	target := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if target == nil {
		return references.Position{}, false
	}
	if _, ok := g.crafting.active.chests[*target]; !ok {
		return references.Position{}, false
	}

	return *target, true
}

// StorageChestContents returns a copy of a chest of the loaded deed map.
func (g *GameState) StorageChestContents(position references.Position) (StorageChest, bool) {
	if g.crafting.active == nil {
		return StorageChest{}, false
	}
	chest, ok := g.crafting.active.chests[position]
	if !ok {
		return StorageChest{}, false
	}

	return *chest, true
}

// TransferStorageChest moves every unit of a category into (store) or out of (take) a chest of
// the loaded deed map, each stack stopping at the receiving stack's cap, and returns how many
// units moved. It spends no time and no karma; the scene completes the Open command once.
func (g *GameState) TransferStorageChest(position references.Position, category StorageChestCategory, store bool) int {
	if g.crafting.active == nil {
		return 0
	}
	chest, ok := g.crafting.active.chests[position]
	if !ok {
		return 0
	}
	moved := 0
	for _, slot := range g.storageChestSlots(chest, category) {
		if store {
			amount := min(slot.party.Get()-min(slot.keep, slot.party.Get()), slot.limit-min(slot.limit, *slot.chest))
			if amount > 0 && slot.party.DecrementBy(amount) {
				*slot.chest += amount
				moved += int(amount)
			}

			continue
		}
		amount := min(*slot.chest, slot.limit-min(slot.limit, slot.party.Get()))
		if amount > 0 {
			slot.party.IncrementBy(amount)
			*slot.chest -= amount
			moved += int(amount)
		}
	}

	return moved
}

// StorageChestSummary describes a chest's contents, such as "12 Wood, 2 Axe, 300 Gold".
func (g *GameState) StorageChestSummary(position references.Position) string {
	chest, ok := g.StorageChestContents(position)
	if !ok {
		return ""
	}
	parts := make([]string, 0, 8)
	add := func(quantity uint16, name string) {
		if quantity > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", quantity, name))
		}
	}
	for _, material := range references.MaterialCatalog() {
		add(chest.Materials[material], material.Name())
	}
	for _, tool := range references.ToolCatalog() {
		add(chest.Tools[tool], tool.Name())
	}
	for _, piece := range references.BuildPieceCatalog() {
		add(chest.Pieces[piece.ID], piece.Name)
	}
	for index, provision := range StorageChestProvisions {
		add(chest.Provisions[index], references.ProvisionName(provision))
	}
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		add(chest.Equipment[equipment], equipment.String())
	}
	for _, potion := range references.PotionCatalog() {
		add(chest.Potions[potion], potion.Name())
	}
	for _, scroll := range references.ScrollCatalog() {
		add(chest.Scrolls[scroll], scroll.Name())
	}
	add(chest.Gold, "Gold")
	if len(parts) == 0 {
		return "The chest is empty."
	}

	return "The chest holds " + strings.Join(parts, ", ") + "."
}
