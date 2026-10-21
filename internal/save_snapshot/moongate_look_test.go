package save_snapshot

import (
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
