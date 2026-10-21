package save_snapshot

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

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
	schema, _ := compileCurrentNativeSaveSchema(t)
	requireV23RoundTrip(t, schema, v20EvidenceSnapshot())
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

func TestV20DungeonAssessmentsRoundTripAndRejectInvalidCells(t *testing.T) {
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
