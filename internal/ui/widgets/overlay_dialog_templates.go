package widgets

// =====================================
// GENERIC TEMPLATES
// =====================================
//
// Templates set common appearance bundles (sizing, position, effects, alignment).
// Duration is NEVER set by templates (except AsBriefFeedback which includes 2s).
//
// For feature-specific styling, define local helper functions instead:
//
//	func (g *GameScene) buildMyFeatureOverlay(msg string) *widgets.OverlayDialog {
//	    return widgets.NewOverlayDialog(msg).
//	        WithSizing(widgets.SizingMedium).
//	        WithPosition(widgets.PositionScreenCenter).
//	        WithEffects(widgets.EffectsStandard).
//	        WithContentAlignment(widgets.ContentAlignCenter).
//	        WithTextAlignment(widgets.TextAlignCenter).
//	        WithUserDismissed(g.keyboard).
//	        Build()
//	}
//
// Usage:
//
//	// Timed feedback
//	dialog := NewOverlayDialog("Destroyed!").
//	    AsOverlayFeedback().   // Template sets appearance
//	    WithDuration(1000).    // Caller sets duration
//	    Build()
//
//	// User-dismissed confirmation
//	dialog := NewOverlayDialog("Press any key...").
//	    AsOverlayFeedback().           // Same appearance
//	    WithUserDismissed(keyboard).   // Different duration
//	    Build()

// AsOverlayFeedback configures the dialog for typical overlay feedback messages.
// Sets: Auto-fit sizing, map-center-lower position, no effects, centered alignment.
// Does NOT set duration - caller must choose WithDuration() or WithUserDismissed().
//
// Use for: Action feedback, brief status messages, spell effects, item pickup.
func (b *OverlayDialogBuilder) AsOverlayFeedback() *OverlayDialogBuilder {
	return b.
		WithSizing(SizingAutoFitDefault).
		WithPosition(PositionMapCenterLower).
		WithEffects(EffectsNone).
		WithContentAlignment(ContentAlignCenter).
		WithTextAlignment(TextAlignCenter)
}

// AsConfirmation configures the dialog for simple confirmation messages.
// Sets: Responsive sizing, map-center-lower position, no effects, centered alignment.
// Does NOT set duration - caller must choose WithDuration() or WithUserDismissed().
//
// Use for: Brief acknowledgments and simple prompts.
func (b *OverlayDialogBuilder) AsConfirmation() *OverlayDialogBuilder {
	return b.AsOverlayFeedback()
}

// AsBriefFeedback configures for quick auto-dismiss feedback messages.
// Sets: Auto-fit sizing, map-center-lower position, no effects, centered alignment.
// ALSO sets: 2 second duration (auto-dismiss).
//
// This is an exception to the "templates don't set duration" rule because
// 2-second auto-dismiss feedback is an extremely common pattern.
// Override with WithDuration() if a different duration is needed.
//
// Use for: "No effect!", "Poisoned!", brief action confirmations.
func (b *OverlayDialogBuilder) AsBriefFeedback() *OverlayDialogBuilder {
	return b.
		WithSizing(SizingAutoFitDefault).
		WithPosition(PositionMapCenterLower).
		WithEffects(EffectsNone).
		WithContentAlignment(ContentAlignCenter).
		WithTextAlignment(TextAlignCenter).
		WithDuration(2000)
}
