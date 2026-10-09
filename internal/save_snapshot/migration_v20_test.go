package save_snapshot

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

// Historical fixtures must model the frozen wire shape, never current runtime
// structs with just a downgraded version marker.
func removeV20Fields(body map[string]any) {
	removeV24Fields(body)
	delete(body, "openDoor")
	delete(body, "atlasLandmarks")
	delete(body, "moonstoneJournal")
	if units, ok := body["mapUnits"].([]any); ok {
		for _, value := range units {
			delete(value.(map[string]any), "surfaceChestAssessment")
		}
	}
	if dungeon, ok := body["dungeon"].(map[string]any); ok {
		delete(dungeon, "chestAssessments")
	}
	if observations, ok := body["dungeonJournal"].([]any); ok {
		for _, value := range observations {
			observation := value.(map[string]any)
			delete(observation, "sites")
			observation["entranceSeen"], observation["open"] = false, false
			observation["entranceX"], observation["entranceY"], observation["entranceFloor"] = 0, 0, 0
		}
	}
}

func v20EvidenceSnapshot() SaveSnapshot {
	snapshot := validSnapshot()
	at := journal.OptionalGameTime{Known: true, Time: journal.GameTime{Year: 139, Month: 4, Day: 5, Hour: 21}}
	site := journal.SiteLocation{Map: references.BritanniaUnderworld, X: 240, Y: 73}
	snapshot.DungeonJournal[0] = journal.DungeonObservation{Known: true, WordUsed: true}
	snapshot.DungeonJournal[0].Sites[0] = journal.DungeonSiteObservation{Seen: true, Location: site, Open: true,
		ObservedAt: at, Survey: journal.SurveyedLocation{Measured: true, Location: site, RecordedAt: at}}
	snapshot.DungeonJournal[0].Sites[1] = journal.DungeonSiteObservation{Seen: true,
		Location: journal.SiteLocation{Map: references.BritanniaUnderworld, Floor: -1, X: 5, Y: 7}}
	context := journal.MoonstoneContext{Source: journal.MoonstoneSourceGate, Observer: site, HasSextant: true}
	snapshot.MoonstoneJournal[3] = journal.MoonstoneObservation{OriginalSiteRetired: true, Known: true, FirstSource: journal.MoonstoneSourceGate,
		DiscoveredAt: at, SiteKnown: true, Site: site, Placed: true, LastContext: context, ObservedAt: at,
		Survey: journal.SurveyedLocation{Measured: true, Location: site, RecordedAt: at}, SurveyContext: context}
	snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{validLoreKnowledgeEntry(KnowledgeCategoryJournal, JournalKnowledgeOppressionPassword)}
	snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{{Sequence: 1, Kind: QuestJournalEventKnowledgeLearned,
		Subject: "journal.oppression.password", Source: KnowledgeSourceDialogue, SourceName: "Flain",
		OccurredOn: &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5}}}
	snapshot.Presentation.Visits.Record(journal.ChronicleTab, 51)
	number := 0x81
	snapshot.MapUnits = []MapUnitSnapshot{{Kind: MapUnitKindChest, Position: validPosition(), SurfaceChestNumber: &number,
		AIOverrideProvenance: string(references.AIOverrideProvenanceNone), SurfaceChestAssessment: int(references.ChestTrapped)}}
	return snapshot
}

func TestV20EarnedEvidenceRoundTripAndSchema(t *testing.T) {
	schema, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s20.schema.json")
	snapshot := v20EvidenceSnapshot()
	data := historicalV20TestData(t, 20, snapshot)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, snapshot, decoded.Snapshot)
	// Frozen v20 accepted historical surveys; promotion clears them after a move.
	observation := &snapshot.MoonstoneJournal[3]
	observation.Site = journal.SiteLocation{Map: references.Britain, X: 3, Y: 4}
	observation.LastContext = journal.MoonstoneContext{Source: journal.MoonstoneSourceBurial, Observer: observation.Site}
	observation.ObservedAt.Time.Day++
	data = historicalV20TestData(t, 20, snapshot)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err = Decode(data)
	require.NoError(t, err)
	observation.Survey, observation.SurveyContext = journal.SurveyedLocation{}, journal.MoonstoneContext{}
	require.Equal(t, snapshot, decoded.Snapshot)
}

func TestV20RejectsInvalidEvidenceInCodecAndSchema(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for name, mutate := range map[string]func(*SaveSnapshot){
		"assessment overflow":       func(s *SaveSnapshot) { s.MapUnits[0].SurfaceChestAssessment = 256 },
		"assessment negative":       func(s *SaveSnapshot) { s.MapUnits[0].SurfaceChestAssessment = -1 },
		"armed marked safe":         func(s *SaveSnapshot) { s.MapUnits[0].SurfaceChestAssessment = 2 },
		"unarmed marked trapped":    func(s *SaveSnapshot) { *s.MapUnits[0].SurfaceChestNumber = 1 },
		"nonchest certainty":        func(s *SaveSnapshot) { s.MapUnits[0].SurfaceChestNumber = nil },
		"unknown dungeon":           func(s *SaveSnapshot) { s.DungeonJournal[0].Known = false },
		"unseen site payload":       func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[0].Seen = false },
		"wrong plane":               func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[1].Location.Floor = 0 },
		"noncanonical survey":       func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[0].Location.X++ },
		"underworld survey":         func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[1].Survey = s.DungeonJournal[0].Sites[0].Survey },
		"unknown moonstone payload": func(s *SaveSnapshot) { s.MoonstoneJournal[3].Known = false },
		"unknown first source":      func(s *SaveSnapshot) { s.MoonstoneJournal[3].FirstSource = "unknown" },
		"possession invented date":  func(s *SaveSnapshot) { s.MoonstoneJournal[3].FirstSource = journal.MoonstoneSourcePossession },
		"unknown site payload":      func(s *SaveSnapshot) { s.MoonstoneJournal[3].SiteKnown = false },
		"invalid map":               func(s *SaveSnapshot) { s.MoonstoneJournal[3].Site.Map = 255 },
		"invalid date":              func(s *SaveSnapshot) { s.MoonstoneJournal[3].ObservedAt.Time.Month = 14 },
		"backward observation":      func(s *SaveSnapshot) { s.MoonstoneJournal[3].ObservedAt.Time.Year-- },
		"observer mismatch":         func(s *SaveSnapshot) { s.MoonstoneJournal[3].LastContext.Observer.X++ },
		"pickup still placed":       func(s *SaveSnapshot) { s.MoonstoneJournal[3].LastContext.Source = journal.MoonstoneSourcePickup },
		"survey without instrument": func(s *SaveSnapshot) { s.MoonstoneJournal[3].SurveyContext.HasSextant = false },
		"survey without date":       func(s *SaveSnapshot) { s.MoonstoneJournal[3].Survey.RecordedAt = journal.OptionalGameTime{} },
		"daylight survey":           func(s *SaveSnapshot) { s.MoonstoneJournal[3].Survey.RecordedAt.Time.Hour = 12 },
		"absent survey payload":     func(s *SaveSnapshot) { s.MoonstoneJournal[3].Survey.Measured = false },
		"future survey":             func(s *SaveSnapshot) { s.MoonstoneJournal[3].Survey.RecordedAt.Time.Day++ },
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := v20EvidenceSnapshot()
			mutate(&snapshot)
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

func TestV20RequiresNewConcreteMembers(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for _, path := range [][]string{{"moonstoneJournal"}, {"dungeonJournal", "sites"}, {"mapUnits", "surfaceChestAssessment"}} {
		for _, operation := range []string{"missing", "null", "short", "extra"} {
			t.Run(fmt.Sprint(path, operation), func(t *testing.T) {
				document := currentEnvelopeDocument(t, v20EvidenceSnapshot())
				parent := document["snapshot"].(map[string]any)
				if len(path) == 2 {
					parent = parent[path[0]].([]any)[0].(map[string]any)
				}
				member := path[len(path)-1]
				switch operation {
				case "missing":
					delete(parent, member)
				case "null":
					parent[member] = nil
				case "short":
					parent[member] = []any{}
				case "extra":
					parent["unexpected"] = true
				}
				data, err := json.Marshal(document)
				require.NoError(t, err)
				_, err = Decode(data)
				require.Error(t, err)
				require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			})
		}
	}
}

func historicalV20TestData(t *testing.T, version int, snapshot SaveSnapshot) []byte {
	t.Helper()
	switch version {
	case 12:
		return versionTwelveEnvelopeData(t, snapshot, nil)
	case 13:
		return versionThirteenEnvelopeData(t, snapshot, nil)
	case 14:
		return versionFourteenEnvelopeData(t, snapshot, nil)
	case 15:
		return versionFifteenEnvelopeData(t, snapshot, nil)
	default:
		// Marshal the source evidence directly: v20 legitimately allowed fields
		// whose semantics are rejected by the current-placement v21 validator.
		encoded, err := json.Marshal(validEnvelope(snapshot))
		require.NoError(t, err)
		var document map[string]any
		require.NoError(t, json.Unmarshal(encoded, &document))
		body := document["snapshot"].(map[string]any)
		body["version"] = version
		removeV24Fields(body)
		if version <= 22 {
			delete(body, "openDoor")
		}
		if version <= 21 {
			removeV22Fields(body)
		}
		if version == 20 {
			removeV21Fields(body)
		} else if version < 20 {
			removeV20Fields(body)
		}
		if version == 16 {
			delete(body, "dungeonJournal")
			delete(body, "lastConversation")
		}
		data, err := json.Marshal(document)
		require.NoError(t, err)
		return data
	}
}

func TestV20HistoricalPromotionAndFrozenNestedShapes(t *testing.T) {
	for version := 12; version <= 19; version++ {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Inventory.Moonstones[2] = 1
			number := 0x81
			snapshot.MapUnits[0] = MapUnitSnapshot{Kind: MapUnitKindChest, Position: validPosition(), SurfaceChestNumber: &number,
				AIOverrideProvenance: string(references.AIOverrideProvenanceNone)}
			if version < 14 {
				snapshot.MapUnits[0].Kind = MapUnitKindNPC
			}
			data := historicalV20TestData(t, version, snapshot)
			decoded, err := Decode(data)
			require.NoError(t, err)
			require.Equal(t, journal.MoonstoneObservation{OriginalSiteRetired: true, Known: true, FirstSource: journal.MoonstoneSourcePossession}, decoded.Snapshot.MoonstoneJournal[2])
			require.Zero(t, decoded.Snapshot.MoonstoneJournal[0], "raw found/placement bytes never backfill knowledge")
			require.Zero(t, decoded.Snapshot.MapUnits[0].SurfaceChestAssessment, "armed bit never backfills inspection")
			for _, field := range []string{"moonstoneJournal", "surfaceChestAssessment", "oppression.password"} {
				var document map[string]any
				require.NoError(t, json.Unmarshal(data, &document))
				body := document["snapshot"].(map[string]any)
				switch field {
				case "moonstoneJournal":
					body[field] = currentEnvelopeDocument(t, snapshot)["snapshot"].(map[string]any)[field]
				case "surfaceChestAssessment":
					body["mapUnits"].([]any)[0].(map[string]any)[field] = 0
				case "oppression.password":
					body["redux"].(map[string]any)["playerKnowledge"] = []PlayerKnowledgeSnapshot{validLoreKnowledgeEntry(KnowledgeCategoryJournal, field)}
				}
				invalid, marshalErr := json.Marshal(document)
				require.NoError(t, marshalErr)
				_, err = Decode(invalid)
				require.Error(t, err, "v%d must reject %s", version, field)
			}
		})
	}
}

func TestV20HistoricalDungeonSitesPreservePrivateEvidence(t *testing.T) {
	for version := 17; version <= 19; version++ {
		for _, floor := range []int{0, -1} {
			document := currentEnvelopeDocument(t, validSnapshot())
			body := document["snapshot"].(map[string]any)
			body["version"] = version
			removeV20Fields(body)
			old := body["dungeonJournal"].([]any)[0].(map[string]any)
			old["known"], old["wordUsed"], old["entranceSeen"], old["open"] = true, true, true, true
			old["entranceX"], old["entranceY"], old["entranceFloor"] = 11, 29, floor
			data, err := json.Marshal(document)
			require.NoError(t, err)
			decoded, err := Decode(data)
			require.NoError(t, err)
			slot := journal.DungeonSurfaceSite
			if floor == -1 {
				slot = journal.DungeonUnderworldSite
			}
			require.Equal(t, journal.DungeonSiteObservation{Seen: true, Open: true,
				Location: journal.SiteLocation{Map: references.BritanniaUnderworld, Floor: references.FloorNumber(floor), X: 11, Y: 29}}, decoded.Snapshot.DungeonJournal[0].Sites[slot])
			require.Zero(t, decoded.Snapshot.DungeonJournal[0].Sites[1-slot])
			old["sites"] = []any{}
			data, err = json.Marshal(document)
			require.NoError(t, err)
			_, err = Decode(data)
			require.Error(t, err, "historical DTO must reject current sites")
		}
	}
}

func historicalJournalEnvelopeData(t *testing.T, snapshot SaveSnapshot) []byte {
	t.Helper()
	data, err := json.Marshal(validEnvelope(snapshot))
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(data, &document))
	removeV20Fields(document["snapshot"].(map[string]any))
	data, err = json.Marshal(document)
	require.NoError(t, err)
	return data
}

func TestV20DungeonAssessmentsAndHistoricalDefaults(t *testing.T) {
	snapshot := validSnapshot()
	snapshot.World.Position = PositionSnapshot{Location: 33, X: 2, Y: 3, Floor: 4}
	snapshot.Dungeon = &DungeonSnapshot{Location: 33, Cells: make([]int, DungeonCellCount)}
	for index, cell := range []int{0x40, 0x41, 0x40, 0x41} {
		snapshot.Dungeon.Cells[index] = cell
		snapshot.Dungeon.ChestAssessments[index] = index
	}
	schema, _ := compileCurrentNativeSaveSchema(t)
	data, err := EncodeCanonical(validEnvelope(snapshot))
	require.NoError(t, err)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, snapshot, decoded.Snapshot)
	for version := 12; version <= 19; version++ {
		old := historicalV20TestData(t, version, snapshot)
		decoded, err = Decode(old)
		require.NoError(t, err)
		require.Zero(t, decoded.Snapshot.Dungeon.ChestAssessments)
		require.Equal(t, snapshot.Dungeon.Cells, decoded.Snapshot.Dungeon.Cells)
		var document map[string]any
		require.NoError(t, json.Unmarshal(old, &document))
		document["snapshot"].(map[string]any)["dungeon"].(map[string]any)["chestAssessments"] = snapshot.Dungeon.ChestAssessments
		old, err = json.Marshal(document)
		require.NoError(t, err)
		_, err = Decode(old)
		require.Error(t, err, "v%d physical dungeon must reject new fields", version)
	}
	for _, invalid := range []any{nil, []int{}, make([]int, 511), make([]int, 513), "unknown"} {
		document := currentEnvelopeDocument(t, snapshot)
		dungeon := document["snapshot"].(map[string]any)["dungeon"].(map[string]any)
		if invalid == nil {
			delete(dungeon, "chestAssessments")
		} else {
			dungeon["chestAssessments"] = invalid
		}
		data, err = json.Marshal(document)
		require.NoError(t, err)
		_, err = Decode(data)
		require.Error(t, err)
		require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	}
	for _, test := range []struct{ cell, assessment int }{{0, 1}, {0x41, 2}, {0x40, 3}, {0x41, 256}, {0x40, -1}} {
		snapshot.Dungeon.Cells[511] = test.cell
		snapshot.Dungeon.ChestAssessments[511] = test.assessment
		data, err = json.Marshal(validEnvelope(snapshot))
		require.NoError(t, err)
		_, err = Decode(data)
		require.Error(t, err)
		require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	}
}

func TestV20OppressionHistoricalEventAndPresentationLimits(t *testing.T) {
	for version := 16; version <= 19; version++ {
		for _, field := range []string{"event", "visit"} {
			data := historicalV20TestData(t, version, validSnapshot())
			var document map[string]any
			require.NoError(t, json.Unmarshal(data, &document))
			body := document["snapshot"].(map[string]any)
			if field == "event" {
				body["redux"].(map[string]any)["questJournal"] = v20EvidenceSnapshot().Redux.QuestJournal
			} else {
				presentation := journal.EmptyPresentation()
				presentation.Visits.Record(journal.ChronicleTab, chronicleLimit(version)+1)
				body["presentation"] = presentation
			}
			data, err := json.Marshal(document)
			require.NoError(t, err)
			_, err = Decode(data)
			require.Error(t, err, "v%d %s cannot use current catalog", version, field)
			schema, _ := compileNativeSaveSchemaFile(t, fmt.Sprintf("native-save-e2-s%d.schema.json", version))
			require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
		}
	}
}
