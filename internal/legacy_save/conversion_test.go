package legacy_save

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestConvertPopulatesCurrentNativeSnapshot(t *testing.T) {
	gam, ool := fullConversionBundleBytes()
	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	snapshot := result.Snapshot
	if len(snapshot.AtlasLandmarks) != 0 {
		t.Fatal("legacy import must not infer earned atlas observations")
	}
	if !reflect.DeepEqual(snapshot.Presentation, journal.EmptyPresentation()) {
		t.Fatalf("legacy import must start with quiet empty presentation: %#v", snapshot.Presentation)
	}
	if issues := save_snapshot.Validate(snapshot); len(issues) != 0 {
		t.Fatalf("converted snapshot validation issues = %#v", issues)
	}
	if snapshot.Version != save_snapshot.CurrentVersion || snapshot.NeedsMapUnitReconstruction() || len(snapshot.MapUnits) == 0 || len(snapshot.ItemStacks) != 0 || len(snapshot.MapMutations) != 0 {
		t.Fatalf("native boundary fields = version %d redux %#v units %#v stacks %#v mutations %#v", snapshot.Version, snapshot.Redux, snapshot.MapUnits, snapshot.ItemStacks, snapshot.MapMutations)
	}
	if snapshot.Continuation.ActivePlayerIndex != 2 || snapshot.Continuation.BeggarKarmaCounter != 255 || snapshot.Continuation.IsCriminal || snapshot.Continuation.HMSCapeAutomaticWaitActorSkip {
		t.Fatalf("continuation = %#v, want active player 2, beggar counter 255, and false Redux defaults", snapshot.Continuation)
	}
	if !snapshot.Redux.LordBritishPassageOpen {
		t.Fatal("legacy wooden-box ownership did not infer the permanent Lord British passage")
	}
	for _, entry := range snapshot.Redux.PlayerKnowledge {
		if entry.Category != save_snapshot.KnowledgeCategorySpell {
			t.Fatalf("legacy conversion fabricated lore knowledge: %#v", entry)
		}
	}

	wantPmaps := []int{int(PartyMapActive), int(PartyMapActive), int(PartyMapActive), int(PartyMapMartyr), int(PartyMapUnrecruited), 1, 40}
	for index, want := range wantPmaps {
		if got := snapshot.Party.Roster[index].PartyMap; got != want {
			t.Fatalf("roster[%d].partyMap = %#x, want %#x", index, got, want)
		}
	}
	avatar := snapshot.Party.Roster[0]
	if avatar.Name != "ABCDEFGHI" || avatar.Gender != int(GenderMale) || avatar.Class != int(ClassAvatar) || avatar.Status != int(StatusGood) ||
		avatar.ArmorClass != 3 || avatar.Helmet != 1 || avatar.Armor != 2 || avatar.Weapon != 3 || avatar.Shield != 4 || avatar.Ring != 5 || avatar.Amulet != 6 {
		t.Fatalf("Avatar fields = %#v", avatar)
	}
	if snapshot.Party.Karma != 77 {
		t.Fatalf("karma = %d, want 77", snapshot.Party.Karma)
	}
	wantMet := []save_snapshot.NPCFlagSnapshot{{Location: 0, NPC: 0}, {Location: 31, NPC: 31}}
	wantDead := []save_snapshot.NPCFlagSnapshot{{Location: 16, NPC: 31}, {Location: 17, NPC: 30}, {Location: 31, NPC: 31}}
	if !reflect.DeepEqual(snapshot.Party.MetNPCs, wantMet) || !reflect.DeepEqual(snapshot.Party.DeadNPCs, wantDead) {
		t.Fatalf("NPC flags = met %#v dead %#v", snapshot.Party.MetNPCs, snapshot.Party.DeadNPCs)
	}

	inventory := snapshot.Inventory
	if inventory.Food != 9999 || inventory.Gold != 8888 || inventory.Keys != 7 || inventory.Gems != 8 || inventory.Torches != 9 || inventory.SkullKeys != 10 {
		t.Fatalf("provisions = %#v", inventory)
	}
	if inventory.Equipment[0] != 1 || inventory.Equipment[47] != 48 || inventory.Spells[0] != 2 || inventory.Spells[47] != 49 || inventory.Scrolls[7] != 10 || inventory.Potions[7] != 11 || inventory.Reagents[7] != 12 {
		t.Fatalf("inventory arrays = %#v", inventory)
	}
	wantQuestItems := [save_snapshot.QuestItemCount]uint16{1, 1, 1, 1, 1, 1, 4, 3, 1, 1, 1, 1, 1, 1}
	if inventory.QuestItems != wantQuestItems {
		t.Fatalf("quest items = %#v, want %#v", inventory.QuestItems, wantQuestItems)
	}
	if inventory.Moonstones != [save_snapshot.MoonstoneCount]uint16{0, 1} {
		t.Fatalf("moonstone inventory = %#v", inventory.Moonstones)
	}
	wantMoonstoneJournal := [8]journal.MoonstoneObservation{}
	wantMoonstoneJournal[1] = journal.MoonstoneObservation{OriginalSiteRetired: true, Known: true, FirstSource: journal.MoonstoneSourcePossession}
	if snapshot.MoonstoneJournal != wantMoonstoneJournal {
		t.Fatalf("legacy moonstone knowledge must reflect only carried inventory: %#v", snapshot.MoonstoneJournal)
	}
	for _, unit := range snapshot.MapUnits {
		if unit.SurfaceChestAssessment != 0 {
			t.Fatal("legacy chest trap bytes must not disclose an assessment")
		}
	}

	if snapshot.World.Calendar.Month != 13 || snapshot.World.Position.Floor != -1 || snapshot.World.LastLargeMapPosition != snapshot.World.Position || snapshot.World.LastProcessedHour != 22 || snapshot.World.LastCamped != 23 || snapshot.World.LastWandererMonth != 13 || snapshot.World.IsSailing {
		t.Fatalf("world = %#v", snapshot.World)
	}
	wantSearchFound := bundle.Game.SearchFlags
	for bit := references.SearchLocationCount; bit < len(wantSearchFound)*8; bit++ {
		wantSearchFound[bit/8] &^= 1 << (bit & 7)
	}
	if snapshot.Progression.SearchFound != wantSearchFound || snapshot.Progression.SkullKeysFoundDay != 24 || snapshot.Progression.ReagentCachesFoundDay != [3]byte{25, 26, 27} {
		t.Fatalf("search progression = %#v", snapshot.Progression)
	}
	if got := snapshot.Progression.Moonstones[0]; got.State != 0 || got.IsFound || got.Map != 0 {
		t.Fatalf("buried moonstone = %#v", got)
	}
	if got := snapshot.Progression.Moonstones[1]; got.State != 0xff || got.IsFound || got.Map != 0xff {
		t.Fatalf("inventory moonstone = %#v", got)
	}
	if got := snapshot.Progression.Moonstones[2]; got.State != 0 || got.IsFound || got.Map != 2 {
		t.Fatalf("buried moonstone = %#v", got)
	}
	if !snapshot.Progression.Shrines[0].Active || snapshot.Progression.Shrines[0].Completed || !snapshot.Progression.Shrines[0].Ruined || !snapshot.Progression.Shrines[6].Completed || snapshot.Progression.Shrines[1].Ruined {
		t.Fatalf("shrine progression = %#v", snapshot.Progression.Shrines)
	}
	if !snapshot.Progression.Dungeons[0] || snapshot.Progression.Dungeons[1] || !snapshot.Progression.Dungeons[7] || snapshot.Progression.DungeonRooms != bundle.Game.DungeonRooms {
		t.Fatalf("dungeon progression = dungeons %#v rooms %#v", snapshot.Progression.Dungeons, snapshot.Progression.DungeonRooms)
	}
	if snapshot.Progression.Shadowlords.Alive != [save_snapshot.ShadowlordCount]bool{false, true, true} || snapshot.Progression.Shadowlords.TownAssignments != [save_snapshot.ShadowlordCount]int{} || snapshot.Progression.Shadowlords.LastCalled != 2 || snapshot.Progression.Shadowlords.Summoned || snapshot.Progression.Shadowlords.AutomaticPresence != -1 {
		t.Fatalf("Shadowlord progression = %#v", snapshot.Progression.Shadowlords)
	}
	if !snapshot.Progression.HMSCapeActivated {
		t.Fatal("original rigged plans high bit did not enable the Redux HMS Cape preference")
	}
	if snapshot.Effects.DurationType != 'P' || snapshot.Effects.DurationTurns != 37 || snapshot.Effects.MagicLightMinutesRemaining != 44 || snapshot.Effects.CarriedTorchMinutesRemaining != 55 {
		t.Fatalf("effects = %#v", snapshot.Effects)
	}
	if snapshot.Wind.Direction != 3 || snapshot.Wind.UpdateCounter != 12 ||
		snapshot.Wind.NextTickMs != environment.WindTickIntervalMs {
		t.Fatalf("wind = %#v", snapshot.Wind)
	}
	if snapshot.PartyVehicle == nil || snapshot.PartyVehicle.Type != int(references.FrigateVehicle) || snapshot.PartyVehicle.CurrentDirection != int(references.Right) || snapshot.PartyVehicle.PreviousDirection != int(references.Right) || snapshot.PartyVehicle.Skiffs != 5 || snapshot.PartyVehicle.Hull != 99 || snapshot.PartyVehicle.SailState != 1 || snapshot.PartyVehicle.HasHMSCape {
		t.Fatalf("party vehicle = %#v", snapshot.PartyVehicle)
	}

	hasher := sha256.New()
	_, _ = hasher.Write(gam)
	_, _ = hasher.Write(ool)
	wantSeed := binary.LittleEndian.Uint64(hasher.Sum(nil)[:8])
	if snapshot.Random != (save_snapshot.RandomSnapshot{Seed: wantSeed, SourceCalls: 0}) {
		t.Fatalf("random = %#v, want seed %d", snapshot.Random, wantSeed)
	}
	for index := 1; index < len(result.Warnings); index++ {
		previous, current := result.Warnings[index-1], result.Warnings[index]
		if previous.Path > current.Path || (previous.Path == current.Path && previous.Code > current.Code) {
			t.Fatalf("warnings are not stable-sorted: %#v", result.Warnings)
		}
	}
	if !hasConversionWarning(result.Warnings, "world.isSailing", "automatic_sailing_not_restored") {
		t.Fatalf("required warnings missing: %#v", result.Warnings)
	}
}

func TestConvertHMSCapePlansImportsOwnershipAndRiggedHighBitSeparately(t *testing.T) {
	tests := []struct {
		name    string
		plans   byte
		owned   uint16
		enabled bool
	}{
		{name: "not owned", plans: 0},
		{name: "owned but not rigged", plans: 1, owned: 1},
		{name: "rigged high bit", plans: 0x81, owned: 1, enabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gam, ool := fullConversionBundleBytes()
			gam[specialItemsOffset+1] = test.plans
			bundle, err := DecodeBundle(gam, ool)
			if err != nil {
				t.Fatal(err)
			}
			result, err := Convert(bundle)
			if err != nil {
				t.Fatal(err)
			}

			if got := result.Snapshot.Inventory.QuestItems[references.HMSCape]; got != test.owned {
				t.Fatalf("plans ownership = %d, want %d", got, test.owned)
			}
			if got := result.Snapshot.Progression.HMSCapeActivated; got != test.enabled {
				t.Fatalf("HMS Cape preference = %v, want %v", got, test.enabled)
			}
		})
	}
}

func TestConvertInfersLordBritishPassageFromOriginalNPCDeadFlag(t *testing.T) {
	gam, ool := fullConversionBundleBytes()
	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}
	bundle.Game.Inventory.SpecialItems.WoodenBox = 0
	bundle.Game.DeadNPCs = [128]byte{}
	byteIndex := lordBritishNPCFlagLocation*npcFlagBytesPerLocation + lordBritishSandalwoodBoxNPC/bitsPerByte
	bundle.Game.DeadNPCs[byteIndex] = byte(1) << uint(lordBritishSandalwoodBoxNPC%bitsPerByte)

	result, err := Convert(bundle)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if !result.Snapshot.Redux.LordBritishPassageOpen {
		t.Fatal("legacy NPC-dead flag did not infer the permanent Lord British passage")
	}
	if result.Snapshot.Inventory.QuestItems[references.WoodenBox] != 0 {
		t.Fatal("NPC-dead-only legacy state unexpectedly granted the Wooden Box")
	}
	if !hasNPCFlag(result.Snapshot.Party.DeadNPCs, lordBritishNPCFlagLocation, lordBritishSandalwoodBoxNPC) {
		t.Fatal("converted snapshot lost the Lord British castle NPC-dead flag")
	}
}

func TestConvertPreservesValidShadowlordTownAssignments(t *testing.T) {
	t.Parallel()
	gam, ool := validBundleBytes()
	copy(gam[shadowlordLordsOffset:shadowlordLordsOffset+3], []byte{1, 0xff, 8})

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	shadowlords := result.Snapshot.Progression.Shadowlords
	if shadowlords.Alive != [save_snapshot.ShadowlordCount]bool{true, false, true} || shadowlords.TownAssignments != [save_snapshot.ShadowlordCount]int{1, 0, 8} {
		t.Fatalf("converted Shadowlords = %#v", shadowlords)
	}
}

func TestConvertNormalizesDuplicateShadowlordTownAssignments(t *testing.T) {
	t.Parallel()
	gam, ool := validBundleBytes()
	copy(gam[shadowlordLordsOffset:shadowlordLordsOffset+3], []byte{1, 1, 8})

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Snapshot.Progression.Shadowlords.TownAssignments; got != [save_snapshot.ShadowlordCount]int{1, 0, 8} {
		t.Fatalf("converted assignments = %#v", got)
	}
	if !hasConversionWarning(result.Warnings, "progression.shadowlords.townAssignments[1]", "duplicate_source_value") {
		t.Fatalf("duplicate assignment warning missing: %#v", result.Warnings)
	}
}

func TestConvertNormalizesSourcePartyPrefixFromPartyCount(t *testing.T) {
	gam, ool := validBundleBytes()
	gam[partyCountOffset] = 5

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}
	result, err := Convert(bundle)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	for index := range 5 {
		if got := result.Snapshot.Party.Roster[index].PartyMap; got != int(PartyMapActive) {
			t.Fatalf("roster[%d].partyMap = %#x, want active", index, got)
		}
	}
	if got := result.Snapshot.Party.Roster[5].PartyMap; got != int(PartyMapUnrecruited) {
		t.Fatalf("roster[5].partyMap = %#x, want unrecruited", got)
	}
	if !hasConversionWarning(result.Warnings, "party.roster", "party_prefix_pmap_normalized") {
		t.Fatalf("party-prefix normalization warning missing: %#v", result.Warnings)
	}
}

func TestConvertNormalizesOriginalActivePlayerSafely(t *testing.T) {
	tests := []struct {
		name        string
		active      byte
		status      CharacterStatus
		zeroHP      bool
		want        int
		wantWarning bool
	}{
		{name: "none sentinel", active: originalNothing, status: StatusGood, want: -1},
		{name: "eligible active character", active: 1, status: StatusPoisoned, want: 1},
		{name: "outside roster", active: characterCount, status: StatusGood, want: -1, wantWarning: true},
		{name: "outside active party", active: 3, status: StatusGood, want: -1, wantWarning: true},
		{name: "dead", active: 1, status: StatusDead, want: -1, wantWarning: true},
		{name: "sleeping", active: 1, status: StatusAsleep, want: -1, wantWarning: true},
		{name: "zero hp", active: 1, status: StatusGood, zeroHP: true, want: -1, wantWarning: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gam, ool := validBundleBytes()
			gam[activePlayerOffset] = test.active
			gam[charactersOffset+characterRecordSize+statusFieldOffset] = byte(test.status)
			if test.zeroHP {
				binary.LittleEndian.PutUint16(gam[charactersOffset+characterRecordSize+currentHPFieldOffset:], 0)
			}
			bundle, err := DecodeBundle(gam, ool)
			if err != nil {
				t.Fatal(err)
			}

			result, err := Convert(bundle)
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Snapshot.Continuation.ActivePlayerIndex; got != test.want {
				t.Fatalf("active player = %d, want %d", got, test.want)
			}
			if got := hasConversionWarning(result.Warnings, "continuation.activePlayerIndex", "invalid_source_value"); got != test.wantWarning {
				t.Fatalf("active-player warning = %v, want %v: %#v", got, test.wantWarning, result.Warnings)
			}
		})
	}
}

func TestConvertPreservesFullOriginalLastGaveByteRange(t *testing.T) {
	for _, lastGave := range []byte{0, 1, 100, 255} {
		gam, ool := validBundleBytes()
		gam[lastGaveOffset] = lastGave
		bundle, err := DecodeBundle(gam, ool)
		if err != nil {
			t.Fatal(err)
		}
		result, err := Convert(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if got := result.Snapshot.Continuation.BeggarKarmaCounter; got != int(lastGave) {
			t.Fatalf("last_gave %d converted to %d", lastGave, got)
		}
	}
}

func TestConvertClampsBoardedFrigateSkiffsToReduxCapacity(t *testing.T) {
	gam, ool := validBundleBytes()
	gam[playerShapeOffset] = originalShipShape
	writeObject(gam[activeObjectsOffset:], 0, ObjectRecord{
		Tile: originalShipShape, Shape: originalShipShape, Number: defaultShipHull, Misc: originalSkiffCountMask,
	})

	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}
	result, err := Convert(bundle)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result.Snapshot.PartyVehicle == nil ||
		result.Snapshot.PartyVehicle.Skiffs != references.MaxFrigateSkiffs {
		t.Fatalf("converted frigate = %#v, want %d skiffs", result.Snapshot.PartyVehicle, references.MaxFrigateSkiffs)
	}
	if !hasConversionWarning(result.Warnings, "partyVehicle.skiffs", "redux_capacity_clamped") {
		t.Fatalf("capacity warning missing: %#v", result.Warnings)
	}
}

func TestConvertUsesActiveOutdoorListWithoutMergingStaleOOL(t *testing.T) {
	bundle := validConversionBundle(t)
	clear(bundle.Game.ActiveObjects[:])
	clear(bundle.Objects.Britannia[:])
	clear(bundle.Objects.Underworld[:])
	bundle.Game.Position.Location = 0
	bundle.Game.Position.Floor = byte(references.OVERWORLD)
	bundle.Game.ActiveObjects[7] = ObjectRecord{Tile: 0x10, Shape: 0x10, X: 10, Y: 11, Level: byte(references.OVERWORLD)}
	bundle.Objects.Britannia[8] = ObjectRecord{Tile: originalCarpetObjectShape, Shape: originalCarpetObjectShape, X: 12, Y: 13}
	ratTile := byte(originalNPCShape + 4*int(references.MonsterRat))
	bundle.Objects.Underworld[9] = ObjectRecord{Tile: ratTile, Shape: ratTile, X: 14, Y: 15, Level: 0xff}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshot.MapUnits) != 2 {
		t.Fatalf("map units = %#v, want active horse and inactive-underworld rat", result.Snapshot.MapUnits)
	}
	if result.Snapshot.MapUnits[0].NPCNum != 9 || result.Snapshot.MapUnits[0].World != int(references.UNDERWORLD) ||
		result.Snapshot.MapUnits[1].NPCNum != 7 || result.Snapshot.MapUnits[1].World != int(references.OVERWORLD) {
		t.Fatalf("map units = %#v, active authority or source slots changed", result.Snapshot.MapUnits)
	}
}

func TestConvertSemanticObjectPayloads(t *testing.T) {
	bundle := validConversionBundle(t)
	clear(bundle.Game.ActiveObjects[:])
	clear(bundle.Objects.Britannia[:])
	clear(bundle.Objects.Underworld[:])
	bundle.Game.Position.Location = 0
	bundle.Game.Position.Floor = byte(references.OVERWORLD)
	bundle.Game.ActiveObjects[1] = ObjectRecord{Tile: originalChestShape, Shape: originalChestShape, X: 1, Y: 2, Number: 0x9e}
	bundle.Game.ActiveObjects[2] = ObjectRecord{Tile: originalPirateShape + 1, Shape: originalPirateShape + 1, X: 3, Y: 4, Number: 43}
	ratTile := byte(originalNPCShape + 4*int(references.MonsterRat))
	bundle.Game.ActiveObjects[3] = ObjectRecord{Tile: ratTile, Shape: ratTile, X: 5, Y: 6, Number: 77}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	units := result.Snapshot.MapUnits
	if len(units) != 3 || units[0].Kind != "chest" || units[0].SurfaceChestNumber == nil || *units[0].SurfaceChestNumber != 0x9e {
		t.Fatalf("converted chest = %#v", units)
	}
	if units[1].Kind != "enemy" || units[1].EnemyVariant != "pirate" || units[1].EnemyCurrentHP != 43 || units[1].EnemyFacing != int(references.Right) {
		t.Fatalf("converted pirate = %#v", units[1])
	}
	if units[2].Kind != "enemy" || units[2].ReferenceID != int(references.MonsterRat) || units[2].EnemyCurrentHP != references.MonsterBaseHitPoints(references.MonsterRat) {
		t.Fatalf("converted rat = %#v", units[2])
	}
}

func TestConvertOmitsOffWorldObjectsWithoutChangingListAuthority(t *testing.T) {
	for _, activeFloor := range []byte{0, 0xff} {
		t.Run(fmt.Sprintf("active_floor_%d", signedFloor(activeFloor)), func(t *testing.T) {
			bundle := validConversionBundle(t)
			bundle.Game.Position.Location = 0
			bundle.Game.Position.Floor = activeFloor
			bundle.Game.Position.X, bundle.Game.Position.Y = 162, 237
			bundle.Game.PlayerShape = originalSkiffShape + 1
			inactiveFloor := activeFloor ^ 0xff
			inactive, stale := &bundle.Objects.Underworld, &bundle.Objects.Britannia
			inactivePath := "ool.underworld"
			if activeFloor == 0xff {
				inactive, stale = stale, inactive
				inactivePath = "ool.britannia"
			}
			// Slot zero is party-vehicle state, not an outdoor map object.
			bundle.Game.ActiveObjects[0] = ObjectRecord{Tile: originalSkiffShape + 1, Level: inactiveFloor}
			bundle.Game.ActiveObjects[1] = ObjectRecord{Tile: originalSkiffShape, X: 40, Y: 41, Level: inactiveFloor}
			bundle.Game.ActiveObjects[2] = ObjectRecord{Tile: originalSkiffShape, X: 20, Y: 21, Level: activeFloor}
			bundle.Game.ActiveObjects[3] = ObjectRecord{Tile: originalGoldShape, X: 42, Y: 43, Level: inactiveFloor, Number: 10}
			bundle.Game.ActiveObjects[7] = ObjectRecord{Tile: originalCarpetObjectShape, X: 22, Y: 23, Level: activeFloor}
			// Synthetic six-vehicle off-world placement regression.
			omittedSlots := []int{1, 2, 4, 5, 10, 11}
			for _, slot := range omittedSlots {
				inactive[slot] = ObjectRecord{Tile: originalSkiffShape, X: byte(50 + slot), Y: byte(70 + slot), Level: activeFloor}
			}
			inactive[3] = ObjectRecord{Tile: originalSkiffShape, X: 24, Y: 25, Level: inactiveFloor}
			inactive[12] = ObjectRecord{Tile: originalCarpetObjectShape, X: 26, Y: 27, Level: inactiveFloor}
			inactive[13] = ObjectRecord{Tile: originalGoldShape, X: 44, Y: 45, Level: activeFloor, Number: 10}
			stale[6] = ObjectRecord{Tile: originalSkiffShape, X: 80, Y: 81, Level: activeFloor}
			stale[8] = ObjectRecord{Tile: originalSkiffShape, X: 82, Y: 83, Level: inactiveFloor}

			result, err := Convert(bundle)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := result.Snapshot
			if issues := save_snapshot.Validate(snapshot); len(issues) != 0 {
				t.Fatalf("converted snapshot validation issues = %#v", issues)
			}
			if len(snapshot.MapUnits) != 4 || len(snapshot.ItemStacks) != 0 {
				t.Fatalf("units = %#v stacks = %#v, want only four on-world vehicles", snapshot.MapUnits, snapshot.ItemStacks)
			}
			for index, world := range []int{-1, 0} {
				slots, records := []int{3, 12}, inactive
				if world == signedFloor(activeFloor) {
					slots, records = []int{2, 7}, &bundle.Game.ActiveObjects
				}
				for order, slot := range slots {
					unit, record := snapshot.MapUnits[index*2+order], records[slot]
					wantPosition := save_snapshot.PositionSnapshot{Location: 0, X: int(record.X), Y: int(record.Y), Floor: world}
					if unit.World != world || unit.NPCNum != slot || unit.ControllerOrder != order || unit.Position != wantPosition || unit.Kind != save_snapshot.MapUnitKindVehicle {
						t.Fatalf("unit = %#v, want world %d slot %d order %d position %#v", unit, world, slot, order, wantPosition)
					}
				}
			}
			wantPartyPosition := save_snapshot.PositionSnapshot{Location: 0, X: 162, Y: 237, Floor: signedFloor(activeFloor)}
			if snapshot.World.Position != wantPartyPosition || snapshot.PartyVehicle == nil || snapshot.PartyVehicle.Type != int(references.SkiffVehicle) || snapshot.PartyVehicle.CurrentDirection != int(references.Right) {
				t.Fatalf("party position = %#v vehicle = %#v", snapshot.World.Position, snapshot.PartyVehicle)
			}
			wantPaths := []string{"gam.activeObjects[1].level", "gam.activeObjects[3].level", inactivePath + "[13].level"}
			for _, slot := range omittedSlots {
				wantPaths = append(wantPaths, fmt.Sprintf("%s[%d].level", inactivePath, slot))
			}
			slices.Sort(wantPaths)
			var gotPaths []string
			for _, warning := range result.Warnings {
				if warning.Code == "off_world_object_omitted" {
					gotPaths = append(gotPaths, warning.Path)
				}
			}
			if !reflect.DeepEqual(gotPaths, wantPaths) {
				t.Fatalf("off-world warning paths = %#v, want %#v", gotPaths, wantPaths)
			}
			repeated, err := Convert(bundle)
			if err != nil || !reflect.DeepEqual(result, repeated) {
				t.Fatalf("repeated conversion differs: result %#v error %v", repeated, err)
			}
		})
	}
}

func TestConvertPreservesSmallMapObjectFloors(t *testing.T) {
	for _, activeFloor := range []byte{0xff, 0, 1} {
		t.Run(fmt.Sprintf("active_floor_%d", signedFloor(activeFloor)), func(t *testing.T) {
			bundle := validConversionBundle(t)
			bundle.Game.Position.Location = MapID(references.Britain)
			bundle.Game.Position.Floor = activeFloor
			for index, floor := range []byte{0xff, 0, 1} {
				bundle.Game.ActiveObjects[index+1] = ObjectRecord{Tile: originalChestShape, X: byte(10 + index), Y: 11, Level: floor}
			}
			bundle.Game.ActiveObjects[4] = ObjectRecord{Tile: originalGoldShape, X: 14, Y: 15, Level: activeFloor, Number: 10}
			offFloor := byte(0)
			if activeFloor == 0 {
				offFloor = 1
			}
			bundle.Game.ActiveObjects[5] = ObjectRecord{Tile: originalGoldShape, X: 16, Y: 17, Level: offFloor, Number: 10}

			result, err := Convert(bundle)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Snapshot.NeedsMapUnitReconstruction() || len(result.Snapshot.MapUnits) != 3 {
				t.Fatalf("small-map units = %#v", result.Snapshot.MapUnits)
			}
			for index, unit := range result.Snapshot.MapUnits {
				wantPosition := save_snapshot.PositionSnapshot{Location: int(references.Britain), X: 10 + index, Y: 11, Floor: index - 1}
				if unit.World != int(references.OVERWORLD) || unit.Position != wantPosition || unit.NPCNum != index+1 || unit.ControllerOrder != index {
					t.Fatalf("small-map unit = %#v, want slot %d position %#v", unit, index+1, wantPosition)
				}
			}
			if len(result.Snapshot.ItemStacks) != 1 || result.Snapshot.ItemStacks[0].Position.Floor != signedFloor(activeFloor) {
				t.Fatalf("small-map stacks = %#v", result.Snapshot.ItemStacks)
			}
			if !hasConversionWarning(result.Warnings, "gam.activeObjects[5]", "inactive_floor_loose_item_omitted") {
				t.Fatalf("off-floor loose-item warning missing: %#v", result.Warnings)
			}
			for _, warning := range result.Warnings {
				if warning.Code == "off_world_object_omitted" {
					t.Fatalf("small-map floor incorrectly treated as a world: %#v", warning)
				}
			}
		})
	}
}

func TestConvertPendingShipDelivery(t *testing.T) {
	bundle := validConversionBundle(t)
	clear(bundle.Game.ActiveObjects[:])
	clear(bundle.Objects.Britannia[:])
	clear(bundle.Objects.Underworld[:])
	bundle.Game.Position.Location = MapID(references.Britain)
	bundle.Game.Position.Floor = 0
	bundle.Game.Ship.X, bundle.Game.Ship.Y = 20, 21
	bundle.Game.Dungeon.Skiffs = 0x85

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshot.MapUnits) != 1 || result.Snapshot.MapUnits[0].Vehicle == nil {
		t.Fatalf("pending delivery units = %#v", result.Snapshot.MapUnits)
	}
	unit := result.Snapshot.MapUnits[0]
	if !unit.Vehicle.PendingDelivery || unit.Vehicle.Type != int(references.FrigateVehicle) || unit.Vehicle.Skiffs != 5 || unit.Position.X != 20 || unit.Position.Y != 21 {
		t.Fatalf("pending delivery = %#v", unit)
	}
}

func TestConvertUnsupportedObjectWarnsWithExactSourcePath(t *testing.T) {
	bundle := validConversionBundle(t)
	clear(bundle.Game.ActiveObjects[:])
	clear(bundle.Objects.Britannia[:])
	clear(bundle.Objects.Underworld[:])
	bundle.Game.Position.Location = 0
	bundle.Game.Position.Floor = byte(references.OVERWORLD)
	bundle.Game.ActiveObjects[7] = ObjectRecord{Tile: originalDeadMonsterShape, Shape: originalDeadMonsterShape, X: 4, Y: 5}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if !hasConversionWarning(result.Warnings, "gam.activeObjects[7]", "transient_corpse_omitted") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestConvertSafeActiveLooseItemsIntoOnePositionStack(t *testing.T) {
	bundle := validConversionBundle(t)
	clear(bundle.Game.ActiveObjects[:])
	clear(bundle.Objects.Britannia[:])
	clear(bundle.Objects.Underworld[:])
	bundle.Game.Position.Location = 0
	bundle.Game.Position.Floor = byte(references.OVERWORLD)
	bundle.Game.ActiveObjects[4] = ObjectRecord{Tile: originalGoldShape, Shape: originalGoldShape, X: 6, Y: 7, Number: 23}
	bundle.Game.ActiveObjects[5] = ObjectRecord{Tile: originalWeaponShape, Shape: originalWeaponShape, X: 6, Y: 7, Number: byte(references.Arrows)}
	bundle.Game.ActiveObjects[6] = ObjectRecord{Tile: originalPotionShape, Shape: originalPotionShape, X: 8, Y: 9, Number: 8}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshot.ItemStacks) != 1 || len(result.Snapshot.ItemStacks[0].Items) != 2 {
		t.Fatalf("item stacks = %#v", result.Snapshot.ItemStacks)
	}
	items := result.Snapshot.ItemStacks[0].Items
	if items[0] != (save_snapshot.ItemSnapshot{Type: int(references.ItemTypeEquipment), ID: int(references.Arrows), Quantity: 5}) ||
		items[1] != (save_snapshot.ItemSnapshot{Type: int(references.ItemTypeProvision), ID: int(references.Gold), Quantity: 23}) {
		t.Fatalf("stack items = %#v", items)
	}
	if !hasConversionWarning(result.Warnings, "gam.activeObjects[6].number", "invalid_loose_item_number") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestConvertLooseMoonstoneRequiresMatchingProgressionState(t *testing.T) {
	bundle := validConversionBundle(t)
	clear(bundle.Game.ActiveObjects[:])
	clear(bundle.Objects.Britannia[:])
	clear(bundle.Objects.Underworld[:])
	bundle.Game.Position.Location = 0
	bundle.Game.Position.Floor = byte(references.OVERWORLD)
	bundle.Game.Moonstones[2] = Moonstone{X: 10, Y: 11, Map: 0, Level: 0}
	bundle.Game.ActiveObjects[3] = ObjectRecord{Tile: originalMoonstoneShape, Shape: originalMoonstoneShape, X: 10, Y: 11, Number: 2}
	bundle.Game.ActiveObjects[4] = ObjectRecord{Tile: originalMoonstoneShape, Shape: originalMoonstoneShape, X: 12, Y: 13, Number: 3}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshot.ItemStacks) != 1 || !result.Snapshot.Progression.Moonstones[2].IsFound {
		t.Fatalf("moonstone conversion = stacks %#v progression %#v", result.Snapshot.ItemStacks, result.Snapshot.Progression.Moonstones[2])
	}
	if !hasConversionWarning(result.Warnings, "gam.activeObjects[4].number", "moonstone_state_mismatch") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestConvertInactiveLooseItemWarnsInsteadOfUsingActiveItemStack(t *testing.T) {
	bundle := validConversionBundle(t)
	clear(bundle.Game.ActiveObjects[:])
	clear(bundle.Objects.Britannia[:])
	clear(bundle.Objects.Underworld[:])
	bundle.Game.Position.Location = 0
	bundle.Game.Position.Floor = byte(references.OVERWORLD)
	bundle.Objects.Underworld[12] = ObjectRecord{Tile: originalGoldShape, Shape: originalGoldShape, X: 4, Y: 5, Level: 0xff, Number: 10}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshot.ItemStacks) != 0 || !hasConversionWarning(result.Warnings, "ool.underworld[12]", "inactive_loose_item_omitted") {
		t.Fatalf("stacks %#v warnings %#v", result.Snapshot.ItemStacks, result.Warnings)
	}
}

func validConversionBundle(t *testing.T) Bundle {
	t.Helper()
	gam, ool := validBundleBytes()
	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatal(err)
	}

	return bundle
}

func TestConvertDeterministicSeedAndWarnings(t *testing.T) {
	gam, ool := fullConversionBundleBytes()
	firstBundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatal(err)
	}
	secondBundle, err := DecodeBundle(append([]byte(nil), gam...), append([]byte(nil), ool...))
	if err != nil {
		t.Fatal(err)
	}
	first, err := Convert(firstBundle)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Convert(secondBundle)
	if err != nil {
		t.Fatal(err)
	}
	if first.Snapshot.Random != second.Snapshot.Random || !reflect.DeepEqual(first.Warnings, second.Warnings) {
		t.Fatalf("same source converted differently: first %#v second %#v", first, second)
	}

	changedGam := append([]byte(nil), gam...)
	binary.LittleEndian.PutUint16(changedGam[goldOffset:], 8887)
	changedBundle, err := DecodeBundle(changedGam, ool)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := Convert(changedBundle)
	if err != nil {
		t.Fatal(err)
	}
	if first.Snapshot.Random.Seed == changed.Snapshot.Random.Seed {
		t.Fatal("different source state produced the same deterministic seed")
	}
}

func TestConvertWarningOrderingAndSafeShadowlordMapping(t *testing.T) {
	gam, ool := validBundleBytes()
	gam[playerShapeOffset] = 0x30
	gam[lastCalledOffset] = 7
	gam[spellOffset] = 1
	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if result.Snapshot.Progression.Shadowlords.LastCalled != -1 || result.Snapshot.PartyVehicle != nil {
		t.Fatalf("unsafe source values were retained: shadowlords %#v vehicle %#v", result.Snapshot.Progression.Shadowlords, result.Snapshot.PartyVehicle)
	}
	wantPaths := []string{"effects", "mapUnits", "partyVehicle", "progression.shadowlords.lastCalled", "redux", "world.lastLargeMapPosition"}
	if len(result.Warnings) != len(wantPaths) {
		t.Fatalf("warnings = %#v, want paths %#v", result.Warnings, wantPaths)
	}
	for index, want := range wantPaths {
		if result.Warnings[index].Path != want {
			t.Fatalf("warning[%d] = %#v, want path %q", index, result.Warnings[index], want)
		}
	}
}

func TestConvertUsesDurableActiveSpellAndOmitsTransientActionFlags(t *testing.T) {
	tests := []struct {
		name            string
		activeSpell     byte
		duration        byte
		spell           byte
		magic           byte
		smashed         byte
		wantEffects     save_snapshot.EffectsSnapshot
		wantWarningPath string
		wantWarningCode string
	}{
		{
			name:        "duration spell",
			activeSpell: 'P',
			duration:    37,
			wantEffects: save_snapshot.EffectsSnapshot{DurationType: 'P', DurationTurns: 37},
		},
		{
			name:        "Quickness import starts unsaved surface cadence at false",
			activeSpell: 'Q',
			duration:    37,
			wantEffects: save_snapshot.EffectsSnapshot{DurationType: 'Q', DurationTurns: 37},
		},
		{
			name:        "legacy smashed",
			smashed:     25,
			wantEffects: save_snapshot.EffectsSnapshot{Smashed: 25},
		},
		{
			name:        "amulet",
			activeSpell: originalAmuletEffect,
			duration:    255,
			wantEffects: save_snapshot.EffectsSnapshot{Amulet: save_snapshot.ToggleSnapshot{Active: true, Turns: 255}},
		},
		{
			name:        "crown",
			activeSpell: originalCrownEffect,
			duration:    255,
			wantEffects: save_snapshot.EffectsSnapshot{Crown: save_snapshot.ToggleSnapshot{Active: true, Turns: 255}},
		},
		{
			name:        "badge",
			activeSpell: originalBadgeEffect,
			duration:    255,
			wantEffects: save_snapshot.EffectsSnapshot{Badge: save_snapshot.ToggleSnapshot{Active: true, Turns: 255}},
		},
		{
			name:            "transient action flags",
			spell:           1,
			magic:           1,
			wantEffects:     save_snapshot.EffectsSnapshot{},
			wantWarningPath: "effects",
			wantWarningCode: "transient_action_flags_omitted",
		},
		{
			name:            "unknown active effect",
			activeSpell:     'X',
			duration:        20,
			wantEffects:     save_snapshot.EffectsSnapshot{},
			wantWarningPath: "effects.activeSpell",
			wantWarningCode: "unsupported_source_value",
		},
		{
			name:            "duration effect without duration",
			activeSpell:     'Q',
			wantEffects:     save_snapshot.EffectsSnapshot{},
			wantWarningPath: "effects.activeSpell",
			wantWarningCode: "invalid_duration_omitted",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gam, ool := validBundleBytes()
			gam[activeSpellOffset] = test.activeSpell
			gam[spellDurationOffset] = test.duration
			gam[spellOffset] = test.spell
			gam[magicOffset] = test.magic
			gam[smashedOffset] = test.smashed
			bundle, err := DecodeBundle(gam, ool)
			if err != nil {
				t.Fatal(err)
			}

			result, err := Convert(bundle)
			if err != nil {
				t.Fatal(err)
			}
			if result.Snapshot.Effects != test.wantEffects {
				t.Fatalf("effects = %#v, want %#v", result.Snapshot.Effects, test.wantEffects)
			}
			for _, warning := range result.Warnings {
				if warning.Path == "effects.drunk" {
					t.Fatalf("conversion emitted unavailable-drunk warning: %#v", warning)
				}
			}
			if test.wantWarningCode != "" && !hasConversionWarning(result.Warnings, test.wantWarningPath, test.wantWarningCode) {
				t.Fatalf("warning %s/%s missing: %#v", test.wantWarningPath, test.wantWarningCode, result.Warnings)
			}
		})
	}
}

func TestConvertPreservesActiveDungeonAndAggregatesInvalidNativeState(t *testing.T) {
	t.Run("dungeon", func(t *testing.T) {
		gam, ool := validBundleBytes()
		gam[locationOffset], gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 33, 1, 2, 3
		gam[dungeonLastDirOffset], gam[dungeonFacingOffset], gam[dungeonTypeOffset] = 5, byte(DungeonNorth), byte(DungeonTypeDungeon)
		for index := range save_snapshot.DungeonCellCount {
			gam[dungeonMapOffset+index] = byte(index)
		}
		writeObject(gam[activeObjectsOffset:], 1, ObjectRecord{Tile: 2, Shape: 2, X: 6, Y: 5, Level: 1, Number: byte(references.MonsterSpider), Misc: originalNothing})
		writeObject(gam[activeObjectsOffset:], 2, ObjectRecord{X: 5, Y: 5})
		bundle, err := DecodeBundle(gam, ool)
		if err != nil {
			t.Fatal(err)
		}

		result, err := Convert(bundle)
		if err != nil {
			t.Fatal(err)
		}
		want := &save_snapshot.DungeonSnapshot{
			Location: 33, Cells: make([]int, save_snapshot.DungeonCellCount),
			Facing: int(DungeonNorth), LastDirection: 5,
			Wanderer: &save_snapshot.DungeonWandererSnapshot{
				Monster: int(references.MonsterSpider), Floor: 1, X: 6, Y: 5,
				PreviousX: 5, PreviousY: 5, OnCeiling: true,
			},
		}
		for index := range want.Cells {
			want.Cells[index] = index & 0xff
		}
		if !reflect.DeepEqual(result.Snapshot.Dungeon, want) {
			t.Fatalf("dungeon snapshot = %#v, want %#v", result.Snapshot.Dungeon, want)
		}
	})

	t.Run("dungeon rat uses tile zero", func(t *testing.T) {
		gam, ool := validBundleBytes()
		gam[locationOffset], gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 33, 1, 2, 3
		gam[dungeonTypeOffset] = byte(DungeonTypeDungeon)
		writeObject(gam[activeObjectsOffset:], 1, ObjectRecord{
			Tile: 0, X: 6, Y: 5, Level: 1, Number: byte(references.MonsterRat),
		})
		bundle, err := DecodeBundle(gam, ool)
		if err != nil {
			t.Fatal(err)
		}

		result, err := Convert(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if result.Snapshot.Dungeon.Wanderer == nil ||
			result.Snapshot.Dungeon.Wanderer.Monster != int(references.MonsterRat) {
			t.Fatalf("tile-zero dungeon wanderer = %#v", result.Snapshot.Dungeon.Wanderer)
		}
	})

	t.Run("native validation", func(t *testing.T) {
		gam, ool := validBundleBytes()
		bundle, err := DecodeBundle(gam, ool)
		if err != nil {
			t.Fatal(err)
		}
		bundle.Game.Wind = 9
		bundle.Game.Inventory.Food = 10000

		result, err := Convert(bundle)
		if !reflect.DeepEqual(result, ConversionResult{}) {
			t.Fatalf("invalid conversion returned partial result: %#v", result)
		}
		var validationError *save_snapshot.ValidationError
		if !errors.Is(err, save_snapshot.ErrInvalidSnapshot) || !errors.As(err, &validationError) || len(validationError.Issues) != 2 {
			t.Fatalf("Convert() validation error = %#v", err)
		}
	})
}

func TestConvertedSnapshotReconstructionHintIsNotPersisted(t *testing.T) {
	gam, ool := fullConversionBundleBytes()
	bundle, err := DecodeBundle(gam, ool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Convert(bundle)
	if err != nil {
		t.Fatal(err)
	}
	envelope := save_snapshot.Envelope{
		Format:          save_snapshot.EnvelopeFormat,
		EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{
			Name:        "Legacy conversion",
			Description: "Synthetic original save conversion.",
			CreatedAt:   time.Date(2026, time.August, 14, 12, 0, 0, 0, time.UTC),
			GameVersion: "test",
		},
		Snapshot: result.Snapshot,
	}
	encoded, err := save_snapshot.EncodeCanonical(envelope)
	if err != nil {
		t.Fatalf("EncodeCanonical(transient conversion) error = %v", err)
	}
	decoded, err := save_snapshot.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode(encoded conversion) error = %v", err)
	}
	if decoded.Snapshot.NeedsMapUnitReconstruction() {
		t.Fatal("transient map-unit reconstruction hint persisted in native JSON")
	}
	if !reflect.DeepEqual(envelope, decoded) {
		t.Fatal("current native conversion round trip must preserve every authoritative field and metadata")
	}
	for _, dungeon := range decoded.Snapshot.DungeonJournal {
		for _, site := range dungeon.Sites {
			if site.Survey != (journal.SurveyedLocation{}) {
				t.Fatal("legacy import must not invent original or flexible survey evidence")
			}
		}
	}
	for _, moonstone := range decoded.Snapshot.MoonstoneJournal {
		if moonstone.Survey != (journal.SurveyedLocation{}) {
			t.Fatal("carried legacy moonstones must not acquire measurement-policy provenance")
		}
	}
}

func fullConversionBundleBytes() ([]byte, []byte) {
	gam, ool := validBundleBytes()
	copy(gam[charactersOffset:charactersOffset+9], "ABCDEFGHI")
	for index := range characterCount {
		base := charactersOffset + index*characterRecordSize
		gam[base+0x18] = byte(index + 3)
		for equipmentIndex := range 6 {
			gam[base+0x19+equipmentIndex] = byte(index*6 + equipmentIndex + 1)
		}
		gam[base+partyMapFieldOffset] = byte(PartyMapUnrecruited)
	}
	for index := 0; index < 3; index++ {
		gam[charactersOffset+index*characterRecordSize+partyMapFieldOffset] = byte(PartyMapActive)
	}
	gam[charactersOffset+3*characterRecordSize+partyMapFieldOffset] = byte(PartyMapMartyr)
	gam[charactersOffset+4*characterRecordSize+partyMapFieldOffset] = byte(PartyMapUnrecruited)
	gam[charactersOffset+5*characterRecordSize+partyMapFieldOffset] = 1
	gam[charactersOffset+6*characterRecordSize+partyMapFieldOffset] = 40
	gam[partyCountOffset] = 3
	gam[activePlayerOffset] = 2
	gam[lastGaveOffset] = 0xff

	binary.LittleEndian.PutUint16(gam[foodOffset:], 9999)
	binary.LittleEndian.PutUint16(gam[goldOffset:], 8888)
	gam[keysOffset], gam[gemsOffset], gam[torchesOffset], gam[skullKeysOffset] = 7, 8, 9, 10
	gam[grapplesOffset], gam[carpetsOffset] = 3, 4
	gam[amuletOffset], gam[crownOffset], gam[sceptreOffset] = 0xff, 0xff, 0xff
	copy(gam[shardsOffset:shardsOffset+4], []byte{0xff, 0xff, 0xff, 0})
	copy(gam[specialItemsOffset:specialItemsOffset+6], []byte{0xff, 0x81, 0xff, 0xff, 0xff, 0xff})
	fillSequence(gam[equipmentOffset:equipmentOffset+48], 1)
	fillSequence(gam[preparedSpellsOffset:preparedSpellsOffset+48], 2)
	fillSequence(gam[scrollsOffset:scrollsOffset+8], 3)
	fillSequence(gam[potionsOffset:potionsOffset+8], 4)
	fillSequence(gam[reagentsOffset:reagentsOffset+8], 5)
	gam[skullKeysDayOffset] = 24
	copy(gam[reagentCacheDaysOffset:reagentCacheDaysOffset+3], []byte{25, 26, 27})

	for index := range 8 {
		gam[moonstoneXOffset+index] = byte(20 + index)
		gam[moonstoneYOffset+index] = byte(30 + index)
		gam[moonstoneMapOffset+index] = byte(index)
		gam[moonstoneLevelOffset+index] = byte(index % 2)
	}
	gam[moonstoneMapOffset+1] = 0xff
	gam[searchFlagsOffset+(21>>3)] |= 1 << (21 & 7)

	binary.LittleEndian.PutUint16(gam[yearOffset:], 999)
	gam[monthOffset], gam[dayOffset], gam[hourOffset], gam[minuteOffset] = 13, 28, 23, 59
	gam[oldHourOffset], gam[lastCampedOffset], gam[lastWandererOffset] = 22, 23, 13
	gam[karmaOffset] = 77
	gam[locationOffset], gam[floorOffset], gam[partyXOffset], gam[partyYOffset] = 0, 0xff, 220, 230
	gam[activeSpellOffset], gam[spellDurationOffset] = 'P', 37
	gam[magicLightOffset], gam[torchTurnsOffset] = 44, 55
	gam[windOffset], gam[windUpdateCountOffset] = 3, 12

	gam[shrineActiveOffset], gam[shrineCompleteOffset] = 0x81, 0x40
	clear(gam[closedShrinesOffset : closedShrinesOffset+8])
	gam[closedShrinesOffset] = 0xff
	clear(gam[openDungeonsOffset : openDungeonsOffset+8])
	gam[openDungeonsOffset], gam[openDungeonsOffset+7] = 0x80, 0xff
	fillSequence(gam[dungeonRoomsOffset:dungeonRoomsOffset+14], 41)
	copy(gam[shadowlordLordsOffset:shadowlordLordsOffset+3], []byte{0xff, 0, 0xfe})
	gam[lastCalledOffset] = 2

	gam[metNPCsOffset] = 0x01
	gam[metNPCsOffset+127] = 0x80
	gam[deadNPCsOffset+17*4+3] = 0x40
	gam[deadNPCsOffset+127] = 0x80

	gam[playerShapeOffset] = originalShipShape + 1
	gam[sailingOffset] = 1
	gam[skiffsOffset] = 0x85
	writeObject(gam[activeObjectsOffset:], 0, ObjectRecord{
		Tile: originalShipShape + 1, Shape: originalShipShape + 1,
		Number: defaultShipHull, Misc: 5,
	})

	return gam, ool
}

func hasConversionWarning(warnings []ConversionWarning, path, code string) bool {
	for _, warning := range warnings {
		if warning.Path == path && warning.Code == code {
			return true
		}
	}

	return false
}
