package main

import "github.com/bradhannah/UltimaVRedux/internal/logging"

// ScenarioCommandsReady reports whether a scripted scenario may dispatch its
// next command without overlapping interactive or scripted gameplay work.
func (g *GameScene) ScenarioCommandsReady() bool {
	return !g.gameState.IsInCombat() &&
		!g.dialogStack.HasOpenDialog() &&
		!g.gameState.EventQueueManager.HasActiveWorkflows() &&
		g.miniMapScene == nil &&
		g.gameState.ShrineScene == nil &&
		g.gameState.CanAcceptGameplayCommands()
}

// ScenarioRunsAllowed rejects runs that can never make progress because the
// production finale has taken permanent gameplay ownership.
func (g *GameScene) ScenarioRunsAllowed() bool {
	return g.gameState.CanAcceptGameplayCommands()
}

// PrepareForSceneReplacement stops live playback before the candidate is activated.
func (g *GameScene) PrepareForSceneReplacement() {
	g.clearKnowledgeNotices()
	g.clearBattleSummaries()
	g.resetProvisionChanges()
	g.surfaceCamera = surfaceCameraProjection{}
	g.terrainDither.enabled = false
	g.clearTimedOverlay()
	g.clearTransientActionFeedback()
	g.clearTransientIcons()
	if g.audioManager == nil {
		return
	}
	if err := g.audioManager.StopBackgroundMusic(); err != nil {
		logging.Warn(logging.CategoryAudio, "Failed to stop music before scene replacement: %v", err)
	}
}

// Dispose releases scene-owned resources. Repeated calls are harmless.
func (g *GameScene) Dispose() {
	g.clearSaveQuestion()
	if g.disposed {
		return
	}
	g.disposed = true
	g.closeRunicReference()
	g.sky = skyPresentation{}
	if g.borders.skyBacking != nil {
		g.borders.skyBacking.Deallocate()
		g.borders.skyBacking = nil
	}
	g.resetProvisionChanges()
	g.closeJournalSession(g.journalSession, false)
	g.closeAtlasSession(true)
	g.clearKnowledgeNotices()
	for _, widget := range g.dialogStack.Dialogs() {
		if dialog, ok := widget.(*LinearTalkDialog); ok {
			dialog.Close()
		}
	}
	g.terrainDither.image.Dispose()
	g.outerTerrainDither.Dispose()
	g.terrainDither = terrainDitherFrame{}
	g.surfaceCamera = surfaceCameraProjection{}
	if g.surfaceTerrainImage != nil {
		g.surfaceTerrainImage.Deallocate()
		g.surfaceTerrainImage = nil
	}
	g.active = false
	if g.gameState != nil && g.gameState.EventQueueManager != nil {
		if g.gameState.IsCombatPartySpellTargeting() {
			g.gameState.CancelCombatSpellTargeting()
			g.secondaryKeyState = PrimaryInput
		}
		g.gameState.EventQueueManager.Shutdown()
	}
	g.clearTimedOverlay()
	g.clearTransientActionFeedback()
	g.clearTransientIcons()
	g.clearShadowlordPresentationForSceneTransition()
	g.clearBattleSummaries()
	g.clearLordBritishFinalePresentation(false)
	g.avatarDissolve.Cleanup()
	g.wandererDissolve.Cleanup()
	g.clearCombatVanishEffects()
	g.clearCombatSignatureEffects()
	g.combatAllegiance = combatAllegianceState{}
	g.clearCombatInvocationEffects()
	g.clearCombatTeleportEffect()
	for index := range g.dungeonSkeletonCrumbles {
		g.dungeonSkeletonCrumbles[index].cleanup()
	}
	g.dungeonSkeletonCrumbles = nil
	for index := range g.dungeonPitReveals {
		g.dungeonPitReveals[index].cleanup()
	}
	g.dungeonPitReveals = nil
	g.StopAllTileDissolves()
	g.visualEffects = g.visualEffects[:0]
	g.screenFlashIntensity = 0
	g.screenShakeOffsetX = 0
	g.screenShakeOffsetY = 0
	g.screenInvertIntensity = 0
	g.inputBlocked = false
	if g.characterSummary != nil {
		g.characterSummary.Dispose()
	}
	if g.audioManager == nil {
		return
	}
	if err := g.audioManager.Close(); err != nil {
		logging.Warn(logging.CategoryAudio, "Failed to close audio during scene disposal: %v", err)
	}
	g.audioManager = nil
}

func (g *GameScene) clearTransientIcons() {
	if g.activeEffectsOverlay != nil {
		g.activeEffectsOverlay.Clear()
	}
}
