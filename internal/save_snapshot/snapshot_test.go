package save_snapshot

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSnapshotPreservesOrderedSixteenSlotRoster(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Party.Roster[15] = CharacterSnapshot{
		Occupied: true, Name: "Last", Class: 'B', Status: 'G', Gender: 0x0c, Level: 2,
		Strength: 10, Dexterity: 12, Intelligence: 11, CurrentHP: 20, MaximumHP: 20,
		CurrentMP: 5, PartyMap: 0xff,
	}

	data, err := EncodeCanonical(validEnvelope(snapshot))
	if err != nil {
		t.Fatalf("EncodeCanonical() error = %v", err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(decoded.Snapshot.Party.Roster) != RosterSlotCount {
		t.Fatalf("roster length = %d, want %d", len(decoded.Snapshot.Party.Roster), RosterSlotCount)
	}
	if !reflect.DeepEqual(decoded.Snapshot.Party.Roster, snapshot.Party.Roster) {
		t.Fatal("roster order or vacant slots changed during round trip")
	}
}

func TestSnapshotRoundTripPreservesDungeonRooms(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	want := [DungeonRoomProgressBytes]byte{0x01, 0x80, 0x55, 0xaa, 0xff, 0x13, 0x37, 0x42, 0x7f, 0x02, 0x04, 0x08, 0x10, 0x20}
	snapshot.Progression.DungeonRooms = want

	data, err := EncodeCanonical(validEnvelope(snapshot))
	if err != nil {
		t.Fatalf("EncodeCanonical() error = %v", err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got := decoded.Snapshot.Progression.DungeonRooms; got != want {
		t.Fatalf("dungeon rooms = %#v, want %#v", got, want)
	}
}

func TestValidateReturnsStructuredIssuesWithoutMutation(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Version = 99
	snapshot.Party.MetNPCs = []NPCFlagSnapshot{{Location: 1, NPC: 2}, {Location: 1, NPC: 2}, {Location: 1, NPC: 32}}
	snapshot.World.Calendar.Hour = 24
	snapshot.Wind.Direction = 99
	snapshot.ItemStacks = []ItemStackSnapshot{{Position: validPosition(), Items: []ItemSnapshot{{Type: 9, ID: 5, Quantity: 0}}}}
	snapshot.MapMutations = []MapMutationSnapshot{{Position: validPosition(), Kind: "unknown", Value: -2}}
	before := snapshot

	issues := Validate(snapshot)
	if len(issues) < 7 {
		t.Fatalf("Validate() returned %d issues, want comprehensive issues: %#v", len(issues), issues)
	}
	if !hasIssue(issues, "version", "unsupported_version") ||
		!hasIssue(issues, "party.metNpcs[1]", "duplicate") ||
		!hasIssue(issues, "party.metNpcs[2]", "out_of_range") ||
		!hasIssue(issues, "mapMutations[0].kind", "unsupported") {
		t.Fatalf("Validate() issues missing expected paths/codes: %#v", issues)
	}
	if !reflect.DeepEqual(snapshot, before) {
		t.Fatal("Validate() mutated its input")
	}
}

func TestValidateAcceptsCompleteSemanticSnapshot(t *testing.T) {
	t.Parallel()
	if issues := Validate(validSnapshot()); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateSurfaceChestAuthority(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		kind   string
		number int
		path   string
		code   string
	}{
		{name: "NPC byte", kind: "npc", number: 255},
		{name: "semantic chest byte", kind: "chest", number: 255},
		{name: "negative", kind: "npc", number: -1, path: "mapUnits[0].surfaceChestNumber", code: "out_of_range"},
		{name: "above byte", kind: "npc", number: 256, path: "mapUnits[0].surfaceChestNumber", code: "out_of_range"},
		{name: "enemy", kind: "enemy", number: 30, path: "mapUnits[0].surfaceChestNumber", code: "inconsistent"},
		{name: "vehicle", kind: "vehicle", number: 30, path: "mapUnits[0].surfaceChestNumber", code: "inconsistent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			unit := &snapshot.MapUnits[0]
			unit.Kind = test.kind
			unit.SurfaceChestNumber = &test.number
			switch test.kind {
			case "npc", "chest":
				unit.EnemyVariant = ""
				unit.EnemyCurrentHP = 0
			case "vehicle":
				unit.EnemyVariant = ""
				unit.EnemyCurrentHP = 0
				unit.Vehicle = &VehicleSnapshot{Type: 1, Hull: 99}
			}
			issues := Validate(snapshot)
			if test.path == "" && len(issues) != 0 {
				t.Fatalf("Validate() issues = %#v", issues)
			}
			if test.path != "" && !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateSemanticChestRequiresSurfaceState(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	unit := &snapshot.MapUnits[0]
	unit.Kind = "chest"
	unit.EnemyVariant = ""
	unit.EnemyCurrentHP = 0
	unit.Shadowlord = nil
	if issues := Validate(snapshot); !hasIssue(issues, "mapUnits[0].surfaceChestNumber", "required") {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateAllowsDormantAutomaticAIOverrideProvenanceAfterAssignmentChange(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		presence   references.Shadowlord
		aiType     references.AiType
		provenance references.AIOverrideProvenance
	}{
		{
			name: "Astaroth law flee", presence: references.Nosfentor, aiType: references.FleeFromAvatar,
			provenance: references.AIOverrideProvenanceAutomaticAstaroth,
		},
		{
			name: "Nosfentor law hostility", presence: references.Astaroth, aiType: references.ExtortOrAttackOrFollow,
			provenance: references.AIOverrideProvenanceAutomaticNosfentor,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.World.Position.Location = int(references.Britain)
			unit := &snapshot.MapUnits[0]
			unit.Kind = "npc"
			unit.EnemyVariant = ""
			unit.EnemyCurrentHP = 0
			unit.Position.Location = int(references.Britain)
			unit.OverriddenAI = int(test.aiType)
			unit.AIOverrideProvenance = string(test.provenance)
			snapshot.Progression.Shadowlords.AutomaticPresence = int(test.presence)
			snapshot.Progression.Shadowlords.TownAssignments[test.presence] = int(references.Britain)

			if issues := Validate(snapshot); len(issues) != 0 {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateHostileAIOverrideProvenanceActions(t *testing.T) {
	t.Parallel()
	for _, provenance := range []references.AIOverrideProvenance{
		references.AIOverrideProvenanceAutomaticAstaroth,
		references.AIOverrideProvenanceAutomaticNosfentor,
		references.AIOverrideProvenanceGuardHostility,
	} {
		for action := references.Unset; action <= references.StoneGargoyleTrigger; action++ {
			if !action.IsValid() {
				continue
			}
			t.Run(fmt.Sprintf("%s/action%d", provenance, action), func(t *testing.T) {
				snapshot := validSnapshot()
				snapshot.World.Position.Location = int(references.Britain)
				unit := &snapshot.MapUnits[0]
				unit.Kind = MapUnitKindNPC
				unit.EnemyVariant = ""
				unit.EnemyCurrentHP = 0
				unit.Position.Location = int(references.Britain)
				unit.OverriddenAI = int(action)
				unit.AIOverrideProvenance = string(provenance)

				valid := action == references.ExtortOrAttackOrFollow || action == references.DrudgeWorthThing ||
					(action == references.FleeFromAvatar && provenance != references.AIOverrideProvenanceGuardHostility)
				issues := Validate(snapshot)
				if valid {
					if len(issues) != 0 {
						t.Fatalf("Validate() rejected supported action: %#v", issues)
					}
				} else if !hasIssue(issues, "mapUnits[0].aiOverrideProvenance", "inconsistent") {
					t.Fatalf("Validate() did not reject incompatible action: %#v", issues)
				}
			})
		}
	}
}

func TestValidateAutomaticAIOverrideProvenanceRequiresActiveLocation(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.World.Position.Location = int(references.Britain)
	unit := &snapshot.MapUnits[0]
	unit.Kind = "npc"
	unit.EnemyVariant = ""
	unit.EnemyCurrentHP = 0
	unit.Position.Location = int(references.Yew)
	unit.OverriddenAI = int(references.ExtortOrAttackOrFollow)
	unit.AIOverrideProvenance = string(references.AIOverrideProvenanceAutomaticAstaroth)

	if issues := Validate(snapshot); !hasIssue(issues, "mapUnits[0].aiOverrideProvenance", "inconsistent") {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateRejectsUnsupportedOverriddenAIWithNoProvenance(t *testing.T) {
	t.Parallel()
	for _, aiType := range []int{-2, 5, 9, 99, 113} {
		snapshot := validSnapshot()
		snapshot.MapUnits[0].OverriddenAI = aiType
		snapshot.MapUnits[0].AIOverrideProvenance = string(references.AIOverrideProvenanceNone)

		if issues := Validate(snapshot); !hasIssue(issues, "mapUnits[0].overriddenAi", "unsupported") {
			t.Fatalf("Validate(overriddenAi=%d) issues = %#v", aiType, issues)
		}
	}
}

func TestQuestJournalRoundTripPreservesOrderedSemanticEvents(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Progression.Shadowlords.Alive[references.Astaroth] = false
	snapshot.Progression.Shadowlords.Alive[references.Nosfentor] = false
	snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{
		{
			Sequence: 1, Kind: QuestJournalEventShadowlordDefeated, Subject: QuestJournalSubjectAstaroth,
			OccurredOn: &BritannianDateSnapshot{Year: 139, Month: 13, Day: 28},
			Source:     QuestJournalSourceGameplay,
		},
		{
			Sequence: 2, Kind: QuestJournalEventShadowlordDefeated, Subject: QuestJournalSubjectNosfentor,
			Source: QuestJournalSourceMigration, Acknowledged: true,
		},
	}

	data, err := EncodeCanonical(validEnvelope(snapshot))
	if err != nil {
		t.Fatalf("EncodeCanonical() error = %v", err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !reflect.DeepEqual(decoded.Snapshot.Redux.QuestJournal, snapshot.Redux.QuestJournal) {
		t.Fatalf("quest journal changed\nbefore: %#v\nafter:  %#v", snapshot.Redux.QuestJournal, decoded.Snapshot.Redux.QuestJournal)
	}
}

func TestValidateQuestJournalInvariants(t *testing.T) {
	t.Parallel()
	validEvent := func() QuestJournalEventSnapshot {
		return QuestJournalEventSnapshot{
			Sequence: 1, Kind: QuestJournalEventShadowlordDefeated, Subject: QuestJournalSubjectFaulinei,
			OccurredOn: &BritannianDateSnapshot{Year: 139, Month: 1, Day: 1}, Source: QuestJournalSourceGameplay,
		}
	}
	for _, test := range []struct {
		name string
		set  func(*SaveSnapshot)
		path string
		code string
	}{
		{name: "nil", set: func(snapshot *SaveSnapshot) { snapshot.Redux.QuestJournal = nil }, path: "redux.questJournal", code: "required"},
		{name: "sequence", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.Sequence = 2
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0].sequence", code: "not_contiguous"},
		{name: "duplicate", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			second := event
			second.Sequence = 2
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event, second}
		}, path: "redux.questJournal[1]", code: "duplicate"},
		{name: "kind", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.Kind = "future"
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0].kind", code: "unsupported"},
		{name: "subject", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.Subject = "unknown"
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0].subject", code: "unsupported"},
		{name: "source", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.Source = "unknown"
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0].source", code: "unsupported"},
		{name: "date", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.OccurredOn.Month = 14
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0].occurredOn", code: "out_of_range"},
		{name: "gameplay date required", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.OccurredOn = nil
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0].occurredOn", code: "required"},
		{name: "migration date must be null", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.Source = QuestJournalSourceMigration
			event.Acknowledged = true
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0]", code: "inconsistent"},
		{name: "migration must be acknowledged", set: func(snapshot *SaveSnapshot) {
			event := validEvent()
			event.Source = QuestJournalSourceMigration
			event.OccurredOn = nil
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{event}
		}, path: "redux.questJournal[0]", code: "inconsistent"},
		{name: "authoritative Shadowlord remains alive", set: func(snapshot *SaveSnapshot) {
			snapshot.Progression.Shadowlords.Alive[references.Faulinei] = true
			snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{validEvent()}
		}, path: "redux.questJournal[0]", code: "inconsistent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Progression.Shadowlords.Alive[references.Faulinei] = false
			test.set(&snapshot)
			if issues := Validate(snapshot); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateShadowlordTownAssignments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(*SaveSnapshot)
		path string
		code string
	}{
		{name: "out of range", set: func(snapshot *SaveSnapshot) { snapshot.Progression.Shadowlords.TownAssignments[0] = 9 }, path: "progression.shadowlords.townAssignments[0]", code: "out_of_range"},
		{name: "duplicate", set: func(snapshot *SaveSnapshot) {
			snapshot.Progression.Shadowlords.TownAssignments = [ShadowlordCount]int{1, 1, 0}
		}, path: "progression.shadowlords.townAssignments[1]", code: "duplicate"},
		{name: "defeated assigned", set: func(snapshot *SaveSnapshot) {
			snapshot.Progression.Shadowlords.Alive[1] = false
			snapshot.Progression.Shadowlords.TownAssignments[1] = 2
		}, path: "progression.shadowlords.townAssignments[1]", code: "inconsistent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			snapshot := validSnapshot()
			test.set(&snapshot)
			if issues := Validate(snapshot); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateNPCFlagsUseNativeZeroBasedSmallMapBoundaries(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Party.DeadNPCs = []NPCFlagSnapshot{
		{Location: 0, NPC: 0},
		{Location: int(references.SerpentsHold - references.Moonglow), NPC: 31},
	}
	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate() rejected NPC flag boundaries: %#v", issues)
	}

	for _, location := range []int{-1, 32} {
		invalid := validSnapshot()
		invalid.Party.DeadNPCs = []NPCFlagSnapshot{{Location: location, NPC: 0}}
		if issues := Validate(invalid); !hasIssue(issues, "party.deadNpcs[0]", "out_of_range") {
			t.Fatalf("Validate(location=%d) issues = %#v", location, issues)
		}
	}
}

func TestValidateRejectsInvalidScheduleSlots(t *testing.T) {
	t.Parallel()
	for _, value := range []int{-1, 3} {
		snapshot := validSnapshot()
		snapshot.MapUnits[0].ArrivedScheduleSlot = &value
		if issues := Validate(snapshot); !hasIssue(issues, "mapUnits[0].arrivedScheduleSlot", "out_of_range") {
			t.Fatalf("Validate(arrivedScheduleSlot=%d) issues = %#v", value, issues)
		}

		snapshot = validSnapshot()
		snapshot.MapUnits[0].ConsumedTalk3ScheduleSlot = &value
		if issues := Validate(snapshot); !hasIssue(issues, "mapUnits[0].consumedTalk3ScheduleSlot", "out_of_range") {
			t.Fatalf("Validate(consumedTalk3ScheduleSlot=%d) issues = %#v", value, issues)
		}
	}
}

func TestValidateAllowsConsumedTalk3ScheduleSlotOnlyForNPCUnits(t *testing.T) {
	t.Parallel()
	value := 1
	snapshot := validSnapshot()
	snapshot.MapUnits[0].ConsumedTalk3ScheduleSlot = &value
	issues := Validate(snapshot)
	if !hasIssue(issues, "mapUnits[0].consumedTalk3ScheduleSlot", "inconsistent") {
		t.Fatalf("Validate(enemy consumedTalk3ScheduleSlot) issues = %#v", issues)
	}

	snapshot.MapUnits[0].Kind = "npc"
	snapshot.MapUnits[0].EnemyVariant = ""
	snapshot.MapUnits[0].EnemyCurrentHP = 0
	issues = Validate(snapshot)
	if hasIssue(issues, "mapUnits[0].consumedTalk3ScheduleSlot", "inconsistent") ||
		hasIssue(issues, "mapUnits[0].consumedTalk3ScheduleSlot", "out_of_range") {
		t.Fatalf("Validate(NPC consumedTalk3ScheduleSlot) issues = %#v", issues)
	}
}

func TestValidateActiveDungeonCrossFieldRules(t *testing.T) {
	t.Parallel()
	validDungeon := func() SaveSnapshot {
		snapshot := validSnapshot()
		snapshot.World.Position = PositionSnapshot{Location: 33, X: 2, Y: 3, Floor: 4}
		snapshot.Dungeon = &DungeonSnapshot{
			Location: 33, Cells: make([]int, DungeonCellCount), Facing: 1,
			LastDirection: 5, QuicknessSkipNext: true,
			Wanderer: &DungeonWandererSnapshot{
				Monster: 22, Floor: 4, X: 4, Y: 5, PreviousX: 3, PreviousY: 5, OnCeiling: true,
			},
		}

		return snapshot
	}
	if issues := Validate(validDungeon()); len(issues) != 0 {
		t.Fatalf("Validate(valid dungeon) issues = %#v", issues)
	}

	tests := []struct {
		name string
		edit func(*SaveSnapshot)
		path string
		code string
	}{
		{name: "missing dungeon", edit: func(s *SaveSnapshot) { s.Dungeon = nil }, path: "dungeon", code: "required"},
		{name: "identity mismatch", edit: func(s *SaveSnapshot) { s.Dungeon.Location = 34 }, path: "dungeon.location", code: "inconsistent"},
		{name: "short cells", edit: func(s *SaveSnapshot) { s.Dungeon.Cells = s.Dungeon.Cells[:DungeonCellCount-1] }, path: "dungeon.cells", code: "wrong_length"},
		{name: "raw cell", edit: func(s *SaveSnapshot) { s.Dungeon.Cells[511] = 256 }, path: "dungeon.cells[511]", code: "out_of_range"},
		{name: "facing", edit: func(s *SaveSnapshot) { s.Dungeon.Facing = 4 }, path: "dungeon.facing", code: "out_of_range"},
		{name: "facing overflow", edit: func(s *SaveSnapshot) { s.Dungeon.Facing = 256 }, path: "dungeon.facing", code: "out_of_range"},
		{name: "vertical direction", edit: func(s *SaveSnapshot) { s.Dungeon.LastDirection = 6 }, path: "dungeon.lastDirection", code: "out_of_range"},
		{name: "direction overflow", edit: func(s *SaveSnapshot) { s.Dungeon.LastDirection = 261 }, path: "dungeon.lastDirection", code: "out_of_range"},
		{name: "wanderer monster", edit: func(s *SaveSnapshot) { s.Dungeon.Wanderer.Monster = 48 }, path: "dungeon.wanderer.monster", code: "out_of_range"},
		{name: "non-wandering monster", edit: func(s *SaveSnapshot) { s.Dungeon.Wanderer.Monster = 39 }, path: "dungeon.wanderer.monster", code: "unsupported"},
		{name: "non-ceiling monster", edit: func(s *SaveSnapshot) { s.Dungeon.Wanderer.Monster = 28 }, path: "dungeon.wanderer.onCeiling", code: "inconsistent"},
		{name: "wanderer floor", edit: func(s *SaveSnapshot) { s.Dungeon.Wanderer.Floor = 8 }, path: "dungeon.wanderer", code: "out_of_range"},
		{name: "wanderer current", edit: func(s *SaveSnapshot) { s.Dungeon.Wanderer.X = 8 }, path: "dungeon.wanderer", code: "out_of_range"},
		{name: "wanderer previous", edit: func(s *SaveSnapshot) { s.Dungeon.Wanderer.PreviousY = -1 }, path: "dungeon.wanderer", code: "out_of_range"},
		{name: "party occupancy", edit: func(s *SaveSnapshot) { s.Dungeon.Wanderer.X, s.Dungeon.Wanderer.Y = 2, 3 }, path: "dungeon.wanderer", code: "occupied"},
		{name: "vehicle", edit: func(s *SaveSnapshot) { s.PartyVehicle = &VehicleSnapshot{Type: 1} }, path: "partyVehicle", code: "inconsistent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validDungeon()
			test.edit(&snapshot)
			if issues := Validate(snapshot); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateSurfaceSnapshotRequiresNullDungeon(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Dungeon = &DungeonSnapshot{Cells: make([]int, DungeonCellCount)}

	if issues := Validate(snapshot); !hasIssue(issues, "dungeon", "inconsistent") {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateContinuationRanges(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		edit  func(*ContinuationSnapshot)
		path  string
		valid bool
	}{
		{name: "active player absent", edit: func(continuation *ContinuationSnapshot) { continuation.ActivePlayerIndex = -1 }, valid: true},
		{name: "active player first", edit: func(continuation *ContinuationSnapshot) { continuation.ActivePlayerIndex = 0 }, valid: true},
		{name: "active player last", edit: func(continuation *ContinuationSnapshot) { continuation.ActivePlayerIndex = 5 }, valid: true},
		{name: "active player below range", edit: func(continuation *ContinuationSnapshot) { continuation.ActivePlayerIndex = -2 }, path: "continuation.activePlayerIndex"},
		{name: "active player above range", edit: func(continuation *ContinuationSnapshot) { continuation.ActivePlayerIndex = 6 }, path: "continuation.activePlayerIndex"},
		{name: "beggar counter zero", edit: func(continuation *ContinuationSnapshot) { continuation.BeggarKarmaCounter = 0 }, valid: true},
		{name: "beggar counter byte maximum", edit: func(continuation *ContinuationSnapshot) { continuation.BeggarKarmaCounter = 255 }, valid: true},
		{name: "beggar counter negative", edit: func(continuation *ContinuationSnapshot) { continuation.BeggarKarmaCounter = -1 }, path: "continuation.beggarKarmaCounter"},
		{name: "beggar counter above byte", edit: func(continuation *ContinuationSnapshot) { continuation.BeggarKarmaCounter = 256 }, path: "continuation.beggarKarmaCounter"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			test.edit(&snapshot.Continuation)
			issues := Validate(snapshot)
			if test.valid && len(issues) != 0 {
				t.Fatalf("Validate() issues = %#v", issues)
			}
			if !test.valid && !hasIssue(issues, test.path, "out_of_range") {
				t.Fatalf("Validate() issues = %#v, want %s out_of_range", issues, test.path)
			}
		})
	}
}

func TestValidateReconstructMapUnitsAllowsSemanticOverlay(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		mapUnits []MapUnitSnapshot
		valid    bool
	}{
		{name: "nil", mapUnits: nil, valid: true},
		{name: "empty", mapUnits: []MapUnitSnapshot{}, valid: true},
		{name: "semantic units", mapUnits: validSnapshot().MapUnits, valid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.MapUnits = test.mapUnits
			snapshot.MarkMapUnitsForReconstruction()
			issues := Validate(snapshot)
			if got := !hasIssue(issues, "mapUnits", "inconsistent"); got != test.valid {
				t.Fatalf("Validate() issues = %#v, valid = %v", issues, test.valid)
			}
		})
	}
}

func TestValidateDungeonDiscoveriesRequiresSortedUniqueSupportedCells(t *testing.T) {
	t.Parallel()
	valid := []DungeonDiscoverySnapshot{
		{Location: 33, Floor: 0, X: 1, Y: 2, Kind: DungeonDiscoveryHiddenDoor},
		{Location: 33, Floor: 1, X: 1, Y: 2, Kind: DungeonDiscoveryHiddenPit},
	}
	snapshot := validSnapshot()
	snapshot.Redux.DungeonDiscoveries = valid
	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate(valid discoveries) issues = %#v", issues)
	}

	for _, test := range []struct {
		name        string
		discoveries []DungeonDiscoverySnapshot
		path        string
		code        string
	}{
		{
			name: "null", discoveries: nil,
			path: "redux.dungeonDiscoveries", code: "required",
		},
		{
			name: "out of range",
			discoveries: []DungeonDiscoverySnapshot{
				{Location: 33, Floor: 8, X: 1, Y: 2, Kind: DungeonDiscoveryHiddenPit},
			},
			path: "redux.dungeonDiscoveries[0]", code: "out_of_range",
		},
		{
			name: "unsupported kind",
			discoveries: []DungeonDiscoverySnapshot{
				{Location: 33, Floor: 0, X: 1, Y: 2, Kind: "other"},
			},
			path: "redux.dungeonDiscoveries[0].kind", code: "unsupported",
		},
		{
			name: "duplicate coordinate",
			discoveries: []DungeonDiscoverySnapshot{
				{Location: 33, Floor: 0, X: 1, Y: 2, Kind: DungeonDiscoveryHiddenDoor},
				{Location: 33, Floor: 0, X: 1, Y: 2, Kind: DungeonDiscoveryHiddenPit},
			},
			path: "redux.dungeonDiscoveries[1]", code: "duplicate",
		},
		{
			name: "not sorted", discoveries: []DungeonDiscoverySnapshot{valid[1], valid[0]},
			path: "redux.dungeonDiscoveries[1]", code: "not_sorted",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := validSnapshot()
			candidate.Redux.DungeonDiscoveries = test.discoveries
			if issues := Validate(candidate); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestSnapshotRoundTripPreservesDungeonDiscoveries(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Redux.DungeonDiscoveries = []DungeonDiscoverySnapshot{
		{Location: 33, Floor: 0, X: 1, Y: 2, Kind: DungeonDiscoveryHiddenDoor},
		{Location: 40, Floor: 7, X: 7, Y: 7, Kind: DungeonDiscoveryHiddenPit},
	}

	data, err := EncodeCanonical(validEnvelope(snapshot))
	if err != nil {
		t.Fatalf("EncodeCanonical() error = %v", err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !reflect.DeepEqual(snapshot.Redux.DungeonDiscoveries, decoded.Snapshot.Redux.DungeonDiscoveries) {
		t.Fatalf(
			"dungeon discoveries = %#v, want %#v",
			decoded.Snapshot.Redux.DungeonDiscoveries,
			snapshot.Redux.DungeonDiscoveries,
		)
	}
}

func TestSnapshotRoundTripPreservesPlayerKnowledge(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Redux.PlayerKnowledge = append(validLoreKnowledge(), validPlayerKnowledge()...)

	data, err := EncodeCanonical(validEnvelope(snapshot))
	if err != nil {
		t.Fatalf("EncodeCanonical() error = %v", err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !reflect.DeepEqual(decoded.Snapshot.Redux.PlayerKnowledge, snapshot.Redux.PlayerKnowledge) {
		t.Fatalf("player knowledge = %#v, want %#v", decoded.Snapshot.Redux.PlayerKnowledge, snapshot.Redux.PlayerKnowledge)
	}
}

func TestValidateCurrentLoreKnowledge(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Redux.PlayerKnowledge = validLoreKnowledge()
	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate(valid lore knowledge) issues = %#v", issues)
	}
	for _, entry := range validLoreKnowledge() {
		t.Run(entry.Category+"."+entry.Key+" independently", func(t *testing.T) {
			candidate := validSnapshot()
			candidate.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{entry}
			if issues := Validate(candidate); len(issues) != 0 {
				t.Fatalf("Validate(independent lore knowledge) issues = %#v", issues)
			}
		})
	}

	tests := []struct {
		name string
		edit func(*PlayerKnowledgeSnapshot)
		path string
		code string
	}{
		{name: "unknown Shadowlord key", edit: func(entry *PlayerKnowledgeSnapshot) { entry.Key = "unknown.name" }, path: "redux.playerKnowledge[0].key", code: "unsupported"},
		{name: "shrine key in Shadowlord category", edit: func(entry *PlayerKnowledgeSnapshot) { entry.Key = ShrineKnowledgeHonestyMantra }, path: "redux.playerKnowledge[0].key", code: "unsupported"},
		{name: "unsupported source", edit: func(entry *PlayerKnowledgeSnapshot) { entry.Source.Kind = "book" }, path: "redux.playerKnowledge[0].source.kind", code: "unsupported"},
		{name: "experimentation source", edit: func(entry *PlayerKnowledgeSnapshot) {
			entry.Source = KnowledgeSourceSnapshot{Kind: KnowledgeSourceExperimentation, DisplayName: experimentationKnowledgeDisplayName}
		}, path: "redux.playerKnowledge[0].source.kind", code: "inconsistent"},
		{name: "migration source", edit: func(entry *PlayerKnowledgeSnapshot) {
			entry.Source = KnowledgeSourceSnapshot{Kind: KnowledgeSourceMigration, DisplayName: migratedKnowledgeDisplayName}
		}, path: "redux.playerKnowledge[0].source.kind", code: "inconsistent"},
		{name: "blank dialogue source", edit: func(entry *PlayerKnowledgeSnapshot) { entry.Source.DisplayName = "  " }, path: "redux.playerKnowledge[0].source.displayName", code: "required"},
		{name: "missing date", edit: func(entry *PlayerKnowledgeSnapshot) { entry.DiscoveredOn = nil }, path: "redux.playerKnowledge[0].discoveredOn", code: "required"},
		{name: "invalid date", edit: func(entry *PlayerKnowledgeSnapshot) { entry.DiscoveredOn.Day = 29 }, path: "redux.playerKnowledge[0].discoveredOn", code: "out_of_range"},
		{name: "unread", edit: func(entry *PlayerKnowledgeSnapshot) { entry.Unread = true }, path: "redux.playerKnowledge[0].unread", code: "inconsistent"},
		{name: "spell formula", edit: func(entry *PlayerKnowledgeSnapshot) {
			entry.SpellFormula = &SpellFormulaKnowledgeSnapshot{State: SpellFormulaConfirmed, ConfirmedMask: 1, CandidateMasks: []uint16{}}
		}, path: "redux.playerKnowledge[0].spellFormula", code: "inconsistent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := validSnapshot()
			candidate.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{
				validLoreKnowledgeEntry(KnowledgeCategoryShadowlord, ShadowlordKnowledgeAstarothName),
			}
			test.edit(&candidate.Redux.PlayerKnowledge[0])
			if issues := Validate(candidate); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidatePlayerKnowledgeStrictCurrentValues(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Redux.PlayerKnowledge = validPlayerKnowledge()
	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate(valid player knowledge) issues = %#v", issues)
	}
	optionalDate := validSnapshot()
	optionalDate.Redux.PlayerKnowledge = validPlayerKnowledge()
	optionalDate.Redux.PlayerKnowledge[0].DiscoveredOn = nil
	if issues := Validate(optionalDate); len(issues) != 0 {
		t.Fatalf("Validate(optional knowledge date) issues = %#v", issues)
	}

	tests := []struct {
		name string
		edit func(*SaveSnapshot)
		path string
		code string
	}{
		{name: "null entries", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge = nil }, path: "redux.playerKnowledge", code: "required"},
		{name: "unsupported category", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].Category = "rumor" }, path: "redux.playerKnowledge[0].category", code: "unsupported"},
		{name: "unsupported hidden spell", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].Key = "in-lor" }, path: "redux.playerKnowledge[0].key", code: "unsupported"},
		{name: "unsupported source", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].Source.Kind = "book" }, path: "redux.playerKnowledge[0].source.kind", code: "unsupported"},
		{name: "blank dialogue source", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].Source.DisplayName = "  " }, path: "redux.playerKnowledge[0].source.displayName", code: "required"},
		{name: "invalid experimentation display", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[1].Source.DisplayName = "Mixing" }, path: "redux.playerKnowledge[1].source.displayName", code: "inconsistent"},
		{name: "invalid migration display", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[2].Source.DisplayName = "Old save" }, path: "redux.playerKnowledge[2].source.displayName", code: "inconsistent"},
		{name: "invalid date", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].DiscoveredOn.Day = 29 }, path: "redux.playerKnowledge[0].discoveredOn", code: "out_of_range"},
		{name: "duplicate key", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[1] = s.Redux.PlayerKnowledge[0] }, path: "redux.playerKnowledge[1]", code: "duplicate"},
		{name: "unsorted", edit: func(s *SaveSnapshot) {
			s.Redux.PlayerKnowledge[0], s.Redux.PlayerKnowledge[1] = s.Redux.PlayerKnowledge[1], s.Redux.PlayerKnowledge[0]
		}, path: "redux.playerKnowledge[1]", code: "not_sorted"},
		{name: "unsupported formula state", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].SpellFormula.State = "guessed" }, path: "redux.playerKnowledge[0].spellFormula.state", code: "unsupported"},
		{name: "null candidates", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].SpellFormula.CandidateMasks = nil }, path: "redux.playerKnowledge[0].spellFormula.candidateMasks", code: "required"},
		{name: "uncertain formula shape", edit: func(s *SaveSnapshot) {
			s.Redux.PlayerKnowledge[0].SpellFormula.ConfirmedMask = 1
			s.Redux.PlayerKnowledge[0].SpellFormula.CandidateMasks = []uint16{1}
		}, path: "redux.playerKnowledge[0].spellFormula", code: "inconsistent"},
		{name: "zero candidate", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].SpellFormula.CandidateMasks[0] = 0 }, path: "redux.playerKnowledge[0].spellFormula.candidateMasks[0]", code: "out_of_range"},
		{name: "candidate exceeds reagents", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[0].SpellFormula.CandidateMasks[0] = 0x100 }, path: "redux.playerKnowledge[0].spellFormula.candidateMasks[0]", code: "out_of_range"},
		{name: "duplicate candidate", edit: func(s *SaveSnapshot) {
			s.Redux.PlayerKnowledge[0].SpellFormula.CandidateMasks[1] = s.Redux.PlayerKnowledge[0].SpellFormula.CandidateMasks[0]
		}, path: "redux.playerKnowledge[0].spellFormula.candidateMasks[1]", code: "duplicate"},
		{name: "noncanonical confirmed formula", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[1].SpellFormula.ConfirmedMask = 1 }, path: "redux.playerKnowledge[1].spellFormula.confirmedMask", code: "inconsistent"},
		{name: "confirmed formula candidates", edit: func(s *SaveSnapshot) { s.Redux.PlayerKnowledge[1].SpellFormula.CandidateMasks = []uint16{1} }, path: "redux.playerKnowledge[1].spellFormula.candidateMasks", code: "inconsistent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := validSnapshot()
			candidate.Redux.PlayerKnowledge = validPlayerKnowledge()
			test.edit(&candidate)
			if issues := Validate(candidate); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateRejectsNonAuthenticDayTwentyNine(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.World.Calendar.Day = 29
	issues := Validate(snapshot)
	if !hasIssue(issues, "world.calendar", "out_of_range") {
		t.Fatalf("Validate() issues = %#v, want authentic 28-day rejection", issues)
	}
}

func TestValidateAllowsNoSpriteMapMutation(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.MapMutations = []MapMutationSnapshot{{Position: validPosition(), Kind: MapMutationSetTile, Value: -1}}

	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateRejectsInvalidIndependentEffectStates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		effect EffectsSnapshot
		path   string
		code   string
	}{
		{name: "negative light", effect: EffectsSnapshot{MagicLightMinutesRemaining: -1}, path: "effects.magicLightMinutesRemaining", code: "negative"},
		{name: "light in active slot", effect: EffectsSnapshot{DurationType: 'L', DurationTurns: 20}, path: "effects.durationType", code: "unsupported"},
		{name: "unknown active spell", effect: EffectsSnapshot{DurationType: 'X', DurationTurns: 20}, path: "effects.durationType", code: "unsupported"},
		{name: "active spell without duration", effect: EffectsSnapshot{DurationType: 'P'}, path: "effects.durationType", code: "inconsistent"},
		{name: "duration without active spell", effect: EffectsSnapshot{DurationTurns: 20}, path: "effects.durationType", code: "inconsistent"},
		{name: "negative drunk", effect: EffectsSnapshot{Drunk: -1}, path: "effects.drunk", code: "out_of_range"},
		{name: "negative smashed", effect: EffectsSnapshot{Smashed: -1}, path: "effects.smashed", code: "out_of_range"},
		{name: "excessive smashed", effect: EffectsSnapshot{Smashed: maxSnapshotSmashed + 1}, path: "effects.smashed", code: "out_of_range"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Effects = test.effect
			if issues := Validate(snapshot); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateAcceptsUncappedNonnegativeDrunk(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Effects.Drunk = 10_000
	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestSnapshotRoundTripPreservesIndependentEffectCombinations(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		effects EffectsSnapshot
	}{
		{name: "light only", effects: EffectsSnapshot{MagicLightMinutesRemaining: 83}},
		{name: "active spell only", effects: EffectsSnapshot{DurationType: 'P', DurationTurns: 37}},
		{name: "simultaneous", effects: EffectsSnapshot{DurationType: 'T', DurationTurns: 20, MagicLightMinutesRemaining: 83}},
		{name: "intoxication", effects: EffectsSnapshot{Drunk: 10_000, Smashed: maxSnapshotSmashed}},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Effects = test.effects
			data, err := EncodeCanonical(validEnvelope(snapshot))
			if err != nil {
				t.Fatalf("EncodeCanonical() error = %v", err)
			}
			decoded, err := Decode(data)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if !reflect.DeepEqual(decoded.Snapshot.Effects, test.effects) {
				t.Fatalf("effects = %#v, want %#v", decoded.Snapshot.Effects, test.effects)
			}
		})
	}
}

func TestValidateAcceptsSemanticPartyMapStatesAndArmorClass(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	states := []int{0, 0xff, 2, 0x7f}
	for index, partyMap := range states {
		snapshot.Party.Roster[index] = snapshot.Party.Roster[0]
		snapshot.Party.Roster[index].Name = fmt.Sprintf("Player%d", index)
		snapshot.Party.Roster[index].PartyMap = partyMap
		snapshot.Party.Roster[index].ArmorClass = 17 + index
	}
	snapshot.Party.Roster[0].PartyMap = 0

	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateAcceptsCurrentCharacterIdentityValues(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		values []int
		set    func(*CharacterSnapshot, int)
	}{
		{name: "gender", values: []int{0x0b, 0x0c}, set: func(character *CharacterSnapshot, value int) { character.Gender = value }},
		{name: "class", values: []int{'A', 'F', 'B', 'M'}, set: func(character *CharacterSnapshot, value int) { character.Class = value }},
		{name: "status", values: []int{'G', 'P', 'C', 'S', 'D'}, set: func(character *CharacterSnapshot, value int) { character.Status = value }},
		{name: "party map", values: []int{0, 1, 40, 0x7f, 0xff}, set: func(character *CharacterSnapshot, value int) { character.PartyMap = value }},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, value := range test.values {
				snapshot := validSnapshot()
				character := snapshot.Party.Roster[0]
				character.Name = "Companion"
				character.Class = 'F'
				character.PartyMap = 0xff
				test.set(&character, value)
				snapshot.Party.Roster[1] = character
				if issues := Validate(snapshot); len(issues) != 0 {
					t.Fatalf("Validate(%s=%#x) issues = %#v", test.name, value, issues)
				}
			}
		})
	}
}

func TestValidateRejectsUnsupportedCurrentCharacterIdentityValues(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		set  func(*CharacterSnapshot)
		path string
	}{
		{name: "gender", set: func(character *CharacterSnapshot) { character.Gender = 0x0d }, path: "party.roster[0].gender"},
		{name: "class", set: func(character *CharacterSnapshot) { character.Class = 'X' }, path: "party.roster[0].class"},
		{name: "class overflow", set: func(character *CharacterSnapshot) { character.Class = 'A' + 256 }, path: "party.roster[0].class"},
		{name: "status", set: func(character *CharacterSnapshot) { character.Status = 'X' }, path: "party.roster[0].status"},
		{name: "status overflow", set: func(character *CharacterSnapshot) { character.Status = 'G' + 256 }, path: "party.roster[0].status"},
		{name: "party map", set: func(character *CharacterSnapshot) { character.PartyMap = 41 }, path: "party.roster[0].partyMap"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			test.set(&snapshot.Party.Roster[0])
			if issues := Validate(snapshot); !hasIssue(issues, test.path, "unsupported") {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidateRequiresAvatarInActiveRosterSlotZero(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		set  func(*SaveSnapshot)
		path string
		code string
	}{
		{name: "vacant", set: func(snapshot *SaveSnapshot) { snapshot.Party.Roster[0] = CharacterSnapshot{} }, path: "party.roster[0].occupied", code: "required"},
		{name: "other class", set: func(snapshot *SaveSnapshot) { snapshot.Party.Roster[0].Class = 'F' }, path: "party.roster[0].class", code: "inconsistent"},
		{name: "inactive", set: func(snapshot *SaveSnapshot) { snapshot.Party.Roster[0].PartyMap = 1 }, path: "party.roster[0].partyMap", code: "inconsistent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			test.set(&snapshot)
			if issues := Validate(snapshot); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}

	snapshot := validSnapshot()
	snapshot.Party.Roster[0].Name = "Custom"
	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate(player-named Avatar) issues = %#v", issues)
	}
}

func TestValidateRejectsMoreThanSixActiveCharacters(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	for index := 1; index < 7; index++ {
		snapshot.Party.Roster[index] = snapshot.Party.Roster[0]
		snapshot.Party.Roster[index].Name = fmt.Sprintf("Player%d", index)
		snapshot.Party.Roster[index].Class = 'F'
	}

	if issues := Validate(snapshot); !hasIssue(issues, "party.roster[6].partyMap", "noncontiguous_active_party") {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateRejectsUnpackedActiveRoster(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Party.Roster[1] = snapshot.Party.Roster[0]
	snapshot.Party.Roster[1].Name = "Inactive"
	snapshot.Party.Roster[1].PartyMap = 0xff
	snapshot.Party.Roster[2] = snapshot.Party.Roster[0]
	snapshot.Party.Roster[2].Name = "Active"

	issues := Validate(snapshot)
	if !hasIssue(issues, "party.roster[2].partyMap", "noncontiguous_active_party") {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateRejectsWorkflowOwnedAutomaticSailing(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.World.IsSailing = true
	snapshot.PartyVehicle = &VehicleSnapshot{Type: 3, Hull: 99, SailState: 1}

	issues := Validate(snapshot)
	if !hasIssue(issues, "world.isSailing", "unsupported") {
		t.Fatalf("Validate() issues = %#v", issues)
	}
}

func TestValidateVehicleSkiffConstraints(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		vehicle VehicleSnapshot
		path    string
		code    string
		valid   bool
	}{
		{name: "frigate zero", vehicle: VehicleSnapshot{Type: 4, Hull: 99}, valid: true},
		{name: "frigate maximum", vehicle: VehicleSnapshot{Type: 4, Skiffs: 12, Hull: 99}, valid: true},
		{name: "negative", vehicle: VehicleSnapshot{Type: 4, Skiffs: -1, Hull: 99}, path: "partyVehicle", code: "out_of_range"},
		{name: "above maximum", vehicle: VehicleSnapshot{Type: 4, Skiffs: 13, Hull: 99}, path: "partyVehicle", code: "out_of_range"},
		{name: "non-frigate", vehicle: VehicleSnapshot{Type: 3, Skiffs: 1, Hull: 99}, path: "partyVehicle.skiffs", code: "inconsistent"},
		{name: "destroyed frigate", vehicle: VehicleSnapshot{Type: 4, Hull: 0}, path: "partyVehicle.hull", code: "inconsistent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.PartyVehicle = &test.vehicle
			issues := Validate(snapshot)
			if test.valid && len(issues) != 0 {
				t.Fatalf("Validate() issues = %#v", issues)
			}
			if !test.valid && !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func TestValidatePendingDeliveryConstraints(t *testing.T) {
	t.Parallel()
	validPending := func() SaveSnapshot {
		snapshot := validSnapshot()
		snapshot.World.Position.Floor = int(references.UNDERWORLD)
		snapshot.World.LastLargeMapPosition = snapshot.World.Position
		snapshot.MapUnits = []MapUnitSnapshot{{
			World: int(references.OVERWORLD), Kind: "vehicle", Position: PositionSnapshot{
				Location: int(references.BritanniaUnderworld), X: 10, Y: 20, Floor: int(references.OVERWORLD),
			}, Visible: true, AIOverrideProvenance: string(references.AIOverrideProvenanceNone), Vehicle: &VehicleSnapshot{
				Type: int(references.FrigateVehicle), CurrentDirection: int(references.Right),
				PreviousDirection: int(references.Right), Skiffs: 2, Hull: 99, PendingDelivery: true,
			},
		}}

		return snapshot
	}
	if issues := Validate(validPending()); len(issues) != 0 {
		t.Fatalf("Validate(valid pending delivery) issues = %#v", issues)
	}
	standaloneSkiff := validPending()
	standaloneSkiff.MapUnits[0].Vehicle.Type = int(references.SkiffVehicle)
	standaloneSkiff.MapUnits[0].Vehicle.Skiffs = 0
	if issues := Validate(standaloneSkiff); len(issues) != 0 {
		t.Fatalf("Validate(valid pending standalone skiff) issues = %#v", issues)
	}

	for _, test := range []struct {
		name string
		edit func(*SaveSnapshot)
		path string
		code string
	}{
		{name: "party vehicle", edit: func(s *SaveSnapshot) { s.PartyVehicle = &VehicleSnapshot{Type: 4, Hull: 99, PendingDelivery: true} }, path: "partyVehicle.pendingDelivery", code: "inconsistent"},
		{name: "non-vehicle unit", edit: func(s *SaveSnapshot) { s.MapUnits[0].Kind = "npc" }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "carpet", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.Type = int(references.CarpetVehicle) }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "wrong location", edit: func(s *SaveSnapshot) { s.MapUnits[0].Position.Location = int(references.Britain) }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "wrong controller", edit: func(s *SaveSnapshot) { s.MapUnits[0].World = int(references.UNDERWORLD) }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "wrong floor", edit: func(s *SaveSnapshot) { s.MapUnits[0].Position.Floor = int(references.UNDERWORLD) }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "hidden", edit: func(s *SaveSnapshot) { s.MapUnits[0].Visible = false }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "owned", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.OwnedByNPC = true }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "damaged hull", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.Hull = 98 }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "raised sail", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.SailState = 1 }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "current direction", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.CurrentDirection = int(references.Left) }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "previous direction", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.PreviousDirection = int(references.Left) }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "HMS Cape", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.HasHMSCape = true }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "frigate has too few skiffs", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.Skiffs = 1 }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "frigate has too many skiffs", edit: func(s *SaveSnapshot) { s.MapUnits[0].Vehicle.Skiffs = 13 }, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "standalone skiff carries skiffs", edit: func(s *SaveSnapshot) {
			s.MapUnits[0].Vehicle.Type = int(references.SkiffVehicle)
			s.MapUnits[0].Vehicle.Skiffs = 1
		}, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
		{name: "duplicate", edit: func(s *SaveSnapshot) {
			duplicate := s.MapUnits[0]
			duplicate.ControllerOrder, duplicate.NPCNum = 1, 1
			vehicle := *duplicate.Vehicle
			duplicate.Vehicle = &vehicle
			s.MapUnits = append(s.MapUnits, duplicate)
		}, path: "mapUnits[1].vehicle.pendingDelivery", code: "duplicate"},
		{name: "stable active overworld", edit: func(s *SaveSnapshot) {
			s.World.Position = PositionSnapshot{Location: int(references.BritanniaUnderworld), X: 1, Y: 2, Floor: int(references.OVERWORLD)}
		}, path: "mapUnits[0].vehicle.pendingDelivery", code: "inconsistent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validPending()
			test.edit(&snapshot)
			if issues := Validate(snapshot); !hasIssue(issues, test.path, test.code) {
				t.Fatalf("Validate() issues = %#v", issues)
			}
		})
	}
}

func hasIssue(issues []Issue, path, code string) bool {
	for _, issue := range issues {
		if issue.Path == path && issue.Code == code {
			return true
		}
	}
	return false
}

func validPlayerKnowledge() []PlayerKnowledgeSnapshot {
	return []PlayerKnowledgeSnapshot{
		{
			Category:     KnowledgeCategorySpell,
			Key:          SpellKnowledgeAnYlem,
			Source:       KnowledgeSourceSnapshot{Kind: KnowledgeSourceDialogue, DisplayName: "Goeth"},
			DiscoveredOn: &BritannianDateSnapshot{Year: 100, Month: 4, Day: 12},
			Unread:       true,
			SpellFormula: &SpellFormulaKnowledgeSnapshot{
				State: SpellFormulaUncertain, CandidateMasks: []uint16{0x81, 0x14},
			},
		},
		{
			Category:     KnowledgeCategorySpell,
			Key:          SpellKnowledgeInXenMani,
			Source:       KnowledgeSourceSnapshot{Kind: KnowledgeSourceExperimentation, DisplayName: experimentationKnowledgeDisplayName},
			DiscoveredOn: &BritannianDateSnapshot{Year: 100, Month: 4, Day: 13},
			SpellFormula: &SpellFormulaKnowledgeSnapshot{
				State: SpellFormulaConfirmed, ConfirmedMask: references.GetSpellReagents(references.InXenMani),
				CandidateMasks: make([]uint16, 0),
			},
		},
		{
			Category: KnowledgeCategorySpell,
			Key:      SpellKnowledgeRelXenBet,
			Source:   KnowledgeSourceSnapshot{Kind: KnowledgeSourceMigration, DisplayName: migratedKnowledgeDisplayName},
		},
	}
}

func validLoreKnowledge() []PlayerKnowledgeSnapshot {
	return []PlayerKnowledgeSnapshot{
		validLoreKnowledgeEntry(KnowledgeCategoryShadowlord, ShadowlordKnowledgeAstarothAssociation),
		validLoreKnowledgeEntry(KnowledgeCategoryShadowlord, ShadowlordKnowledgeAstarothName),
		validLoreKnowledgeEntry(KnowledgeCategoryShadowlord, ShadowlordKnowledgeFaulineiAssociation),
		validLoreKnowledgeEntry(KnowledgeCategoryShadowlord, ShadowlordKnowledgeFaulineiName),
		validLoreKnowledgeEntry(KnowledgeCategoryShadowlord, ShadowlordKnowledgeNosfentorAssociation),
		validLoreKnowledgeEntry(KnowledgeCategoryShadowlord, ShadowlordKnowledgeNosfentorName),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeCompassionMantra),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeHonestyMantra),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeHonorMantra),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeHumilityMantra),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeJusticeMantra),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeSacrificeMantra),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeSpiritualityMantra),
		validLoreKnowledgeEntry(KnowledgeCategoryShrine, ShrineKnowledgeValourMantra),
	}
}

func validLoreKnowledgeEntry(category, key string) PlayerKnowledgeSnapshot {
	return PlayerKnowledgeSnapshot{
		Category:     category,
		Key:          key,
		Source:       KnowledgeSourceSnapshot{Kind: KnowledgeSourceDialogue, DisplayName: "Savant"},
		DiscoveredOn: &BritannianDateSnapshot{Year: 100, Month: 4, Day: 12},
	}
}
