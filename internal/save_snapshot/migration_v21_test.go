package save_snapshot

import (
	"encoding/json"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestV21RequiresRetirement(t *testing.T) {
	currentSchema, _ := compileCurrentNativeSaveSchema(t)
	for _, value := range []any{false, true, nil, 1, "false"} {
		document := currentEnvelopeDocument(t, validSnapshot())
		document["snapshot"].(map[string]any)["moonstoneJournal"].([]any)[0].(map[string]any)["originalSiteRetired"] = value
		data, err := json.Marshal(document)
		require.NoError(t, err)
		_, err = Decode(data)
		if value == false {
			require.NoError(t, err)
			require.NoError(t, currentSchema.Validate(decodeJSONSchemaInstance(t, data)))
		} else {
			require.Error(t, err)
			require.Error(t, currentSchema.Validate(decodeJSONSchemaInstance(t, data)))
		}
	}
	document := currentEnvelopeDocument(t, validSnapshot())
	for _, value := range document["snapshot"].(map[string]any)["moonstoneJournal"].([]any) {
		delete(value.(map[string]any), "originalSiteRetired")
	}
	data, err := json.Marshal(document)
	require.NoError(t, err)
	_, err = Decode(data)
	require.ErrorContains(t, err, "originalSiteRetired")
	require.Error(t, currentSchema.Validate(decodeJSONSchemaInstance(t, data)))
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
