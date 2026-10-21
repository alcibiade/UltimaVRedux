package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/legacy_save"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestPlayerSaveRejectsUnsafeSnapshotWithoutReplacingScene(t *testing.T) {
	gameReferences := &references.GameReferences{
		LocationReferences: &references.LocationReferences{},
		EnemyReferences:    &references.EnemyReferences{},
		TileReferences:     references.NewTileReferences(),
	}
	for _, kind := range []string{"unknown terrain", "unknown object", "extreme clock", "wrong controller world"} {
		t.Run(kind, func(t *testing.T) {
			snapshot := validCommandTestSnapshot()
			snapshot.World.Position.Location = int(references.BritanniaUnderworld)
			snapshot.World.LastLargeMapPosition = snapshot.World.Position
			snapshot.Wind.NextTickMs = 55
			require.NoError(t, game_state.ValidateSaveSnapshotCompatibility(snapshot, gameReferences))
			switch kind {
			case "unknown terrain", "unknown object":
				mutation := save_snapshot.MapMutationSnapshot{Position: snapshot.World.Position, Kind: save_snapshot.MapMutationSetTile, Value: 999999}
				if kind == "unknown object" {
					mutation.Kind = save_snapshot.MapMutationSetObject
				}
				snapshot.MapMutations = []save_snapshot.MapMutationSnapshot{mutation}
			case "extreme clock":
				snapshot.World.ElapsedMs, snapshot.Wind.NextTickMs = math.MaxInt64-100, math.MaxInt64-99
			case "wrong controller world":
				chest := 0
				snapshot.MapUnits = []save_snapshot.MapUnitSnapshot{{
					Kind: save_snapshot.MapUnitKindChest, World: -1, Position: snapshot.World.Position,
					SurfaceChestNumber: &chest, AIOverrideProvenance: string(references.AIOverrideProvenanceNone),
				}}
			}
			repository, err := player_save.NewRepository(t.TempDir(), func(candidate save_snapshot.SaveSnapshot) error {
				return game_state.ValidateSaveSnapshotCompatibility(candidate, gameReferences)
			})
			require.NoError(t, err)
			envelope := playerSaveTestRecord(player_save.Slot01, "", snapshot).Envelope
			record, err := repository.Save(player_save.Slot01, envelope, "")
			require.NoError(t, err)
			require.Equal(t, player_save.SlotIncompatible, repository.List()[0].Status)
			oldScene := &playerSaveTestScene{
				snapshot: validCommandTestSnapshot(), playthrough: playthroughMetadata{id: testPlaythroughID},
			}
			game := NewGame(oldScene)
			generation := game.DebugSceneGeneration()
			game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
				t.Fatal("incompatible snapshot reached scene construction")
				return nil, nil
			})
			require.ErrorIs(t, game.loadPlayerSaveSlot(player_save.Slot01, record.Revision), player_save.ErrIncompatible)
			assert.Same(t, oldScene, game.currentScene)
			assert.Equal(t, generation, game.DebugSceneGeneration())
			assert.Zero(t, oldScene.disposals)
		})
	}
}

type playerSaveTestRepository struct {
	records             map[player_save.SlotID]player_save.Record
	summaries           []player_save.SlotSummary
	latest              player_save.Record
	latestErr           error
	compatibilityErr    error
	saveErr             error
	deleteErr           error
	restoreErr          error
	saves               []playerSaveTestSave
	loads               []player_save.SlotID
	deletes             []playerSaveTestDelete
	restores            []playerSaveTestRestore
	summariesProvider   func() []player_save.SlotSummary
	listCalls           int
	latestCalls         int
	nextRevision        int
	compatibilityChecks int
	events              *[]string
}

type playerSaveTestSave struct {
	slot             player_save.SlotID
	envelope         save_snapshot.Envelope
	expectedRevision string
}

type playerSaveTestDelete struct {
	slot             player_save.SlotID
	expectedRevision string
}

type playerSaveTestRestore struct {
	record           player_save.Record
	expectedRevision string
}

func (repository *playerSaveTestRepository) List() []player_save.SlotSummary {
	repository.listCalls++
	if repository.summariesProvider != nil {
		return repository.summariesProvider()
	}
	return append([]player_save.SlotSummary(nil), repository.summaries...)
}

func (repository *playerSaveTestRepository) Load(slot player_save.SlotID) (player_save.Record, error) {
	repository.loads = append(repository.loads, slot)
	if repository.events != nil {
		*repository.events = append(*repository.events, "load:"+string(slot))
	}
	record, ok := repository.records[slot]
	if !ok {
		return player_save.Record{}, &player_save.NotFoundError{Slot: slot}
	}

	return record, nil
}

func (repository *playerSaveTestRepository) Save(
	slot player_save.SlotID,
	envelope save_snapshot.Envelope,
	expectedRevision string,
) (player_save.Record, error) {
	repository.saves = append(repository.saves, playerSaveTestSave{slot: slot, envelope: envelope, expectedRevision: expectedRevision})
	if repository.events != nil {
		*repository.events = append(*repository.events, "save:"+string(slot))
	}
	if repository.saveErr != nil {
		return player_save.Record{}, repository.saveErr
	}
	current, occupied := repository.records[slot]
	actual := ""
	if occupied {
		actual = current.Revision
	}
	if actual != expectedRevision {
		return player_save.Record{}, &player_save.StaleRevisionError{Slot: slot, Expected: expectedRevision, Actual: actual}
	}
	repository.nextRevision++
	record := player_save.Record{Slot: slot, Envelope: envelope, Revision: fmt.Sprintf("revision-%d", repository.nextRevision)}
	if repository.records == nil {
		repository.records = make(map[player_save.SlotID]player_save.Record)
	}
	repository.records[slot] = record

	return record, nil
}

func (repository *playerSaveTestRepository) Delete(slot player_save.SlotID, expectedRevision string) error {
	repository.deletes = append(repository.deletes, playerSaveTestDelete{slot: slot, expectedRevision: expectedRevision})
	if repository.deleteErr != nil {
		return repository.deleteErr
	}
	record, ok := repository.records[slot]
	if !ok {
		return &player_save.NotFoundError{Slot: slot}
	}
	if record.Revision != expectedRevision {
		return &player_save.StaleRevisionError{Slot: slot, Expected: expectedRevision, Actual: record.Revision}
	}
	delete(repository.records, slot)

	return nil
}

func (repository *playerSaveTestRepository) Restore(record player_save.Record, expectedRevision string) error {
	repository.restores = append(repository.restores, playerSaveTestRestore{
		record: record, expectedRevision: expectedRevision,
	})
	if repository.restoreErr != nil {
		return repository.restoreErr
	}
	current, ok := repository.records[record.Slot]
	if !ok {
		return &player_save.NotFoundError{Slot: record.Slot}
	}
	if current.Revision != expectedRevision {
		return &player_save.StaleRevisionError{
			Slot: record.Slot, Expected: expectedRevision, Actual: current.Revision,
		}
	}
	repository.records[record.Slot] = record

	return nil
}

func (repository *playerSaveTestRepository) Latest() (player_save.Record, error) {
	repository.latestCalls++
	if repository.latestErr != nil {
		return player_save.Record{}, repository.latestErr
	}
	if repository.latest.Slot.Valid() {
		return repository.latest, nil
	}

	return player_save.Record{}, &player_save.NotFoundError{}
}

func (repository *playerSaveTestRepository) ValidateCompatibility(save_snapshot.SaveSnapshot) error {
	repository.compatibilityChecks++

	return repository.compatibilityErr
}

type playerSaveTestScene struct {
	onActivate          func()
	playthrough         playthroughMetadata
	configuration       *config.UltimaVConfiguration
	snapshot            save_snapshot.SaveSnapshot
	captureErr          error
	preflightErr        error
	activatePanic       bool
	captures            int
	preflights          int
	activations         int
	disposals           int
	slot                player_save.SlotID
	revision            string
	events              *[]string
	inputPreparations   int
	provenanceAtCapture bool
	doomRunID           string
}

const testPlaythroughID = "0123456789abcdef0123456789abcdef"

func (scene *playerSaveTestScene) setPlaythroughMetadata(metadata playthroughMetadata) {
	scene.playthrough = metadata
}
func (scene *playerSaveTestScene) playthroughMetadata() playthroughMetadata {
	return scene.playthrough
}

func (scene *playerSaveTestScene) Update(_ *Game) error { return nil }
func (scene *playerSaveTestScene) Draw(_ *ebiten.Image) {}
func (scene *playerSaveTestScene) GetUltimaConfiguration() *config.UltimaVConfiguration {
	return scene.configuration
}
func (scene *playerSaveTestScene) InvalidateResolution() {}
func (scene *playerSaveTestScene) CaptureSaveSnapshot() (save_snapshot.SaveSnapshot, error) {
	scene.captures++
	_, _, scene.provenanceAtCapture = scene.playerSaveProvenance()
	if scene.events != nil {
		*scene.events = append(*scene.events, "capture")
	}
	if scene.captureErr != nil {
		return save_snapshot.SaveSnapshot{}, scene.captureErr
	}

	return scene.snapshot, nil
}
func (scene *playerSaveTestScene) setPlayerSaveProvenance(slot player_save.SlotID, revision string) {
	scene.slot, scene.revision = slot, revision
}
func (scene *playerSaveTestScene) playerSaveProvenance() (player_save.SlotID, string, bool) {
	return scene.slot, scene.revision, scene.slot.Valid() && scene.revision != ""
}

// The fake implements doomRunAssociationOwner, which player saves detect with a
// type assertion; dropping any of these methods silently skips that path.
func (scene *playerSaveTestScene) setDoomRunAssociation(runID string) {
	scene.doomRunID = runID
}
func (scene *playerSaveTestScene) doomRunAssociation() string {
	return scene.doomRunID
}
func (scene *playerSaveTestScene) doomRunAssociationForPlayerSave() string {
	return scene.doomRunID
}
func (scene *playerSaveTestScene) PreflightActivation() error {
	scene.preflights++
	if scene.events != nil {
		*scene.events = append(*scene.events, "preflight")
	}

	return scene.preflightErr
}
func (scene *playerSaveTestScene) Activate() {
	if scene.onActivate != nil {
		scene.onActivate()
	}
	scene.activations++
	if scene.activatePanic {
		panic("activation failed")
	}
}
func (scene *playerSaveTestScene) Dispose() { scene.disposals++ }
func (scene *playerSaveTestScene) PrepareInputForSceneActivation() {
	scene.inputPreparations++
}

type playerSaveNoCaptureScene struct {
	playthrough playthroughMetadata
	preflights  int
	disposals   int
	slot        player_save.SlotID
	revision    string
}

func (scene *playerSaveNoCaptureScene) setPlaythroughMetadata(metadata playthroughMetadata) {
	scene.playthrough = metadata
}
func (scene *playerSaveNoCaptureScene) playthroughMetadata() playthroughMetadata {
	return scene.playthrough
}

func (scene *playerSaveNoCaptureScene) Update(_ *Game) error { return nil }
func (scene *playerSaveNoCaptureScene) Draw(_ *ebiten.Image) {}
func (scene *playerSaveNoCaptureScene) GetUltimaConfiguration() *config.UltimaVConfiguration {
	return nil
}
func (scene *playerSaveNoCaptureScene) InvalidateResolution() {}
func (scene *playerSaveNoCaptureScene) setPlayerSaveProvenance(slot player_save.SlotID, revision string) {
	scene.slot, scene.revision = slot, revision
}
func (scene *playerSaveNoCaptureScene) playerSaveProvenance() (player_save.SlotID, string, bool) {
	return scene.slot, scene.revision, scene.slot.Valid() && scene.revision != ""
}
func (scene *playerSaveNoCaptureScene) PreflightActivation() error {
	scene.preflights++

	return nil
}
func (scene *playerSaveNoCaptureScene) Dispose() { scene.disposals++ }

func TestPlayerSaveListAndSaveCaptureExactlyOnce(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	oldRecord := playerSaveTestRecord(player_save.Slot03, "reviewed", snapshot)
	repository := &playerSaveTestRepository{
		records:   map[player_save.SlotID]player_save.Record{player_save.Slot03: oldRecord},
		summaries: []player_save.SlotSummary{{Slot: player_save.Slot03, Status: player_save.SlotReady, Revision: "reviewed"}},
	}
	scene := &playerSaveTestScene{snapshot: snapshot, playthrough: playthroughMetadata{id: testPlaythroughID}}
	game := NewGame(scene)
	game.ConfigurePlayerSaveRepository(repository, nil)
	now := time.Date(2026, time.August, 14, 11, 12, 13, 14, time.FixedZone("test", -7*60*60))
	game.playerSaveNow = func() time.Time { return now }
	game.playerSaveGameVersion = "v-test"

	slots, err := game.listPlayerSaveSlots()
	require.NoError(t, err)
	require.Equal(t, repository.summaries, slots)
	record, err := game.savePlayerSaveSlot(player_save.Slot03, "reviewed", "")
	require.NoError(t, err)
	assert.Equal(t, 1, scene.captures)
	require.Len(t, repository.saves, 1)
	save := repository.saves[0]
	assert.Equal(t, "reviewed", save.expectedRevision)
	assert.Equal(t, "Slot 3", save.envelope.Metadata.Name)
	assert.Equal(t, playerSaveGameplayDescription, save.envelope.Metadata.Description)
	assert.Equal(t, now.UTC(), save.envelope.Metadata.CreatedAt)
	assert.Equal(t, time.UTC, save.envelope.Metadata.CreatedAt.Location())
	assert.Equal(t, "v-test", save.envelope.Metadata.GameVersion)
	assert.Equal(t, save_snapshot.CurrentVersion, save.envelope.Snapshot.Version)
	assert.Equal(t, player_save.Slot03, scene.slot)
	assert.Equal(t, record.Revision, scene.revision)
}

func TestPlayerSaveLoadRequiresReviewedRevisionAndLatestLoadsFromCurrentScene(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	record := playerSaveTestRecord(player_save.Slot05, "current", snapshot)
	repository := &playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{player_save.Slot05: record}, latest: record}
	oldScene := &playerSaveTestScene{
		configuration: &config.UltimaVConfiguration{}, playthrough: playthroughMetadata{id: otherTestPlaythroughID},
	}
	newScene := &playerSaveTestScene{}
	builds := 0
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(repository, func(configuration *config.UltimaVConfiguration, got save_snapshot.SaveSnapshot) (Scene, error) {
		builds++
		assert.Same(t, oldScene.configuration, configuration)
		assert.Equal(t, snapshot, got)

		return newScene, nil
	})

	err := game.loadPlayerSaveSlot(player_save.Slot05, "stale")
	var stale *player_save.StaleRevisionError
	require.ErrorIs(t, err, player_save.ErrStaleRevision)
	require.ErrorAs(t, err, &stale)
	assert.Equal(t, "stale", stale.Expected)
	assert.Equal(t, "current", stale.Actual)
	assert.Zero(t, builds)
	assert.Same(t, oldScene, game.currentScene)

	require.NoError(t, game.loadLatestPlayerSave())
	assert.Equal(t, 1, builds)
	assert.Same(t, newScene, game.currentScene)
	assert.Equal(t, player_save.Slot05, newScene.slot)
	assert.Equal(t, "current", newScene.revision)
	assert.Equal(t, 1, newScene.inputPreparations)
}

func TestPlayerSaveCandidateFailureLeavesCurrentSceneActive(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	record := playerSaveTestRecord(player_save.Slot02, "reviewed", snapshot)
	constructionErr := errors.New("builder rejected snapshot")
	tests := []struct {
		name      string
		builder   func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error)
		candidate *playerSaveTestScene
	}{
		{
			name: "construction",
			builder: func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
				return nil, constructionErr
			},
		},
		{
			name:      "activation",
			candidate: &playerSaveTestScene{activatePanic: true},
			builder: func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
				return nil, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldScene := &playerSaveTestScene{playthrough: playthroughMetadata{id: otherTestPlaythroughID}}
			repository := &playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{player_save.Slot02: record}}
			game := NewGame(oldScene)
			builder := test.builder
			if test.candidate != nil {
				builder = func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
					return test.candidate, nil
				}
			}
			game.ConfigurePlayerSaveRepository(repository, builder)

			err := game.loadPlayerSaveSlot(player_save.Slot02, "reviewed")
			require.Error(t, err)
			assert.Same(t, oldScene, game.currentScene)
			assert.Equal(t, 0, oldScene.disposals)
			if test.candidate != nil {
				assert.Equal(t, 1, test.candidate.disposals)
				assert.Equal(t, "reviewed", test.candidate.revision)
			}
		})
	}
}

func TestPlayerSaveQuickSaveRequiresAndUpdatesActiveSlot(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	repository := &playerSaveTestRepository{records: make(map[player_save.SlotID]player_save.Record)}
	scene := &playerSaveTestScene{snapshot: snapshot, playthrough: playthroughMetadata{id: testPlaythroughID}}
	game := NewGame(scene)
	game.ConfigurePlayerSaveRepository(repository, nil)

	_, err := game.quickSavePlayerSave()
	var noActive *NoActivePlayerSaveSlotError
	require.ErrorIs(t, err, ErrNoActivePlayerSaveSlot)
	require.ErrorAs(t, err, &noActive)
	assert.Zero(t, scene.captures)

	repository.records[player_save.Slot07] = playerSaveTestRecord(player_save.Slot07, "loaded", snapshot)
	scene.setPlayerSaveProvenance(player_save.Slot07, "loaded")
	record, err := game.quickSavePlayerSave()
	require.NoError(t, err)
	assert.Equal(t, 1, scene.captures)
	require.Len(t, repository.saves, 1)
	assert.Equal(t, "loaded", repository.saves[0].expectedRevision)
	assert.Equal(t, record.Revision, scene.revision)
}

func TestPlayerSaveDoomRunAssociationSurvivesOrdinarySaveLoad(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	snapshot.World.Position.Location = int(references.Doom)
	repository := &playerSaveTestRepository{records: make(map[player_save.SlotID]player_save.Record)}
	source := &playerSaveTestScene{
		snapshot: snapshot, doomRunID: "opaque-doom-run", playthrough: playthroughMetadata{id: testPlaythroughID},
	}
	game := NewGame(source)
	game.ConfigurePlayerSaveRepository(repository, nil)

	record, err := game.savePlayerSaveSlot(player_save.Slot01, "", "")
	require.NoError(t, err)
	assert.Equal(t, "opaque-doom-run", record.Envelope.Metadata.DoomRunID)

	candidate := &playerSaveTestScene{}
	game.playerSaveSceneBuilder = func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		return candidate, nil
	}
	require.NoError(t, game.loadPlayerSaveRecord(record))
	assert.Equal(t, "opaque-doom-run", candidate.doomRunID)
}

func TestPlayerSaveWithoutAssociatedDoomMetadataCannotClaimRecovery(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	snapshot.World.Position.Location = int(references.Doom)
	record := playerSaveTestRecord(player_save.Slot01, "revision", snapshot)
	candidate := &playerSaveTestScene{doomRunID: "stale"}
	game := NewGame(&playerSaveTestScene{playthrough: playthroughMetadata{id: otherTestPlaythroughID}})
	game.ConfigurePlayerSaveRepository(&playerSaveTestRepository{}, func(
		*config.UltimaVConfiguration,
		save_snapshot.SaveSnapshot,
	) (Scene, error) {
		return candidate, nil
	})

	require.NoError(t, game.loadPlayerSaveRecord(record))
	assert.Empty(t, candidate.doomRunID)
}

func TestPlayerSaveRunMetadataOutsideDoomIsClearedOnLoad(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	snapshot.World.Position.Location = int(references.Britain)
	record := playerSaveTestRecord(player_save.Slot01, "revision", snapshot)
	record.Envelope.Metadata.DoomRunID = "unrelated-run"
	candidate := &playerSaveTestScene{doomRunID: "stale"}
	game := NewGame(&playerSaveTestScene{playthrough: playthroughMetadata{id: otherTestPlaythroughID}})
	game.ConfigurePlayerSaveRepository(&playerSaveTestRepository{}, func(
		*config.UltimaVConfiguration,
		save_snapshot.SaveSnapshot,
	) (Scene, error) {
		return candidate, nil
	})

	require.NoError(t, game.loadPlayerSaveRecord(record))
	assert.Empty(t, candidate.doomRunID)
}

func TestPlayerSaveDoomMetadataRejectionDisposesCandidate(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	snapshot.World.Position.Location = int(references.Doom)
	record := playerSaveTestRecord(player_save.Slot01, "revision", snapshot)
	record.Envelope.Metadata.DoomRunID = "associated-run"
	candidate := &playerSaveNoCaptureScene{}
	source := &playerSaveTestScene{playthrough: playthroughMetadata{id: otherTestPlaythroughID}}
	game := NewGame(source)
	game.ConfigurePlayerSaveRepository(&playerSaveTestRepository{}, func(
		*config.UltimaVConfiguration,
		save_snapshot.SaveSnapshot,
	) (Scene, error) {
		return candidate, nil
	})

	err := game.loadPlayerSaveRecord(record)

	require.ErrorIs(t, err, ErrPlayerSaveSceneIneligible)
	assert.Same(t, source, game.currentScene)
	assert.Equal(t, 1, candidate.disposals)
}

func TestPlayerSaveLegacyDungeonConversionSucceeds(t *testing.T) {
	gamPath := writePlayerSaveLegacyBundle(t, true)

	prepared, err := (&Game{playthroughIDProvider: func() (string, error) { return testPlaythroughID, nil }}).prepareLegacyPlayerSave(gamPath)
	require.NoError(t, err)
	require.NotNil(t, prepared.snapshot.Dungeon)
	assert.Equal(t, int(references.Deceit), prepared.snapshot.Dungeon.Location)
}

func TestPlayerSaveLegacyPathFailureDoesNotWrite(t *testing.T) {
	_, err := (&Game{playthroughIDProvider: func() (string, error) { return testPlaythroughID, nil }}).prepareLegacyPlayerSave(filepath.Join(t.TempDir(), "SAVED.GAM"))
	require.Error(t, err)
}

func TestPrepareLegacyPlayerSaveReadsAndConvertsWithoutWriting(t *testing.T) {
	gamPath := writePlayerSaveLegacyBundle(t, false)
	directory := filepath.Dir(gamPath)
	before, err := os.ReadDir(directory)
	require.NoError(t, err)

	prepared, err := (&Game{playthroughIDProvider: func() (string, error) { return testPlaythroughID, nil }}).prepareLegacyPlayerSave(gamPath)
	require.NoError(t, err)
	after, err := os.ReadDir(directory)
	require.NoError(t, err)

	assert.Equal(t, save_snapshot.CurrentVersion, prepared.snapshot.Version)
	assert.NotEmpty(t, prepared.warnings)
	assert.Equal(t, before, after)
}

func TestBCarpetLegacyFixturePreparesAndImportsIntoNativeSlot(t *testing.T) {
	prepared, err := (&Game{playthroughIDProvider: func() (string, error) { return testPlaythroughID, nil }}).prepareLegacyPlayerSave(filepath.Join(
		"..", "..", "internal", "legacy_save", "testdata", "b_carpet", "SAVED.GAM",
	))
	require.NoError(t, err)
	assert.Equal(t, "Avatar", prepared.snapshot.Party.Roster[0].Name)
	assert.Equal(t, save_snapshot.ToggleSnapshot{Active: true, Turns: 255}, prepared.snapshot.Effects.Amulet)
	require.NotNil(t, prepared.snapshot.PartyVehicle)
	assert.Equal(t, int(references.CarpetVehicle), prepared.snapshot.PartyVehicle.Type)
	assert.NotEmpty(t, prepared.warnings)
	assert.False(t, prepared.snapshot.NeedsMapUnitReconstruction())
	assert.NotEmpty(t, prepared.snapshot.MapUnits)

	repository := &playerSaveTestRepository{records: make(map[player_save.SlotID]player_save.Record)}
	oldScene := &playerSaveTestScene{}
	canonical := canonicalPlayerSaveTestSnapshot()
	canonical.Continuation = prepared.snapshot.Continuation
	loadedScene := &playerSaveTestScene{snapshot: canonical}
	game := NewGame(oldScene)
	game.playerSaveNow = func() time.Time { return time.Date(2026, time.August, 14, 20, 0, 0, 0, time.UTC) }
	game.ConfigurePlayerSaveRepository(repository, func(_ *config.UltimaVConfiguration, snapshot save_snapshot.SaveSnapshot) (Scene, error) {
		assert.Equal(t, prepared.snapshot, snapshot)

		return loadedScene, nil
	})

	require.NoError(t, game.importPreparedLegacyPlayerSave(player_save.Slot02, "", prepared))
	require.Len(t, repository.saves, 1)
	assert.Equal(t, canonical, repository.saves[0].envelope.Snapshot)
	assert.Equal(t, prepared.snapshot.Continuation, repository.saves[0].envelope.Snapshot.Continuation)
	assertCanonicalPreparedPlayerSave(t, repository.saves[0].envelope.Snapshot)
	assert.Equal(t, 1, loadedScene.captures)
	assert.False(t, loadedScene.provenanceAtCapture)
	assert.Same(t, loadedScene, game.currentScene)
}

func TestPlayerSavePreparedLegacyPreflightFailureDoesNotWrite(t *testing.T) {
	prepared, err := (&Game{playthroughIDProvider: func() (string, error) { return testPlaythroughID, nil }}).prepareLegacyPlayerSave(writePlayerSaveLegacyBundle(t, false))
	require.NoError(t, err)

	preflightErr := errors.New("candidate cannot activate")
	existing := playerSaveTestRecord(player_save.Slot06, "existing", validCommandTestSnapshot())
	repository := &playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{player_save.Slot06: existing}}
	oldScene := &playerSaveTestScene{}
	candidate := &playerSaveTestScene{preflightErr: preflightErr}
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		return candidate, nil
	})

	err = game.importPreparedLegacyPlayerSave(player_save.Slot06, "existing", prepared)
	require.ErrorIs(t, err, preflightErr)
	assert.Empty(t, repository.saves)
	assert.Equal(t, existing, repository.records[player_save.Slot06])
	assert.Same(t, oldScene, game.currentScene)
	assert.Equal(t, 1, candidate.preflights)
	assert.Zero(t, candidate.captures)
	assert.Equal(t, 1, candidate.disposals)
}

func TestPlayerSavePreparedCandidateRequiresCaptureWithoutReviewingOrWriting(t *testing.T) {
	prepared := validCommandTestSnapshot()
	existing := playerSaveTestRecord(player_save.Slot06, "existing", validCommandTestSnapshot())
	repository := &playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{player_save.Slot06: existing}}
	oldScene := &playerSaveTestScene{}
	candidate := &playerSaveNoCaptureScene{}
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		return candidate, nil
	})

	err := game.commitPreparedPlayerSave(player_save.Slot06, "existing", prepared, playerSaveLegacyDescription, playthroughMetadata{id: testPlaythroughID})
	require.ErrorIs(t, err, ErrPlayerSaveSceneIneligible)
	assert.Same(t, oldScene, game.currentScene)
	assert.Equal(t, existing, repository.records[player_save.Slot06])
	assert.Empty(t, repository.loads)
	assert.Empty(t, repository.saves)
	assert.Equal(t, 1, candidate.preflights)
	assert.Equal(t, 1, candidate.disposals)
	assert.Empty(t, candidate.revision)
}

func TestPlayerSavePreparedCaptureFailureWithoutReviewingOrWriting(t *testing.T) {
	captureErr := errors.New("candidate capture failed")
	existing := playerSaveTestRecord(player_save.Slot06, "existing", validCommandTestSnapshot())
	repository := &playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{player_save.Slot06: existing}}
	oldScene := &playerSaveTestScene{}
	candidate := &playerSaveTestScene{captureErr: captureErr}
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		return candidate, nil
	})

	err := game.commitPreparedPlayerSave(player_save.Slot06, "existing", validCommandTestSnapshot(), playerSaveLegacyDescription, playthroughMetadata{id: testPlaythroughID})
	require.ErrorIs(t, err, captureErr)
	assert.Same(t, oldScene, game.currentScene)
	assert.Equal(t, existing, repository.records[player_save.Slot06])
	assert.Empty(t, repository.loads)
	assert.Empty(t, repository.saves)
	assert.Equal(t, 1, candidate.preflights)
	assert.Equal(t, 1, candidate.captures)
	assert.Equal(t, 1, candidate.disposals)
	assert.False(t, candidate.provenanceAtCapture)
	assert.Empty(t, candidate.revision)
}

func TestPlayerSavePreparedCaptureRejectsReconstructionBridgeWithoutWriting(t *testing.T) {
	captured := validCommandTestSnapshot()
	captured.MarkMapUnitsForReconstruction()
	existing := playerSaveTestRecord(player_save.Slot06, "existing", validCommandTestSnapshot())
	repository := &playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{player_save.Slot06: existing}}
	oldScene := &playerSaveTestScene{}
	candidate := &playerSaveTestScene{snapshot: captured}
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		return candidate, nil
	})

	err := game.commitPreparedPlayerSave(player_save.Slot06, "existing", validCommandTestSnapshot(), playerSaveLegacyDescription, playthroughMetadata{id: testPlaythroughID})
	require.ErrorIs(t, err, ErrPlayerSaveSceneIneligible)
	assert.Same(t, oldScene, game.currentScene)
	assert.Equal(t, existing, repository.records[player_save.Slot06])
	assert.Empty(t, repository.loads)
	assert.Empty(t, repository.saves)
	assert.Equal(t, 1, candidate.captures)
	assert.Equal(t, 1, candidate.disposals)
	assert.False(t, candidate.provenanceAtCapture)
}

func TestPlayerSavePreparedLegacyCompatibilityFailureDoesNotConstructOrWrite(t *testing.T) {
	prepared := preparedLegacyPlayerSave{snapshot: validCommandTestSnapshot(), playthrough: playthroughMetadata{id: testPlaythroughID}}
	compatibilityErr := errors.New("snapshot references are unavailable")
	repository := &playerSaveTestRepository{compatibilityErr: compatibilityErr}
	builds := 0
	game := NewGame(&playerSaveTestScene{})
	game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		builds++

		return &playerSaveTestScene{}, nil
	})

	err := game.importPreparedLegacyPlayerSave(player_save.Slot01, "", prepared)
	require.ErrorIs(t, err, compatibilityErr)
	assert.Equal(t, 1, repository.compatibilityChecks)
	assert.Zero(t, builds)
	assert.Empty(t, repository.saves)
}

func TestPlayerSavePreparedLegacyActivationFailureRollsBackDestination(t *testing.T) {
	prepared := preparedLegacyPlayerSave{snapshot: validCommandTestSnapshot(), playthrough: playthroughMetadata{id: testPlaythroughID}}
	for _, test := range []struct {
		name     string
		slot     player_save.SlotID
		existing *player_save.Record
	}{
		{name: "empty", slot: player_save.Slot04},
		{name: "occupied", slot: player_save.Slot05, existing: func() *player_save.Record {
			record := playerSaveTestRecord(player_save.Slot05, "existing", validCommandTestSnapshot())

			return &record
		}()},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &playerSaveTestRepository{records: make(map[player_save.SlotID]player_save.Record)}
			expectedRevision := ""
			if test.existing != nil {
				repository.records[test.slot] = *test.existing
				expectedRevision = test.existing.Revision
			}
			oldScene := &playerSaveTestScene{}
			candidate := &playerSaveTestScene{snapshot: canonicalPlayerSaveTestSnapshot(), activatePanic: true}
			game := NewGame(oldScene)
			game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
				return candidate, nil
			})

			err := game.importPreparedLegacyPlayerSave(test.slot, expectedRevision, prepared)
			require.Error(t, err)
			assert.Same(t, oldScene, game.currentScene)
			if test.existing == nil {
				assert.NotContains(t, repository.records, test.slot)
				require.Len(t, repository.deletes, 1)
			} else {
				restored := repository.records[test.slot]
				assert.Equal(t, test.existing.Envelope, restored.Envelope)
				require.Len(t, repository.saves, 1)
				require.Len(t, repository.restores, 1)
			}
		})
	}
}

func TestPlayerSavePreparedActivationFailureReportsRollbackFailure(t *testing.T) {
	rollbackErr := errors.New("rollback failed")
	repository := &playerSaveTestRepository{
		records:   make(map[player_save.SlotID]player_save.Record),
		deleteErr: rollbackErr,
	}
	oldScene := &playerSaveTestScene{}
	candidate := &playerSaveTestScene{snapshot: canonicalPlayerSaveTestSnapshot(), activatePanic: true}
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(repository, func(
		*config.UltimaVConfiguration,
		save_snapshot.SaveSnapshot,
	) (Scene, error) {
		return candidate, nil
	})

	err := game.commitPreparedPlayerSave(
		player_save.Slot04,
		"",
		validCommandTestSnapshot(),
		playerSaveNewGameDescription,
		playthroughMetadata{id: testPlaythroughID},
	)

	require.ErrorIs(t, err, rollbackErr)
	require.ErrorContains(t, err, "activating prepared player save")
	require.ErrorContains(t, err, "rolling back prepared player save")
	assert.Same(t, oldScene, game.currentScene)
	require.Len(t, repository.saves, 1)
	require.Len(t, repository.deletes, 1)
}

func TestPlayerSavePreparedActivationFailureReportsRestoreFailure(t *testing.T) {
	rollbackErr := errors.New("restore failed")
	existing := playerSaveTestRecord(player_save.Slot05, "existing", validCommandTestSnapshot())
	repository := &playerSaveTestRepository{
		records:    map[player_save.SlotID]player_save.Record{player_save.Slot05: existing},
		restoreErr: rollbackErr,
	}
	oldScene := &playerSaveTestScene{}
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(repository, func(
		*config.UltimaVConfiguration,
		save_snapshot.SaveSnapshot,
	) (Scene, error) {
		return &playerSaveTestScene{snapshot: canonicalPlayerSaveTestSnapshot(), activatePanic: true}, nil
	})

	err := game.commitPreparedPlayerSave(
		player_save.Slot05,
		existing.Revision,
		validCommandTestSnapshot(),
		playerSaveNewGameDescription,
		playthroughMetadata{id: testPlaythroughID},
	)

	require.ErrorIs(t, err, rollbackErr)
	require.ErrorContains(t, err, "activating prepared player save")
	require.ErrorContains(t, err, "rolling back prepared player save")
	assert.Same(t, oldScene, game.currentScene)
	require.Len(t, repository.saves, 1)
	require.Len(t, repository.restores, 1)
}

func TestPlayerSaveLegacyImportPersistsCanonicalNativeSnapshotThenLoadsNormally(t *testing.T) {
	gamPath := writePlayerSaveLegacyBundle(t, false)
	events := make([]string, 0)
	repository := &playerSaveTestRepository{records: make(map[player_save.SlotID]player_save.Record), events: &events}
	oldScene := &playerSaveTestScene{}
	canonical := canonicalPlayerSaveTestSnapshot()
	loadedScene := &playerSaveTestScene{snapshot: canonical, events: &events}
	game := NewGame(oldScene)
	game.playerSaveNow = func() time.Time { return time.Date(2026, time.August, 14, 18, 0, 0, 0, time.UTC) }
	game.playerSaveGameVersion = "import-test"
	game.ConfigurePlayerSaveRepository(repository, func(_ *config.UltimaVConfiguration, snapshot save_snapshot.SaveSnapshot) (Scene, error) {
		events = append(events, "build")
		assert.Equal(t, save_snapshot.CurrentVersion, snapshot.Version)

		return loadedScene, nil
	})

	prepared, err := game.prepareLegacyPlayerSave(gamPath)
	require.NoError(t, err)
	require.True(t, prepared.snapshot.NeedsMapUnitReconstruction())
	canonical.Continuation = prepared.snapshot.Continuation
	loadedScene.snapshot = canonical
	require.NoError(t, os.Remove(gamPath))
	require.NoError(t, os.Remove(filepath.Join(filepath.Dir(gamPath), "SAVED.OOL")))
	err = game.importPreparedLegacyPlayerSave(player_save.Slot08, "", prepared)
	require.NoError(t, err)
	require.NotEmpty(t, prepared.warnings)
	assert.Equal(t, []string{"build", "preflight", "capture", "save:slot-08", "preflight"}, events)
	require.Len(t, repository.saves, 1)
	save := repository.saves[0]
	assert.Equal(t, save_snapshot.CurrentVersion, save.envelope.Snapshot.Version)
	assert.Equal(t, "Slot 8", save.envelope.Metadata.Name)
	assert.Equal(t, playerSaveLegacyDescription, save.envelope.Metadata.Description)
	assert.Equal(t, "import-test", save.envelope.Metadata.GameVersion)
	assert.Equal(t, canonical, save.envelope.Snapshot)
	assert.Equal(t, prepared.snapshot.Continuation, save.envelope.Snapshot.Continuation)
	assertCanonicalPreparedPlayerSave(t, save.envelope.Snapshot)
	assert.Same(t, loadedScene, game.currentScene)
	assert.Equal(t, player_save.Slot08, loadedScene.slot)
	assert.Equal(t, repository.records[player_save.Slot08].Revision, loadedScene.revision)
	assert.Empty(t, repository.loads)
	assert.Equal(t, 1, loadedScene.captures)
	assert.False(t, loadedScene.provenanceAtCapture)
}

func canonicalPlayerSaveTestSnapshot() save_snapshot.SaveSnapshot {
	return withCanonicalPlayerSaveTestMapUnits(validCommandTestSnapshot())
}

func canonicalPlayerSaveTestSnapshotFrom(snapshot save_snapshot.SaveSnapshot) save_snapshot.SaveSnapshot {
	canonical := canonicalPlayerSaveTestSnapshot()
	canonical.Continuation = snapshot.Continuation

	return canonical
}

func withCanonicalPlayerSaveTestMapUnits(snapshot save_snapshot.SaveSnapshot) save_snapshot.SaveSnapshot {
	snapshot.MapUnits = []save_snapshot.MapUnitSnapshot{{
		World:                0,
		ControllerOrder:      0,
		Kind:                 "npc",
		ReferenceID:          0,
		NPCNum:               0,
		Position:             snapshot.World.Position,
		Visible:              true,
		AIOverrideProvenance: string(references.AIOverrideProvenanceNone),
	}}

	return snapshot
}

func assertCanonicalPreparedPlayerSave(t *testing.T, snapshot save_snapshot.SaveSnapshot) {
	t.Helper()
	assert.Empty(t, save_snapshot.Validate(snapshot))
	assert.False(t, snapshot.NeedsMapUnitReconstruction())
	require.NotEmpty(t, snapshot.MapUnits)
}

func playerSaveTestRecord(slot player_save.SlotID, revision string, snapshot save_snapshot.SaveSnapshot) player_save.Record {
	return player_save.Record{
		Slot: slot,
		Envelope: save_snapshot.Envelope{
			Format:          save_snapshot.EnvelopeFormat,
			EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
			Metadata: save_snapshot.Metadata{
				Name:          playerSaveSlotName(slot),
				Description:   playerSaveGameplayDescription,
				CreatedAt:     time.Date(2026, time.August, 14, 12, 0, 0, 0, time.UTC),
				GameVersion:   "test",
				PlaythroughID: testPlaythroughID,
			},
			Snapshot: snapshot,
		},
		Revision: revision,
	}
}

func writePlayerSaveLegacyBundle(t *testing.T, dungeon bool) string {
	t.Helper()
	directory := t.TempDir()
	gam := make([]byte, legacy_save.SavedGameSize)
	ool := make([]byte, legacy_save.SavedObjectsSize)
	for index := range 16 {
		base := 0x0002 + index*32
		copy(gam[base:base+9], fmt.Sprintf("C%02d", index))
		gam[base+0x09] = byte(legacy_save.GenderMale)
		gam[base+0x0a] = byte(legacy_save.ClassFighter)
		gam[base+0x0b] = byte(legacy_save.StatusGood)
		gam[base+0x0c], gam[base+0x0d], gam[base+0x0e], gam[base+0x0f] = 20, 20, 20, 10
		binary.LittleEndian.PutUint16(gam[base+0x10:], 100)
		binary.LittleEndian.PutUint16(gam[base+0x12:], 100)
		gam[base+0x16] = 1
		if index < 3 {
			gam[base+0x1f] = byte(legacy_save.PartyMapActive)
		} else {
			gam[base+0x1f] = byte(legacy_save.PartyMapUnrecruited)
		}
	}
	gam[0x0002+0x0a] = byte(legacy_save.ClassAvatar)
	gam[0x02b5] = 3
	gam[0x02d5] = 0xff
	binary.LittleEndian.PutUint16(gam[0x02ce:], 139)
	gam[0x02d7], gam[0x02d8], gam[0x02d9], gam[0x02db] = 4, 5, 8, 35
	gam[0x02e2] = 75
	gam[0x02ed], gam[0x02ef], gam[0x02f0], gam[0x02f1] = 13, 0, 15, 15
	if dungeon {
		gam[0x02ed], gam[0x02f0], gam[0x02f1] = 33, 7, 7
		gam[0x105e] = byte(legacy_save.DungeonTypeDungeon)
		const activeDungeonWandererNumberOffset = 0x06b4 + 8 + 5
		gam[activeDungeonWandererNumberOffset] = 0xff
	}
	gamPath := filepath.Join(directory, "SAVED.GAM")
	require.NoError(t, os.WriteFile(gamPath, gam, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "SAVED.OOL"), ool, 0o600))

	return gamPath
}
