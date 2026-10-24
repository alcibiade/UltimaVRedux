package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

func TestSaveSlotBrowserImportFailureNoticeVisibleOnEmptySlot(t *testing.T) {
	const notice = "Import failed. The save data could not be validated. Check the game log for details."
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setResponsiveDialogTestResolution(t, size.X, size.Y)
			browser := newTestSaveSlotBrowser(newWidgetTestKeyboard(0), []SaveSlotView{{
				ID: "slot-01", Label: "Slot 1", Status: "Empty", PrimaryEnabled: true,
				Details: []SaveSlotDetail{{Value: "No save data."}},
			}}, SaveSlotBrowserConfig{
				Title: "Import Legacy Save", PrimaryActionLabel: "Import", FontSize: FontSizeBody,
				InitialSelectionID: "slot-01", Notice: notice,
			})
			require.Contains(t, strings.Join(strings.Fields(browser.detailOutput.GetOutputStr(false)), " "), notice)
			require.Zero(t, browser.detailPage)
			require.LessOrEqual(t, browser.detailOutput.GetPanelContentLineCount(), browser.detailLineCapacity,
				"the import error must be visible without paging")
			if size.X != 1366 && size.X != 2560 {
				return
			}
			screen := ebiten.NewImageWithOptions(browser.detailTextRect, nil)
			t.Cleanup(screen.Dispose)
			browser.drawDetails(screen)
			requireImageRegionContainsColor(t, screen, browser.detailTextRect, DialogStatusWarningColor)
		})
	}
}
