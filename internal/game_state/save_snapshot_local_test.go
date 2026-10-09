package game_state

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

// Sources are opt-in, read-only and never embedded in repository fixtures.
func TestLocalNativeSaveHydration(t *testing.T) {
	paths := filepath.SplitList(os.Getenv("U5_NATIVE_SAVE_TEST_PATHS"))
	if len(paths) == 0 {
		// Private saves are an optional extra proof, not a provisioned-data gate.
		t.Log("no local saves requested; set U5_NATIVE_SAVE_TEST_PATHS for the opt-in proof")
		return
	}
	refs := getBaseGameReferences()
	require.NotNil(t, refs, "opt-in hydration requires installed original game data")
	require.NotNil(t, cachedIntegrationCfg)
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			before := sha256.Sum256(data)
			copyRoot := t.TempDir()
			copyPath := filepath.Join(copyRoot, "slot-01.json")
			require.NoError(t, os.WriteFile(copyPath, data, 0o600))
			t.Cleanup(func() {
				after, readErr := os.ReadFile(path)
				require.NoError(t, readErr)
				require.Equal(t, before, sha256.Sum256(after), "source bytes must remain untouched")
			})
			envelope, err := save_snapshot.Decode(data)
			require.NoError(t, err)
			repository, err := player_save.NewRepository(copyRoot, func(snapshot save_snapshot.SaveSnapshot) error {
				return ValidateSaveSnapshotCompatibility(snapshot, refs)
			})
			require.NoError(t, err)
			require.Equal(t, player_save.SlotReady, repository.List()[0].Status, "actual slot must not be marked corrupt or incompatible")
			record, err := repository.Load(player_save.Slot01)
			require.NoError(t, err)
			require.Equal(t, envelope, record.Envelope)
			var originalMetadata struct {
				Metadata save_snapshot.Metadata `json:"metadata"`
			}
			require.NoError(t, json.Unmarshal(data, &originalMetadata))
			require.Equal(t, originalMetadata.Metadata, envelope.Metadata, "original metadata and IDs survive promotion")
			state, err := NewGameStateFromSaveSnapshot(envelope.Snapshot, cachedIntegrationCfg, refs, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
			require.NoError(t, err)
			state.SystemCallbacks = NewMockSystemCallbacks(t).ToSystemCallbacks()
			state.SetupEnvironmentalSystemCallbacks()
			recaptured, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			// These two records are scene-owned and carried alongside GameState.
			recaptured.Presentation = envelope.Snapshot.Presentation
			recaptured.LastConversation = envelope.Snapshot.LastConversation
			require.Equal(t, envelope.Snapshot, recaptured, "full authoritative hydration/recapture parity")
			assertHistoricalPersistedBase(t, data, recaptured)
			envelope.Snapshot = recaptured
			encoded, err := save_snapshot.EncodeCanonical(envelope)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(copyPath, encoded, 0o600))
			encoded, err = os.ReadFile(copyPath)
			require.NoError(t, err)
			reloaded, err := save_snapshot.Decode(encoded)
			require.NoError(t, err)
			require.Equal(t, envelope, reloaded, "metadata, presentation and gameplay all survive current encoding")
			require.Equal(t, player_save.SlotReady, repository.List()[0].Status, "recaptured current slot remains loadable")
		})
	}
}

// Compare against the original wire document, not merely against the migrated
// snapshot: a migration that silently drops an old field must fail this test.
func assertHistoricalPersistedBase(t *testing.T, original []byte, recaptured save_snapshot.SaveSnapshot) {
	t.Helper()
	var document map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(original, &document))
	var old map[string]any
	decoder := json.NewDecoder(bytes.NewReader(document["snapshot"]))
	decoder.UseNumber() // Preserve all 64 bits of RNG counters and visit revisions.
	require.NoError(t, decoder.Decode(&old))
	version := persistedInteger(t, old["version"])
	if version < 17 || version > int64(save_snapshot.CurrentVersion) {
		t.Fatalf("local base parity fixture requires v17-current, got %d", version)
	}
	encoded, err := json.Marshal(recaptured)
	require.NoError(t, err)
	var current map[string]any
	decoder = json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&current))
	current["version"] = old["version"]
	if version < 24 {
		stripV24PromotedPersonalState(t, old, current, recaptured)
	}
	if version <= 22 {
		delete(current, "openDoor")
	}
	if version <= 21 {
		stripPromotedSurveyPolicy(t, current)
	}
	if version >= 20 {
		if version == 20 {
			assertV20PersistedMoonstones(t, document["snapshot"], recaptured)
			delete(old, "moonstoneJournal")
			delete(current, "moonstoneJournal")
		}
		require.Equal(t, old, current, "every unchanged persisted v20/current field must survive migration and hydration")
		t.Logf("v%d: persisted base, current-placement moonstone migration, hydration/recapture and metadata preserved", version)
		return
	}
	delete(current, "moonstoneJournal")
	delete(current, "atlasLandmarks")
	require.Empty(t, recaptured.AtlasLandmarks, "old saves cannot acquire inferred atlas observations")
	if units, ok := current["mapUnits"].([]any); ok {
		for _, value := range units {
			delete(value.(map[string]any), "surfaceChestAssessment")
		}
	}
	if dungeon, ok := current["dungeon"].(map[string]any); ok {
		delete(dungeon, "chestAssessments")
	}
	for index, value := range old["dungeonJournal"].([]any) {
		observation := value.(map[string]any)
		actual := recaptured.DungeonJournal[index]
		require.Equal(t, observation["known"], actual.Known)
		require.Equal(t, observation["wordUsed"], actual.WordUsed)
		if observation["entranceSeen"].(bool) {
			slot := journal.DungeonSurfaceSite
			if persistedInteger(t, observation["entranceFloor"]) == -1 {
				slot = journal.DungeonUnderworldSite
			}
			site := actual.Sites[slot]
			require.True(t, site.Seen)
			require.Equal(t, observation["open"], site.Open)
			require.EqualValues(t, persistedInteger(t, observation["entranceX"]), site.Location.X)
			require.EqualValues(t, persistedInteger(t, observation["entranceY"]), site.Location.Y)
			require.EqualValues(t, persistedInteger(t, observation["entranceFloor"]), site.Location.Floor)
			require.False(t, site.Survey.Measured)
		}
	}
	delete(old, "dungeonJournal")
	delete(current, "dungeonJournal")
	require.Equal(t, old, current, "every persisted base field must survive migration and hydration")
	t.Logf("v%d: full persisted base, dungeon evidence, hydration/recapture and metadata preserved", version)
}

func stripV24PromotedPersonalState(t *testing.T, old, current map[string]any, recaptured save_snapshot.SaveSnapshot) {
	t.Helper()
	removeSnapshotV24Fields(current)
	require.Zero(t, recaptured.AtlasNotes.LastID)
	require.Empty(t, recaptured.AtlasNotes.Records)
	require.Zero(t, recaptured.AtlasFleet.LastID)
	require.False(t, recaptured.CodexRevelationComplete)
	party := old["party"].(map[string]any)
	if flags, ok := party["deadNpcs"].([]any); ok {
		kept := make([]any, 0, len(flags))
		for _, value := range flags {
			flag := value.(map[string]any)
			daily := false
			for index, source := range references.LordBritishBasementChests() {
				if persistedInteger(t, flag["location"]) == int64(references.LordBritishsCastle-references.Moonglow) && persistedInteger(t, flag["npc"]) == int64(source.NPCNumber) {
					daily = true
					calendar := recaptured.World.Calendar
					require.Equal(t, &save_snapshot.BritannianDateSnapshot{Year: calendar.Year, Month: calendar.Month, Day: calendar.Day}, recaptured.BasementChestHistory[index].LastOpened)
				}
			}
			if !daily {
				kept = append(kept, value)
			}
		}
		party["deadNpcs"] = kept
	}
}

// removeSnapshotV24Fields projects the V23-and-earlier wire shape for historical
// fixtures and parity comparisons, including separately boarded/parked vehicles.
func removeSnapshotV24Fields(body map[string]any) {
	for _, field := range []string{"basementChestHistory", "atlasNotes", "atlasFleet", "codexRevelationComplete"} {
		delete(body, field)
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

// V22 adds only explicit original-policy false to these historical surveys.
// Assert the value before removing that migration-only member from comparison;
// do not discard any original coordinate, source, date, number or collection.
func stripPromotedSurveyPolicy(t *testing.T, current map[string]any) {
	t.Helper()
	for _, value := range current["dungeonJournal"].([]any) {
		for _, site := range value.(map[string]any)["sites"].([]any) {
			survey := site.(map[string]any)["survey"].(map[string]any)
			require.Equal(t, false, survey["flexible"])
			delete(survey, "flexible")
		}
	}
	for _, family := range []string{"moonstoneJournal", "atlasLandmarks"} {
		if values, ok := current[family].([]any); ok {
			for _, value := range values {
				survey := value.(map[string]any)["survey"].(map[string]any)
				require.Equal(t, false, survey["flexible"])
				delete(survey, "flexible")
			}
		}
	}
}

func assertV20PersistedMoonstones(t *testing.T, original json.RawMessage, recaptured save_snapshot.SaveSnapshot) {
	t.Helper()
	var old struct {
		MoonstoneJournal [8]journal.MoonstoneObservation `json:"moonstoneJournal"`
		Inventory        save_snapshot.InventorySnapshot `json:"inventory"`
	}
	require.NoError(t, json.Unmarshal(original, &old))
	for phase, observation := range old.MoonstoneJournal {
		carried := old.Inventory.Moonstones[phase] > 0
		if !observation.Known && carried {
			observation = journal.MoonstoneObservation{Known: true, FirstSource: journal.MoonstoneSourcePossession}
		}
		if observation.Known {
			canonical := moonstoneJournalSite(references.ChartedMoonstoneOrigins()[phase].Location)
			handled := func(source journal.MoonstoneSource) bool {
				return source == journal.MoonstoneSourcePickup || source == journal.MoonstoneSourceBurial
			}
			observation.OriginalSiteRetired = carried || handled(observation.FirstSource) ||
				handled(observation.LastContext.Source) || handled(observation.SurveyContext.Source) ||
				observation.SiteKnown && observation.Site != canonical ||
				observation.Survey.Measured && observation.Survey.Location != observation.Site
			if carried || !observation.Placed || !observation.SiteKnown || observation.Survey.Location != observation.Site ||
				observation.SurveyContext.Source == journal.MoonstoneSourcePickup ||
				observation.LastContext.Source == journal.MoonstoneSourceBurial && observation.Survey.RecordedAt != observation.ObservedAt {
				observation.Survey, observation.SurveyContext = journal.SurveyedLocation{}, journal.MoonstoneContext{}
			}
		}
		require.Equal(t, observation, recaptured.MoonstoneJournal[phase], "v20 phase %d must preserve all evidence except intentional origin retirement/stale survey removal", phase)
	}
}

func persistedInteger(t *testing.T, value any) int64 {
	t.Helper()
	number, ok := value.(json.Number)
	require.True(t, ok)
	integer, err := number.Int64()
	require.NoError(t, err)
	return integer
}
