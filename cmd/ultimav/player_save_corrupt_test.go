package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestCorruptSlotRequiresReviewedBytesAndRemainsUnloadable(t *testing.T) {
	for _, revision := range []string{"", "reviewed"} {
		summary := player_save.SlotSummary{Slot: player_save.Slot01, Status: player_save.SlotCorrupt, Revision: revision}
		assert.False(t, playerSaveSlotView(summary, playerSaveBrowserLoad).PrimaryEnabled)
		assert.Equal(t, revision != "", playerSaveSlotView(summary, playerSaveBrowserWrite).PrimaryEnabled)
		assert.Contains(t, playerSaveDescriptionNotice(summary, testPlaythroughID), "unreadable save data")
		assert.Contains(t, playerSaveDescriptionNotice(summary, testPlaythroughID), "permanently replaced")
	}
}

func TestImportCanReplaceCorruptSlotWithRevisionAndRollbackSafety(t *testing.T) {
	for _, data := range []string{"", "truncated save {"} {
		for _, outcome := range []string{"success", "activation failure", "stale"} {
			t.Run(outcome+"/"+data, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, "slot-01.json")
				require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
				repository, err := player_save.NewRepository(root, nil)
				require.NoError(t, err)
				menu := newIntroInputTestScene()
				game := NewGame(menu)
				candidate := &playerSaveTestScene{snapshot: canonicalPlayerSaveTestSnapshot(), activatePanic: outcome == "activation failure"}
				game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
					assert.Zero(t, menu.dialogStack.Count(), "all import UI must close before candidate capture")
					return candidate, nil
				})
				prepared := &preparedLegacyPlayerSave{snapshot: candidate.snapshot, playthrough: playthroughMetadata{id: testPlaythroughID}}
				menu.openPlayerSaveBrowserWithState(game, playerSaveBrowserWrite, prepared, string(player_save.Slot01), "")
				browser, ok := menu.dialogStack.PeekTopModalDialog().(*widgets.SaveSlotBrowser)
				require.True(t, ok)
				view, selected := browser.GetSelectedView()
				require.True(t, selected)
				require.True(t, view.PrimaryEnabled)
				menu.keyboard.SimulateKeyPress(ebiten.KeyEnter)
				require.NoError(t, game.Update())
				editor, ok := menu.dialogStack.PeekTopModalDialog().(*widgets.SaveDescriptionEntryModal)
				require.True(t, ok)
				editor.SetText("b_skiff")
				if outcome == "stale" {
					data += "changed"
					require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
				}
				editor.Submit()
				if outcome == "success" {
					assert.Same(t, candidate, game.currentScene)
					assert.Zero(t, menu.dialogStack.Count())
					record, loadErr := repository.Load(player_save.Slot01)
					require.NoError(t, loadErr)
					assert.Equal(t, "b_skiff", record.Envelope.Metadata.UserDescription)
					assert.Equal(t, testPlaythroughID, record.Envelope.Metadata.PlaythroughID)
					return
				}
				assert.Same(t, menu, game.currentScene)
				after, readErr := os.ReadFile(path)
				require.NoError(t, readErr)
				assert.Equal(t, []byte(data), after)
				overlay, ok := menu.dialogStack.PeekTopModalDialog().(*widgets.OverlayDialog)
				require.True(t, ok)
				assert.Contains(t, overlay.GetMessage(), "Import to Slot 1 failed")
				overlay.Dismiss()
				assert.IsType(t, &widgets.SaveSlotBrowser{}, menu.dialogStack.PeekTopModalDialog())
				assert.Equal(t, "b_skiff", prepared.draft.text)
			})
		}
	}
}
