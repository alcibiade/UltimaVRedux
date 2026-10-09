package new_game

import (
	"reflect"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/legacy_save"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestPrepareInitialSnapshotWithOriginalTemplates(t *testing.T) {
	directory := originaldata.Directory(t, files.INIT_GAM, files.INIT_OOL)

	conversion, err := PrepareInitialSnapshot(directory, testCreationResult(party_state.Male))
	if err != nil {
		t.Fatal(err)
	}
	if issues := save_snapshot.Validate(conversion.Snapshot); len(issues) != 0 {
		t.Fatalf("original prepared snapshot validation issues = %#v", issues)
	}

	snapshot := conversion.Snapshot
	avatar := snapshot.Party.Roster[0]
	if avatar.Name != "Avatar" || avatar.Gender != int(legacy_save.GenderMale) || avatar.Class != int(legacy_save.ClassAvatar) || avatar.Status != int(legacy_save.StatusGood) || avatar.Strength != 20 || avatar.Dexterity != 15 || avatar.Intelligence != 23 || avatar.CurrentMP != 23 || avatar.CurrentHP != 60 || avatar.MaximumHP != 60 || avatar.Experience != 150 || avatar.Level != 2 {
		t.Fatalf("original Avatar state = %#v", avatar)
	}
	if snapshot.Party.Roster[1].Name != "Shamino" || snapshot.Party.Roster[2].Name != "Iolo" {
		t.Fatalf("original active roster prefix = %#v", snapshot.Party.Roster[:3])
	}
	for index, character := range snapshot.Party.Roster[:3] {
		if !character.Occupied || character.PartyMap != int(legacy_save.PartyMapActive) {
			t.Fatalf("original roster[%d] = %#v", index, character)
		}
	}
	if snapshot.Inventory.Food != 63 || snapshot.Inventory.Gold != 150 || snapshot.Inventory.Keys != 2 || snapshot.Inventory.Torches != 4 || snapshot.Party.Karma != 75 {
		t.Fatalf("original resources = party %#v inventory %#v", snapshot.Party, snapshot.Inventory)
	}
	calendar := snapshot.World.Calendar
	position := snapshot.World.Position
	if calendar.Year != 139 || calendar.Month != 4 || calendar.Day != 5 || calendar.Hour != 8 || calendar.Minute != 35 || position.Location != 13 || position.Floor != 0 || position.X != 15 || position.Y != 15 {
		t.Fatalf("original starting world = calendar %#v position %#v", calendar, position)
	}
	if snapshot.Random != (save_snapshot.RandomSnapshot{Seed: 42, SourceCalls: 64}) {
		t.Fatalf("original prepared random metadata = %#v", snapshot.Random)
	}
	if snapshot.Continuation.ActivePlayerIndex != -1 || snapshot.Continuation.BeggarKarmaCounter != 0 {
		t.Fatalf("original initial continuation = %#v", snapshot.Continuation)
	}

	gameConfig, err := config.NewUltimaVConfigurationWithPath(directory)
	if err != nil {
		t.Fatalf("loading selected original configuration: %v", err)
	}
	gameReferences, err := references.NewGameReferences(gameConfig)
	if err != nil {
		t.Fatalf("loading selected original reference data: %v", err)
	}
	if err := game_state.ValidateSaveSnapshotCompatibility(snapshot, gameReferences); err != nil {
		t.Fatalf("prepared snapshot compatibility error = %v", err)
	}
	hydrated, err := game_state.NewGameStateFromSaveSnapshot(snapshot, gameConfig, gameReferences, 20, 12)
	if err != nil {
		t.Fatalf("hydrating prepared snapshot: %v", err)
	}
	for _, spell := range hydrated.CurrentSpellCatalog() {
		if spell.ID == references.RelXenBet.ID() || spell.ID == references.InXenMani.ID() {
			t.Fatalf("fresh game exposes hidden spell %+v", spell)
		}
	}
	openingRats := map[int]references.Position{
		1: {X: 23, Y: 8},
		2: {X: 13, Y: 5},
		3: {X: 3, Y: 6},
	}
	for slot, wantPosition := range openingRats {
		found := false
		for _, unit := range *hydrated.CurrentNPCAIController.GetNpcs() {
			friendly, ok := unit.(*map_units.NPCFriendly)
			if !ok || friendly.MapUnitDetails().NPCNum != slot {
				continue
			}
			monster, classified := friendly.NPCReference.SmallMapCombatMonsterType()
			if !classified || monster != references.MonsterRat || friendly.NPCReference.Location != references.IolosHut ||
				friendly.Pos() != wantPosition || friendly.Floor() != references.MainFloor {
				t.Fatalf("opening Iolo's Hut rat slot %d = unit %#v reference %#v", slot, friendly.MapUnitDetails(), friendly.NPCReference)
			}
			found = true
			break
		}
		if !found {
			t.Fatalf("opening Iolo's Hut rat slot %d is absent", slot)
		}
	}
	if smith := hydrated.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
		(references.Position{X: 27, Y: 3}), references.MainFloor,
	); smith != nil {
		t.Fatalf("Smith is present in the opening Iolo's Hut scene: %#v", smith.MapUnitDetails())
	}
	recaptured, err := hydrated.CaptureSaveSnapshot()
	if err != nil {
		t.Fatalf("recapturing prepared snapshot: %v", err)
	}
	if recaptured.NeedsMapUnitReconstruction() || len(recaptured.MapUnits) == 0 {
		t.Fatalf("recaptured OOL bridge = redux %#v map units %d", recaptured.Redux, len(recaptured.MapUnits))
	}
	if recaptured.Random != snapshot.Random ||
		recaptured.Continuation != snapshot.Continuation ||
		!reflect.DeepEqual(recaptured.Party, snapshot.Party) ||
		!reflect.DeepEqual(recaptured.Inventory, snapshot.Inventory) ||
		!reflect.DeepEqual(recaptured.World, snapshot.World) ||
		!reflect.DeepEqual(recaptured.Progression, snapshot.Progression) {
		t.Fatal("hydrate/capture changed durable prepared state")
	}
}
