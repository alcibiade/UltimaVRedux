package save_snapshot

import (
	"encoding/json"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func atlasEvidence() journal.AtlasLandmarkObservation {
	site := journal.SiteLocation{X: 233, Y: 66}
	at := journal.OptionalGameTime{Known: true, Time: journal.GameTime{Year: 139, Month: 4, Day: 5, Hour: 21}}
	return journal.AtlasLandmarkObservation{SiteKey: 101, Site: site, Kind: "shrine", Name: "the shrine of Honesty",
		Source: journal.AtlasSourceLook, NameSource: journal.AtlasSourceShrineIntroduction,
		ObservedAt: at, NamedAt: at, Survey: journal.SurveyedLocation{Measured: true, Location: site, RecordedAt: at},
		SurveySource: journal.AtlasSourceLook}
}

func TestAtlasV20CodecSchemaAndHistoricalBoundaries(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	snapshot := validSnapshot()
	snapshot.AtlasLandmarks = []journal.AtlasLandmarkObservation{atlasEvidence()}
	data, err := EncodeCanonical(validEnvelope(snapshot))
	require.NoError(t, err)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, snapshot, decoded.Snapshot)
	for version := 12; version <= 19; version++ {
		data = historicalV20TestData(t, version, snapshot)
		decoded, err = Decode(data)
		require.NoError(t, err)
		require.Empty(t, decoded.Snapshot.AtlasLandmarks)
		var document map[string]any
		require.NoError(t, json.Unmarshal(data, &document))
		document["snapshot"].(map[string]any)["atlasLandmarks"] = snapshot.AtlasLandmarks
		data, err = json.Marshal(document)
		require.NoError(t, err)
		_, err = Decode(data)
		require.Error(t, err, "historical v%d must reject atlas evidence", version)
	}
	for _, member := range []string{"atlasLandmarks", "site", "surveySource", "namedAt"} {
		document := currentEnvelopeDocument(t, snapshot)
		body := document["snapshot"].(map[string]any)
		if member == "atlasLandmarks" {
			delete(body, member)
		} else {
			delete(body["atlasLandmarks"].([]any)[0].(map[string]any), member)
		}
		data, err = json.Marshal(document)
		require.NoError(t, err)
		_, err = Decode(data)
		require.Error(t, err)
		require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	}
}

func TestAtlasV20RejectsInvalidEvidence(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for name, mutate := range map[string]func(*journal.AtlasLandmarkObservation){
		"Ararat":               func(o *journal.AtlasLandmarkObservation) { o.SiteKey = uint8(references.Ararat) },
		"dungeon":              func(o *journal.AtlasLandmarkObservation) { o.SiteKey = uint8(references.Deceit) },
		"sentinel":             func(o *journal.AtlasLandmarkObservation) { o.SiteKey = 108 },
		"wrong shrine":         func(o *journal.AtlasLandmarkObservation) { o.Site.X++ },
		"underworld":           func(o *journal.AtlasLandmarkObservation) { o.Site.Floor = -1 },
		"invented source":      func(o *journal.AtlasLandmarkObservation) { o.Source = "chart" },
		"look naming":          func(o *journal.AtlasLandmarkObservation) { o.NameSource = journal.AtlasSourceLook },
		"absent name evidence": func(o *journal.AtlasLandmarkObservation) { o.Name = "" },
		"survey wrong site":    func(o *journal.AtlasLandmarkObservation) { o.Survey.Location.Y++ },
		"survey no source":     func(o *journal.AtlasLandmarkObservation) { o.SurveySource = "" },
		"daylight":             func(o *journal.AtlasLandmarkObservation) { o.Survey.RecordedAt.Time.Hour = 12 },
		"invalid date":         func(o *journal.AtlasLandmarkObservation) { o.ObservedAt.Time.Month = 14 },
		"future name":          func(o *journal.AtlasLandmarkObservation) { o.NamedAt.Time.Year++ },
		"future survey":        func(o *journal.AtlasLandmarkObservation) { o.Survey.RecordedAt.Time.Year++ },
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.AtlasLandmarks = []journal.AtlasLandmarkObservation{atlasEvidence()}
			mutate(&snapshot.AtlasLandmarks[0])
			data, err := json.Marshal(validEnvelope(snapshot))
			require.NoError(t, err)
			_, err = Decode(data)
			require.Error(t, err)
			require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
		})
	}
	snapshot := validSnapshot()
	snapshot.AtlasLandmarks = []journal.AtlasLandmarkObservation{atlasEvidence(), atlasEvidence()}
	_, err := EncodeCanonical(validEnvelope(snapshot))
	require.Error(t, err, "duplicate site identities are invalid")
}
