package save_snapshot

import (
	"encoding/json"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func removeV21Fields(body map[string]any) {
	for _, value := range body["moonstoneJournal"].([]any) {
		delete(value.(map[string]any), "originalSiteRetired")
	}
}

func TestV21RequiresRetirementAndV20RejectsIt(t *testing.T) {
	currentSchema, _ := compileCurrentNativeSaveSchema(t)
	v21Schema, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s21.schema.json")
	oldSchema, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s20.schema.json")
	for _, version := range []int{20, 21, CurrentVersion} {
		for _, value := range []any{false, true, nil, 1, "false"} {
			document := currentEnvelopeDocument(t, validSnapshot())
			body := document["snapshot"].(map[string]any)
			body["version"] = version
			if version <= 21 {
				removeV22Fields(body)
			}
			if version == 20 {
				removeV21Fields(body)
			}
			body["moonstoneJournal"].([]any)[0].(map[string]any)["originalSiteRetired"] = value
			data, err := json.Marshal(document)
			require.NoError(t, err)
			_, err = Decode(data)
			schema := currentSchema
			if version == 20 {
				schema = oldSchema
			} else if version == 21 {
				schema = v21Schema
			}
			if version >= 21 && value == false {
				require.NoError(t, err)
				require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			} else {
				require.Error(t, err)
				require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			}
		}
	}
	document := currentEnvelopeDocument(t, validSnapshot())
	removeV21Fields(document["snapshot"].(map[string]any))
	data, err := json.Marshal(document)
	require.NoError(t, err)
	_, err = Decode(data)
	require.ErrorContains(t, err, "originalSiteRetired")
	require.Error(t, currentSchema.Validate(decodeJSONSchemaInstance(t, data)))
}

func TestV20RetirementMigrationUsesOnlyProvenEvidence(t *testing.T) {
	for name, test := range map[string]struct {
		mutate            func(*SaveSnapshot)
		retired, measured bool
	}{
		"original observed": {func(*SaveSnapshot) {}, false, true},
		"carried":           {func(s *SaveSnapshot) { s.Inventory.Moonstones[3] = 1 }, true, false},
		"pickup": {func(s *SaveSnapshot) {
			o := &s.MoonstoneJournal[3]
			o.LastContext.Source, o.Placed = journal.MoonstoneSourcePickup, false
		}, true, false},
		"first pickup":  {func(s *SaveSnapshot) { s.MoonstoneJournal[3].FirstSource = journal.MoonstoneSourcePickup }, true, true},
		"burial":        {func(s *SaveSnapshot) { s.MoonstoneJournal[3].LastContext.Source = journal.MoonstoneSourceBurial }, true, true},
		"first burial":  {func(s *SaveSnapshot) { s.MoonstoneJournal[3].FirstSource = journal.MoonstoneSourceBurial }, true, true},
		"survey burial": {func(s *SaveSnapshot) { s.MoonstoneJournal[3].SurveyContext.Source = journal.MoonstoneSourceBurial }, true, true},
		"pickup survey at placed site": {func(s *SaveSnapshot) {
			s.MoonstoneJournal[3].SurveyContext.Source = journal.MoonstoneSourcePickup
		}, true, false},
		"reburied same site without new measurement": {func(s *SaveSnapshot) {
			o := &s.MoonstoneJournal[3]
			o.LastContext.Source = journal.MoonstoneSourceBurial
			o.ObservedAt.Time.Day++
		}, true, false},
		"moved unmeasured": {func(s *SaveSnapshot) {
			o := &s.MoonstoneJournal[3]
			o.Site.X++
			o.LastContext.Observer = o.Site
		}, true, false},
		"moved measured": {func(s *SaveSnapshot) {
			o := &s.MoonstoneJournal[3]
			o.Site.X++
			o.LastContext.Observer, o.Survey.Location, o.SurveyContext.Observer = o.Site, o.Site, o.Site
		}, true, true},
		"returned with historical survey": {func(s *SaveSnapshot) {
			o := &s.MoonstoneJournal[3]
			o.Survey.Location.X++
			o.SurveyContext.Observer = o.Survey.Location
		}, true, false},
		"unknown hidden movement": {func(s *SaveSnapshot) {
			s.MoonstoneJournal[3] = journal.MoonstoneObservation{}
			s.Progression.Moonstones[3] = MoonstoneSnapshot{X: 2, Y: 3, State: 1, IsFound: true}
		}, false, false},
		"unknown carried": {func(s *SaveSnapshot) {
			s.MoonstoneJournal[3] = journal.MoonstoneObservation{}
			s.Inventory.Moonstones[3] = 1
		}, true, false},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := v20EvidenceSnapshot()
			o := &snapshot.MoonstoneJournal[3]
			original := references.ChartedMoonstoneOrigins()[3].Location
			site := journal.SiteLocation{Map: original.Location, Floor: original.Floor, X: int(original.Position.X), Y: int(original.Position.Y)}
			o.Site, o.LastContext.Observer, o.Survey.Location, o.SurveyContext.Observer = site, site, site, site
			o.OriginalSiteRetired = false
			test.mutate(&snapshot)
			data := historicalV20TestData(t, 20, snapshot)
			schema, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s20.schema.json")
			require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)), "frozen v20 must accept its real historical values")
			decoded, err := Decode(data)
			require.NoError(t, err)
			got := decoded.Snapshot.MoonstoneJournal[3]
			require.Equal(t, test.retired, got.OriginalSiteRetired)
			require.Equal(t, test.measured, got.Survey.Measured)
			if !test.measured {
				require.Zero(t, got.Survey)
				require.Zero(t, got.SurveyContext)
			}
			if name == "unknown carried" {
				require.Equal(t, journal.MoonstoneObservation{Known: true, FirstSource: journal.MoonstoneSourcePossession, OriginalSiteRetired: true}, got)
			}
			current, err := EncodeCanonical(decoded)
			require.NoError(t, err)
			currentSchema, _ := compileCurrentNativeSaveSchema(t)
			require.NoError(t, currentSchema.Validate(decodeJSONSchemaInstance(t, current)))
			reloaded, err := Decode(current)
			require.NoError(t, err)
			decoded.legacyExcerpts = false
			require.Equal(t, decoded, reloaded)
		})
	}
}

func TestV21RejectsHistoricalOrCarriedSurvey(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for _, mutate := range []func(*SaveSnapshot){
		func(s *SaveSnapshot) { s.MoonstoneJournal[3].Site.X++ },
		func(s *SaveSnapshot) { s.Inventory.Moonstones[3] = 1 },
		func(s *SaveSnapshot) { s.MoonstoneJournal[3].SurveyContext.Source = journal.MoonstoneSourcePickup },
		func(s *SaveSnapshot) {
			s.MoonstoneJournal[3].LastContext.Source = journal.MoonstoneSourceBurial
			s.MoonstoneJournal[3].ObservedAt.Time.Day++
		},
		func(s *SaveSnapshot) {
			s.MoonstoneJournal[3].Placed = false
			s.MoonstoneJournal[3].LastContext.Source = journal.MoonstoneSourcePickup
		},
	} {
		snapshot := v20EvidenceSnapshot()
		mutate(&snapshot)
		data, err := json.Marshal(validEnvelope(snapshot))
		require.NoError(t, err)
		_, err = Decode(data)
		require.Error(t, err)
		require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	}
}
