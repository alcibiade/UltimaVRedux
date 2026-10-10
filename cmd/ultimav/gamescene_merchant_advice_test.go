package main

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestMerchantAdviceNoticeWorkflowCleanup(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		keyboard := input.NewKeyboard(0)
		scene := &GameScene{keyboard: keyboard}
		parent := &merchantRegistryTestDialog{name: "merchant"}
		decoy := &merchantRegistryTestDialog{name: "unrelated"}
		workflow := &merchantTalkWorkflowState{}
		workflow.pushOwnedDialog(&scene.dialogStack, parent)
		notice := widgets.NewTavernAdviceNotice(keyboard, func() image.Rectangle { return image.Rect(640, 40, 1240, 680) }, "No advice.")
		workflow.tavernNotice = notice
		workflow.pushOwnedDialog(&scene.dialogStack, notice)
		if !cancel {
			require.True(t, scene.handleTavernActionDialog(game_state.MerchantSession{}, nil, false,
				tavernShopWorkflowProse{}, "", nil, workflow))
			keyboard.SimulateKeyPress(ebiten.KeyEscape)
			notice.Update()
			require.True(t, notice.IsDismissed())
			require.True(t, scene.handleTavernActionDialog(game_state.MerchantSession{}, nil, false,
				tavernShopWorkflowProse{}, "", nil, workflow))
			require.Nil(t, workflow.tavernNotice)
			require.False(t, parent.IsDismissed())
		}
		scene.dialogStack.PushModalDialog(decoy)
		workflow.dismissAndRemoveOwnedDialogs(&scene.dialogStack)
		require.True(t, parent.IsDismissed())
		require.True(t, notice.IsDismissed())
		require.False(t, decoy.IsDismissed())
		require.False(t, scene.dialogStack.RemoveWidgetInstance(notice))
		require.True(t, scene.dialogStack.RemoveWidgetInstance(decoy))
		require.False(t, scene.dialogStack.HasOpenDialog())
	}
}
