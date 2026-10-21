package widgets

import (
	"image"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

type physicalQuantityKeys struct {
	held     map[ebiten.Key]bool
	keyboard *input.Keyboard
	elapsed  time.Duration
}

func newPhysicalQuantityKeys() *physicalQuantityKeys {
	keys := &physicalQuantityKeys{held: map[ebiten.Key]bool{}}
	// A gameplay repeat delay must not slow fresh horizontal taps.
	keys.keyboard = input.NewKeyboardWithKeyStateSource(250, func(key ebiten.Key) bool { return keys.held[key] })
	keys.keyboard.SetElapsedMillisecondsSource(func() int64 { return keys.elapsed.Milliseconds() })

	return keys
}

func (k *physicalQuantityKeys) tick(dialog Widget) {
	ticks := ebiten.TPS()
	if ticks <= 0 {
		ticks = ebiten.DefaultTPS
	}
	k.elapsed += time.Second / time.Duration(ticks)
	k.keyboard.ClearFrameConsumption()
	dialog.Update()
}

func (k *physicalQuantityKeys) release(dialog Widget) {
	clear(k.held)
	k.tick(dialog)
}

type quantityDialogFixture struct {
	dialog   Widget
	quantity func() int
	consume  func() (int, bool)
	maximum  int
	maxKey   ebiten.Key
	aliases  bool
}

func quantityDialogKinds() []string {
	return []string{"arms buy", "arms sell", "guild", "reagents", "tavern provisions", "loaded skiffs"}
}

func newQuantityDialogFixture(t *testing.T, kind string, keyboard *input.Keyboard) quantityDialogFixture {
	t.Helper()
	switch kind {
	case "arms buy":
		quote := testArmsShopState().BuyQuotes[0]
		quote.MaxQuantity = 8
		dialog := newTestArmsTradeDialog(keyboard, ArmsTradeIntent{Mode: ArmsShopBuy, BuyQuote: quote})
		return quantityDialogFixture{
			dialog: dialog, quantity: func() int { return int(dialog.GetQuantity()) },
			maximum: int(quote.MaxQuantity), maxKey: ebiten.KeyM, aliases: true,
			consume: func() (int, bool) {
				request, ok := dialog.ConsumePendingRequest()
				if ok {
					require.Equal(t, ArmsShopBuy, request.Mode)
					require.Equal(t, quote.Equipment, request.Buy.Equipment)
				}
				return int(request.Buy.Quantity), ok
			},
		}
	case "arms sell":
		quote := game_state.ArmsSellQuote{
			Equipment: references.Dagger, BasePrice: 50, UnitOffer: 40, Gold: 500,
			QuantityBefore: 8, MaxQuantity: 8, Sellable: true,
		}
		dialog := newTestArmsTradeDialog(keyboard, ArmsTradeIntent{Mode: ArmsShopSell, SellQuote: quote})
		return quantityDialogFixture{
			dialog: dialog, quantity: func() int { return int(dialog.GetQuantity()) },
			maximum: int(quote.MaxQuantity), maxKey: ebiten.KeyM, aliases: true,
			consume: func() (int, bool) {
				request, ok := dialog.ConsumePendingRequest()
				if ok {
					require.Equal(t, ArmsShopSell, request.Mode)
					require.Equal(t, quote.Equipment, request.Sell.Equipment)
				}
				return int(request.Sell.Quantity), ok
			},
		}
	case "guild":
		quote := testGuildShopState().Offers[2]
		dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
			Keyboard: keyboard, Quote: quote, Pitch: "Five torches for twelve gold.",
		})
		return quantityDialogFixture{
			dialog: dialog, quantity: func() int { return int(dialog.GetBundleCount()) },
			maximum: int(quote.MaxBundles), maxKey: ebiten.KeyM, aliases: true,
			consume: func() (int, bool) {
				request, ok := dialog.ConsumePendingPurchaseRequest()
				if ok {
					require.Equal(t, quote.Provision, request.Provision)
				}
				return int(request.BundleCount), ok
			},
		}
	case "reagents":
		quote := testReagentShopState().Offers[0]
		dialog := NewReagentPurchaseDialog(ReagentPurchaseDialogConfig{Keyboard: keyboard, Quote: quote})
		return quantityDialogFixture{
			dialog: dialog, quantity: func() int { return int(dialog.GetBundleCount()) },
			maximum: len(quote.ApprovalViews), maxKey: ebiten.KeyM, aliases: true,
			consume: func() (int, bool) {
				request, ok := dialog.ConsumePendingPurchaseRequest()
				if ok {
					require.Equal(t, quote.Reagent, request.Reagent)
					require.Equal(t, quote.ApprovalViews[request.BundleCount-1], dialog.GetApprovalView())
				}
				return int(request.BundleCount), ok
			},
		}
	case "tavern provisions":
		intent := testTavernProvisionIntent(t)
		dialog := NewTavernActionDialog(TavernActionDialogConfig{Keyboard: keyboard, Intent: intent})
		return quantityDialogFixture{
			dialog: dialog, quantity: dialog.GetProvisionPackCount,
			maximum: intent.Provision.MaximumUsefulPacks, maxKey: ebiten.KeyM, aliases: true,
			consume: func() (int, bool) {
				request, ok := dialog.ConsumePendingRequest()
				if ok {
					want := intent.Request
					want.Provision.PackCount = dialog.GetProvisionPackCount()
					require.Equal(t, want, request, "quantity must preserve every reviewed comparison field")
				}
				return request.Provision.PackCount, ok
			},
		}
	case "loaded skiffs":
		state := testPendingFrigateShipShopState(true)
		intent := testShipPurchaseIntent(state, state.SkiffQuote, false)
		dialog := NewShipPurchaseDialog(ShipPurchaseDialogConfig{Keyboard: keyboard, Intent: intent})
		return quantityDialogFixture{
			dialog: dialog, quantity: dialog.GetQuantity,
			maximum: len(intent.Quote.ApprovalViews), maxKey: ebiten.KeyM, aliases: true,
			consume: func() (int, bool) {
				request, ok := dialog.ConsumePendingPurchaseRequest()
				if ok {
					want := intent.Request
					approval := intent.Quote.ApprovalViews[dialog.GetQuantity()-1]
					want.Quantity, want.ReviewedTotalPrice = approval.Quantity, approval.TotalPrice
					require.Equal(t, want, request)
				}
				return request.Quantity, ok
			},
		}
	default:
		t.Fatalf("unknown quantity dialog %q", kind)
		return quantityDialogFixture{}
	}
}

func TestMerchantQuantityPhysicalRepeatTimingAndBounds(t *testing.T) {
	previousTPS := ebiten.TPS()
	ebiten.SetTPS(60)
	t.Cleanup(func() { ebiten.SetTPS(previousTPS) })
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyLeft] = true
			for range 40 {
				keys.tick(f.dialog)
			}
			require.Equal(t, 1, f.quantity(), "minimum never underflows")
			keys.release(f.dialog)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 2, f.quantity(), "fresh press is immediate")
			for range 17 {
				keys.tick(f.dialog)
				require.Equal(t, 2, f.quantity(), "before 300ms")
			}
			keys.tick(f.dialog)
			require.Equal(t, 3, f.quantity(), "300ms first repeat")
			for range 7 {
				keys.tick(f.dialog)
				require.Equal(t, 3, f.quantity(), "before the next 120ms interval")
			}
			keys.tick(f.dialog)
			require.Equal(t, 4, f.quantity())
			keys.release(f.dialog)
			if f.maxKey != ebiten.KeyMax {
				keys.held[f.maxKey] = true
				keys.tick(f.dialog)
				keys.release(f.dialog)
			}
			keys.held[ebiten.KeyRight] = true
			for range f.maximum*8 + 20 {
				keys.tick(f.dialog)
			}
			require.Equal(t, f.maximum, f.quantity(), "maximum clamps without wrapping")
			_, pending := f.consume()
			require.False(t, pending, "holding an arrow never emits a transaction")
			keys.release(f.dialog)
			keys.held[ebiten.KeyNumpadEnter] = true
			keys.tick(f.dialog)
			quantity, pending := f.consume()
			require.True(t, pending)
			require.Equal(t, f.maximum, quantity)
			for range 40 {
				keys.tick(f.dialog)
			}
			_, duplicate := f.consume()
			require.False(t, duplicate)
		})
	}
}

func TestMerchantQuantityPhysicalRapidTapsAndDirectionSwitch(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			keys.release(f.dialog)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 3, f.quantity(), "second fresh tap must not inherit gameplay delay")
			keys.held[ebiten.KeyRight] = false
			keys.held[ebiten.KeyLeft] = true
			keys.tick(f.dialog)
			require.Equal(t, 2, f.quantity(), "switching direction without an idle update is immediate")
			keys.held[ebiten.KeyLeft] = false
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 3, f.quantity())
		})
	}
}

func TestMerchantQuantityPhysicalLeftRepeatTimingAndRelease(t *testing.T) {
	previousTPS := ebiten.TPS()
	ebiten.SetTPS(100) // Exact 10ms updates also verify cadence is independent of 60 TPS.
	t.Cleanup(func() { ebiten.SetTPS(previousTPS) })
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyRight] = true
			for range f.maximum*12 + 30 {
				keys.tick(f.dialog)
			}
			require.Equal(t, f.maximum, f.quantity())
			keys.release(f.dialog)
			keys.held[ebiten.KeyLeft] = true
			keys.tick(f.dialog)
			require.Equal(t, f.maximum-1, f.quantity(), "Left takes one immediate step")
			for range 29 {
				keys.tick(f.dialog)
				require.Equal(t, f.maximum-1, f.quantity(), "no repeat before 300ms")
			}
			keys.tick(f.dialog)
			require.Equal(t, f.maximum-2, f.quantity(), "repeat at 300ms")
			for range 11 {
				keys.tick(f.dialog)
				require.Equal(t, f.maximum-2, f.quantity(), "no repeat before another 120ms")
			}
			keys.tick(f.dialog)
			require.Equal(t, f.maximum-3, f.quantity())
			keys.release(f.dialog)
			for range 60 {
				keys.tick(f.dialog)
			}
			require.Equal(t, f.maximum-3, f.quantity(), "release stops repetition")
			keys.held[ebiten.KeyLeft] = true
			keys.tick(f.dialog)
			require.Equal(t, f.maximum-4, f.quantity(), "repress starts a new immediate step")
			for range 29 {
				keys.tick(f.dialog)
				require.Equal(t, f.maximum-4, f.quantity(), "repress restarts the initial delay")
			}
			_, pending := f.consume()
			require.False(t, pending)
		})
	}
}

func TestMerchantQuantityPhysicalModifiedKeysRequireFreshPlainPress(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			for _, modifier := range []ebiten.Key{
				ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
				ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
				ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight,
				ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight,
			} {
				t.Run(modifier.String(), func(t *testing.T) {
					keys := newPhysicalQuantityKeys()
					f := newQuantityDialogFixture(t, kind, keys.keyboard)
					for _, key := range []ebiten.Key{ebiten.KeyRight, ebiten.KeyDown} {
						keys.held[key], keys.held[modifier] = true, true
						for range 60 {
							keys.tick(f.dialog)
						}
						require.Equal(t, 1, f.quantity(), "modified arrows never adjust or page quantity")
						keys.held[modifier] = false
						for range 60 {
							keys.tick(f.dialog)
						}
						require.Equal(t, 1, f.quantity(), "releasing only the modifier cannot replay the arrow")
						keys.release(f.dialog)
					}
					keys.held[ebiten.KeyRight] = true
					keys.tick(f.dialog)
					require.Equal(t, 2, f.quantity())
					keys.held[modifier] = true
					keys.tick(f.dialog)
					keys.held[modifier] = false
					for range 60 {
						keys.tick(f.dialog)
					}
					require.Equal(t, 2, f.quantity(), "adding a modifier cancels an already repeating hold")
					_, pending := f.consume()
					require.False(t, pending)
				})
			}
		})
	}
}

func TestMerchantQuantityPhysicalAliasesRemainTapOnly(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyDown] = true
			for range 60 {
				keys.tick(f.dialog)
			}
			want := 1
			if f.aliases {
				want = 2
			}
			require.Equal(t, want, f.quantity())
			keys.release(f.dialog)
			keys.held[ebiten.KeyUp] = true
			for range 60 {
				keys.tick(f.dialog)
			}
			require.Equal(t, 1, f.quantity())
		})
	}
}

func TestMerchantQuantityPhysicalConflictsAndConfirmationNeverReplay(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyLeft], keys.held[ebiten.KeyRight], keys.held[ebiten.KeyEnter] = true, true, true
			keys.tick(f.dialog)
			keys.held[ebiten.KeyLeft] = false
			for range 60 {
				keys.tick(f.dialog)
			}
			require.Equal(t, 1, f.quantity(), "a remaining conflicting key is not a fresh intent")
			keys.held[ebiten.KeyRight] = false
			keys.tick(f.dialog)
			_, pending := f.consume()
			require.False(t, pending, "Enter pressed during adjustment cannot confirm after arrows release")
			keys.release(f.dialog)
			keys.held[ebiten.KeyEnter] = true
			keys.tick(f.dialog)
			quantity, pending := f.consume()
			require.True(t, pending)
			require.Equal(t, 1, quantity)
			keys.tick(f.dialog)
			_, duplicate := f.consume()
			require.False(t, duplicate)
		})
	}
}

func TestMerchantQuantityPhysicalHeldAdjustmentCannotQueueConfirmation(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			keys.held[ebiten.KeyNumpadEnter] = true
			keys.tick(f.dialog) // A throttled hold still owns this update.
			require.Equal(t, 2, f.quantity())
			keys.held[ebiten.KeyRight] = false
			for range 40 {
				keys.tick(f.dialog)
			}
			_, pending := f.consume()
			require.False(t, pending)
			keys.release(f.dialog)
			keys.held[ebiten.KeyNumpadEnter] = true
			keys.tick(f.dialog)
			quantity, pending := f.consume()
			require.True(t, pending)
			require.Equal(t, 2, quantity)
		})
	}
}

func TestMerchantQuantityPhysicalFocusCoverAndSuspensionRequireRelease(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			var stack DialogStack
			stack.PushModalDialog(f.dialog)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 2, f.quantity())
			cover := newTestButtonListModal("Cover quantity")
			stack.PushModalDialog(cover)
			// Covered generic dialogs receive no Update calls from the scene.
			stack.MustRemoveWidgetInstance(cover)
			for range 60 {
				keys.tick(f.dialog)
			}
			require.Equal(t, 2, f.quantity())
			keys.release(f.dialog)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 3, f.quantity())
			restore := keys.keyboard.SuspendInputPolling()
			// Application focus loss also suspends the dialog rather than polling it.
			restore()
			for range 60 {
				keys.tick(f.dialog)
			}
			require.Equal(t, 3, f.quantity())
			keys.release(f.dialog)
			stack.PushModalDialog(cover)
			stack.MustRemoveWidgetInstance(cover)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 4, f.quantity(), "fresh input after focus returns is immediate")
		})
	}
}

func TestMerchantQuantityPhysicalDismissalDoesNotLeakIntoNextModal(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			keys.held[ebiten.KeyEscape] = true
			keys.tick(f.dialog)
			_, pending := f.consume()
			require.False(t, pending)
			require.False(t, keys.keyboard.IsKeyPressedAndRegisterRepeatable(ebiten.KeyRight),
				"dismissed modal's hold is suppressed even for gameplay repeat")
			next := newQuantityDialogFixture(t, kind, keys.keyboard)
			for range 60 {
				keys.tick(next.dialog)
			}
			require.Equal(t, 1, next.quantity())
			_, pending = next.consume()
			require.False(t, pending)
			keys.release(next.dialog)
			keys.held[ebiten.KeyRight] = true
			keys.tick(next.dialog)
			require.Equal(t, 2, next.quantity(), "next modal is usable after physical release")
		})
	}
}

func TestMerchantQuantityPhysicalPromptCloseDoesNotConfirm(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			var stack DialogStack
			stack.PushModalDialog(f.dialog)
			cover := newTestButtonListModal("Prompt above quantity")
			stack.PushModalDialog(cover)
			keys.held[ebiten.KeyEnter], keys.held[ebiten.KeyRight] = true, true
			stack.MustRemoveWidgetInstance(cover)
			for range 60 {
				keys.tick(f.dialog)
			}
			require.Equal(t, 1, f.quantity(), "keys first pressed while covered do not become quantity intents")
			_, pending := f.consume()
			require.False(t, pending, "the covering prompt's Enter cannot approve the returning dialog")
			keys.release(f.dialog)
			keys.held[ebiten.KeyEnter] = true
			keys.tick(f.dialog)
			_, pending = f.consume()
			require.True(t, pending)
		})
	}
}

func TestMerchantQuantityPhysicalSuspensionWithoutUpdatesRejectsNewHolds(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 2, f.quantity())
			for range 60 {
				restore := keys.keyboard.SuspendInputPolling()
				// Match the scene: no selector Update while the window lacks focus.
				// The user can release the old arrow and press new controls meanwhile.
				keys.held[ebiten.KeyRight] = false
				keys.held[ebiten.KeyLeft], keys.held[ebiten.KeyEnter] = true, true
				keys.keyboard.ClearFrameConsumption()
				restore()
			}
			keys.keyboard.ResetReleaseDebouncing()
			for range 60 {
				keys.tick(f.dialog)
			}
			require.Equal(t, 2, f.quantity(), "new holds during suspension are not fresh focused presses")
			keys.held[ebiten.KeyLeft] = false
			keys.tick(f.dialog)
			_, pending := f.consume()
			require.False(t, pending, "Enter cannot replay when the suspended arrow is released")
			keys.release(f.dialog)
			keys.held[ebiten.KeyLeft] = true
			keys.tick(f.dialog)
			require.Equal(t, 1, f.quantity(), "an actual release rearms the selector")
		})
	}
}

func TestMerchantQuantityPhysicalRemovalSuppressesGameplayHold(t *testing.T) {
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			var stack DialogStack
			stack.PushModalDialog(f.dialog)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 2, f.quantity())
			stack.MustRemoveWidgetInstance(f.dialog)
			keys.keyboard.ResetReleaseDebouncing()
			for range 60 {
				keys.keyboard.ClearFrameConsumption()
				require.False(t, keys.keyboard.IsKeyPressedAndRegisterRepeatable(ebiten.KeyRight),
					"removing the widget without another Update must suppress the hold for gameplay")
			}
			_, pending := f.consume()
			require.False(t, pending)
		})
	}
}

func TestMerchantQuantityPhysicalResizePreservesRepeat(t *testing.T) {
	previousTPS := ebiten.TPS()
	ebiten.SetTPS(60)
	t.Cleanup(func() { ebiten.SetTPS(previousTPS) })
	for _, kind := range quantityDialogKinds() {
		t.Run(kind, func(t *testing.T) {
			setMerchantShellTestResolution(t, 320, 180)
			keys := newPhysicalQuantityKeys()
			f := newQuantityDialogFixture(t, kind, keys.keyboard)
			keys.held[ebiten.KeyRight] = true
			keys.tick(f.dialog)
			require.Equal(t, 2, f.quantity())
			for range 17 {
				keys.tick(f.dialog)
			}
			setMerchantShellTestResolution(t, 1280, 720)
			bounds, ok := f.dialog.(interface{ GetBounds() image.Rectangle })
			require.True(t, ok)
			require.False(t, bounds.GetBounds().Empty()) // Rebuild layout as Draw would.
			keys.tick(f.dialog)
			require.Equal(t, 3, f.quantity(), "layout rebuild neither restarts nor loses the held-key cadence")
			_, pending := f.consume()
			require.False(t, pending)
		})
	}
}
