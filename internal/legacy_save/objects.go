package legacy_save

import (
	"fmt"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const duplicateSingletonLooseItemWarning = "duplicate_singleton_loose_item"

type legacyObjectSource struct {
	path     string
	records  [objectCount]ObjectRecord
	world    references.World
	location int
	smallMap bool
	active   bool
}

//nolint:funlen // Source authority branches are clearest beside the ordered-list conversion.
func convertMapObjects(bundle Bundle, snapshot *save_snapshot.SaveSnapshot, warnings *[]ConversionWarning) {
	game := bundle.Game
	sources := []legacyObjectSource{
		{
			path: "ool.britannia", records: bundle.Objects.Britannia, world: references.OVERWORLD,
			location: int(references.BritanniaUnderworld), smallMap: false, active: false,
		},
		{
			path: "ool.underworld", records: bundle.Objects.Underworld, world: references.UNDERWORLD,
			location: int(references.BritanniaUnderworld), smallMap: false, active: false,
		},
	}
	mapType := references.Location(game.Position.Location).GetMapType()
	switch mapType {
	case references.LargeMapType:
		activeWorld := references.World(signedFloor(game.Position.Floor))
		active := legacyObjectSource{
			path: "gam.activeObjects", records: game.ActiveObjects, world: activeWorld,
			location: int(references.BritanniaUnderworld), smallMap: false, active: true,
		}
		if activeWorld == references.OVERWORLD {
			sources[0] = active
		} else {
			sources[1] = active
		}
	case references.SmallMapType:
		sources = append(sources, legacyObjectSource{
			path: "gam.activeObjects", records: game.ActiveObjects, world: references.OVERWORLD,
			location: int(game.Position.Location), smallMap: true, active: true,
		})
		snapshot.MarkMapUnitsForReconstruction()
		addConversionWarning(
			warnings,
			"mapUnits",
			"legacy_small_map_npcs_reconstructed",
			"authored small-map NPCs are reconstructed while supported semantic objects are overlaid "+
				"from the active object list",
		)
	case references.CombatMapType, references.DungeonMapType, references.CampingMapType:
		// These contexts do not use the active list as persistent surface authority.
	}

	orders := make(map[[2]int]int)
	snapshot.MapUnits = make([]save_snapshot.MapUnitSnapshot, 0)
	for _, source := range sources {
		for slot := 1; slot < len(source.records); slot++ {
			record := source.records[slot]
			if record.Tile == 0 {
				continue
			}
			path := fmt.Sprintf("%s[%d]", source.path, slot)
			// Original noncombat lookup/rendering ignores records on another level.
			if !source.smallMap && signedFloor(record.Level) != int(source.world) {
				addConversionWarning(
					warnings,
					path+".level",
					"off_world_object_omitted",
					"object level does not match its authoritative world list and the object was omitted",
				)

				continue
			}
			if convertLooseItem(game, source, record, path, snapshot, warnings) {
				continue
			}
			unit, supported := convertObjectRecord(game, source, slot, record, path, warnings)
			if !supported {
				continue
			}
			controller := [2]int{unit.Position.Location, unit.World}
			unit.ControllerOrder = orders[controller]
			orders[controller]++
			snapshot.MapUnits = append(snapshot.MapUnits, unit)
		}
	}
	convertPendingShipDelivery(game, snapshot, warnings, orders)
	sort.Slice(snapshot.MapUnits, func(i, j int) bool {
		left, right := snapshot.MapUnits[i], snapshot.MapUnits[j]
		if left.Position.Location != right.Position.Location {
			return left.Position.Location < right.Position.Location
		}
		if left.World != right.World {
			return left.World < right.World
		}

		return left.ControllerOrder < right.ControllerOrder
	})
}

func convertLooseItem(
	game SavedGame,
	source legacyObjectSource,
	record ObjectRecord,
	path string,
	snapshot *save_snapshot.SaveSnapshot,
	warnings *[]ConversionWarning,
) bool {
	item, recognized, warningCode := looseItemSnapshot(game, source, record)
	if !recognized {
		return false
	}
	if !source.active {
		addConversionWarning(
			warnings,
			path,
			"inactive_loose_item_omitted",
			"native item stacks retain only the active map, so this inactive-world loose item was omitted",
		)

		return true
	}
	if signedFloor(record.Level) != signedFloor(game.Position.Floor) {
		addConversionWarning(
			warnings,
			path,
			"inactive_floor_loose_item_omitted",
			"native item stacks retain only the active floor, so this off-floor loose item was omitted",
		)

		return true
	}
	if warningCode != "" {
		addConversionWarning(
			warnings,
			path+".number",
			warningCode,
			"loose item payload has no safe semantic native mapping and was omitted",
		)

		return true
	}
	if record.Tile == originalMoonstoneShape {
		snapshot.Progression.Moonstones[record.Number].IsFound = true
	}
	position := save_snapshot.PositionSnapshot{
		Location: source.location, X: int(record.X), Y: int(record.Y), Floor: signedFloor(record.Level),
	}
	for index := range snapshot.ItemStacks {
		if snapshot.ItemStacks[index].Position == position {
			snapshot.ItemStacks[index].Items = prependItemSnapshot(snapshot.ItemStacks[index].Items, item)

			return true
		}
	}
	snapshot.ItemStacks = append(snapshot.ItemStacks, save_snapshot.ItemStackSnapshot{
		Position: position,
		Items:    []save_snapshot.ItemSnapshot{item},
	})

	return true
}

func prependItemSnapshot(
	items []save_snapshot.ItemSnapshot,
	item save_snapshot.ItemSnapshot,
) []save_snapshot.ItemSnapshot {
	return append([]save_snapshot.ItemSnapshot{item}, items...)
}

//nolint:cyclop,funlen,gocyclo // The original tile-to-payload table is safer to audit as one switch.
func looseItemSnapshot(
	game SavedGame,
	source legacyObjectSource,
	record ObjectRecord,
) (save_snapshot.ItemSnapshot, bool, string) {
	quantity := uint16(record.Number)
	switch record.Tile {
	case originalGoldShape:
		return quantityItem(references.Gold, quantity)
	case originalFoodShape:
		return quantityItem(references.Food, quantity)
	case originalTorchShape:
		return quantityItem(references.Torches, quantity)
	case originalGemShape:
		return quantityItem(references.Gem, quantity)
	case originalKeyShape:
		provision := references.Key
		quantity = uint16(record.Number & originalKeyQuantityMask)
		if record.Number&originalFrigateDelivery != 0 {
			provision = references.SkullKeys
		}

		return quantityItem(provision, quantity)
	case originalPotionShape:
		return indexedItem(references.ItemTypePotion, int(record.Number), int(references.Blue), int(references.White))
	case originalScrollShape:
		if record.Number == originalNothing {
			if game.Inventory.SpecialItems.HMSCapePlans != 0 {
				return emptyItemSnapshot(), true, duplicateSingletonLooseItemWarning
			}

			return itemSnapshot(references.HMSCape, 1), true, ""
		}

		return indexedItem(
			references.ItemTypeScroll,
			int(record.Number),
			int(references.ScrollVasLor),
			int(references.ScrollAnTym),
		)
	case originalHelmShape:
		return indexedItem(
			references.ItemTypeEquipment,
			int(record.Number),
			int(references.LeatherHelm),
			int(references.SpikedHelm),
		)
	case originalShieldShape:
		return indexedItem(
			references.ItemTypeEquipment,
			int(record.Number),
			int(references.SmallShield),
			int(references.JewelShield),
		)
	case originalArmorShape:
		return indexedItem(
			references.ItemTypeEquipment,
			int(record.Number),
			int(references.ClothArmour),
			int(references.MysticArmour),
		)
	case originalWeaponShape:
		item, recognized, warning := indexedItem(
			references.ItemTypeEquipment,
			int(record.Number),
			int(references.Dagger),
			int(references.MysticSword),
		)
		if warning == "" &&
			(record.Number == byte(references.Arrows) || record.Number == byte(references.Quarrels)) {
			item.Quantity = 5
		}

		return item, recognized, warning
	case originalRingShape:
		return indexedItem(
			references.ItemTypeEquipment,
			int(record.Number),
			int(references.RingInvisibility),
			int(references.RingRegeneration),
		)
	case originalAmuletShape:
		return indexedItem(
			references.ItemTypeEquipment,
			int(record.Number),
			int(references.AmuletOfTurning),
			int(references.Ankh),
		)
	case originalMoonstoneShape:
		index := int(record.Number)
		if index < 0 || index >= len(game.Moonstones) || snapshotMoonstoneMismatch(game.Moonstones[index], source, record) {
			return emptyItemSnapshot(), true, "moonstone_state_mismatch"
		}

		return itemSnapshot(references.Moonstone(index), 1), true, ""
	case originalShardShape:
		if record.Number < originalShardPayloadBase || record.Number > originalShardPayloadBase+2 {
			return emptyItemSnapshot(), true, "invalid_loose_item_number"
		}
		shardIndex := record.Number - originalShardPayloadBase
		shard := references.ShardOfFalsehood + references.QuestItem(shardIndex)
		if game.Inventory.Shards.Source[shardIndex] != 0 {
			return emptyItemSnapshot(), true, duplicateSingletonLooseItemWarning
		}

		return itemSnapshot(shard, 1), true, ""
	case originalCrownShape:
		if game.Inventory.Crown != 0 {
			return emptyItemSnapshot(), true, duplicateSingletonLooseItemWarning
		}

		return itemSnapshot(references.Crown, 1), true, ""
	case originalSceptreShape:
		if game.Inventory.Sceptre != 0 {
			return emptyItemSnapshot(), true, duplicateSingletonLooseItemWarning
		}

		return itemSnapshot(references.Sceptre, 1), true, ""
	case originalQuestAmuletShape:
		if game.Inventory.Amulet != 0 {
			return emptyItemSnapshot(), true, duplicateSingletonLooseItemWarning
		}

		return itemSnapshot(references.Amulet, 1), true, ""
	default:
		return emptyItemSnapshot(), false, ""
	}
}

func quantityItem(item references.Item, quantity uint16) (save_snapshot.ItemSnapshot, bool, string) {
	if quantity == 0 {
		return emptyItemSnapshot(), true, "zero_quantity_loose_item_omitted"
	}

	return itemSnapshot(item, quantity), true, ""
}

func indexedItem(itemType references.ItemType, id, first, last int) (save_snapshot.ItemSnapshot, bool, string) {
	if id < first || id > last {
		return emptyItemSnapshot(), true, "invalid_loose_item_number"
	}

	return save_snapshot.ItemSnapshot{Type: int(itemType), ID: id, Quantity: 1}, true, ""
}

func itemSnapshot(item references.Item, quantity uint16) save_snapshot.ItemSnapshot {
	return save_snapshot.ItemSnapshot{Type: int(item.Type()), ID: item.ID(), Quantity: quantity}
}

func emptyItemSnapshot() save_snapshot.ItemSnapshot {
	var item save_snapshot.ItemSnapshot

	return item
}

func snapshotMoonstoneMismatch(stone Moonstone, source legacyObjectSource, record ObjectRecord) bool {
	return stone.Map == originalNothing || int(stone.Map) != source.location || stone.Level != record.Level ||
		stone.X != record.X || stone.Y != record.Y
}

//nolint:cyclop,funlen,nestif // Object categories intentionally mirror the original tile classifier.
func convertObjectRecord(
	game SavedGame,
	source legacyObjectSource,
	slot int,
	record ObjectRecord,
	path string,
	warnings *[]ConversionWarning,
) (save_snapshot.MapUnitSnapshot, bool) {
	base := record.Tile & originalObjectBaseMask
	unit := newLegacyMapUnit(source, slot, record)

	if vehicle, ok := convertObjectVehicle(record, path, warnings); ok {
		unit.Kind = save_snapshot.MapUnitKindVehicle
		unit.Vehicle = vehicle

		return unit, true
	}
	if base == originalPirateShape {
		unit.Kind = save_snapshot.MapUnitKindEnemy
		unit.ReferenceID = int(references.MonsterStocks)
		unit.EnemyVariant = "pirate"
		unit.EnemyCurrentHP = int(record.Number)
		unit.EnemyFacing = originalDirection(record.Shape & originalDirectionMask)

		return unit, true
	}
	if record.Tile == originalChestShape {
		number := int(record.Number)
		unit.Kind = save_snapshot.MapUnitKindChest
		unit.SurfaceChestNumber = &number

		return unit, true
	}
	if base >= originalNPCShape {
		monster := references.MonsterType((base - originalNPCShape) / originalMonsterFrames)
		if monster >= references.MonsterSeaHorse && monster.IsValid() && monster != references.MonsterForceField {
			if source.smallMap && monster != references.MonsterShadowlord {
				addConversionWarning(
					warnings,
					path,
					"legacy_small_map_actor_reconstructed",
					"authored small-map actor state cannot be associated with an exact NPC reference and was reconstructed",
				)

				return emptyMapUnitSnapshot(), false
			}
			unit.Kind = save_snapshot.MapUnitKindEnemy
			unit.ReferenceID = int(monster)
			unit.EnemyVariant = "standard"
			unit.EnemyCurrentHP = references.MonsterBaseHitPoints(monster)
			switch monster { //nolint:exhaustive // Only variants with distinct persisted state need cases.
			case references.MonsterWhirlpool:
				unit.EnemyVariant = "whirlpool"
			case references.MonsterShadowlord:
				if !source.smallMap || game.CurrentShadowlord >= save_snapshot.ShadowlordCount {
					addConversionWarning(
						warnings,
						path,
						"ambiguous_shadowlord_identity",
						"Shadowlord object lacks a provable identity and was omitted",
					)

					return emptyMapUnitSnapshot(), false
				}
				identity := int(game.CurrentShadowlord)
				unit.EnemyVariant = "shadowlord"
				unit.Shadowlord = &identity
			}

			return unit, true
		}
	}
	if source.smallMap && base >= originalNPCShape {
		return emptyMapUnitSnapshot(), false
	}

	code := "unsupported_object_record"
	message := "legacy object record has no safe semantic native mapping and was omitted"
	if record.Tile == originalDeadPCShape || record.Tile == originalDeadMonsterShape {
		code = "transient_corpse_omitted"
		message = "legacy corpse state is transient and was omitted"
	}
	addConversionWarning(warnings, path, code, message)

	return emptyMapUnitSnapshot(), false
}

func newLegacyMapUnit(source legacyObjectSource, slot int, record ObjectRecord) save_snapshot.MapUnitSnapshot {
	var unit save_snapshot.MapUnitSnapshot
	unit.World = int(source.world)
	unit.NPCNum = slot
	unit.Position = save_snapshot.PositionSnapshot{
		Location: source.location,
		X:        int(record.X),
		Y:        int(record.Y),
		Floor:    signedFloor(record.Level),
	}
	unit.Visible = true
	unit.AIOverrideProvenance = string(references.AIOverrideProvenanceNone)
	unit.CurrentPath = make([]save_snapshot.PositionSnapshot, 0)

	return unit
}

func emptyMapUnitSnapshot() save_snapshot.MapUnitSnapshot {
	var unit save_snapshot.MapUnitSnapshot

	return unit
}

func convertObjectVehicle(
	record ObjectRecord,
	path string,
	warnings *[]ConversionWarning,
) (*save_snapshot.VehicleSnapshot, bool) {
	var vehicleType references.VehicleType
	var direction int
	base := record.Tile & originalObjectBaseMask
	switch {
	case record.Tile&0xfe == originalHorseObjectShape:
		vehicleType = references.HorseVehicle
		direction = horizontalShapeDirection(record.Shape)
	case record.Tile == originalCarpetObjectShape:
		vehicleType = references.CarpetVehicle
		direction = int(references.Right)
	case base == originalSkiffShape:
		vehicleType = references.SkiffVehicle
		direction = originalDirection(record.Shape & originalDirectionMask)
	case base == originalShipShape || base == originalAnchorShape:
		vehicleType = references.FrigateVehicle
		direction = originalDirection(record.Shape & originalDirectionMask)
	default:
		return nil, false
	}

	vehicle := &save_snapshot.VehicleSnapshot{
		VesselID:          0, // Unregistered: original saves have no Atlas vessel IDs
		Type:              int(vehicleType),
		CurrentDirection:  direction,
		PreviousDirection: direction,
		Skiffs:            0,
		OwnedByNPC:        false,
		Hull:              defaultShipHull,
		SailState:         0,
		HasHMSCape:        false,
		PendingDelivery:   false,
	}
	if vehicleType == references.FrigateVehicle {
		vehicle.Hull = int(record.Number)
		if vehicle.Hull < 1 || vehicle.Hull > defaultShipHull {
			addConversionWarning(
				warnings,
				path+".number",
				"invalid_ship_hull",
				"frigate hull was outside 1 through 99 and the object was omitted",
			)

			return nil, false
		}
		vehicle.Skiffs = int(record.Misc & originalSkiffCountMask)
		if vehicle.Skiffs > references.MaxFrigateSkiffs {
			vehicle.Skiffs = references.MaxFrigateSkiffs
			addConversionWarning(
				warnings,
				path+".misc",
				"redux_capacity_clamped",
				fmt.Sprintf("frigate skiffs were clamped to %d", references.MaxFrigateSkiffs),
			)
		}
	}

	return vehicle, true
}

func convertPendingShipDelivery(
	game SavedGame,
	snapshot *save_snapshot.SaveSnapshot,
	warnings *[]ConversionWarning,
	orders map[[2]int]int,
) {
	if game.Dungeon.Skiffs < originalDeliveryMarker {
		return
	}
	if game.Position.Location == 0 && signedFloor(game.Position.Floor) == int(references.OVERWORLD) {
		addConversionWarning(
			warnings,
			"gam.skiffs",
			"inconsistent_pending_delivery",
			"pending ship delivery marker on the active Britannia map was omitted",
		)

		return
	}
	vehicle := pendingDeliveryVehicle(game, warnings)
	controller := [2]int{int(references.BritanniaUnderworld), int(references.OVERWORLD)}
	slot, ok := firstUnusedLegacySlot(snapshot.MapUnits, references.OVERWORLD)
	if !ok {
		addConversionWarning(
			warnings,
			"gam.skiffs",
			"object_capacity_exhausted",
			"pending ship delivery was omitted because Britannia has no free object slot",
		)

		return
	}
	var unit save_snapshot.MapUnitSnapshot
	unit.World = int(references.OVERWORLD)
	unit.ControllerOrder = orders[controller]
	unit.Kind = save_snapshot.MapUnitKindVehicle
	unit.NPCNum = slot
	unit.Position = save_snapshot.PositionSnapshot{
		Location: int(references.BritanniaUnderworld),
		X:        int(game.Ship.X),
		Y:        int(game.Ship.Y),
		Floor:    int(references.OVERWORLD),
	}
	unit.Visible = true
	unit.AIOverrideProvenance = string(references.AIOverrideProvenanceNone)
	unit.CurrentPath = make([]save_snapshot.PositionSnapshot, 0)
	unit.Vehicle = vehicle
	snapshot.MapUnits = append(snapshot.MapUnits, unit)
}

// pendingDeliveryVehicle returns the skiff or frigate awaiting delivery
// according to the GAM marker.
func pendingDeliveryVehicle(game SavedGame, warnings *[]ConversionWarning) *save_snapshot.VehicleSnapshot {
	vehicle := &save_snapshot.VehicleSnapshot{
		VesselID:          0, // Unregistered: original saves have no Atlas vessel IDs
		Type:              int(references.SkiffVehicle),
		CurrentDirection:  int(references.Right),
		PreviousDirection: int(references.Right),
		Skiffs:            0,
		OwnedByNPC:        false,
		Hull:              defaultShipHull,
		SailState:         0,
		HasHMSCape:        false,
		PendingDelivery:   true,
	}
	if game.Dungeon.Skiffs&originalFrigateDelivery == 0 {
		return vehicle
	}
	vehicle.Type = int(references.FrigateVehicle)
	vehicle.Skiffs = int(game.Dungeon.Skiffs & originalSkiffCountMask)
	// The original adds skiffs to an awaiting frigate without limit; Redux
	// clamps it like a moored one rather than refusing the whole import.
	if vehicle.Skiffs > references.MaxFrigateSkiffs {
		vehicle.Skiffs = references.MaxFrigateSkiffs
		addConversionWarning(
			warnings,
			"gam.skiffs",
			"redux_capacity_clamped",
			fmt.Sprintf("pending frigate skiffs were clamped to %d", references.MaxFrigateSkiffs),
		)
	}

	return vehicle
}

func firstUnusedLegacySlot(units []save_snapshot.MapUnitSnapshot, world references.World) (int, bool) {
	used := make(map[int]struct{})
	for _, unit := range units {
		if unit.World == int(world) && unit.Position.Location == int(references.BritanniaUnderworld) {
			used[unit.NPCNum] = struct{}{}
		}
	}
	for slot := 1; slot < objectCount; slot++ {
		if _, exists := used[slot]; !exists {
			return slot, true
		}
	}

	return 0, false
}
