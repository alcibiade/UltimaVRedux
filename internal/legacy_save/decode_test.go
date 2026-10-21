package legacy_save

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestDecodeBundleRequiresExactLengthsBeforeParsing(t *testing.T) {
	bundle, err := DecodeBundle(make([]byte, SavedGameSize-1), make([]byte, SavedObjectsSize+1))
	if bundle != (Bundle{}) {
		t.Fatal("invalid lengths returned a partial bundle")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("DecodeBundle() error = %T %v, want *ValidationError", err, err)
	}
	if !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("DecodeBundle() error does not wrap ErrInvalidBundle: %v", err)
	}
	if len(validationError.Issues) != 2 {
		t.Fatalf("issue count = %d, want 2: %#v", len(validationError.Issues), validationError.Issues)
	}
	want := []ValidationIssue{
		{Path: "gam", Code: "invalid_length", Offset: -1, Message: fmt.Sprintf("SAVED.GAM must be exactly %d bytes; got %d", SavedGameSize, SavedGameSize-1)},
		{Path: "ool", Code: "invalid_length", Offset: -1, Message: fmt.Sprintf("SAVED.OOL must be exactly %d bytes; got %d", SavedObjectsSize, SavedObjectsSize+1)},
	}
	if !reflect.DeepEqual(validationError.Issues, want) {
		t.Fatalf("issues = %#v, want %#v", validationError.Issues, want)
	}
}

func TestDecodeBundleDecodesVerifiedRegionsWithoutNeighborBleed(t *testing.T) {
	gam, ool := validBundleBytes()

	for _, offset := range []int{
		0x0000, 0x0001, 0x02cc, 0x02cd, 0x02dc, 0x02ee, 0x02f2, 0x0302,
		0x0321, 0x0327, 0x0329, 0x0348, 0x03b0, 0x03b2, 0x03b3, 0x05b3,
	} {
		gam[offset] = 0xee
	}

	for index := 0; index < characterCount; index++ {
		base := charactersOffset + index*characterRecordSize
		name := fmt.Sprintf("P%02d", index)
		clear(gam[base : base+characterRecordSize])
		copy(gam[base:base+9], name)
		gam[base+0x09] = byte([]Gender{GenderMale, GenderFemale}[index%2])
		gam[base+0x0a] = byte([]CharacterClass{ClassAvatar, ClassFighter, ClassBard, ClassWizard}[index%4])
		gam[base+0x0b] = byte([]CharacterStatus{StatusGood, StatusPoisoned, StatusCharmed, StatusAsleep, StatusDead}[index%5])
		gam[base+0x0c] = byte(10 + index)
		gam[base+0x0d] = byte(20 + index)
		gam[base+0x0e] = byte(30 + index)
		gam[base+0x0f] = byte(5 + index)
		binary.LittleEndian.PutUint16(gam[base+0x10:], uint16(100+index))
		binary.LittleEndian.PutUint16(gam[base+0x12:], uint16(200+index))
		binary.LittleEndian.PutUint16(gam[base+0x14:], uint16(300+index))
		gam[base+0x16] = byte(index + 1)
		gam[base+0x17] = byte(index + 2)
		gam[base+0x18] = byte(index + 3)
		for equipmentIndex := 0; equipmentIndex < 6; equipmentIndex++ {
			gam[base+0x19+equipmentIndex] = byte(index*6 + equipmentIndex + 1)
		}
		if index < 3 {
			gam[base+0x1f] = byte(PartyMapActive)
		} else {
			gam[base+0x1f] = byte(PartyMapUnrecruited)
		}
	}

	binary.LittleEndian.PutUint16(gam[foodOffset:], 0x1234)
	binary.LittleEndian.PutUint16(gam[goldOffset:], 0x5678)
	gam[keysOffset], gam[gemsOffset], gam[torchesOffset], gam[skullKeysOffset] = 1, 2, 3, 4
	gam[grapplesOffset], gam[carpetsOffset] = 5, 6
	gam[skullKeysDayOffset], gam[amuletOffset], gam[crownOffset], gam[sceptreOffset] = 7, 8, 9, 10
	copy(gam[shardsOffset:shardsOffset+4], []byte{5, 6, 7, 0xa5})
	copy(gam[specialItemsOffset:specialItemsOffset+6], []byte{8, 9, 10, 11, 12, 13})
	fillSequence(gam[equipmentOffset:equipmentOffset+48], 20)
	fillSequence(gam[preparedSpellsOffset:preparedSpellsOffset+48], 70)
	fillSequence(gam[scrollsOffset:scrollsOffset+8], 120)
	fillSequence(gam[potionsOffset:potionsOffset+8], 130)
	fillSequence(gam[moonstoneXOffset:moonstoneXOffset+8], 140)
	fillSequence(gam[moonstoneYOffset:moonstoneYOffset+8], 150)
	fillSequence(gam[moonstoneMapOffset:moonstoneMapOffset+8], 160)
	fillSequence(gam[moonstoneLevelOffset:moonstoneLevelOffset+8], 170)
	fillSequence(gam[reagentsOffset:reagentsOffset+8], 180)
	copy(gam[reagentCacheDaysOffset:reagentCacheDaysOffset+3], []byte{11, 12, 13})
	gam[partyCountOffset] = 3
	fillSequence(gam[searchFlagsOffset:searchFlagsOffset+24], 190)
	binary.LittleEndian.PutUint16(gam[yearOffset:], 0x1234)
	gam[monthOffset], gam[dayOffset], gam[hourOffset], gam[minuteOffset] = 13, 28, 23, 59
	gam[activeSpellOffset], gam[activePlayerOffset], gam[playerShapeOffset], gam[oldHourOffset], gam[windUpdateCountOffset] = 'P', 2, originalPlayerShape, 22, 14
	gam[combatTimeCountOffset] = 7
	gam[karmaOffset], gam[lastGaveOffset] = 77, 254
	gam[lastCampedOffset], gam[lastWandererOffset] = 15, 13
	gam[spellDurationOffset], gam[spellOffset], gam[magicOffset], gam[windOffset] = 16, 17, 18, 4
	gam[locationOffset], gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 0, 0xff, 0xfe, 0xfd
	gam[magicLightOffset], gam[torchTurnsOffset] = 200, 201
	copy(gam[shadowlordLordsOffset:shadowlordLordsOffset+3], []byte{1, 0xff, 8})
	gam[lastCalledOffset] = 2
	gam[shrineActiveOffset], gam[shrineCompleteOffset] = 0x55, 0xaa
	fillSequence(gam[openDungeonsOffset:openDungeonsOffset+8], 0x80)
	fillSequence(gam[closedShrinesOffset:closedShrinesOffset+8], 0x90)
	fillSequence(gam[dungeonRoomsOffset:dungeonRoomsOffset+14], 202)
	gam[shipXOffset], gam[shipYOffset], gam[sailingOffset] = 31, 32, 1
	gam[smashedOffset] = 25
	gam[easyOffset] = 1
	fillSequence(gam[deadNPCsOffset:deadNPCsOffset+128], 1)
	fillSequence(gam[metNPCsOffset:metNPCsOffset+128], 129)
	writeObject(gam[activeObjectsOffset:], 0, ObjectRecord{1, 2, 3, 4, 5, 6, 7, 8})
	writeObject(gam[activeObjectsOffset:], 31, ObjectRecord{9, 10, 11, 12, 13, 14, 15, 16})
	gam[dungeonFacingOffset], gam[dungeonTypeOffset], gam[skiffsOffset] = byte(DungeonWest), byte(DungeonTypeMine), 17

	writeObject(ool, 0, ObjectRecord{21, 22, 23, 24, 25, 26, 27, 28})
	writeObject(ool, 31, ObjectRecord{29, 30, 31, 32, 33, 34, 35, 36})
	writeObject(ool[SavedObjectsSize/2:], 0, ObjectRecord{41, 42, 43, 44, 45, 46, 47, 48})
	writeObject(ool[SavedObjectsSize/2:], 31, ObjectRecord{49, 50, 51, 52, 53, 54, 55, 56})

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}

	for index, character := range bundle.Game.Characters {
		if character.Name != fmt.Sprintf("P%02d", index) || character.CurrentHP != uint16(100+index) || character.MaximumHP != uint16(200+index) || character.Experience != uint16(300+index) {
			t.Fatalf("character %d decoded incorrectly: %#v", index, character)
		}
		wantEquipment := EquippedItems{
			Helmet: byte(index*6 + 1),
			Armor:  byte(index*6 + 2),
			Hands:  [2]byte{byte(index*6 + 3), byte(index*6 + 4)},
			Ring:   byte(index*6 + 5),
			Amulet: byte(index*6 + 6),
		}
		if character.Equipment != wantEquipment {
			t.Fatalf("character %d equipment = %#v, want %#v", index, character.Equipment, wantEquipment)
		}
	}

	inventory := bundle.Game.Inventory
	if inventory.Food != 0x1234 || inventory.Gold != 0x5678 || inventory.Keys != 1 || inventory.Gems != 2 || inventory.Torches != 3 || inventory.SkullKeys != 4 || inventory.Grapples != 5 || inventory.Carpets != 6 || inventory.Amulet != 8 || inventory.Crown != 9 || inventory.Sceptre != 10 {
		t.Fatalf("provisions decoded incorrectly: %#v", inventory)
	}
	if inventory.Shards.Source != [4]byte{5, 6, 7, 0xa5} || inventory.Shards.Falsehood != 5 || inventory.Shards.Hatred != 6 || inventory.Shards.Cowardice != 7 {
		t.Fatalf("shard mapping decoded incorrectly: %#v", inventory.Shards)
	}
	if inventory.SpecialItems != (SpecialItems{8, 9, 10, 11, 12, 13}) {
		t.Fatalf("special items decoded incorrectly: %#v", inventory.SpecialItems)
	}
	if inventory.Equipment[0] != 20 || inventory.Equipment[47] != 67 || inventory.PreparedSpells[0] != 70 || inventory.PreparedSpells[47] != 117 || inventory.Scrolls[7] != 127 || inventory.Potions[7] != 137 || inventory.Reagents[7] != 187 {
		t.Fatalf("inventory arrays decoded incorrectly: %#v", inventory)
	}
	for index, moonstone := range bundle.Game.Moonstones {
		want := Moonstone{byte(140 + index), byte(150 + index), byte(160 + index), byte(170 + index)}
		if moonstone != want {
			t.Fatalf("moonstone %d = %#v, want %#v", index, moonstone, want)
		}
	}
	if bundle.Game.PartyCount != 3 || bundle.Game.SearchFlags[0] != 190 || bundle.Game.SearchFlags[23] != 213 {
		t.Fatalf("party/search fields decoded incorrectly: count=%d search=%#v", bundle.Game.PartyCount, bundle.Game.SearchFlags)
	}
	if bundle.Game.Calendar != (Calendar{Year: 0x1234, Month: 13, Day: 28, Hour: 23, Minute: 59}) || bundle.Game.Karma != 77 {
		t.Fatalf("calendar/karma decoded incorrectly: %#v karma=%d", bundle.Game.Calendar, bundle.Game.Karma)
	}
	if bundle.Game.SkullKeysFoundDay != 7 || bundle.Game.ReagentCacheDays != [3]byte{11, 12, 13} || bundle.Game.ActiveSpell != 'P' || bundle.Game.ActivePlayer != 2 || bundle.Game.PlayerShape != originalPlayerShape || bundle.Game.OldHour != 22 || bundle.Game.CombatActionTimeCount != 7 || bundle.Game.WindUpdateCount != 14 || bundle.Game.LastGave != 254 || bundle.Game.LastCamped != 15 || bundle.Game.LastWandererMonth != 13 || bundle.Game.SpellDuration != 16 || bundle.Game.Spell != 17 || bundle.Game.Magic != 18 || bundle.Game.Wind != 4 || bundle.Game.Easy != 1 {
		t.Fatalf("extended source fields decoded incorrectly: %#v", bundle.Game)
	}
	if bundle.Game.Position != (WorldPosition{Location: 0, Floor: 0xff, X: 0xfe, Y: 0xfd}) || bundle.Game.MagicLight != 200 || bundle.Game.TorchTurns != 201 {
		t.Fatalf("position/torch decoded incorrectly: %#v torch=%d", bundle.Game.Position, bundle.Game.TorchTurns)
	}
	if bundle.Game.Shrines != (ShrineProgress{ActiveBits: 0x55, CompletedBits: 0xaa, Closed: [8]byte{0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97}}) {
		t.Fatalf("shrine bits include neighboring bytes: %#v", bundle.Game.Shrines)
	}
	if bundle.Game.ShadowlordLords != [3]byte{1, 0xff, 8} || bundle.Game.LastCalledShadowlord != 2 || bundle.Game.OpenDungeons[0] != 0x80 || bundle.Game.OpenDungeons[7] != 0x87 || bundle.Game.DungeonRooms[0] != 202 || bundle.Game.DungeonRooms[13] != 215 {
		t.Fatalf("progression regions decoded incorrectly: lords=%#v rooms=%#v", bundle.Game.ShadowlordLords, bundle.Game.DungeonRooms)
	}
	if bundle.Game.Ship != (ShipState{X: 31, Y: 32, Sailing: 1}) || bundle.Game.Smashed != 25 {
		t.Fatalf("ship/intoxication fields decoded incorrectly: ship=%#v smashed=%d", bundle.Game.Ship, bundle.Game.Smashed)
	}
	if bundle.Game.DeadNPCs[0] != 1 || bundle.Game.DeadNPCs[127] != 128 || bundle.Game.MetNPCs[0] != 129 || bundle.Game.MetNPCs[127] != 0 {
		t.Fatalf("NPC regions decoded incorrectly: dead edges=%d/%d met edges=%d/%d", bundle.Game.DeadNPCs[0], bundle.Game.DeadNPCs[127], bundle.Game.MetNPCs[0], bundle.Game.MetNPCs[127])
	}
	if bundle.Game.ActiveObjects[0] != (ObjectRecord{1, 2, 3, 4, 5, 6, 7, 8}) || bundle.Game.ActiveObjects[31] != (ObjectRecord{9, 10, 11, 12, 13, 14, 15, 16}) {
		t.Fatalf("active objects decoded incorrectly: first=%#v last=%#v", bundle.Game.ActiveObjects[0], bundle.Game.ActiveObjects[31])
	}
	if bundle.Objects.Britannia[0].Tile != 21 || bundle.Objects.Britannia[31].Tile != 29 || bundle.Objects.Underworld[0].Tile != 41 || bundle.Objects.Underworld[31].Tile != 49 {
		t.Fatalf("SAVED.OOL half ordering is wrong: %#v", bundle.Objects)
	}
	if bundle.Game.Dungeon.Active || bundle.Game.Dungeon.Facing != DungeonWest || bundle.Game.Dungeon.Type != DungeonTypeMine || bundle.Game.Dungeon.Skiffs != 17 {
		t.Fatalf("dungeon tail decoded incorrectly: %#v", bundle.Game.Dungeon)
	}

	gam[charactersOffset] = 'X'
	oollFirstTile := bundle.Objects.Britannia[0].Tile
	oollUnderworldTile := bundle.Objects.Underworld[0].Tile
	ool[0], ool[SavedObjectsSize/2] = 0xff, 0xff
	if bundle.Game.Characters[0].Name != "P00" || bundle.Objects.Britannia[0].Tile != oollFirstTile || bundle.Objects.Underworld[0].Tile != oollUnderworldTile {
		t.Fatal("decoded bundle aliases source buffers")
	}
}

func TestDecodeBundleAcceptsAllPartyMapStates(t *testing.T) {
	gam, ool := validBundleBytes()
	partyMaps := [characterCount]PartyMap{
		PartyMapActive, PartyMapActive, PartyMapActive, PartyMapActive, PartyMapActive, PartyMapActive,
		PartyMapMartyr, PartyMapUnrecruited, 1, 2, 3, 7, 20, 24, 32, 40,
	}
	for index, partyMap := range partyMaps {
		gam[charactersOffset+index*characterRecordSize+0x1f] = byte(partyMap)
	}
	gam[partyCountOffset] = 6

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() rejected valid pmap states: %v", err)
	}
	for index, want := range partyMaps {
		if got := bundle.Game.Characters[index].PartyMap; got != want {
			t.Fatalf("character %d pmap = %#x, want %#x", index, got, want)
		}
	}
	if !PartyMap(1).IsInnLocation() || !PartyMap(40).IsInnLocation() || PartyMapMartyr.IsInnLocation() || PartyMapUnrecruited.IsInnLocation() {
		t.Fatal("PartyMap.IsInnLocation() does not preserve pmap state boundaries")
	}
}

func TestDecodeBundleAcceptsPrintableNineByteName(t *testing.T) {
	gam, ool := validBundleBytes()
	copy(gam[charactersOffset:charactersOffset+9], "ABCDEFGHI")

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() rejected printable nine-byte name: %v", err)
	}
	if bundle.Game.Characters[0].Name != "ABCDEFGHI" {
		t.Fatalf("name = %q, want ABCDEFGHI", bundle.Game.Characters[0].Name)
	}

	gam[charactersOffset+8] = 0x1f
	_, err = DecodeBundle(gam, ool)
	assertValidationIssue(t, err, "gam.characters[0].name", "non_printable", charactersOffset+8)
}

func TestDecodeBundleUsesAuthoritativePartyCountPrefix(t *testing.T) {
	t.Run("generator five-member prefix with three active pmap values", func(t *testing.T) {
		gam, ool := validBundleBytes()
		gam[partyCountOffset] = 5

		bundle, err := DecodeBundle(gam, ool)
		if err != nil {
			t.Fatalf("DecodeBundle() rejected source-generated party prefix: %v", err)
		}
		if bundle.Game.PartyCount != 5 ||
			bundle.Game.Characters[3].PartyMap != PartyMapUnrecruited ||
			bundle.Game.Characters[4].PartyMap != PartyMapUnrecruited {
			t.Fatalf("decoded generator party state = count %d pmap %#x/%#x",
				bundle.Game.PartyCount,
				bundle.Game.Characters[3].PartyMap,
				bundle.Game.Characters[4].PartyMap,
			)
		}
	})

	t.Run("missing named character in active prefix", func(t *testing.T) {
		gam, ool := validBundleBytes()
		gam[partyCountOffset] = 4
		clear(gam[charactersOffset+3*characterRecordSize : charactersOffset+3*characterRecordSize+9])

		_, err := DecodeBundle(gam, ool)
		assertValidationIssue(
			t,
			err,
			"gam.characters[3].name",
			"missing_active_character",
			charactersOffset+3*characterRecordSize,
		)
	})

	t.Run("empty party", func(t *testing.T) {
		gam, ool := validBundleBytes()
		gam[partyCountOffset] = 0

		_, err := DecodeBundle(gam, ool)
		assertValidationIssue(t, err, "gam.partyCount", "out_of_range", partyCountOffset)
	})
}

func TestDecodeBundleAggregatesSemanticValidation(t *testing.T) {
	gam, ool := validBundleBytes()

	for index := 0; index < 7; index++ {
		gam[charactersOffset+index*characterRecordSize+0x1f] = byte(PartyMapActive)
	}
	clear(gam[charactersOffset : charactersOffset+9])
	copy(gam[charactersOffset:charactersOffset+9], "ABCDEFGHI")
	second := charactersOffset + characterRecordSize
	gam[second], gam[second+1] = 0x1f, 0
	third := charactersOffset + 2*characterRecordSize
	gam[third+0x09], gam[third+0x0a], gam[third+0x0b] = 1, 'X', 'X'
	fourth := charactersOffset + 3*characterRecordSize
	binary.LittleEndian.PutUint16(gam[fourth+0x10:], 51)
	binary.LittleEndian.PutUint16(gam[fourth+0x12:], 50)
	ninth := charactersOffset + 8*characterRecordSize
	gam[ninth+0x1f] = 41
	gam[partyCountOffset] = 7
	binary.LittleEndian.PutUint16(gam[yearOffset:], 0)
	gam[monthOffset], gam[dayOffset], gam[hourOffset], gam[minuteOffset] = 14, 0, 24, 60
	gam[locationOffset], gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 1, 8, 32, 33

	bundle, err := DecodeBundle(gam, ool)
	if bundle != (Bundle{}) {
		t.Fatal("semantic validation returned a partial bundle")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("DecodeBundle() error = %T %v, want *ValidationError", err, err)
	}

	want := map[string]int{
		"gam.characters[1].name/non_printable":     second,
		"gam.characters[2].gender/unknown_value":   third + 0x09,
		"gam.characters[2].class/unknown_value":    third + 0x0a,
		"gam.characters[2].status/unknown_value":   third + 0x0b,
		"gam.characters[3].currentHP/inconsistent": fourth + 0x10,
		"gam.characters[8].partyMap/unknown_value": ninth + 0x1f,
		"gam.partyCount/out_of_range":              partyCountOffset,
		"gam.calendar.year/out_of_range":           yearOffset,
		"gam.calendar.month/out_of_range":          monthOffset,
		"gam.calendar.day/out_of_range":            dayOffset,
		"gam.calendar.hour/out_of_range":           hourOffset,
		"gam.calendar.minute/out_of_range":         minuteOffset,
		"gam.position.floor/out_of_range":          floorOffset,
		"gam.position.x/out_of_range":              partyXOffset,
		"gam.position.y/out_of_range":              partyYOffset,
	}
	if len(validationError.Issues) != len(want) {
		t.Fatalf("issue count = %d, want %d: %#v", len(validationError.Issues), len(want), validationError.Issues)
	}
	for _, issue := range validationError.Issues {
		key := issue.Path + "/" + issue.Code
		offset, ok := want[key]
		if !ok {
			t.Errorf("unexpected issue: %#v", issue)
			continue
		}
		if issue.Offset != offset {
			t.Errorf("issue %s offset = %#x, want %#x", key, issue.Offset, offset)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing issues: %#v", want)
	}
}

func TestDecodeBundleIgnoresEnumsInVacantRosterRecord(t *testing.T) {
	gam, ool := validBundleBytes()
	base := charactersOffset + 15*characterRecordSize
	clear(gam[base : base+characterRecordSize])
	gam[base+0x09], gam[base+0x0a], gam[base+0x0b], gam[base+0x1f] = 1, 2, 3, 41

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() rejected nonsemantic vacant-record bytes: %v", err)
	}
	if bundle.Game.Characters[15].Occupied {
		t.Fatal("zero-leading name buffer decoded as occupied")
	}
}

func TestDungeonDetectionValidationAndStateDecoding(t *testing.T) {
	gam, ool := validBundleBytes()
	gam[locationOffset], gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 33, 7, 7, 7
	gam[dungeonLastDirOffset], gam[dungeonFacingOffset], gam[dungeonTypeOffset], gam[skiffsOffset] = 4, byte(DungeonWest), byte(DungeonTypeDungeon), 9
	gam[dungeonMapOffset], gam[dungeonMapOffset+511] = 0x12, 0xfe

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}
	if !bundle.IsDungeon() || !bundle.Game.Dungeon.Active || bundle.Game.Dungeon.LastDirection != 4 ||
		bundle.Game.Dungeon.Facing != DungeonWest || bundle.Game.Dungeon.Type != DungeonTypeDungeon ||
		bundle.Game.Dungeon.Skiffs != 9 || bundle.Game.Dungeon.Cells[0] != 0x12 || bundle.Game.Dungeon.Cells[511] != 0xfe {
		t.Fatalf("dungeon context decoded incorrectly: %#v", bundle.Game.Dungeon)
	}

	gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 8, 8, 8
	gam[dungeonLastDirOffset], gam[dungeonFacingOffset], gam[dungeonTypeOffset] = 6, 4, 0
	invalid, err := DecodeBundle(gam, ool)
	if invalid != (Bundle{}) {
		t.Fatal("invalid dungeon returned a partial bundle")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) || len(validationError.Issues) != 6 {
		t.Fatalf("invalid dungeon issues = %#v, error %v", validationError, err)
	}
}

func validBundleBytes() ([]byte, []byte) {
	gam := make([]byte, SavedGameSize)
	ool := make([]byte, SavedObjectsSize)
	for index := 0; index < characterCount; index++ {
		base := charactersOffset + index*characterRecordSize
		copy(gam[base:base+9], fmt.Sprintf("C%02d", index))
		gam[base+0x09] = byte(GenderMale)
		gam[base+0x0a] = byte(ClassFighter)
		gam[base+0x0b] = byte(StatusGood)
		gam[base+0x0c], gam[base+0x0d], gam[base+0x0e], gam[base+0x0f] = 20, 20, 20, 10
		binary.LittleEndian.PutUint16(gam[base+0x10:], 100)
		binary.LittleEndian.PutUint16(gam[base+0x12:], 100)
		binary.LittleEndian.PutUint16(gam[base+0x14:], uint16(index*100))
		gam[base+0x16] = 1
		if index < 3 {
			gam[base+0x1f] = byte(PartyMapActive)
		} else {
			gam[base+0x1f] = byte(PartyMapUnrecruited)
		}
	}
	gam[charactersOffset+classFieldOffset] = byte(ClassAvatar)
	gam[partyCountOffset] = 3
	gam[activePlayerOffset] = originalNothing
	binary.LittleEndian.PutUint16(gam[yearOffset:], 139)
	gam[monthOffset], gam[dayOffset], gam[hourOffset], gam[minuteOffset] = 4, 5, 8, 35
	gam[karmaOffset] = 75
	gam[locationOffset], gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 13, 0, 15, 15

	return gam, ool
}

func fillSequence(destination []byte, first byte) {
	for index := range destination {
		destination[index] = first + byte(index)
	}
}

func writeObject(destination []byte, index int, object ObjectRecord) {
	offset := index * objectRecordSize
	destination[offset] = object.Tile
	destination[offset+1] = object.Shape
	destination[offset+2] = object.X
	destination[offset+3] = object.Y
	destination[offset+4] = object.Level
	destination[offset+5] = object.Number
	destination[offset+6] = object.Animation
	destination[offset+7] = object.Misc
}

// IsDungeon reports whether SAVED.GAM was captured on map 33 through 40.
func (bundle Bundle) IsDungeon() bool {
	return bundle.Game.Dungeon.Active
}
