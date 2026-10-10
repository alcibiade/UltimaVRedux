package save_snapshot

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func removeV24Fields(body map[string]any) {
	for _, member := range []string{"basementChestHistory", "atlasNotes", "atlasFleet", "codexRevelationComplete"} {
		delete(body, member)
	}
	if vehicle, ok := body["partyVehicle"].(map[string]any); ok {
		delete(vehicle, "vesselId")
	}
	if units, ok := body["mapUnits"].([]any); ok {
		for _, value := range units {
			unit := value.(map[string]any)
			delete(unit, "surfaceChestAuthored")
			delete(unit, "surfaceChestTransient")
			if vehicle, ok := unit["vehicle"].(map[string]any); ok {
				delete(vehicle, "vesselId")
			}
		}
	}
}

func v24PersonalSnapshot() SaveSnapshot {
	snapshot := validSnapshot()
	snapshot.BasementChestHistory[0].LastOpened = &BritannianDateSnapshot{Year: 139, Month: 13, Day: 28}
	snapshot.AtlasNotes = journal.AtlasNotesState{LastID: 9, Records: []journal.AtlasNoteRecord{{
		ID: 3, Position: journal.SiteLocation{X: 255, Y: 0}, Label: "Shrine landing",
		Description: "Walk north.  Keep the spacing.\n\nBring more supplies.",
		RecordedAt:  journal.GameTime{Year: 139, Month: 13, Day: 28, Hour: 23, Minute: 59},
	}}}
	snapshot.AtlasFleet.LastID = 8
	snapshot.PartyVehicle = &VehicleSnapshot{Type: int(references.FrigateVehicle), VesselID: 5, Skiffs: 2, Hull: 70}
	snapshot.MapUnits = append(snapshot.MapUnits, MapUnitSnapshot{Kind: MapUnitKindVehicle, NPCNum: 3,
		ControllerOrder: 1, Position: validPosition(), AIOverrideProvenance: "none",
		Vehicle: &VehicleSnapshot{Type: int(references.SkiffVehicle), VesselID: 8, Hull: 99}})
	return snapshot
}

func addV24CodexWord(snapshot *SaveSnapshot) {
	date := &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5}
	snapshot.Redux.PlayerKnowledge = append(snapshot.Redux.PlayerKnowledge, PlayerKnowledgeSnapshot{
		Category: KnowledgeCategoryDungeon, Key: DungeonKnowledgeDoomWordOfPower,
		Source: KnowledgeSourceSnapshot{Kind: KnowledgeSourceRevelation, DisplayName: "Codex of Ultimate Wisdom"}, DiscoveredOn: date,
	})
	snapshot.Redux.QuestJournal = append(snapshot.Redux.QuestJournal, QuestJournalEventSnapshot{
		Sequence: 1, Kind: QuestJournalEventKnowledgeLearned, Subject: "dungeon.doom.word_of_power",
		OccurredOn: date, Source: QuestJournalSourceRevelation, SourceName: "Codex of Ultimate Wisdom",
	})
}

func TestV24PersonalCanonicalSchemaParity(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for _, complete := range []bool{false, true} {
		snapshot := v24PersonalSnapshot()
		addV24CodexWord(&snapshot)
		snapshot.CodexRevelationComplete = complete
		requireV23RoundTrip(t, schema, snapshot)
	}
	snapshot := v24PersonalSnapshot()
	snapshot.AtlasNotes.Records[0].Label = strings.Repeat("a", journal.MaxAtlasNoteLabelRunes)
	snapshot.AtlasNotes.Records[0].Description = strings.Repeat("\n", journal.MaxAtlasNoteDescriptionRunes)
	requireV23RoundTrip(t, schema, snapshot)
}

func TestV24PersonalValidationSchemaParity(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for name, mutate := range map[string]func(*SaveSnapshot){
		"invalid opening date": func(s *SaveSnapshot) { s.BasementChestHistory[0].LastOpened.Day = 29 },
		"obsolete death flag": func(s *SaveSnapshot) {
			s.Party.DeadNPCs = []NPCFlagSnapshot{{Location: int(references.LordBritishsCastle - references.Moonglow), NPC: 23}}
		},
		"long label":          func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Label = strings.Repeat("x", 33) },
		"untrimmed label":     func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Label = " landing" },
		"blank label":         func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Label = " " },
		"unicode label":       func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Label = "Caf\u00e9" },
		"label newline":       func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Label = "landing\n" },
		"description control": func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Description = "a\rb" },
		"long description":    func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Description = strings.Repeat("x", 501) },
		"note date":           func(s *SaveSnapshot) { s.AtlasNotes.Records[0].RecordedAt.Month = 0 },
		"note floor":          func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Position.Floor = -1 },
		"note map":            func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Position.Map = references.Britain },
		"note coordinate":     func(s *SaveSnapshot) { s.AtlasNotes.Records[0].Position.X = 256 },
		"note zero id":        func(s *SaveSnapshot) { s.AtlasNotes.Records[0].ID = 0 },
		"note watermark":      func(s *SaveSnapshot) { s.AtlasNotes.LastID = 2 },
		"note unsafe id":      func(s *SaveSnapshot) { s.AtlasNotes.LastID = journal.MaxAtlasIdentity + 1 },
		"duplicate note id": func(s *SaveSnapshot) {
			n := s.AtlasNotes.Records[0]
			n.Position.X--
			s.AtlasNotes.Records = append(s.AtlasNotes.Records, n)
		},
		"duplicate note position": func(s *SaveSnapshot) {
			n := s.AtlasNotes.Records[0]
			n.ID++
			s.AtlasNotes.Records = append(s.AtlasNotes.Records, n)
		},
		"too many notes":                  func(s *SaveSnapshot) { s.AtlasNotes.Records = make([]journal.AtlasNoteRecord, 257) },
		"duplicate parked boarded vessel": func(s *SaveSnapshot) { s.MapUnits[1].Vehicle.VesselID = s.PartyVehicle.VesselID },
		"fleet watermark":                 func(s *SaveSnapshot) { s.AtlasFleet.LastID = 7 },
		"fleet unsafe id":                 func(s *SaveSnapshot) { s.AtlasFleet.LastID = journal.MaxAtlasIdentity + 1 },
		"nonboat registration":            func(s *SaveSnapshot) { s.MapUnits[1].Vehicle.Type = int(references.CarpetVehicle) },
		"chest identity on enemy":         func(s *SaveSnapshot) { s.MapUnits[0].SurfaceChestAuthored = true },
		"codex without word":              func(s *SaveSnapshot) { s.CodexRevelationComplete = true },
		"codex without event": func(s *SaveSnapshot) {
			addV24CodexWord(s)
			s.CodexRevelationComplete = true
			s.Redux.QuestJournal = []QuestJournalEventSnapshot{}
		},
		"codex without matching word": func(s *SaveSnapshot) {
			addV24CodexWord(s)
			s.CodexRevelationComplete = true
			s.Redux.PlayerKnowledge[0].Key = DungeonKnowledgeDeceitWordOfPower
		},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := v24PersonalSnapshot()
			mutate(&snapshot)
			require.NotEmpty(t, Validate(snapshot))
			_, err := EncodeCanonical(validEnvelope(snapshot))
			require.Error(t, err)
			data, err := json.Marshal(validEnvelope(snapshot))
			require.NoError(t, err)
			_, err = Decode(data)
			require.Error(t, err)
			require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
		})
	}
}

func TestV24RequiredPersonalMembers(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for _, path := range [][]string{
		{"basementChestHistory"}, {"atlasNotes"}, {"atlasFleet"}, {"codexRevelationComplete"},
		{"atlasNotes", "lastId"}, {"atlasNotes", "records"}, {"atlasFleet", "lastId"}, {"partyVehicle", "vesselId"},
		{"basementChestHistory", "lastOpened"}, {"basementChestHistory", "lastOpened", "year"},
		{"basementChestHistory", "lastOpened", "month"}, {"basementChestHistory", "lastOpened", "day"},
		{"atlasNotes", "records", "id"}, {"atlasNotes", "records", "position"},
		{"atlasNotes", "records", "label"}, {"atlasNotes", "records", "description"}, {"atlasNotes", "records", "recordedAt"},
		{"mapUnits", "surfaceChestAuthored"}, {"mapUnits", "surfaceChestTransient"},
	} {
		for _, operation := range []string{"missing", "null", "alias", "unknown"} {
			if operation == "null" && (path[len(path)-1] == "records" || path[len(path)-1] == "lastOpened") {
				continue
			}
			t.Run(fmt.Sprint(path, operation), func(t *testing.T) {
				doc := currentEnvelopeDocument(t, v24PersonalSnapshot())
				parent := doc["snapshot"].(map[string]any)
				for _, key := range path[:len(path)-1] {
					if array, ok := parent[key].([]any); ok {
						parent = array[0].(map[string]any)
					} else {
						parent = parent[key].(map[string]any)
					}
				}
				key := path[len(path)-1]
				switch operation {
				case "missing":
					delete(parent, key)
				case "null":
					parent[key] = nil
				case "alias":
					parent[strings.ToUpper(key)] = parent[key]
					delete(parent, key)
				case "unknown":
					parent["invented"] = false
				}
				data, err := json.Marshal(doc)
				require.NoError(t, err)
				_, err = Decode(data)
				require.Error(t, err)
				require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			})
		}
	}
}

func TestV24EveryHistoricalVersionPromotesConservatively(t *testing.T) {
	for version := 12; version <= 23; version++ {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			source := validSnapshot()
			source.PartyVehicle = &VehicleSnapshot{Type: int(references.FrigateVehicle), Hull: 99}
			source.MapUnits = append(source.MapUnits, MapUnitSnapshot{Kind: MapUnitKindVehicle, NPCNum: 3,
				ControllerOrder: 1, Position: validPosition(), AIOverrideProvenance: "none",
				Vehicle: &VehicleSnapshot{Type: int(references.SkiffVehicle), Hull: 99}})
			data := historicalV20TestData(t, version, source)
			var document map[string]any
			require.NoError(t, json.Unmarshal(data, &document))
			body := document["snapshot"].(map[string]any)
			flags := []NPCFlagSnapshot{{Location: int(references.LordBritishsCastle - references.Moonglow), NPC: 23},
				{Location: int(references.LordBritishsCastle - references.Moonglow), NPC: 25},
				{Location: int(references.LordBritishsCastle - references.Moonglow), NPC: 31}, {Location: 17, NPC: 24}}
			body["party"].(map[string]any)["deadNpcs"] = flags
			data, err := json.Marshal(document)
			require.NoError(t, err)
			if version <= 15 {
				schema, _ := compileNativeSaveSchema(t, version)
				require.NoError(t, schema.Validate(archivedEnvelopeOneSchemaInstance(t, data)))
			} else {
				schema, _ := compileNativeSaveSchemaFile(t, fmt.Sprintf("native-save-e2-s%d.schema.json", version))
				require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			}
			decoded, err := Decode(data)
			require.NoError(t, err)
			got := decoded.Snapshot
			require.Equal(t, 24, got.Version)
			require.Equal(t, flags[2:], got.Party.DeadNPCs)
			date := &BritannianDateSnapshot{Year: source.World.Calendar.Year, Month: source.World.Calendar.Month, Day: source.World.Calendar.Day}
			require.Equal(t, date, got.BasementChestHistory[0].LastOpened)
			require.Nil(t, got.BasementChestHistory[1].LastOpened)
			require.Equal(t, date, got.BasementChestHistory[2].LastOpened)
			require.Zero(t, got.AtlasFleet.LastID)
			require.Zero(t, got.AtlasNotes.LastID)
			require.Empty(t, got.AtlasNotes.Records)
			require.False(t, got.CodexRevelationComplete)
			require.Zero(t, got.PartyVehicle.VesselID)
			require.Equal(t, source.Random, got.Random)
			for _, inject := range []func(map[string]any){
				func(b map[string]any) { b["codexRevelationComplete"] = false },
				func(b map[string]any) { b["basementChestHistory"] = []any{} },
				func(b map[string]any) { b["atlasNotes"] = map[string]any{"lastId": 0, "records": nil} },
				func(b map[string]any) { b["atlasFleet"] = map[string]any{"lastId": 0} },
				func(b map[string]any) { b["partyVehicle"].(map[string]any)["vesselId"] = 0 },
				func(b map[string]any) {
					b["mapUnits"].([]any)[1].(map[string]any)["vehicle"].(map[string]any)["vesselId"] = 0
				},
				func(b map[string]any) { b["mapUnits"].([]any)[0].(map[string]any)["surfaceChestTransient"] = false },
			} {
				mutated := cloneJSONDocument(t, document)
				inject(mutated["snapshot"].(map[string]any))
				invalid, marshalErr := json.Marshal(mutated)
				require.NoError(t, marshalErr)
				_, decodeErr := Decode(invalid)
				require.ErrorIs(t, decodeErr, ErrInvalidEnvelope)
			}
		})
	}
}

func TestV24HistoricalWordKnowledgeDoesNotCompleteCodex(t *testing.T) {
	for version := 16; version <= 23; version++ {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			source := validSnapshot()
			addV24CodexWord(&source)
			for i := range source.Progression.Shrines {
				source.Progression.Shrines[i].Completed = true
			}
			source.CodexRevelationComplete = true
			decoded, err := Decode(historicalV20TestData(t, version, source))
			require.NoError(t, err)
			require.False(t, decoded.Snapshot.CodexRevelationComplete, "neither word knowledge nor completed quests proves final acknowledgement")
			require.Equal(t, source.Redux.PlayerKnowledge, decoded.Snapshot.Redux.PlayerKnowledge)
			require.Equal(t, source.Redux.QuestJournal, decoded.Snapshot.Redux.QuestJournal)
		})
	}
}

func TestV24HistoricalBasementSurvivorKeepsAssessmentAndLoot(t *testing.T) {
	for version := 20; version <= 23; version++ {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			source := validSnapshot()
			position := PositionSnapshot{Location: int(references.LordBritishsCastle), Floor: -1, X: 17, Y: 22}
			source.World.Position = position
			number := 0x8c
			source.MapUnits = []MapUnitSnapshot{{Kind: MapUnitKindChest, NPCNum: 24, Position: position,
				AIOverrideProvenance: "none", SurfaceChestNumber: &number, SurfaceChestAssessment: int(references.ChestTrapped), SurfaceChestAuthored: true}}
			source.ItemStacks = []ItemStackSnapshot{{Position: position, Items: []ItemSnapshot{{Type: 9, ID: 5, Quantity: 3}}}}
			source.MapMutations = nil
			decoded, err := Decode(historicalV20TestData(t, version, source))
			require.NoError(t, err)
			require.True(t, decoded.Snapshot.MapUnits[0].SurfaceChestAuthored)
			require.False(t, decoded.Snapshot.MapUnits[0].SurfaceChestTransient)
			require.Equal(t, source.MapUnits, decoded.Snapshot.MapUnits)
			require.Equal(t, source.ItemStacks, decoded.Snapshot.ItemStacks)
			require.Equal(t, source.Random, decoded.Snapshot.Random)
			require.Equal(t, [3]BasementChestDailyRecordSnapshot{}, decoded.Snapshot.BasementChestHistory)
		})
	}
}

func TestV24ChestLifetimeSchemaAndIdentityValidation(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	snapshot := validSnapshot()
	position := PositionSnapshot{Location: int(references.LordBritishsCastle), Floor: -1, X: 17, Y: 22}
	snapshot.World.Position = position
	trapped, safe := 0x8c, 7
	snapshot.MapUnits = []MapUnitSnapshot{
		{Kind: MapUnitKindChest, NPCNum: 24, Position: position, AIOverrideProvenance: "none",
			SurfaceChestNumber: &trapped, SurfaceChestAssessment: int(references.ChestTrapped), SurfaceChestAuthored: true},
		{Kind: MapUnitKindChest, NPCNum: 26, ControllerOrder: 1, Position: position, AIOverrideProvenance: "none",
			SurfaceChestNumber: &safe, SurfaceChestAssessment: int(references.ChestSafe), SurfaceChestTransient: true},
	}
	snapshot.ItemStacks, snapshot.MapMutations = nil, nil
	requireV23RoundTrip(t, schema, snapshot)
	for _, mutate := range []func(*SaveSnapshot){
		func(s *SaveSnapshot) { s.MapUnits[1].SurfaceChestAuthored = true },
		func(s *SaveSnapshot) { s.MapUnits[0].Position.X++ },
		func(s *SaveSnapshot) { s.MapUnits[1].Position.Floor = 0 },
	} {
		invalid := snapshot
		invalid.MapUnits = append([]MapUnitSnapshot(nil), snapshot.MapUnits...)
		mutate(&invalid)
		requireV23Rejected(t, schema, invalid, "mapUnits")
	}
	for _, length := range []int{0, 2, 4} {
		document := currentEnvelopeDocument(t, snapshot)
		body := document["snapshot"].(map[string]any)
		body["basementChestHistory"] = make([]BasementChestDailyRecordSnapshot, length)
		data, err := json.Marshal(document)
		require.NoError(t, err)
		_, err = Decode(data)
		require.ErrorIs(t, err, ErrInvalidEnvelope)
		require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	}
}

func TestV24RegisteredVesselRequiresVehicleUnitSchemaParity(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for _, kind := range []string{MapUnitKindEnemy, MapUnitKindNPC, MapUnitKindChest} {
		t.Run(kind, func(t *testing.T) {
			snapshot := validSnapshot()
			unit := &snapshot.MapUnits[0]
			unit.Kind = kind
			if kind != MapUnitKindEnemy {
				unit.EnemyVariant, unit.EnemyCurrentHP = "", 0
			}
			if kind == MapUnitKindChest {
				number := 7
				unit.SurfaceChestNumber = &number
			}
			unit.Vehicle = &VehicleSnapshot{Type: int(references.SkiffVehicle), Hull: 99}
			requireV23RoundTrip(t, schema, snapshot) // Preserve accepted zero-ID payload compatibility.
			unit.Vehicle.VesselID, snapshot.AtlasFleet.LastID = 1, 1
			requireV23Rejected(t, schema, snapshot, "mapUnits[0].vehicle.vesselId")
		})
	}
}

func TestV24CompleteCodexPreservesValidFirstDiscoverySources(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for _, source := range []string{KnowledgeSourceDialogue, KnowledgeSourceObservation, KnowledgeSourceRevelation} {
		t.Run(source, func(t *testing.T) {
			snapshot := validSnapshot()
			addV24CodexWord(&snapshot)
			snapshot.Redux.PlayerKnowledge[0].Source = KnowledgeSourceSnapshot{Kind: source, DisplayName: "First source"}
			snapshot.Redux.QuestJournal[0].Source, snapshot.Redux.QuestJournal[0].SourceName = source, "First source"
			snapshot.CodexRevelationComplete = true
			requireV23RoundTrip(t, schema, snapshot)
		})
	}
}

func TestV24V12ReconstructionKeepsMigratedBasementHistory(t *testing.T) {
	source := validSnapshot()
	source.World.Position = PositionSnapshot{Location: int(references.LordBritishsCastle), X: 15, Y: 21, Floor: 0}
	data := versionTwelveEnvelopeData(t, source, func(body map[string]any) {
		body["mapUnits"], body["itemStacks"], body["mapMutations"] = nil, nil, nil
		body["redux"].(map[string]any)["reconstructMapUnits"] = true
		body["party"].(map[string]any)["deadNpcs"] = []NPCFlagSnapshot{{Location: int(references.LordBritishsCastle - references.Moonglow), NPC: 23}}
	})
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.True(t, decoded.Snapshot.NeedsMapUnitReconstruction())
	require.Empty(t, decoded.Snapshot.MapUnits)
	require.Empty(t, decoded.Snapshot.Party.DeadNPCs)
	date := source.World.Calendar
	require.Equal(t, &BritannianDateSnapshot{Year: date.Year, Month: date.Month, Day: date.Day}, decoded.Snapshot.BasementChestHistory[0].LastOpened)
}
