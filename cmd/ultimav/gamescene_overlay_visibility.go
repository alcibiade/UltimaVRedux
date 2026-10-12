package main

// Overlay visibility control methods
// These methods allow selective hiding/showing of game overlays for special scenarios
// like camping, cutscenes, or other situations where overlays should not be visible

// HideQuestItems prevents quest items (shards, artifacts) from rendering
func (g *GameScene) HideQuestItems() {
	g.hideQuestItems = true
}

// ShowQuestItems allows quest items to render
func (g *GameScene) ShowQuestItems() {
	g.hideQuestItems = false
}

// HideItemStacks prevents ground items from rendering
func (g *GameScene) HideItemStacks() {
	g.hideItemStacks = true
}

// ShowItemStacks allows ground items to render
func (g *GameScene) ShowItemStacks() {
	g.hideItemStacks = false
}

// HideAmbientSounds prevents ambient sound updates (water, moongates, etc.)
func (g *GameScene) HideAmbientSounds() {
	g.hideAmbientSounds = true
	if g.audioManager != nil {
		g.audioManager.SetAmbientSuppressed(true)
	}
}

// ShowAmbientSounds allows ambient sound updates
func (g *GameScene) ShowAmbientSounds() {
	g.hideAmbientSounds = false
	if g.audioManager != nil {
		g.audioManager.SetAmbientSuppressed(false)
	}
}

// HideAllOverlays hides all overlay types
// Useful for isolated scenes like camping, cutscenes, or special maps
func (g *GameScene) HideAllOverlays() {
	g.clearProvisionFeedback()
	g.provisionChanges.suppressed = true
	g.hideMapUnits = true
	g.hideMoongates = true
	g.HideQuestItems()
	g.HideItemStacks()
	g.HideAmbientSounds()
}

// ShowAllOverlays shows all overlay types (default state)
// Call this to restore normal gameplay rendering
func (g *GameScene) ShowAllOverlays() {
	g.clearProvisionFeedback()
	g.provisionChanges.suppressed = false
	g.hideMapUnits = false
	g.hideMoongates = false
	g.ShowQuestItems()
	g.ShowItemStacks()
	g.ShowAmbientSounds()
}

// HideNonAvatarEntities hides NPCs, quest items, and item stacks but keeps moongates/sounds.
// Used for Stonegate volcano trap to clear the view when lava floods.
func (g *GameScene) HideNonAvatarEntities() {
	g.hideMapUnits = true
	g.hideQuestItems = true
	g.hideItemStacks = true
}

// ShowNonAvatarEntities restores normal entity rendering.
func (g *GameScene) ShowNonAvatarEntities() {
	g.hideMapUnits = false
	g.hideQuestItems = false
	g.hideItemStacks = false
}
