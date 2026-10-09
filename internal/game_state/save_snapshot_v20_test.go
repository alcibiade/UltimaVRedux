package game_state

import (
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestSaveSnapshotV20EarnedEvidenceHydration(t *testing.T) {
	for _, dungeon := range []bool{false, true} {
		t.Run(map[bool]string{false: "surface chest", true: "dungeon chests"}[dungeon], func(t *testing.T) {
			var state *GameState
			if dungeon {
				state = dungeonSaveTestState(t, true, true)
			} else {
				state, _ = NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
					WithPlayerAt(15, 15).WithSystemCallbacks().Build()
			}
			snapshot, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			anchor, floor := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Britain)
			state.atlasLandmarks = []journal.AtlasLandmarkObservation{{SiteKey: uint8(references.Britain),
				Site: journal.SiteLocation{Floor: floor, X: int(anchor.X), Y: int(anchor.Y)}, Kind: "towne", Source: journal.AtlasSourceLook}}
			snapshot, err = state.CaptureSaveSnapshot()
			require.NoError(t, err)
			state.atlasLandmarks[0].Kind = "hut"
			require.Equal(t, "towne", snapshot.AtlasLandmarks[0].Kind, "capture must detach atlas slice")
			at := journal.OptionalGameTime{Known: true, Time: journal.GameTime{Year: 139, Month: 4, Day: 5, Hour: 21}}
			site := journal.SiteLocation{Map: references.BritanniaUnderworld, X: 240, Y: 73}
			snapshot.DungeonJournal[0] = journal.DungeonObservation{Known: true, WordUsed: true}
			snapshot.DungeonJournal[0].Sites[0] = journal.DungeonSiteObservation{Seen: true, Location: site, Open: true,
				ObservedAt: at, Survey: journal.SurveyedLocation{Measured: true, Location: site, RecordedAt: at}}
			snapshot.DungeonJournal[0].Sites[1] = journal.DungeonSiteObservation{Seen: true,
				Location: journal.SiteLocation{Map: references.BritanniaUnderworld, Floor: -1, X: 11, Y: 29}}
			context := journal.MoonstoneContext{Source: journal.MoonstoneSourceGate, Observer: site, HasSextant: true}
			snapshot.MoonstoneJournal[3] = journal.MoonstoneObservation{Known: true, FirstSource: journal.MoonstoneSourceGate,
				DiscoveredAt: at, SiteKnown: true, Site: site, Placed: true, LastContext: context, ObservedAt: at,
				Survey: journal.SurveyedLocation{Measured: true, Location: site, RecordedAt: at}, SurveyContext: context}
			if dungeon {
				for index, cell := range []int{0x40, 0x41, 0x40, 0x41} {
					snapshot.Dungeon.Cells[index] = cell
					snapshot.Dungeon.ChestAssessments[index] = index
				}
			} else {
				number := 0x81
				arrived := 0
				snapshot.MapUnits = []save_snapshot.MapUnitSnapshot{{Kind: save_snapshot.MapUnitKindChest,
					Position:             save_snapshot.PositionSnapshot{Location: 0, X: 10, Y: 20, Floor: 0},
					AIOverrideProvenance: string(references.AIOverrideProvenanceNone),
					ArrivedScheduleSlot:  &arrived,
					SurfaceChestNumber:   &number, SurfaceChestAssessment: int(references.ChestTrapped)}}
			}
			envelope := save_snapshot.Envelope{Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
				Metadata: save_snapshot.Metadata{Name: "V20 evidence", CreatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}, Snapshot: snapshot}
			encoded, err := save_snapshot.EncodeCanonical(envelope)
			require.NoError(t, err)
			decoded, err := save_snapshot.Decode(encoded)
			require.NoError(t, err)
			restored, err := NewGameStateFromSaveSnapshot(decoded.Snapshot, state.GameConfig, state.GameReferences,
				xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
			require.NoError(t, err)
			restored.SystemCallbacks = NewMockSystemCallbacks(t).ToSystemCallbacks()
			restored.SetupEnvironmentalSystemCallbacks()
			decoded.Snapshot.AtlasLandmarks[0].Kind = "hut"
			require.Equal(t, "towne", restored.atlasLandmarks[0].Kind, "hydration must detach atlas slice")
			recaptured, err := restored.CaptureSaveSnapshot()
			require.NoError(t, err)
			require.Equal(t, snapshot, recaptured, "earned evidence and physical state must survive runtime restoration")
			recaptured.AtlasLandmarks[0].Site.X++
			require.Error(t, ValidateSaveSnapshotCompatibility(recaptured, state.GameReferences), "wrong exterior anchor must fail runtime compatibility")
		})
	}
}
