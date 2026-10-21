package legacy_save

import (
	"crypto/sha256"
	"encoding/binary"
)

const (
	charactersOffset        = 0x0002
	characterRecordSize     = 32
	genderFieldOffset       = 0x09
	classFieldOffset        = 0x0a
	statusFieldOffset       = 0x0b
	currentHPFieldOffset    = 0x10
	partyMapFieldOffset     = 0x1f
	foodOffset              = 0x0202
	goldOffset              = 0x0204
	keysOffset              = 0x0206
	gemsOffset              = 0x0207
	torchesOffset           = 0x0208
	grapplesOffset          = 0x0209
	carpetsOffset           = 0x020a
	skullKeysOffset         = 0x020b
	skullKeysDayOffset      = 0x020c
	amuletOffset            = 0x020d
	crownOffset             = 0x020e
	sceptreOffset           = 0x020f
	shardsOffset            = 0x0210
	specialItemsOffset      = 0x0214
	equipmentOffset         = 0x021a
	preparedSpellsOffset    = 0x024a
	scrollsOffset           = 0x027a
	potionsOffset           = 0x0282
	moonstoneXOffset        = 0x028a
	moonstoneYOffset        = 0x0292
	moonstoneMapOffset      = 0x029a
	moonstoneLevelOffset    = 0x02a2
	reagentsOffset          = 0x02aa
	reagentCacheDaysOffset  = 0x02b2
	partyCountOffset        = 0x02b5
	searchFlagsOffset       = 0x02b6
	yearOffset              = 0x02ce
	activeSpellOffset       = 0x02d4
	activePlayerOffset      = 0x02d5
	playerShapeOffset       = 0x02d6
	monthOffset             = 0x02d7
	dayOffset               = 0x02d8
	hourOffset              = 0x02d9
	oldHourOffset           = 0x02da
	minuteOffset            = 0x02db
	combatTimeCountOffset   = 0x02dc
	windUpdateCountOffset   = 0x02dd
	karmaOffset             = 0x02e2
	lastGaveOffset          = 0x02e5
	lastCampedOffset        = 0x02e6
	lastWandererOffset      = 0x02e7
	spellDurationOffset     = 0x02e8
	spellOffset             = 0x02e9
	magicOffset             = 0x02ea
	windOffset              = 0x02ec
	locationOffset          = 0x02ed
	floorOffset             = 0x02ef
	partyXOffset            = 0x02f0
	partyYOffset            = 0x02f1
	magicLightOffset        = 0x0300
	torchTurnsOffset        = 0x0301
	shadowlordLordsOffset   = 0x0322
	lastCalledOffset        = 0x0325
	shrineActiveOffset      = 0x0326
	shrineCompleteOffset    = 0x0328
	openDungeonsOffset      = 0x032a
	closedShrinesOffset     = 0x0332
	dungeonRoomsOffset      = 0x033a
	shipXOffset             = 0x03ad
	shipYOffset             = 0x03ae
	sailingOffset           = 0x03af
	smashedOffset           = 0x03b1
	currentShadowlordOffset = 0x03b2
	easyOffset              = 0x03b3
	dungeonMapOffset        = 0x03b4
	deadNPCsOffset          = 0x05b4
	metNPCsOffset           = 0x0634
	activeObjectsOffset     = 0x06b4
	dungeonLastDirOffset    = 0x105c
	dungeonFacingOffset     = 0x105d
	dungeonTypeOffset       = 0x105e
	skiffsOffset            = 0x105f
)

// DecodeBundle validates exact source lengths, decodes both files, and then
// aggregates semantic validation failures. Invalid input never returns a
// partially decoded bundle.
func DecodeBundle(gam, ool []byte) (Bundle, error) {
	issues := validateLengths(gam, ool)
	if len(issues) != 0 {
		return Bundle{}, newValidationError(issues)
	}

	bundle := decodeBundle(gam, ool)
	issues = validateBundle(bundle)
	if len(issues) != 0 {
		return Bundle{}, newValidationError(issues)
	}

	return bundle, nil
}

func decodeBundle(gam, ool []byte) Bundle {
	var bundle Bundle
	for index := range bundle.Game.Characters {
		bundle.Game.Characters[index] = decodeCharacter(gam, charactersOffset+index*characterRecordSize)
	}
	bundle.Game.Inventory = decodeInventory(gam)
	decodeGameRegions(&bundle.Game, gam)
	decodeObjectRecords(bundle.Objects.Britannia[:], ool[:SavedObjectsSize/2])
	decodeObjectRecords(bundle.Objects.Underworld[:], ool[SavedObjectsSize/2:])
	hasher := sha256.New()
	_, _ = hasher.Write(gam)
	_, _ = hasher.Write(ool)
	copy(bundle.sourceDigest[:], hasher.Sum(nil))

	return bundle
}

func decodeInventory(gam []byte) Inventory {
	var inventory Inventory
	inventory.Food = binary.LittleEndian.Uint16(gam[foodOffset:])
	inventory.Gold = binary.LittleEndian.Uint16(gam[goldOffset:])
	inventory.Keys = gam[keysOffset]
	inventory.Gems = gam[gemsOffset]
	inventory.Torches = gam[torchesOffset]
	inventory.Grapples = gam[grapplesOffset]
	inventory.Carpets = gam[carpetsOffset]
	inventory.SkullKeys = gam[skullKeysOffset]
	inventory.Amulet = gam[amuletOffset]
	inventory.Crown = gam[crownOffset]
	inventory.Sceptre = gam[sceptreOffset]
	copy(inventory.Shards.Source[:], gam[shardsOffset:shardsOffset+4])
	inventory.Shards.Falsehood = inventory.Shards.Source[0]
	inventory.Shards.Hatred = inventory.Shards.Source[1]
	inventory.Shards.Cowardice = inventory.Shards.Source[2]
	inventory.SpecialItems = SpecialItems{
		Spyglass:     gam[specialItemsOffset],
		HMSCapePlans: gam[specialItemsOffset+1],
		Sextant:      gam[specialItemsOffset+2],
		PocketWatch:  gam[specialItemsOffset+3],
		BlackBadge:   gam[specialItemsOffset+4],
		WoodenBox:    gam[specialItemsOffset+5],
	}
	copy(inventory.Equipment[:], gam[equipmentOffset:equipmentOffset+len(inventory.Equipment)])
	copy(inventory.PreparedSpells[:], gam[preparedSpellsOffset:preparedSpellsOffset+len(inventory.PreparedSpells)])
	copy(inventory.Scrolls[:], gam[scrollsOffset:scrollsOffset+len(inventory.Scrolls)])
	copy(inventory.Potions[:], gam[potionsOffset:potionsOffset+len(inventory.Potions)])
	copy(inventory.Reagents[:], gam[reagentsOffset:reagentsOffset+len(inventory.Reagents)])

	return inventory
}

func decodeGameRegions(game *SavedGame, gam []byte) {
	for index := range game.Moonstones {
		game.Moonstones[index] = Moonstone{
			X:     gam[moonstoneXOffset+index],
			Y:     gam[moonstoneYOffset+index],
			Map:   gam[moonstoneMapOffset+index],
			Level: gam[moonstoneLevelOffset+index],
		}
	}

	game.SkullKeysFoundDay = gam[skullKeysDayOffset]
	copy(game.ReagentCacheDays[:], gam[reagentCacheDaysOffset:reagentCacheDaysOffset+len(game.ReagentCacheDays)])
	game.PartyCount = gam[partyCountOffset]
	copy(game.SearchFlags[:], gam[searchFlagsOffset:searchFlagsOffset+len(game.SearchFlags)])
	game.Calendar = Calendar{
		Year:   binary.LittleEndian.Uint16(gam[yearOffset:]),
		Month:  gam[monthOffset],
		Day:    gam[dayOffset],
		Hour:   gam[hourOffset],
		Minute: gam[minuteOffset],
	}
	game.ActiveSpell = gam[activeSpellOffset]
	game.ActivePlayer = gam[activePlayerOffset]
	game.PlayerShape = gam[playerShapeOffset]
	game.OldHour = gam[oldHourOffset]
	game.CombatActionTimeCount = gam[combatTimeCountOffset]
	game.WindUpdateCount = gam[windUpdateCountOffset]
	game.Karma = gam[karmaOffset]
	game.LastGave = gam[lastGaveOffset]
	game.LastCamped = gam[lastCampedOffset]
	game.LastWandererMonth = gam[lastWandererOffset]
	game.SpellDuration = gam[spellDurationOffset]
	game.Spell = gam[spellOffset]
	game.Magic = gam[magicOffset]
	game.Wind = gam[windOffset]
	game.Position = WorldPosition{
		Location: MapID(gam[locationOffset]),
		Floor:    gam[floorOffset],
		X:        gam[partyXOffset],
		Y:        gam[partyYOffset],
	}
	game.MagicLight = gam[magicLightOffset]
	game.TorchTurns = gam[torchTurnsOffset]
	game.Shrines = ShrineProgress{
		ActiveBits:    gam[shrineActiveOffset],
		CompletedBits: gam[shrineCompleteOffset],
		Closed:        [8]byte{},
	}
	copy(game.OpenDungeons[:], gam[openDungeonsOffset:openDungeonsOffset+len(game.OpenDungeons)])
	copy(game.Shrines.Closed[:], gam[closedShrinesOffset:closedShrinesOffset+len(game.Shrines.Closed)])
	copy(game.DungeonRooms[:], gam[dungeonRoomsOffset:dungeonRoomsOffset+len(game.DungeonRooms)])
	copy(game.ShadowlordLords[:], gam[shadowlordLordsOffset:shadowlordLordsOffset+len(game.ShadowlordLords)])
	game.LastCalledShadowlord = gam[lastCalledOffset]
	game.Ship = ShipState{X: gam[shipXOffset], Y: gam[shipYOffset], Sailing: gam[sailingOffset]}
	game.Smashed = gam[smashedOffset]
	game.CurrentShadowlord = gam[currentShadowlordOffset]
	game.Easy = gam[easyOffset]
	copy(game.DeadNPCs[:], gam[deadNPCsOffset:deadNPCsOffset+len(game.DeadNPCs)])
	copy(game.MetNPCs[:], gam[metNPCsOffset:metNPCsOffset+len(game.MetNPCs)])
	decodeObjectRecords(game.ActiveObjects[:], gam[activeObjectsOffset:activeObjectsOffset+objectCount*objectRecordSize])
	game.Dungeon = DungeonState{
		Active:        isDungeonMap(game.Position.Location),
		Cells:         [512]byte{},
		LastDirection: gam[dungeonLastDirOffset],
		Facing:        DungeonFacing(gam[dungeonFacingOffset]),
		Type:          DungeonType(gam[dungeonTypeOffset]),
		Skiffs:        gam[skiffsOffset],
	}
	copy(game.Dungeon.Cells[:], gam[dungeonMapOffset:dungeonMapOffset+len(game.Dungeon.Cells)])
}

func decodeCharacter(gam []byte, offset int) Character {
	var character Character
	copy(character.NameBytes[:], gam[offset:offset+len(character.NameBytes)])
	character.Occupied = character.NameBytes[0] != 0
	terminator := firstZero(character.NameBytes[:])
	if terminator < 0 {
		terminator = len(character.NameBytes)
	}
	character.Name = string(character.NameBytes[:terminator])
	character.Gender = Gender(gam[offset+genderFieldOffset])
	character.Class = CharacterClass(gam[offset+classFieldOffset])
	character.Status = CharacterStatus(gam[offset+statusFieldOffset])
	character.Strength = gam[offset+0x0c]
	character.Dexterity = gam[offset+0x0d]
	character.Intelligence = gam[offset+0x0e]
	character.CurrentMP = gam[offset+0x0f]
	character.CurrentHP = binary.LittleEndian.Uint16(gam[offset+currentHPFieldOffset:])
	character.MaximumHP = binary.LittleEndian.Uint16(gam[offset+0x12:])
	character.Experience = binary.LittleEndian.Uint16(gam[offset+0x14:])
	character.Level = gam[offset+0x16]
	character.MonthsAtInn = gam[offset+0x17]
	character.ArmorClass = gam[offset+0x18]
	character.Equipment = EquippedItems{
		Helmet: gam[offset+0x19],
		Armor:  gam[offset+0x1a],
		Hands:  [2]byte{gam[offset+0x1b], gam[offset+0x1c]},
		Ring:   gam[offset+0x1d],
		Amulet: gam[offset+0x1e],
	}
	character.PartyMap = PartyMap(gam[offset+partyMapFieldOffset])

	return character
}

func decodeObjectRecords(destination []ObjectRecord, source []byte) {
	for index := range destination {
		offset := index * objectRecordSize
		destination[index] = ObjectRecord{
			Tile:      source[offset],
			Shape:     source[offset+1],
			X:         source[offset+2],
			Y:         source[offset+3],
			Level:     source[offset+4],
			Number:    source[offset+5],
			Animation: source[offset+6],
			Misc:      source[offset+7],
		}
	}
}

func firstZero(value []byte) int {
	for index, current := range value {
		if current == 0 {
			return index
		}
	}

	return -1
}

func isDungeonMap(location MapID) bool {
	return location >= 33 && location <= 40
}
