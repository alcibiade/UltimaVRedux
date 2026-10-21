package references

// SearchLocationCount is the number of fixed search records; ref:QUEST-0002.
const SearchLocationCount = 113

// SearchObjectClass is an original fixed-search world-object class; ref:QUEST-0002.
type SearchObjectClass byte

//nolint:revive // Original world-object constants intentionally share this source-class block.
const (
	SearchObjectGold        SearchObjectClass = 0x02
	SearchObjectPotion      SearchObjectClass = 0x03
	SearchObjectScroll      SearchObjectClass = 0x04
	SearchObjectWeapon      SearchObjectClass = 0x05
	SearchObjectShield      SearchObjectClass = 0x06
	SearchObjectKey         SearchObjectClass = 0x07
	SearchObjectGem         SearchObjectClass = 0x08
	SearchObjectHelm        SearchObjectClass = 0x09
	SearchObjectRing        SearchObjectClass = 0x0a
	SearchObjectArmour      SearchObjectClass = 0x0b
	SearchObjectAmulet      SearchObjectClass = 0x0c
	SearchObjectTorch       SearchObjectClass = 0x0d
	SearchObjectFood        SearchObjectClass = 0x0f
	SearchObjectMoonstone   SearchObjectClass = 0x19
	SearchObjectDeadPC      SearchObjectClass = 0x1e
	SearchObjectDeadMonster SearchObjectClass = 0x1f
)

// SearchLocationReference is one fixed-search record in original order; ref:QUEST-0002.
type SearchLocationReference struct {
	Index       int
	ObjectClass SearchObjectClass
	Payload     byte
	// ItemType and Quantity are the decoded carriable result. Corpse records
	// leave ItemType nil and Quantity zero; ObjectClass/Payload remain authoritative.
	ItemType     Item
	Quantity     uint16
	MapNumber    int
	Level        int
	X            int
	Y            int
	Requirements SearchLocationRequirements
}

// SearchLocationRequirements represents special conditions for finding items.
type SearchLocationRequirements struct {
	IsSpecialItem         bool
	RequiresNoKeys        bool
	IsSkullKeys           bool
	RequiresNoGlassSwords bool
	IsRepeatable          bool
}

// SearchLocationDecodedItem is the class-aware interpretation of one objt/objh pair.
type SearchLocationDecodedItem struct {
	RawObjectType     byte
	RawPayload        byte
	Classification    string
	Category          string
	Item              Item
	ItemID            int
	ItemName          string
	PickupQuantity    int
	DiscoveryText     string
	QuantitySemantics string
	Suspicious        bool
	EncodingNote      string
}

// DecodeSearchLocationItem decodes objh according to the record's objt class.
func DecodeSearchLocationItem(location SearchLocationReference) SearchLocationDecodedItem {
	decoded := SearchLocationDecodedItem{
		RawObjectType:  byte(location.ObjectClass),
		RawPayload:     location.Payload,
		ItemID:         -1,
		PickupQuantity: int(location.Payload),
	}

	setItem := func(item Item, quantity int, name, category, discovery, semantics string) {
		decoded.Classification = "inventory_item"
		decoded.Category = category
		decoded.Item = item
		decoded.ItemID = item.ID()
		decoded.ItemName = name
		decoded.PickupQuantity = quantity
		decoded.DiscoveryText = discovery
		decoded.QuantitySemantics = semantics
	}

	switch location.ObjectClass {
	case SearchObjectGold:
		setItem(Gold, int(location.Payload), "Gold", "Provision", "a sack of gold!", "payload_is_pickup_quantity")
	case SearchObjectPotion:
		if location.Payload >= NumberOfPotions {
			return invalidSearchPayload(decoded, "Potion payload is outside the original 0..7 catalog.")
		}
		potion := Potion(location.Payload)
		setItem(potion, 1, potionSearchName(potion), "Potion", "a potion!", "payload_is_potion_subtype")
	case SearchObjectScroll:
		if location.Payload == 0xff {
			setItem(HMSCape, 1, newQuestItemReferences()[HMSCape].FullDisplayName(), "Quest Item", "a scroll!", "payload_is_hms_cape_plans")

			break
		}
		if location.Payload >= NumberOfScrolls {
			return invalidSearchPayload(decoded, "Scroll payload is outside the original 0..7 catalog.")
		}
		scroll := Scroll(location.Payload)
		setItem(scroll, 1, scrollSearchName(scroll), "Scroll", "a scroll!", "payload_is_scroll_subtype")
	case SearchObjectWeapon, SearchObjectShield, SearchObjectHelm, SearchObjectRing,
		SearchObjectArmour, SearchObjectAmulet:
		if location.Payload > byte(Ankh) {
			return invalidSearchPayload(decoded, "Equipment payload is outside the original 0..47 catalog.")
		}
		equipment := Equipment(location.Payload)
		setItem(equipment, 1, equipment.String(), searchEquipmentCategory(location.ObjectClass),
			searchObjectDiscoveryText(location.ObjectClass), "payload_is_equipment_id")
	case SearchObjectKey:
		if location.Payload&0x80 != 0 {
			setItem(SkullKeys, int(location.Payload&0x7f), "Skull Keys", "Provision",
				"a ring of keys!", "high_bit_selects_skull_keys")

			break
		}
		setItem(Key, int(location.Payload), "Keys", "Provision", "a ring of keys!", "payload_is_pickup_quantity")
	case SearchObjectGem:
		setItem(Gem, int(location.Payload), "Gems", "Provision", "a gem!", "payload_is_pickup_quantity")
	case SearchObjectTorch:
		setItem(Torches, int(location.Payload), "Torches", "Provision", "some torches!", "payload_is_pickup_quantity")
	case SearchObjectFood:
		setItem(Food, int(location.Payload), "Food", "Provision", "some food!", "payload_is_pickup_quantity")
	case SearchObjectMoonstone:
		decoded.Classification = "world_object"
		decoded.Category = "Moonstone"
		decoded.ItemName = "Moonstone"
		decoded.PickupQuantity = 0
		decoded.DiscoveryText = "a strange rock!"
		decoded.QuantitySemantics = "payload_is_moonstone_state"
	case SearchObjectDeadPC:
		decoded.Classification = "world_object"
		decoded.Category = "Dead PC"
		decoded.ItemName = "Rotting body"
		decoded.PickupQuantity = 0
		decoded.DiscoveryText = "a rotting body!"
		decoded.QuantitySemantics = "payload_is_object_state"
	case SearchObjectDeadMonster:
		decoded.Classification = "world_object"
		decoded.Category = "Dead Monster"
		decoded.ItemName = "Moldy corpse"
		decoded.PickupQuantity = 0
		decoded.DiscoveryText = "a moldy corpse!"
		decoded.QuantitySemantics = "payload_is_object_state"
	default:
		decoded.Classification = "unknown"
		decoded.Category = "Unknown"
		decoded.Suspicious = true
		decoded.EncodingNote = "Object class has no documented search decoder."
	}

	return decoded
}

func invalidSearchPayload(decoded SearchLocationDecodedItem, note string) SearchLocationDecodedItem {
	decoded.Classification = "unknown"
	decoded.Category = "Unknown"
	decoded.Suspicious = true
	decoded.EncodingNote = note

	return decoded
}

func searchEquipmentCategory(class SearchObjectClass) string {
	//nolint:exhaustive // This helper is called only for equipment object classes.
	switch class {
	case SearchObjectWeapon:
		return "Weapon"
	case SearchObjectShield:
		return "Shield"
	case SearchObjectHelm:
		return "Helm"
	case SearchObjectRing:
		return "Ring"
	case SearchObjectArmour:
		return "Armour"
	case SearchObjectAmulet:
		return "Amulet"
	default:
		return "Equipment"
	}
}

func searchObjectDiscoveryText(class SearchObjectClass) string {
	//nolint:exhaustive // This helper is called only for equipment object classes.
	switch class {
	case SearchObjectWeapon:
		return "a weapon!"
	case SearchObjectShield:
		return "a shield!"
	case SearchObjectHelm:
		return "a helm!"
	case SearchObjectRing:
		return "a ring!"
	case SearchObjectArmour:
		return "some armour!"
	case SearchObjectAmulet:
		return "an amulet!"
	default:
		return "an item!"
	}
}

func potionSearchName(potion Potion) string {
	return newPotionItemReferences()[potion].FullDisplayName()
}

func scrollSearchName(scroll Scroll) string {
	return newScrollItemReferences()[scroll].FullDisplayName()
}

// GetAllSearchLocations returns the 113 fixed-search records in original order; ref:QUEST-0002.
func GetAllSearchLocations() []SearchLocationReference {
	return append([]SearchLocationReference(nil), allSearchLocations...)
}

//nolint:gochecknoglobals // Immutable fixed-search location catalog; ref:QUEST-0002.
var searchLocationObjectTypes = [...]byte{
	0x0b, 0x05, 0x0b, 0x05, 0x0b, 0x05, 0x0b, 0x05, 0x0b, 0x05,
	0x0b, 0x05, 0x04, 0x07, 0x07, 0x05, 0x05, 0x05, 0x08, 0x0b,
	0x0b, 0x03, 0x03, 0x03, 0x04, 0x04, 0x0f, 0x0d, 0x0a, 0x0f,
	0x05, 0x0a, 0x04, 0x0f, 0x0f, 0x05, 0x0d, 0x05, 0x03, 0x04,
	0x1f, 0x1f, 0x1e, 0x1e, 0x1e, 0x05, 0x05, 0x07, 0x02, 0x04,
	0x04, 0x03, 0x03, 0x04, 0x08, 0x1e, 0x03, 0x03, 0x07, 0x0d,
	0x03, 0x08, 0x02, 0x03, 0x04, 0x0c, 0x04, 0x03, 0x0b, 0x0b,
	0x0b, 0x0b, 0x05, 0x07, 0x0a, 0x03, 0x04, 0x04, 0x04, 0x04,
	0x0a, 0x1e, 0x1e, 0x1e, 0x1e, 0x04, 0x0a, 0x08, 0x04, 0x04,
	0x04, 0x0a, 0x07, 0x0c, 0x07, 0x03, 0x04, 0x0c, 0x03, 0x03,
	0x04, 0x0a, 0x0c, 0x03, 0x05, 0x0c, 0x03, 0x0f, 0x03, 0x0f,
	0x0f, 0x03, 0x04,
}

//nolint:gochecknoglobals // Immutable original data.
var searchLocationPayloads = [...]byte{
	0x0f, 0x29, 0x0f, 0x29, 0x0f, 0x29, 0x0f, 0x29, 0x0f, 0x29,
	0x0f, 0x29, 0xff, 0x09, 0x85, 0x27, 0x23, 0x28, 0x01, 0x0a,
	0x0a, 0x05, 0x06, 0x07, 0x05, 0x07, 10, 0x03, 0x2a, 0x05,
	0x14, 0x2b, 0x01, 0x05, 10, 0x26, 0x04, 0x12, 0x03, 0x04,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x15, 0x25, 0x05, 99, 0x01,
	0x01, 0x01, 0x02, 0x06, 0x04, 0x00, 0x01, 0x03, 0x07, 0x09,
	0x00, 0x05, 50, 0x07, 0x05, 0x2d, 0x07, 0x04, 0x0a, 0x0b,
	0x09, 0x0c, 0x1e, 0x07, 0x2c, 0x03, 0x05, 0x05, 0x02, 0x07,
	0x2b, 0x00, 0x00, 0x00, 0x00, 0x06, 0x2c, 0x03, 0x01, 0x01,
	0x01, 0x2a, 0x05, 0x2f, 0x01, 0x03, 0x01, 0x2e, 0x05, 0x04,
	0x05, 0x2a, 0x2e, 0x07, 0x12, 0x2e, 0x01, 20, 0x07, 16,
	13, 0x06, 0x07,
}

//nolint:gochecknoglobals // Immutable original data.
var searchLocationMaps = [...]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x15, 0x12, 0x05, 0x00, 0x12, 0x12, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x02,
	0x02, 0x02, 0x03, 0x03, 0x03, 0x03, 0x03, 0x04, 0x04, 0x04,
	0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x05, 0x05,
	0x05, 0x05, 0x05, 0x05, 0x06, 0x07, 0x07, 0x08, 0x08, 0x09,
	0x0b, 0x0b, 0x0b, 0x0d, 0x0e, 0x0e, 0x0e, 0x0e, 0x11, 0x11,
	0x11, 0x11, 0x11, 0x11, 0x11, 0x12, 0x12, 0x12, 0x12, 0x12,
	0x12, 0x13, 0x13, 0x13, 0x13, 0x13, 0x13, 0x14, 0x15, 0x15,
	0x15, 0x16, 0x16, 0x17, 0x18, 0x18, 0x19, 0x1a, 0x1b, 0x1b,
	0x1c, 0x1e, 0x1e, 0x1e, 0x1e, 0x1e, 0x1f, 0x1f, 0x20, 0x01,
	0x01, 0x11, 0x11,
}

//nolint:gochecknoglobals // Immutable original data.
var searchLocationLevels = [...]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0x00, 0xff, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x01, 0xff, 0xff, 0x01, 0x02, 0x02, 0x02, 0x02,
	0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x02, 0x02, 0x02, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01,
	0x01, 0x02, 0x02,
}

//nolint:gochecknoglobals // Immutable original data.
var searchLocationXs = [...]byte{
	0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9,
	0xe9, 0xe9, 0x0f, 0x08, 0x02, 0x40, 0x07, 0x17, 0x08, 0x19,
	0x19, 0x17, 0x0d, 0x0e, 0x10, 0x11, 0x18, 0x1b, 0x1b, 0x02,
	0x06, 0x18, 0x15, 0x14, 0x1d, 0x1e, 0x01, 0x1d, 0x16, 0x01,
	0x0d, 0x0e, 0x0e, 0x10, 0x12, 0x10, 0x1b, 0x14, 0x1b, 0x0d,
	0x0c, 0x08, 0x08, 0x07, 0x18, 0x05, 0x06, 0x15, 0x19, 0x0a,
	0x15, 0x0e, 0x10, 0x18, 0x10, 0x0e, 0x11, 0x0e, 0x14, 0x15,
	0x16, 0x17, 0x15, 0x07, 0x14, 0x11, 0x0d, 0x0e, 0x10, 0x11,
	0x13, 0x03, 0x05, 0x07, 0x07, 0x05, 0x03, 0x12, 0x0d, 0x0e,
	0x10, 0x0d, 0x03, 0x0f, 0x13, 0x13, 0x19, 0x0b, 0x0b, 0x0a,
	0x13, 0x17, 0x17, 0x14, 0x16, 0x16, 0x06, 0x13, 0x08, 0x13,
	0x14, 0x0c, 0x0c,
}

//nolint:gochecknoglobals // Immutable original data.
var searchLocationYs = [...]byte{
	0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9, 0xe9,
	0xe9, 0xe9, 0x02, 0x06, 0x02, 0x50, 0x06, 0x06, 0x05, 0x06,
	0x08, 0x05, 0x0d, 0x0d, 0x0d, 0x0d, 0x13, 0x03, 0x1d, 0x01,
	0x1a, 0x06, 0x10, 0x0a, 0x01, 0x17, 0x1d, 0x0b, 0x1a, 0x1a,
	0x02, 0x02, 0x04, 0x03, 0x02, 0x03, 0x16, 0x16, 0x08, 0x0b,
	0x0b, 0x15, 0x17, 0x17, 0x06, 0x02, 0x07, 0x12, 0x15, 0x0c,
	0x0f, 0x09, 0x0c, 0x02, 0x0c, 0x10, 0x0c, 0x0c, 0x07, 0x07,
	0x07, 0x07, 0x0d, 0x12, 0x17, 0x12, 0x06, 0x06, 0x06, 0x06,
	0x07, 0x02, 0x07, 0x07, 0x02, 0x07, 0x02, 0x19, 0x02, 0x02,
	0x02, 0x0d, 0x0c, 0x01, 0x16, 0x16, 0x10, 0x04, 0x11, 0x11,
	0x16, 0x09, 0x07, 0x07, 0x13, 0x11, 0x03, 0x07, 0x15, 0x18,
	0x18, 0x0c, 0x0c,
}

//nolint:gochecknoglobals // Derived once from immutable original data.
var allSearchLocations = buildSearchLocations()

func buildSearchLocations() []SearchLocationReference {
	locations := make([]SearchLocationReference, SearchLocationCount)
	for index := range SearchLocationCount {
		level := int(searchLocationLevels[index])
		if searchLocationLevels[index] == 0xff {
			level = -1
		}
		locations[index] = SearchLocationReference{
			Index:        index,
			ObjectClass:  SearchObjectClass(searchLocationObjectTypes[index]),
			Payload:      searchLocationPayloads[index],
			MapNumber:    int(searchLocationMaps[index]),
			Level:        level,
			X:            int(searchLocationXs[index]),
			Y:            int(searchLocationYs[index]),
			Requirements: searchLocationRequirements(index),
		}
		decoded := DecodeSearchLocationItem(locations[index])
		locations[index].ItemType = decoded.Item
		locations[index].Quantity = uint16(decoded.PickupQuantity)
	}

	return locations
}

func searchLocationRequirements(index int) SearchLocationRequirements {
	switch index {
	case 13:
		return SearchLocationRequirements{IsSpecialItem: true, RequiresNoKeys: true}
	case 14:
		return SearchLocationRequirements{IsSpecialItem: true, IsSkullKeys: true, IsRepeatable: true}
	case 15:
		return SearchLocationRequirements{IsSpecialItem: true, RequiresNoGlassSwords: true}
	default:
		return SearchLocationRequirements{}
	}
}
