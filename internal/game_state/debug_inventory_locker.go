package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	debugInventoryGrantQuantity  = 5
	debugInventoryReagentTwenty  = 20
	debugInventoryExtraMutations = 2
)

var (
	errUnsupportedDebugInventoryPreset = errors.New("unsupported debug inventory preset")
	errUnsupportedDebugEquipmentPreset = errors.New("unsupported debug equipment grant preset")
	errInvalidDebugPotion              = errors.New("invalid potion or amount")
	errInvalidDebugScroll              = errors.New("invalid scroll or amount")
	errInvalidDebugReagent             = errors.New("invalid reagent")
	errInvalidDebugEquipment           = errors.New("invalid equipment")
	errUnsupportedDebugProvision       = errors.New("unsupported debug inventory provision")
)

// DebugInventoryProvision is the stable provision identity accepted by debug inventory APIs.
type DebugInventoryProvision string

// Supported debug inventory provisions.
const (
	DebugInventoryProvisionGold      DebugInventoryProvision = "gold"
	DebugInventoryProvisionFood      DebugInventoryProvision = "food"
	DebugInventoryProvisionGems      DebugInventoryProvision = "gems"
	DebugInventoryProvisionTorches   DebugInventoryProvision = "torches"
	DebugInventoryProvisionKeys      DebugInventoryProvision = "keys"
	DebugInventoryProvisionSkullKeys DebugInventoryProvision = "skull_keys"
)

// DebugInventoryPreset identifies an absolute inventory locker preset.
type DebugInventoryPreset string

// Supported absolute debug inventory presets.
const (
	DebugInventoryPresetProvisionsMax           DebugInventoryPreset = "provisions_max"
	DebugInventoryPresetReagentsAll99           DebugInventoryPreset = "reagents_all99"
	DebugInventoryPresetReagentsAll20NoGarlic   DebugInventoryPreset = "reagents_all20_no_garlic"
	DebugInventoryPresetPotionsAll1             DebugInventoryPreset = "potions_all1"
	DebugInventoryPresetPotionsAll99            DebugInventoryPreset = "potions_all99"
	DebugInventoryPresetScrollsAll1             DebugInventoryPreset = "scrolls_all1"
	DebugInventoryPresetScrollsAllMax           DebugInventoryPreset = "scrolls_allmax"
	DebugInventoryPresetEquipmentAllEquippable5 DebugInventoryPreset = "equipment_all_equippable5"
	DebugInventoryPresetEquipmentWeapons5       DebugInventoryPreset = "equipment_weapons5"
	DebugInventoryPresetEquipmentArmour5        DebugInventoryPreset = "equipment_armour5"
	DebugInventoryPresetEquipmentBowsNoAmmo5    DebugInventoryPreset = "equipment_bows_no_ammo5"
)

// DebugInventoryEquipmentGrantPreset identifies an additive equipment locker preset.
type DebugInventoryEquipmentGrantPreset string

// Supported additive equipment grant presets.
const (
	DebugInventoryEquipmentGrantPresetAllEquippable5 DebugInventoryEquipmentGrantPreset = "all_equippable5"
	DebugInventoryEquipmentGrantPresetWeapons5       DebugInventoryEquipmentGrantPreset = "weapons5"
	DebugInventoryEquipmentGrantPresetArmour5        DebugInventoryEquipmentGrantPreset = "armour5"
	DebugInventoryEquipmentGrantPresetBowsNoAmmo5    DebugInventoryEquipmentGrantPreset = "bows_no_ammo5"
)

// DebugInventoryChange describes one detached locker value mutation.
type DebugInventoryChange struct {
	Path   string
	Before uint16
	After  uint16
}

// DebugEquipmentStateChange describes one carried or equipped item removed by a full equipment clear.
type DebugEquipmentStateChange struct {
	Path   string
	Before int
	After  int
}

type debugInventoryMutation struct {
	change DebugInventoryChange
	apply  func()
}

type debugInventoryMutations struct {
	game      *GameState
	snapshot  party_state.InventoryLockerSnapshot
	mutations []debugInventoryMutation
}

// GetDebugInventorySnapshot returns the complete detached locker inventory view.
func (g *GameState) GetDebugInventorySnapshot() party_state.InventoryLockerSnapshot {
	return g.PartyState.Inventory.LockerSnapshot()
}

// SetDebugInventoryProvision assigns one provision quantity, clamped to its authoritative maximum.
func (g *GameState) SetDebugInventoryProvision(
	provision DebugInventoryProvision,
	quantity uint16,
) ([]DebugInventoryChange, error) {
	before, maxQuantity, err := g.debugProvisionValue(provision)
	if err != nil {
		return nil, err
	}
	after := min(quantity, maxQuantity)
	if before == after {
		return []DebugInventoryChange{}, nil
	}

	g.setDebugProvisionValue(provision, after)
	changes := []DebugInventoryChange{{Path: "provisions." + string(provision), Before: before, After: after}}

	return changes, nil
}

// SetDebugInventoryReagent assigns one carried reagent quantity, clamped to 99.
func (g *GameState) SetDebugInventoryReagent(
	reagent references.Reagent,
	quantity uint16,
) ([]DebugInventoryChange, error) {
	if reagent < references.SulfurAsh || reagent > references.MandrakeRoot {
		return nil, fmt.Errorf("%w: %d", errInvalidDebugReagent, reagent)
	}
	before := g.PartyState.Inventory.Reagent.Get(reagent)
	after := min(quantity, party_state.ItemQuantitySmallMax)
	if before == after {
		return []DebugInventoryChange{}, nil
	}

	g.PartyState.Inventory.Reagent.Set(reagent, after)
	changes := []DebugInventoryChange{{Path: fmt.Sprintf("reagents.%d", reagent), Before: before, After: after}}

	return changes, nil
}

// SetDebugInventoryEquipment assigns one carried equipment quantity, clamped to 99.
func (g *GameState) SetDebugInventoryEquipment(
	equipment references.Equipment,
	quantity uint16,
) ([]DebugInventoryChange, error) {
	if equipment < references.LeatherHelm || equipment > references.Ankh {
		return nil, fmt.Errorf("%w: %d", errInvalidDebugEquipment, equipment)
	}
	before := g.PartyState.Inventory.Equipment.Get(equipment)
	after := min(quantity, party_state.ItemQuantitySmallMax)
	if before == after {
		return []DebugInventoryChange{}, nil
	}

	g.PartyState.Inventory.Equipment.Set(equipment, after)
	changes := []DebugInventoryChange{{Path: fmt.Sprintf("equipment.%d", equipment), Before: before, After: after}}

	return changes, nil
}

// ClearDebugEquipmentState destroys all carried equipment and clears every roster loadout.
func (g *GameState) ClearDebugEquipmentState() []DebugEquipmentStateChange {
	changes := make([]DebugEquipmentStateChange, 0)
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		quantity := g.PartyState.Inventory.Equipment.Get(equipment)
		if quantity == 0 {
			continue
		}
		changes = append(changes, DebugEquipmentStateChange{
			Path:   fmt.Sprintf("inventory.equipment.%d.quantity", equipment.ID()),
			Before: int(quantity),
			After:  0,
		})
		g.PartyState.Inventory.Equipment.Set(equipment, 0)
	}

	g.PartyState.ForEachCharacter(func(rosterIndex int, character *party_state.PlayerCharacter) {
		for _, slot := range debugEquipmentSlots() {
			equipment := character.GetEquippedItem(slot)
			if equipment == references.NoEquipment {
				continue
			}
			changes = append(changes, DebugEquipmentStateChange{
				Path:   fmt.Sprintf("party.roster.%d.equipment.%s", rosterIndex, debugEquipmentSlotID(slot)),
				Before: equipment.ID(),
				After:  references.NoEquipment.ID(),
			})
			character.SetEquippedItem(slot, references.NoEquipment)
		}
	})

	return changes
}

func debugEquipmentSlots() []references.EquipmentSlot {
	return []references.EquipmentSlot{
		references.SlotHelm,
		references.SlotArmour,
		references.SlotWeapon,
		references.SlotShield,
		references.SlotRing,
		references.SlotAmulet,
	}
}

func debugEquipmentSlotID(slot references.EquipmentSlot) string {
	switch slot {
	case references.SlotHelm:
		return "helm"
	case references.SlotArmour:
		return "armour"
	case references.SlotWeapon:
		return "weapon"
	case references.SlotShield:
		return "shield"
	case references.SlotRing:
		return "ring"
	case references.SlotAmulet:
		return "amulet"
	case references.SlotNone:
		panic("non-equipment slot has no debug identifier")
	default:
		panic("unknown equipment slot")
	}
}

// ApplyDebugInventoryPreset applies one absolute, idempotent inventory locker preset.
func (g *GameState) ApplyDebugInventoryPreset(preset DebugInventoryPreset) ([]DebugInventoryChange, error) {
	collector := debugInventoryMutations{game: g, snapshot: g.GetDebugInventorySnapshot(), mutations: nil}
	if err := collector.applyPreset(preset); err != nil {
		return nil, err
	}

	return g.applyDebugInventoryMutations(collector.mutations), nil
}

func (m *debugInventoryMutations) applyPreset(preset DebugInventoryPreset) error {
	handlers := map[DebugInventoryPreset]func(){
		DebugInventoryPresetProvisionsMax:           m.appendMaximumProvisions,
		DebugInventoryPresetReagentsAll99:           func() { m.appendReagents(party_state.ItemQuantitySmallMax, false) },
		DebugInventoryPresetReagentsAll20NoGarlic:   func() { m.appendReagents(debugInventoryReagentTwenty, true) },
		DebugInventoryPresetPotionsAll1:             func() { m.appendPotions(1) },
		DebugInventoryPresetPotionsAll99:            func() { m.appendPotions(party_state.ItemQuantitySmallMax) },
		DebugInventoryPresetScrollsAll1:             func() { m.appendScrolls(1) },
		DebugInventoryPresetScrollsAllMax:           func() { m.appendScrolls(party_state.ItemQuantityLargeMax) },
		DebugInventoryPresetEquipmentAllEquippable5: m.appendAllEquippableEquipment,
		DebugInventoryPresetEquipmentWeapons5:       m.appendWeapons,
		DebugInventoryPresetEquipmentArmour5:        m.appendArmour,
		DebugInventoryPresetEquipmentBowsNoAmmo5:    func() { m.appendBowsWithoutAmmo(debugInventoryGrantQuantity) },
	}
	handler, ok := handlers[preset]
	if !ok {
		return fmt.Errorf("%w: %q", errUnsupportedDebugInventoryPreset, preset)
	}
	handler()

	return nil
}

func (m *debugInventoryMutations) appendMaximumProvisions() {
	for _, provision := range debugInventoryProvisions() {
		_, maximum, err := m.game.debugProvisionValue(provision)
		if err != nil {
			panic(fmt.Sprintf("invalid built-in debug provision %q: %v", provision, err))
		}
		m.appendProvision(provision, maximum)
	}
}

func (m *debugInventoryMutations) appendReagents(amount uint16, excludeGarlic bool) {
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		after := amount
		if excludeGarlic && reagent == references.Garlic {
			after = 0
		}
		m.appendReagent(reagent, after)
	}
}

func (m *debugInventoryMutations) appendPotions(amount uint16) {
	for potion := references.Blue; potion <= references.White; potion++ {
		m.appendPotion(potion, amount)
	}
}

func (m *debugInventoryMutations) appendScrolls(amount uint16) {
	for scroll := references.ScrollVasLor; scroll <= references.ScrollAnTym; scroll++ {
		m.appendScroll(scroll, amount)
	}
}

func (m *debugInventoryMutations) appendEquipmentMatching(matches func(references.Equipment) bool, amount uint16) {
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		if matches(equipment) {
			m.appendEquipment(equipment, amount)
		}
	}
}

func (m *debugInventoryMutations) appendBowsWithoutAmmo(amount uint16) {
	for _, equipment := range []references.Equipment{references.Bow, references.Crossbow, references.MagicBow} {
		m.appendEquipment(equipment, amount)
	}
	m.appendEquipment(references.Arrows, 0)
	m.appendEquipment(references.Quarrels, 0)
}

func (m *debugInventoryMutations) appendAllEquippableEquipment() {
	m.appendEquipmentMatching(isEquippableEquipment, debugInventoryGrantQuantity)
}

func (m *debugInventoryMutations) appendWeapons() {
	m.appendEquipmentMatching(isNonAmmoWeapon, debugInventoryGrantQuantity)
}

func (m *debugInventoryMutations) appendArmour() {
	m.appendEquipmentMatching(isArmour, debugInventoryGrantQuantity)
}

func (m *debugInventoryMutations) appendProvision(provision DebugInventoryProvision, after uint16) {
	before, _, err := m.game.debugProvisionValue(provision)
	if err != nil {
		panic(fmt.Sprintf("invalid built-in debug provision %q: %v", provision, err))
	}
	if before != after {
		m.mutations = append(m.mutations, debugInventoryMutation{
			change: DebugInventoryChange{Path: "provisions." + string(provision), Before: before, After: after},
			apply:  func() { m.game.setDebugProvisionValue(provision, after) },
		})
	}
}

func (m *debugInventoryMutations) appendReagent(reagent references.Reagent, after uint16) {
	if before := m.snapshot.Reagents[reagent]; before != after {
		m.mutations = append(m.mutations, debugInventoryMutation{
			change: DebugInventoryChange{Path: fmt.Sprintf("reagents.%d", reagent), Before: before, After: after},
			apply:  func() { m.game.PartyState.Inventory.Reagent.Set(reagent, after) },
		})
	}
}

func (m *debugInventoryMutations) appendPotion(potion references.Potion, after uint16) {
	if before := m.snapshot.Potions[potion]; before != after {
		m.mutations = append(m.mutations, debugInventoryMutation{
			change: DebugInventoryChange{Path: fmt.Sprintf("potions.%d", potion), Before: before, After: after},
			apply:  func() { m.game.PartyState.Inventory.Potions.Set(potion, after) },
		})
	}
}

func (m *debugInventoryMutations) appendScroll(scroll references.Scroll, after uint16) {
	if before := m.snapshot.Scrolls[scroll]; before != after {
		m.mutations = append(m.mutations, debugInventoryMutation{
			change: DebugInventoryChange{Path: fmt.Sprintf("scrolls.%d", scroll), Before: before, After: after},
			apply:  func() { m.game.PartyState.Inventory.Scrolls.Set(scroll, after) },
		})
	}
}

func (m *debugInventoryMutations) appendEquipment(equipment references.Equipment, after uint16) {
	if before := m.snapshot.Equipment[equipment]; before != after {
		m.mutations = append(m.mutations, debugInventoryMutation{
			change: DebugInventoryChange{Path: fmt.Sprintf("equipment.%d", equipment), Before: before, After: after},
			apply:  func() { m.game.PartyState.Inventory.Equipment.Set(equipment, after) },
		})
	}
}

// ApplyDebugInventoryEquipmentGrantPreset adds five matching equipment items, preserving legacy additive semantics.
func (g *GameState) ApplyDebugInventoryEquipmentGrantPreset(
	preset DebugInventoryEquipmentGrantPreset,
) ([]DebugInventoryChange, error) {
	equipment, err := debugEquipmentForGrantPreset(preset)
	if err != nil {
		return nil, err
	}
	collector := debugInventoryMutations{
		game:      g,
		snapshot:  g.GetDebugInventorySnapshot(),
		mutations: make([]debugInventoryMutation, 0, len(equipment)+debugInventoryExtraMutations),
	}
	for _, item := range equipment {
		after := cappedAdd(
			collector.snapshot.Equipment[item],
			debugInventoryGrantQuantity,
			party_state.ItemQuantitySmallMax,
		)
		collector.appendEquipment(item, after)
	}
	if preset == DebugInventoryEquipmentGrantPresetBowsNoAmmo5 {
		collector.appendEquipment(references.Arrows, 0)
		collector.appendEquipment(references.Quarrels, 0)
	}

	return g.applyDebugInventoryMutations(collector.mutations), nil
}

func debugEquipmentForGrantPreset(preset DebugInventoryEquipmentGrantPreset) ([]references.Equipment, error) {
	switch preset {
	case DebugInventoryEquipmentGrantPresetAllEquippable5:
		return debugEquipmentMatching(isEquippableEquipment), nil
	case DebugInventoryEquipmentGrantPresetWeapons5:
		return debugEquipmentMatching(isNonAmmoWeapon), nil
	case DebugInventoryEquipmentGrantPresetArmour5:
		return debugEquipmentMatching(isArmour), nil
	case DebugInventoryEquipmentGrantPresetBowsNoAmmo5:
		return []references.Equipment{references.Bow, references.MagicBow, references.Crossbow}, nil
	default:
		return nil, fmt.Errorf("%w: %q", errUnsupportedDebugEquipmentPreset, preset)
	}
}

func debugEquipmentMatching(matches func(references.Equipment) bool) []references.Equipment {
	equipment := make([]references.Equipment, 0)
	for item := references.LeatherHelm; item <= references.Ankh; item++ {
		if matches(item) {
			equipment = append(equipment, item)
		}
	}

	return equipment
}

func isEquippableEquipment(equipment references.Equipment) bool {
	return equipment.GetSlotType() != references.SlotNone
}

func isNonAmmoWeapon(equipment references.Equipment) bool {
	return equipment.IsWeapon() && !equipment.IsAmmo()
}

func isArmour(equipment references.Equipment) bool {
	return equipment.IsArmour()
}

func (g *GameState) applyDebugInventoryMutations(mutations []debugInventoryMutation) []DebugInventoryChange {
	if len(mutations) == 0 {
		return []DebugInventoryChange{}
	}
	changes := make([]DebugInventoryChange, 0, len(mutations))
	for _, mutation := range mutations {
		mutation.apply()
		changes = append(changes, mutation.change)
	}

	return changes
}

// AddDebugInventoryPotion adds a positive amount to one potion stack, safely capped at 99.
func (g *GameState) AddDebugInventoryPotion(potion references.Potion, amount uint16) ([]DebugInventoryChange, error) {
	if potion < references.Blue || potion > references.White || amount == 0 {
		return nil, fmt.Errorf("%w: potion %d, amount %d", errInvalidDebugPotion, potion, amount)
	}

	return g.addDebugInventoryPotion(potion, amount)
}

// AddDebugInventoryScroll adds a positive amount to one scroll stack, safely capped at 9999.
func (g *GameState) AddDebugInventoryScroll(scroll references.Scroll, amount uint16) ([]DebugInventoryChange, error) {
	if scroll < references.ScrollVasLor || scroll > references.ScrollAnTym || amount == 0 {
		return nil, fmt.Errorf("%w: scroll %d, amount %d", errInvalidDebugScroll, scroll, amount)
	}
	before := g.PartyState.Inventory.Scrolls.Get(scroll)
	after := cappedAdd(before, amount, party_state.ItemQuantityLargeMax)
	if before == after {
		return []DebugInventoryChange{}, nil
	}
	g.PartyState.Inventory.Scrolls.Set(scroll, after)
	changes := []DebugInventoryChange{{Path: fmt.Sprintf("scrolls.%d", scroll), Before: before, After: after}}

	return changes, nil
}

func (g *GameState) addDebugInventoryPotion(potion references.Potion, amount uint16) ([]DebugInventoryChange, error) {
	before := g.PartyState.Inventory.Potions.Get(potion)
	after := cappedAdd(before, amount, party_state.ItemQuantitySmallMax)
	if before == after {
		return []DebugInventoryChange{}, nil
	}
	g.PartyState.Inventory.Potions.Set(potion, after)
	changes := []DebugInventoryChange{{Path: fmt.Sprintf("potions.%d", potion), Before: before, After: after}}

	return changes, nil
}

func cappedAdd(before, amount, maximum uint16) uint16 {
	if amount >= maximum-before {
		return maximum
	}

	return before + amount
}

func (g *GameState) debugProvisionValue(provision DebugInventoryProvision) (uint16, uint16, error) {
	switch provision {
	case DebugInventoryProvisionGold:
		return g.PartyState.Inventory.Gold.Get(), party_state.ItemQuantityLargeMax, nil
	case DebugInventoryProvisionFood:
		return g.PartyState.Inventory.Provisions.Food.Get(), party_state.ItemQuantityLargeMax, nil
	case DebugInventoryProvisionGems:
		return g.PartyState.Inventory.Provisions.Gems.Get(), party_state.ItemQuantitySmallMax, nil
	case DebugInventoryProvisionTorches:
		return g.PartyState.Inventory.Provisions.Torches.Get(), party_state.ItemQuantitySmallMax, nil
	case DebugInventoryProvisionKeys:
		return g.PartyState.Inventory.Provisions.Keys.Get(), party_state.ItemQuantitySmallMax, nil
	case DebugInventoryProvisionSkullKeys:
		return g.PartyState.Inventory.Provisions.SkullKeys.Get(), party_state.ItemQuantitySmallMax, nil
	default:
		return 0, 0, fmt.Errorf("%w: %q", errUnsupportedDebugProvision, provision)
	}
}

func (g *GameState) setDebugProvisionValue(provision DebugInventoryProvision, quantity uint16) {
	switch provision {
	case DebugInventoryProvisionGold:
		g.PartyState.Inventory.Gold.Set(quantity)
	case DebugInventoryProvisionFood:
		g.PartyState.Inventory.Provisions.Food.Set(quantity)
	case DebugInventoryProvisionGems:
		g.PartyState.Inventory.Provisions.Gems.Set(quantity)
	case DebugInventoryProvisionTorches:
		g.PartyState.Inventory.Provisions.Torches.Set(quantity)
	case DebugInventoryProvisionKeys:
		g.PartyState.Inventory.Provisions.Keys.Set(quantity)
	case DebugInventoryProvisionSkullKeys:
		g.PartyState.Inventory.Provisions.SkullKeys.Set(quantity)
	default:
		panic("validated debug inventory provision became invalid")
	}
}

func debugInventoryProvisions() []DebugInventoryProvision {
	return []DebugInventoryProvision{
		DebugInventoryProvisionGold, DebugInventoryProvisionFood, DebugInventoryProvisionGems,
		DebugInventoryProvisionTorches, DebugInventoryProvisionKeys, DebugInventoryProvisionSkullKeys,
	}
}
