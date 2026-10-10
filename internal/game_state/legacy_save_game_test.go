package game_state

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestReadLegacySaveFile(t *testing.T) {
	t.Parallel()

	want := bytes.Repeat([]byte{0x5a}, savedGamFileSize)
	filePath := filepath.Join(t.TempDir(), "SAVED.GAM")
	if err := os.WriteFile(filePath, want, 0o600); err != nil {
		t.Fatalf("writing legacy save fixture: %v", err)
	}

	got, err := ReadLegacySaveFile(filePath)
	if err != nil {
		t.Fatalf("ReadLegacySaveFile() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("ReadLegacySaveFile() returned different save data")
	}
}

func TestReadLegacySaveFileRejectsInvalidSizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
	}{
		{name: "short", size: savedGamFileSize - 1},
		{name: "long", size: savedGamFileSize + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filePath := filepath.Join(t.TempDir(), "SAVED.GAM")
			if err := os.WriteFile(filePath, make([]byte, tt.size), 0o600); err != nil {
				t.Fatalf("writing legacy save fixture: %v", err)
			}

			_, err := ReadLegacySaveFile(filePath)
			if !errors.Is(err, ErrInvalidSaveGameSize) {
				t.Fatalf("ReadLegacySaveFile() error = %v, want ErrInvalidSaveGameSize", err)
			}
		})
	}
}

func TestLoadLegacySaveGameFromBytesRejectsInvalidSize(t *testing.T) {
	t.Parallel()

	gameState := &GameState{}
	err := gameState.LoadLegacySaveGameFromBytes(make([]byte, savedGamFileSize-1))
	if !errors.Is(err, ErrInvalidSaveGameSize) {
		t.Fatalf("LoadLegacySaveGameFromBytes() error = %v, want ErrInvalidSaveGameSize", err)
	}
}

func TestLegacySaveConstructorsReturnRecoverableErrors(t *testing.T) {
	t.Parallel()

	_, err := NewGameStateFromLegacyTemplateBytes(make([]byte, savedGamFileSize-1), nil, nil, 1, 1)
	if !errors.Is(err, ErrInvalidSaveGameSize) {
		t.Fatalf("NewGameStateFromLegacySaveBytes() error = %v, want ErrInvalidSaveGameSize", err)
	}

	_, err = newGameStateFromLegacyTemplateFile(filepath.Join(t.TempDir(), "missing.gam"), nil, nil, 1, 1)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewGameStateFromLegacySaveFile() error = %v, want os.ErrNotExist", err)
	}
}

func TestNewGameStateFromLegacySaveBytesHydratesCurrentSmallMap(t *testing.T) {
	gameReferences := getBaseGameReferences()
	if gameReferences == nil || cachedIntegrationCfg == nil {
		t.Skip("integration test requires original Ultima V game data")
	}

	rawSaveData := make([]byte, savedGamFileSize)
	rawSaveData[0x2ED] = byte(references.IolosHut)
	rawSaveData[0x2EF] = 0
	rawSaveData[0x2F0] = 16
	rawSaveData[0x2F1] = 16
	rawSaveData[0x03B1] = 25
	rawSaveData[0x0215] = 0x81
	copy(rawSaveData[0x0322:0x0322+references.ShadowlordCount], []byte{1, 0xff, 8})
	rawSaveData[0x0325] = byte(references.Nosfentor)

	gameState := mustNewGameStateFromLegacySaveBytes(
		rawSaveData,
		cachedIntegrationCfg,
		gameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)

	layeredMap := gameState.GetLayeredMapByCurrentLocation()
	if layeredMap == nil {
		t.Fatal("legacy small-map save did not initialize its current layered map")
	}
	if _, ok := gameState.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap); !ok {
		t.Fatalf("CurrentNPCAIController = %T, want *ai.NPCAIControllerSmallMap", gameState.CurrentNPCAIController)
	}
	if gameState.drunk != 0 || gameState.smashed != 25 {
		t.Fatalf("legacy intoxication = drunk %d smashed %d, want 0/25", gameState.drunk, gameState.smashed)
	}
	if !gameState.IsHMSCapeActivated() || !gameState.PartyState.Inventory.QuestItems.HasSome(references.HMSCape) {
		t.Fatal("legacy rigged HMS Cape high bit did not import ownership and enabled preference")
	}
	if gameState.ShadowlordState.Alive != [references.ShadowlordCount]bool{true, false, true} || gameState.ShadowlordState.TownAssignments != [references.ShadowlordCount]references.Location{references.Moonglow, 0, references.NewMagincia} || gameState.ShadowlordState.LastCalled != references.Nosfentor {
		t.Fatalf("legacy Shadowlord state = %#v", gameState.ShadowlordState)
	}
	gameState.MapState.PlayerLocation.Location = references.BritanniaUnderworld
	gameState.MapState.PlayerLocation.Floor = references.Basement
	if gameState.IsShardVisible(references.Position{X: 130, Y: 65}, references.ShardOfHatred) {
		t.Fatal("legacy-defeated Astaroth rematerialized the Shard of Hatred")
	}
	if !gameState.IsShardVisible(references.Position{X: 192, Y: 80}, references.ShardOfFalsehood) {
		t.Fatal("legacy living Faulinei did not retain the available Shard of Falsehood")
	}
	gameState.MapState.PlayerLocation.Location = references.IolosHut
	gameState.MapState.PlayerLocation.Floor = references.MainFloor
	gameState.drunk, gameState.smashed = 7, 9
	if err := gameState.LoadLegacySaveGameFromBytes(rawSaveData); err != nil {
		t.Fatalf("reloading legacy save: %v", err)
	}
	if gameState.drunk != 0 || gameState.smashed != 25 {
		t.Fatalf("reloaded legacy intoxication = drunk %d smashed %d, want 0/25", gameState.drunk, gameState.smashed)
	}
	copy(rawSaveData[0x0322:0x0322+references.ShadowlordCount], []byte{1, 1, 8})
	if err := gameState.LoadLegacySaveGameFromBytes(rawSaveData); err != nil {
		t.Fatalf("reloading duplicate legacy Shadowlord assignments: %v", err)
	}
	if gameState.ShadowlordState.TownAssignments != [references.ShadowlordCount]references.Location{references.Moonglow, 0, references.NewMagincia} {
		t.Fatalf("normalized legacy Shadowlord assignments = %#v", gameState.ShadowlordState.TownAssignments)
	}

	layeredMap.RecalculateVisibleTilesWithCustomLights(
		gameState.MapState.PlayerLocation.Position,
		&gameState.MapState.Lighting,
		nil,
		nil,
		nil,
		map_state.PlayerLightAmbient,
	)
}

func TestLegacySaveHydratesContinuationBytesAndClearsReduxOnlyState(t *testing.T) {
	state := getBritain2GameState()
	if state == nil {
		t.Skip("integration test requires original Ultima V game data")
	}
	raw := append([]byte(nil), state.RawSave[:]...)
	raw[0x02d5] = 0
	raw[0x02e5] = 254
	state.isCriminal = true
	state.hmsCapeAutomaticWaitActorSkip = true

	if err := state.LoadLegacySaveGameFromBytes(raw); err != nil {
		t.Fatalf("LoadLegacySaveGameFromBytes() error = %v", err)
	}
	if state.GetActivePlayerIndex() != 0 || state.beggarKarmaCounter != 254 {
		t.Fatalf("legacy continuation = active %d counter %d, want 0/254", state.GetActivePlayerIndex(), state.beggarKarmaCounter)
	}
	if state.IsCriminal() || state.hmsCapeAutomaticWaitActorSkip {
		t.Fatal("legacy load retained Redux-only continuation state")
	}

	raw[0x02d5] = 0xff
	if err := state.LoadLegacySaveGameFromBytes(raw); err != nil {
		t.Fatalf("reloading legacy save: %v", err)
	}
	if state.GetActivePlayerIndex() != -1 {
		t.Fatalf("legacy no-player marker restored index %d, want -1", state.GetActivePlayerIndex())
	}

	raw[0x02d5] = 6
	if err := state.LoadLegacySaveGameFromBytes(raw); err != nil {
		t.Fatalf("reloading invalid legacy selection: %v", err)
	}
	if state.GetActivePlayerIndex() != -1 {
		t.Fatalf("invalid legacy selection restored index %d, want -1", state.GetActivePlayerIndex())
	}
}
