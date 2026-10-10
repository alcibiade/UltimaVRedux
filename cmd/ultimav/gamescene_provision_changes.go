package main

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/mainscreen"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

type provisionChangePresentation struct {
	state      *game_state.GameState
	changes    mainscreen.ProvisionChanges
	panel      mainscreen.ProvisionChangePanel
	observedAt int64
	ammunition mainscreen.AmmunitionCounts
}

func (g *GameScene) resetProvisionChanges() {
	g.provisionChanges = provisionChangePresentation{}
	if g.gameState != nil && !g.disposed {
		g.provisionChanges.state = g.gameState
		g.provisionChanges.changes.Reset(mainscreen.SnapshotProvisions(&g.gameState.PartyState.Inventory), g.gameState.ElapsedMs)
		g.provisionChanges.observedAt = g.gameState.ElapsedMs
		g.provisionChanges.ammunition = mainscreen.SnapshotAmmunition(&g.gameState.PartyState.Inventory)
	}
}

// observeProvisionChanges brackets scene updates, including early modal returns.
// Gross stack counters preserve transactions and consumption within the same
// update; no core callbacks, transaction ordering, RNG, or queue lanes are changed.
func (g *GameScene) observeProvisionChanges() {
	if g.disposed || g.gameState == nil {
		return
	}
	if g.provisionChanges.state != g.gameState {
		g.resetProvisionChanges()
	}
	if !g.provisionChangesAboveDialogs() {
		g.provisionChanges.changes.PauseExpiry(g.gameState.ElapsedMs - g.provisionChanges.observedAt)
	}
	g.provisionChanges.observedAt = g.gameState.ElapsedMs
	g.provisionChanges.changes.Observe(mainscreen.SnapshotProvisions(&g.gameState.PartyState.Inventory), g.gameState.ElapsedMs)
	// Ammo deliberately observes final balances, not gross spent counters: an
	// atomic refill/recovery must not manufacture an intermediate shortage.
	current := mainscreen.SnapshotAmmunition(&g.gameState.PartyState.Inventory)
	for _, notice := range current.ShortageNotices(g.provisionChanges.ammunition) {
		g.ShowActionFeedback(notice, true)
	}
	g.provisionChanges.ammunition = current
	width, height := display.GetManager().GetCurrentSize()
	// GetCurrentSize already returns logical dimensions; refitting can round
	// them differently from Draw and rebuild the panel's fonts every frame.
	g.layoutProvisionChanges(image.Rect(0, 0, width, height))
}

func (g *GameScene) layoutProvisionChanges(screen image.Rectangle) {
	viewport := display.GetManager().MapPopupViewport(screen)
	if viewport.Empty() {
		viewport = screen
		viewport.Max.X = viewport.Min.X + viewport.Dx()*3/4
	}
	g.provisionChanges.panel.Update(g.provisionChanges.changes.Rows(), viewport, screen.Dx())
	bounds := g.provisionChanges.panel.Bounds()
	if bounds.Empty() {
		return
	}
	padding := max(2, viewport.Dx()/128)
	top := viewport.Min.Y + viewport.Dy()*7/10 - bounds.Dy()/2
	// Modal/merchant views and active action receipts reserve the lower map.
	if g.dialogStack.HasOpenDialog() || len(g.transientActionFeedback) > 0 {
		top = viewport.Min.Y + padding
	}
	top = max(viewport.Min.Y+padding, min(top, viewport.Max.Y-padding-bounds.Dy()))
	g.provisionChanges.panel.Place(image.Pt(viewport.Min.X+(viewport.Dx()-bounds.Dx())/2, top))
}

// provisionChangesAboveDialogs admits feedback above the active modal only when
// its measured bounds prove there is room. Merchant parents expose their shell;
// nested transaction dialogs expose their own bounds. Unknown modal layouts
// (including Dungeon View) conservatively render over the feedback instead.
func (g *GameScene) provisionChangesAboveDialogs() bool {
	modal := g.dialogStack.PeekTopModalDialog()
	if modal == nil {
		return true
	}
	var bounds image.Rectangle
	switch dialog := modal.(type) {
	case interface {
		GetShell() *widgets.MerchantDialogShell
	}:
		bounds = dialog.GetShell().GetBounds()
	case interface{ GetBounds() image.Rectangle }:
		bounds = dialog.GetBounds()
	default:
		return false
	}

	return !bounds.Empty() && !g.provisionChanges.panel.Bounds().Overlaps(bounds)
}
