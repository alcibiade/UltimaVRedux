package save_snapshot

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestMoongateLookFirstEvidenceUsesExistingWireRepresentation(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	for _, scenario := range []string{"strict survey", "flexible daytime survey", "unmeasured"} {
		t.Run(scenario, func(t *testing.T) {
			snapshot := v20EvidenceSnapshot()
			observation := &snapshot.MoonstoneJournal[3]
			observation.FirstSource = journal.MoonstoneSourceLook
			observation.LastContext.Source = journal.MoonstoneSourceLook
			observation.LastContext.Observer.X--
			observation.SurveyContext = observation.LastContext
			if scenario == "flexible daytime survey" {
				observation.DiscoveredAt.Time.Hour = 12
				observation.ObservedAt.Time.Hour = 12
				observation.Survey.RecordedAt.Time.Hour = 12
				observation.Survey.Flexible = true
			}
			if scenario == "unmeasured" {
				observation.LastContext.HasSextant = false
				observation.Survey, observation.SurveyContext = journal.SurveyedLocation{}, journal.MoonstoneContext{}
			}
			data, err := EncodeCanonical(validEnvelope(snapshot))
			require.NoError(t, err)
			require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			decoded, err := Decode(data)
			require.NoError(t, err)
			require.Equal(t, snapshot, decoded.Snapshot)
		})
	}
}

func TestHistoricalMoongateLookStillRequiresPriorRecognition(t *testing.T) {
	for _, version := range []int{20, 21, 22, 23} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			schema, _ := compileNativeSaveSchemaFile(t, fmt.Sprintf("native-save-e2-s%d.schema.json", version))
			snapshot := v20EvidenceSnapshot()
			observation := &snapshot.MoonstoneJournal[3]
			observation.LastContext.Source = journal.MoonstoneSourceLook
			observation.LastContext.Observer.X--
			observation.SurveyContext = observation.LastContext
			data := historicalV20TestData(t, version, snapshot)
			require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			_, err := Decode(data)
			require.NoError(t, err)

			observation.FirstSource = journal.MoonstoneSourceLook
			data = historicalV20TestData(t, version, snapshot)
			require.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
			_, err = Decode(data)
			require.Error(t, err)
		})
	}
}
