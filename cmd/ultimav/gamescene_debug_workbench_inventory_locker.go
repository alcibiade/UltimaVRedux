package main

import (
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	inventoryProvisionCount           = 6
	inventoryConsumableCount          = 8
	inventoryPresetArgument           = "preset"
	inventoryExpectedSnapshotArgument = "expectedSnapshot"
	inventoryAmountArgument           = "amount"
)

type inventoryLockerResult struct {
	SnapshotFingerprint string                         `json:"snapshotFingerprint"`
	Provisions          []inventoryProvisionResult     `json:"provisions"`
	Reagents            []inventoryNamedQuantityResult `json:"reagents"`
	Potions             []inventoryNamedQuantityResult `json:"potions"`
	Scrolls             []inventoryNamedQuantityResult `json:"scrolls"`
	Equipment           []inventoryEquipmentResult     `json:"equipment"`
	Equipped            []inventoryEquippedResult      `json:"equipped"`
	UsablePresetReview  usableInventoryPresetTelemetry `json:"usablePresetReview"`
}

type usableInventoryPresetTelemetry struct {
	ReviewFingerprint string                              `json:"reviewFingerprint"`
	Changes           []usableInventoryPresetChangeResult `json:"changes"`
}

type usableInventoryPresetChangeResult struct {
	Category string `json:"category"`
	Path     string `json:"path"`
	Before   uint16 `json:"before"`
	After    uint16 `json:"after"`
}

type inventoryProvisionResult struct {
	Name     string `json:"name"`
	Quantity uint16 `json:"quantity"`
	Maximum  uint16 `json:"maximum"`
}

type inventoryNamedQuantityResult struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Quantity uint16 `json:"quantity"`
	Maximum  uint16 `json:"maximum"`
}

type inventoryEquipmentResult struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Slot     string `json:"slot"`
	Ammo     bool   `json:"ammo"`
	Quantity uint16 `json:"quantity"`
	Maximum  uint16 `json:"maximum"`
}

type inventoryEquippedResult struct {
	RosterIndex int                           `json:"rosterIndex"`
	Name        string                        `json:"name"`
	Slots       []inventoryEquippedSlotResult `json:"slots"`
}

type inventoryEquippedSlotResult struct {
	Slot          string `json:"slot"`
	EquipmentID   int    `json:"equipmentId"`
	EquipmentName string `json:"equipmentName"`
}

func executeInventoryLockerDebugCommand(
	request debugging.Request,
	state *game_state.GameState,
) (debugging.Result, bool) {
	if request.CommandID == debugging.CommandInventoryLockerStatus {
		return debugSuccessResult(request, "Inventory locker status loaded.", newInventoryLockerResult(state)), true
	}

	return executeInventoryLockerMutationCommand(request, state)
}

func executeInventoryLockerMutationCommand(
	request debugging.Request,
	state *game_state.GameState,
) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandInventoryProvisionSet:
		return executeInventoryProvisionSet(request, state), true
	case debugging.CommandInventoryReagentSet:
		return executeInventoryReagentSet(request, state), true
	case debugging.CommandInventoryPotionAdd:
		return executeInventoryPotionAdd(request, state), true
	case debugging.CommandInventoryScrollAdd:
		return executeInventoryScrollAdd(request, state), true
	case debugging.CommandInventoryEquipmentGrantPreset:
		return executeInventoryEquipmentGrantPreset(request, state), true
	case debugging.CommandInventoryEquipmentSet:
		return executeInventoryEquipmentSet(request, state), true
	case debugging.CommandInventoryEquipmentClear:
		return executeInventoryEquipmentClear(request, state), true
	case debugging.CommandInventoryProvisionPreset, debugging.CommandInventoryReagentPreset,
		debugging.CommandInventoryPotionPreset, debugging.CommandInventoryScrollPreset,
		debugging.CommandInventoryEquipmentPreset:
		return executeInventoryPreset(request, state), true
	default:
		return debugging.Result{
			SessionID: "", RequestID: "", CommandID: "", SceneGeneration: 0,
			Invocation:             nil,
			EffectiveExecutionMode: "", Success: false, Summary: "", ErrorCode: "", ErrorMessage: "",
			Changes: nil, Data: nil,
		}, false
	}
}

func executeInventoryReagentSet(request debugging.Request, state *game_state.GameState) debugging.Result {
	reagent, ok := inventoryReagent(mustDebugStringArgument(request, "reagent"))
	if !ok {
		return debugging.FailureResult(request, "invalid_inventory_operation", "unsupported reagent")
	}
	before := newInventoryLockerResult(state)
	if inventoryNamedQuantity(before.Reagents, reagent.ID()) != inventoryLockerQuantity(request, "expectedQuantity") {
		return debugging.FailureResult(request, "stale_inventory", "the reagent quantity changed after review")
	}
	changes, err := state.SetDebugInventoryReagent(reagent, inventoryLockerQuantity(request, "targetQuantity"))
	if err != nil {
		return debugging.FailureResult(request, "invalid_inventory_operation", err.Error())
	}

	return inventoryLockerMutationResult(request, debugging.ExecutionSynchronized, "Reagent updated.", state, changes)
}

func executeInventoryEquipmentSet(request debugging.Request, state *game_state.GameState) debugging.Result {
	equipment, ok := inventoryEquipment(mustDebugStringArgument(request, "equipment"))
	if !ok {
		return debugging.FailureResult(request, "invalid_inventory_operation", "unsupported equipment")
	}
	before := newInventoryLockerResult(state)
	currentQuantity := inventoryEquipmentQuantity(before.Equipment, equipment.ID())
	if currentQuantity != inventoryLockerQuantity(request, "expectedQuantity") {
		return debugging.FailureResult(request, "stale_inventory", "the equipment quantity changed after review")
	}
	changes, err := state.SetDebugInventoryEquipment(equipment, inventoryLockerQuantity(request, "targetQuantity"))
	if err != nil {
		return debugging.FailureResult(request, "invalid_inventory_operation", err.Error())
	}

	return inventoryLockerMutationResult(request, debugging.ExecutionSynchronized, "Equipment updated.", state, changes)
}

func executeInventoryProvisionSet(request debugging.Request, state *game_state.GameState) debugging.Result {
	provision, ok := inventoryProvision(mustDebugStringArgument(request, "provision"))
	if !ok {
		return debugging.FailureResult(request, "invalid_inventory_operation", "unsupported inventory provision")
	}
	before := newInventoryLockerResult(state)
	expectedQuantity := inventoryLockerQuantity(request, "expectedQuantity")
	if inventoryProvisionQuantity(before, string(provision)) != expectedQuantity {
		return debugging.FailureResult(request, "stale_inventory", "the provision quantity changed after review")
	}
	changes, err := state.SetDebugInventoryProvision(provision, inventoryLockerQuantity(request, "targetQuantity"))
	if err != nil {
		return debugging.FailureResult(request, "invalid_inventory_operation", err.Error())
	}

	return inventoryLockerMutationResult(request, debugging.ExecutionSynchronized, "Provision updated.", state, changes)
}

func executeInventoryPreset(request debugging.Request, state *game_state.GameState) debugging.Result {
	if !inventoryLockerMatchesReviewedSnapshot(request, state) {
		return debugging.FailureResult(request, "stale_inventory", "the inventory locker changed after review")
	}
	preset := game_state.DebugInventoryPreset(mustDebugStringArgument(request, inventoryPresetArgument))
	changes, err := state.ApplyDebugInventoryPreset(preset)
	if err != nil {
		return debugging.FailureResult(request, "invalid_inventory_operation", err.Error())
	}

	return inventoryLockerMutationResult(
		request, debugging.ExecutionAlteration, "Inventory preset applied.", state, changes,
	)
}

func executeInventoryEquipmentGrantPreset(request debugging.Request, state *game_state.GameState) debugging.Result {
	if !inventoryLockerMatchesReviewedSnapshot(request, state) {
		return debugging.FailureResult(request, "stale_inventory", "the inventory locker changed after review")
	}
	changes, err := state.ApplyDebugInventoryEquipmentGrantPreset(
		game_state.DebugInventoryEquipmentGrantPreset(mustDebugStringArgument(request, inventoryPresetArgument)),
	)
	if err != nil {
		return debugging.FailureResult(request, "invalid_inventory_operation", err.Error())
	}

	return inventoryLockerMutationResult(request, debugging.ExecutionAlteration, "Equipment granted.", state, changes)
}

func executeInventoryEquipmentClear(request debugging.Request, state *game_state.GameState) debugging.Result {
	if !inventoryLockerMatchesReviewedSnapshot(request, state) {
		return debugging.FailureResult(
			request,
			"stale_equipment_state",
			"the carried equipment or a roster loadout changed after review",
		)
	}
	changes := state.ClearDebugEquipmentState()
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionAlteration,
		"All carried and equipped items cleared.",
		newInventoryLockerResult(state),
	)
	result.Changes = make([]debugging.StateChange, 0, len(changes))
	for _, change := range changes {
		appendSemanticChange(&result.Changes, change.Path, change.Before, change.After)
	}

	return result
}

func executeInventoryPotionAdd(request debugging.Request, state *game_state.GameState) debugging.Result {
	potion, ok := inventoryPotion(mustDebugStringArgument(request, "potion"))
	if !ok {
		return debugging.FailureResult(request, "invalid_inventory_operation", "unsupported potion")
	}
	changes, err := state.AddDebugInventoryPotion(potion, inventoryLockerQuantity(request, inventoryAmountArgument))
	if err != nil {
		return debugging.FailureResult(request, "invalid_inventory_operation", err.Error())
	}

	return inventoryLockerMutationResult(request, debugging.ExecutionLiveSafe, "Potion added.", state, changes)
}

func executeInventoryScrollAdd(request debugging.Request, state *game_state.GameState) debugging.Result {
	scroll, ok := inventoryScroll(mustDebugStringArgument(request, "scroll"))
	if !ok {
		return debugging.FailureResult(request, "invalid_inventory_operation", "unsupported scroll")
	}
	changes, err := state.AddDebugInventoryScroll(scroll, inventoryLockerQuantity(request, inventoryAmountArgument))
	if err != nil {
		return debugging.FailureResult(request, "invalid_inventory_operation", err.Error())
	}

	return inventoryLockerMutationResult(request, debugging.ExecutionLiveSafe, "Scroll added.", state, changes)
}

func inventoryLockerQuantity(request debugging.Request, argument string) uint16 {
	quantity := mustDebugIntegerArgument(request, argument)
	if quantity < 0 || quantity > int(party_state.ItemQuantityLargeMax) {
		panic("validated inventory quantity is outside catalog bounds")
	}

	return uint16(quantity)
}

func inventoryLockerMatchesReviewedSnapshot(request debugging.Request, state *game_state.GameState) bool {
	expectedSnapshot := mustDebugStringArgument(request, inventoryExpectedSnapshotArgument)
	actualSnapshot := inventoryLockerFingerprint(newInventoryLockerResult(state))

	return expectedSnapshot == actualSnapshot
}

func inventoryLockerMutationResult(
	request debugging.Request,
	mode debugging.ExecutionMode,
	summary string,
	state *game_state.GameState,
	changes []game_state.DebugInventoryChange,
) debugging.Result {
	result := debugSuccessResultWithMode(request, mode, summary, newInventoryLockerResult(state))
	result.Changes = make([]debugging.StateChange, 0, len(changes))
	for _, change := range changes {
		appendSemanticChange(&result.Changes, "inventory."+change.Path+".quantity", change.Before, change.After)
	}

	return result
}

func newInventoryLockerResult(state *game_state.GameState) inventoryLockerResult {
	snapshot := state.GetDebugInventorySnapshot()
	review, err := state.ReviewUsableInventoryPreset()
	if err != nil {
		panic("invalid usable inventory preset catalog: " + err.Error())
	}
	result := inventoryLockerResult{
		SnapshotFingerprint: "",
		Provisions:          make([]inventoryProvisionResult, 0, inventoryProvisionCount),
		Reagents:            make([]inventoryNamedQuantityResult, 0, inventoryConsumableCount),
		Potions:             make([]inventoryNamedQuantityResult, 0, inventoryConsumableCount),
		Scrolls:             make([]inventoryNamedQuantityResult, 0, inventoryConsumableCount),
		Equipment:           make([]inventoryEquipmentResult, 0, int(references.Ankh-references.LeatherHelm)+1),
		Equipped:            make([]inventoryEquippedResult, 0, party_state.NRosterSlots),
		UsablePresetReview:  newUsableInventoryPresetTelemetry(review.Fingerprint, review.Changes),
	}
	appendInventoryProvisions(&result, snapshot)
	appendInventoryReagents(&result, snapshot)
	appendInventoryPotions(&result, snapshot)
	appendInventoryScrolls(&result, snapshot)
	appendInventoryEquipment(&result, snapshot)
	appendInventoryEquipped(&result, state)
	result.SnapshotFingerprint = inventoryLockerFingerprint(result)

	return result
}

func newUsableInventoryPresetTelemetry(
	fingerprint string,
	changes []game_state.DebugInventoryChange,
) usableInventoryPresetTelemetry {
	result := usableInventoryPresetTelemetry{
		ReviewFingerprint: fingerprint,
		Changes:           make([]usableInventoryPresetChangeResult, 0, len(changes)),
	}
	for _, change := range changes {
		category, _, _ := strings.Cut(change.Path, ".")
		result.Changes = append(result.Changes, usableInventoryPresetChangeResult{
			Category: category, Path: change.Path, Before: change.Before, After: change.After,
		})
	}

	return result
}

func appendInventoryProvisions(result *inventoryLockerResult, snapshot party_state.InventoryLockerSnapshot) {
	for _, provision := range debugInventoryProvisions() {
		result.Provisions = append(result.Provisions, inventoryProvisionResult{
			Name: string(provision), Quantity: snapshot.Provisions[inventoryProvisionReference(provision)],
			Maximum: inventoryProvisionMaximum(provision),
		})
	}
}

func appendInventoryReagents(result *inventoryLockerResult, snapshot party_state.InventoryLockerSnapshot) {
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		result.Reagents = append(result.Reagents, inventoryNamedQuantityResult{
			ID:       reagent.ID(),
			Name:     inventoryReagentName(reagent),
			Quantity: snapshot.Reagents[reagent],
			Maximum:  party_state.ItemQuantitySmallMax,
		})
	}
}

func appendInventoryPotions(result *inventoryLockerResult, snapshot party_state.InventoryLockerSnapshot) {
	for potion := references.Blue; potion <= references.White; potion++ {
		result.Potions = append(result.Potions, inventoryNamedQuantityResult{
			ID:       potion.ID(),
			Name:     inventoryPotionName(potion),
			Quantity: snapshot.Potions[potion],
			Maximum:  party_state.ItemQuantitySmallMax,
		})
	}
}

func appendInventoryScrolls(result *inventoryLockerResult, snapshot party_state.InventoryLockerSnapshot) {
	for scroll := references.ScrollVasLor; scroll <= references.ScrollAnTym; scroll++ {
		result.Scrolls = append(result.Scrolls, inventoryNamedQuantityResult{
			ID:       scroll.ID(),
			Name:     inventoryScrollName(scroll),
			Quantity: snapshot.Scrolls[scroll],
			Maximum:  party_state.ItemQuantityLargeMax,
		})
	}
}

func appendInventoryEquipment(result *inventoryLockerResult, snapshot party_state.InventoryLockerSnapshot) {
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		result.Equipment = append(result.Equipment, inventoryEquipmentResult{
			ID: equipment.ID(), Name: equipment.String(), Category: equipment.GetCategory().String(),
			Slot: equipment.GetSlotType().String(), Ammo: equipment.IsAmmo(), Quantity: snapshot.Equipment[equipment],
			Maximum: party_state.ItemQuantitySmallMax,
		})
	}
}

func appendInventoryEquipped(result *inventoryLockerResult, state *game_state.GameState) {
	state.PartyState.ForEachCharacter(func(rosterIndex int, character *party_state.PlayerCharacter) {
		entry := inventoryEquippedResult{
			RosterIndex: rosterIndex,
			Name:        character.GetNameAsString(),
			Slots:       make([]inventoryEquippedSlotResult, 0, len(inventoryEquipmentSlots())),
		}
		for _, slot := range inventoryEquipmentSlots() {
			equipment := character.GetEquippedItem(slot)
			name := ""
			if equipment != references.NoEquipment {
				name = equipment.String()
			}
			entry.Slots = append(entry.Slots, inventoryEquippedSlotResult{
				Slot: slot.String(), EquipmentID: equipment.ID(), EquipmentName: name,
			})
		}
		result.Equipped = append(result.Equipped, entry)
	})
}

func inventoryEquipmentSlots() []references.EquipmentSlot {
	return []references.EquipmentSlot{
		references.SlotHelm,
		references.SlotArmour,
		references.SlotWeapon,
		references.SlotShield,
		references.SlotRing,
		references.SlotAmulet,
	}
}

func inventoryLockerFingerprint(result inventoryLockerResult) string {
	result.SnapshotFingerprint = ""

	return string(mustDebugJSON(result))
}

func inventoryProvisionQuantity(result inventoryLockerResult, provision string) uint16 {
	for _, entry := range result.Provisions {
		if entry.Name == provision {
			return entry.Quantity
		}
	}
	panic("complete inventory locker result omitted provision")
}

func inventoryNamedQuantity(entries []inventoryNamedQuantityResult, id int) uint16 {
	for _, entry := range entries {
		if entry.ID == id {
			return entry.Quantity
		}
	}
	panic("complete inventory locker result omitted named quantity")
}

func inventoryEquipmentQuantity(entries []inventoryEquipmentResult, id int) uint16 {
	for _, entry := range entries {
		if entry.ID == id {
			return entry.Quantity
		}
	}
	panic("complete inventory locker result omitted equipment quantity")
}

func inventoryProvision(value string) (game_state.DebugInventoryProvision, bool) {
	for _, provision := range debugInventoryProvisions() {
		if value == string(provision) {
			return provision, true
		}
	}

	return "", false
}

func debugInventoryProvisions() []game_state.DebugInventoryProvision {
	return []game_state.DebugInventoryProvision{
		game_state.DebugInventoryProvisionGold,
		game_state.DebugInventoryProvisionFood,
		game_state.DebugInventoryProvisionGems,
		game_state.DebugInventoryProvisionTorches,
		game_state.DebugInventoryProvisionKeys,
		game_state.DebugInventoryProvisionSkullKeys,
	}
}

func inventoryPotion(value string) (references.Potion, bool) {
	id, err := strconv.Atoi(value)
	if err != nil || id < int(references.Blue) || id > int(references.White) {
		return references.Blue, false
	}

	return references.Potion(id), true
}

func inventoryReagent(value string) (references.Reagent, bool) {
	id, err := strconv.Atoi(value)
	if err != nil || id < references.SulfurAsh.ID() || id > references.MandrakeRoot.ID() {
		return references.SulfurAsh, false
	}

	return references.Reagent(id), true
}

func inventoryEquipment(value string) (references.Equipment, bool) {
	id, err := strconv.Atoi(value)
	if err != nil || id < references.LeatherHelm.ID() || id > references.Ankh.ID() {
		return references.LeatherHelm, false
	}

	return references.Equipment(id), true
}

func inventoryScroll(value string) (references.Scroll, bool) {
	id, err := strconv.Atoi(value)
	if err != nil || id < int(references.ScrollVasLor) || id > int(references.ScrollAnTym) {
		return references.ScrollVasLor, false
	}

	return references.Scroll(id), true
}

func inventoryProvisionReference(provision game_state.DebugInventoryProvision) references.Provision {
	values := map[game_state.DebugInventoryProvision]references.Provision{
		game_state.DebugInventoryProvisionGold:      references.Gold,
		game_state.DebugInventoryProvisionFood:      references.Food,
		game_state.DebugInventoryProvisionGems:      references.Gem,
		game_state.DebugInventoryProvisionTorches:   references.Torches,
		game_state.DebugInventoryProvisionKeys:      references.Key,
		game_state.DebugInventoryProvisionSkullKeys: references.SkullKeys,
	}

	return values[provision]
}

func inventoryProvisionMaximum(provision game_state.DebugInventoryProvision) uint16 {
	if provision == game_state.DebugInventoryProvisionGold || provision == game_state.DebugInventoryProvisionFood {
		return party_state.ItemQuantityLargeMax
	}

	return party_state.ItemQuantitySmallMax
}

func inventoryReagentName(reagent references.Reagent) string {
	return []string{
		"Sulfurous Ash", "Ginseng", "Garlic", "Spider Silk", "Blood Moss", "Black Pearl", "Nightshade", "Mandrake Root",
	}[reagent]
}

func inventoryPotionName(potion references.Potion) string {
	return []string{"Blue", "Yellow", "Red", "Green", "Orange", "Purple", "Black", "White"}[potion]
}

func inventoryScrollName(scroll references.Scroll) string {
	return []string{
		"Vas Lor", "Rel Hur", "In Sanct", "In An", "In Quas Wis", "Kal Xen Corp", "In Mani Corp", "An Tym",
	}[scroll]
}
