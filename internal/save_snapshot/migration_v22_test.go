package save_snapshot

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

// Only the three persisted survey families acquire the v22 member. Nullable
// atlas arrays and optional historical collections retain their original shape.
func surveyJSONMembers(body map[string]any) []map[string]any {
	var surveys []map[string]any
	if observations, ok := body["dungeonJournal"].([]any); ok {
		for _, value := range observations {
			if sites, ok := value.(map[string]any)["sites"].([]any); ok {
				for _, site := range sites {
					surveys = append(surveys, site.(map[string]any)["survey"].(map[string]any))
				}
			}
		}
	}
	for _, family := range []string{"moonstoneJournal", "atlasLandmarks"} {
		if observations, ok := body[family].([]any); ok {
			for _, value := range observations {
				surveys = append(surveys, value.(map[string]any)["survey"].(map[string]any))
			}
		}
	}

	return surveys
}

func removeV22Fields(body map[string]any) {
	removeV24Fields(body)
	delete(body, "openDoor")
	for _, survey := range surveyJSONMembers(body) {
		delete(survey, "flexible")
	}
}

func v22EvidenceSnapshot(hour int, flexible bool) SaveSnapshot {
	snapshot := v20EvidenceSnapshot()
	snapshot.AtlasLandmarks = []journal.AtlasLandmarkObservation{atlasEvidence()}
	at := journal.OptionalGameTime{Known: true, Time: journal.GameTime{Year: 139, Month: 4, Day: 5, Hour: hour}}
	dungeon := &snapshot.DungeonJournal[0].Sites[0]
	dungeon.ObservedAt, dungeon.Survey.RecordedAt, dungeon.Survey.Flexible = at, at, flexible
	moonstone := &snapshot.MoonstoneJournal[3]
	moonstone.DiscoveredAt, moonstone.ObservedAt, moonstone.Survey.RecordedAt, moonstone.Survey.Flexible = at, at, at, flexible
	atlas := &snapshot.AtlasLandmarks[0]
	atlas.ObservedAt, atlas.NamedAt, atlas.Survey.RecordedAt, atlas.Survey.Flexible = at, at, at, flexible

	return snapshot
}

func TestV22SurveyPolicyAllHoursRoundTripAndSchema(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for hour := range 24 {
		for _, flexible := range []bool{false, true} {
			t.Run(fmt.Sprintf("hour-%d-flexible-%t", hour, flexible), func(t *testing.T) {
				snapshot := v22EvidenceSnapshot(hour, flexible)
				data, err := json.Marshal(validEnvelope(snapshot))
				require.NoError(t, err)
				decoded, decodeErr := Decode(data)
				schemaErr := schema.Validate(decodeJSONSchemaInstance(t, data))
				canonical, encodeErr := EncodeCanonical(validEnvelope(snapshot))
				if !flexible && hour >= 6 && hour < 19 {
					require.ErrorContains(t, decodeErr, "nighttime")
					require.Error(t, schemaErr)
					require.Error(t, encodeErr)
					return
				}
				require.NoError(t, decodeErr)
				require.NoError(t, schemaErr)
				require.NoError(t, encodeErr)
				require.Equal(t, snapshot, decoded.Snapshot, "policy, dates, sites and source tags are value-owned")
				reloaded, err := Decode(canonical)
				require.NoError(t, err)
				decoded.legacyExcerpts = false
				require.Equal(t, decoded, reloaded)
			})
		}
	}
}

func TestV22RequiresConcreteFlexibleInEverySurvey(t *testing.T) {
	snapshot := v22EvidenceSnapshot(12, true)
	at := snapshot.AtlasLandmarks[0].ObservedAt
	snapshot.AtlasLandmarks = nil
	for key := 1; key <= 107; key++ {
		if key == 25 || key > 32 && key < 101 {
			continue
		}
		site := journal.SiteLocation{X: key, Y: key}
		kind := "towne"
		if key >= 101 {
			position := references.AtlasShrinePositions()[key-101]
			site.X, site.Y, kind = int(position.X), int(position.Y), "shrine"
		}
		snapshot.AtlasLandmarks = append(snapshot.AtlasLandmarks, journal.AtlasLandmarkObservation{
			SiteKey: uint8(key), Site: site, Kind: kind, Source: journal.AtlasSourceLook,
			ObservedAt: at, Survey: journal.SurveyedLocation{Measured: true, Flexible: true, Location: site, RecordedAt: at},
			SurveySource: journal.AtlasSourceLook,
		})
	}
	schema, _ := compileCurrentNativeSaveSchema(t)
	document := currentEnvelopeDocument(t, snapshot)
	body := document["snapshot"].(map[string]any)
	surveys := surveyJSONMembers(body)
	require.Len(t, surveys, 2*8+8+38)
	for index, survey := range surveys {
		for _, operation := range []string{"missing", "null", "integer", "string", "case alias", "unknown"} {
			t.Run(fmt.Sprintf("survey-%d-%s", index, operation), func(t *testing.T) {
				original := survey["flexible"]
				defer func() {
					survey["flexible"] = original
					delete(survey, "Flexible")
					delete(survey, "setting")
				}()
				switch operation {
				case "missing":
					delete(survey, "flexible")
				case "null":
					survey["flexible"] = nil
				case "integer":
					survey["flexible"] = 1
				case "string":
					survey["flexible"] = "true"
				case "case alias":
					delete(survey, "flexible")
					survey["Flexible"] = original
				case "unknown":
					survey["setting"] = true
				}
				data, err := json.Marshal(document)
				require.NoError(t, err)
				_, err = Decode(data)
				require.ErrorIs(t, err, ErrInvalidEnvelope)
				require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			})
		}
	}
}

func TestV22MixedMeasurementPoliciesRemainPerSurvey(t *testing.T) {
	snapshot := v22EvidenceSnapshot(12, true)
	atlas := &snapshot.AtlasLandmarks[0]
	atlas.ObservedAt.Time.Hour, atlas.NamedAt.Time.Hour, atlas.Survey.RecordedAt.Time.Hour = 21, 21, 21
	atlas.Survey.Flexible = false
	moonstone := &snapshot.MoonstoneJournal[3]
	moonstone.DiscoveredAt.Time.Hour, moonstone.ObservedAt.Time.Hour, moonstone.Survey.RecordedAt.Time.Hour = 19, 19, 19
	envelope := validEnvelope(snapshot)
	data, err := EncodeCanonical(envelope)
	require.NoError(t, err)
	schema, _ := compileCurrentNativeSaveSchema(t)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, envelope, decoded, "a single checkpoint may contain noon flexible, nighttime flexible and nighttime original surveys")
}

func TestV22FlexibleStillRequiresValidSurfaceEvidence(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for name, mutate := range map[string]func(*SaveSnapshot){
		"invalid hour":                func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[0].Survey.RecordedAt.Time.Hour = 24 },
		"invalid month":               func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[0].Survey.RecordedAt.Time.Month = 0 },
		"absent date":                 func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[0].Survey.RecordedAt = journal.OptionalGameTime{} },
		"underworld":                  func(s *SaveSnapshot) { s.DungeonJournal[0].Sites[0].Survey.Location.Floor = -1 },
		"interior":                    func(s *SaveSnapshot) { s.AtlasLandmarks[0].Survey.Location.Map = references.Moonglow },
		"private moonstone interior":  func(s *SaveSnapshot) { s.MoonstoneJournal[3].Survey.Location.Map = references.Moonglow },
		"unmeasured flexible payload": func(s *SaveSnapshot) { s.DungeonJournal[1].Sites[0].Survey.Flexible = true },
		"missing instrument evidence": func(s *SaveSnapshot) { s.MoonstoneJournal[3].SurveyContext.HasSextant = false },
		"observer mismatch":           func(s *SaveSnapshot) { s.MoonstoneJournal[3].SurveyContext.Observer.Map = references.Britain },
		"later atlas survey":          func(s *SaveSnapshot) { s.AtlasLandmarks[0].Survey.RecordedAt.Time.Day++ },
		"future moonstone survey":     func(s *SaveSnapshot) { s.MoonstoneJournal[3].Survey.RecordedAt.Time.Day++ },
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := v22EvidenceSnapshot(12, true)
			mutate(&snapshot)
			data, err := json.Marshal(validEnvelope(snapshot))
			require.NoError(t, err)
			_, err = Decode(data)
			require.ErrorIs(t, err, ErrInvalidSnapshot)
			require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
		})
	}
}

func TestV22HistoricalV20V21SurveyShapesRemainFrozen(t *testing.T) {
	for _, version := range []int{20, 21} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			snapshot := v22EvidenceSnapshot(21, false)
			data := historicalV20TestData(t, version, snapshot)
			schema, _ := compileNativeSaveSchemaFile(t, fmt.Sprintf("native-save-e2-s%d.schema.json", version))
			require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			decoded, err := Decode(data)
			require.NoError(t, err)
			require.Equal(t, snapshot, decoded.Snapshot)
			var document map[string]any
			require.NoError(t, json.Unmarshal(data, &document))
			body := document["snapshot"].(map[string]any)
			for index, survey := range surveyJSONMembers(body) {
				for _, value := range []any{false, true, nil} {
					survey["flexible"] = value
					injected, err := json.Marshal(document)
					require.NoError(t, err)
					_, err = Decode(injected)
					require.ErrorContains(t, err, "flexible", "v%d survey %d must reject even inactive provenance", version, index)
					require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, injected)))
					delete(survey, "flexible")
				}
			}
			for _, family := range []string{"dungeon", "moonstone", "atlas"} {
				daylight := v22EvidenceSnapshot(21, false)
				switch family {
				case "dungeon":
					daylight.DungeonJournal[0].Sites[0].Survey.RecordedAt.Time.Hour = 12
				case "moonstone":
					daylight.MoonstoneJournal[3].Survey.RecordedAt.Time.Hour = 12
				case "atlas":
					daylight.AtlasLandmarks[0].Survey.RecordedAt.Time.Hour = 12
				}
				invalid := historicalV20TestData(t, version, daylight)
				_, err = Decode(invalid)
				require.ErrorContains(t, err, "nighttime", "historical daylight cannot be grandfathered")
				require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, invalid)))
			}
		})
	}
}

func TestV22PromotionPreservesV21RetirementAndNullableAtlas(t *testing.T) {
	for _, empty := range [][]journal.AtlasLandmarkObservation{nil, {}} {
		snapshot := v22EvidenceSnapshot(21, false)
		snapshot.AtlasLandmarks = empty
		snapshot.Inventory.Moonstones[2] = 1
		require.Zero(t, snapshot.MoonstoneJournal[2], "v21 accepted unknown carried records; policy promotion must not backfill them")
		original := references.ChartedMoonstoneOrigins()[3].Location
		moonstone := &snapshot.MoonstoneJournal[3]
		moonstone.Site = journal.SiteLocation{Map: original.Location, Floor: original.Floor, X: int(original.Position.X), Y: int(original.Position.Y)}
		moonstone.LastContext.Observer, moonstone.Survey.Location, moonstone.SurveyContext.Observer = moonstone.Site, moonstone.Site, moonstone.Site
		require.True(t, moonstone.OriginalSiteRetired, "sticky retirement is persisted evidence, not recomputed in v21 promotion")
		decoded, err := Decode(historicalV20TestData(t, 21, snapshot))
		require.NoError(t, err)
		require.Equal(t, snapshot, decoded.Snapshot)
	}
	for version := 12; version <= 21; version++ {
		decoded, err := Decode(historicalV20TestData(t, version, validSnapshot()))
		require.NoError(t, err)
		data, err := json.Marshal(decoded.Snapshot)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(data, &body))
		for _, survey := range surveyJSONMembers(body) {
			require.Equal(t, false, survey["flexible"], "v%d cannot invent flexible acquisition policy", version)
		}
	}
}
