package game_state

// DebugOptionsSnapshot is a detached view of frequently used debug controls.
type DebugOptionsSnapshot struct {
	FreeMove                     bool
	TextPlaceholdersEnabled      bool
	DialogBoundingBoxesShown     bool
	UnlimitedCastingResources    bool
	BypassSpellLevelRestrictions bool
}

// CurrentDebugOptionsSnapshot returns current gameplay and renderer diagnostics.
func (g *GameState) CurrentDebugOptionsSnapshot() DebugOptionsSnapshot {
	return DebugOptionsSnapshot{
		FreeMove:                     g.DebugOptions.FreeMove,
		TextPlaceholdersEnabled:      g.DebugOptions.ShowEmptyLineDebugPlaceholders,
		DialogBoundingBoxesShown:     g.DebugOptions.DialogBoundingBoxes,
		UnlimitedCastingResources:    g.DebugOptions.UnlimitedCastingResources,
		BypassSpellLevelRestrictions: g.DebugOptions.BypassSpellLevelRestrictions,
	}
}

// SetFreeMove explicitly changes collision-free movement.
func (g *GameState) SetFreeMove(enabled bool) (before, after bool) {
	before = g.DebugOptions.FreeMove
	g.DebugOptions.FreeMove = enabled

	return before, g.DebugOptions.FreeMove
}

// SetTextPlaceholders explicitly changes empty-line diagnostic rendering.
func (g *GameState) SetTextPlaceholders(enabled bool) (before, after bool) {
	before = g.DebugOptions.ShowEmptyLineDebugPlaceholders
	g.DebugOptions.ShowEmptyLineDebugPlaceholders = enabled

	return before, g.DebugOptions.ShowEmptyLineDebugPlaceholders
}

// SetDialogBoundingBoxes explicitly changes dialog text-area diagnostic rendering.
func (g *GameState) SetDialogBoundingBoxes(enabled bool) (before, after bool) {
	before = g.DebugOptions.DialogBoundingBoxes
	g.DebugOptions.DialogBoundingBoxes = enabled

	return before, g.DebugOptions.DialogBoundingBoxes
}

// SetUnlimitedCastingResources explicitly changes spell and MP resource enforcement.
func (g *GameState) SetUnlimitedCastingResources(enabled bool) (bool, bool) {
	before := g.DebugOptions.UnlimitedCastingResources
	g.DebugOptions.UnlimitedCastingResources = enabled

	return before, g.DebugOptions.UnlimitedCastingResources
}

// SetBypassSpellLevelRestrictions explicitly changes spell-circle level enforcement.
func (g *GameState) SetBypassSpellLevelRestrictions(enabled bool) (bool, bool) {
	before := g.DebugOptions.BypassSpellLevelRestrictions
	g.DebugOptions.BypassSpellLevelRestrictions = enabled

	return before, g.DebugOptions.BypassSpellLevelRestrictions
}
