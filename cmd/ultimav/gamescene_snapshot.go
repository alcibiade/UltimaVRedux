package main

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

// ErrGameSceneSnapshotNotRepresentable reports uncaptured presentation or input state.
var ErrGameSceneSnapshotNotRepresentable = errors.New("game scene is not representable as a save snapshot")

// ValidateSaveSnapshotEligibility checks scene-owned state not visible to GameState.
func (g *GameScene) ValidateSaveSnapshotEligibility() error {
	if g == nil || g.gameState == nil {
		return fmt.Errorf("%w: game state is required", ErrGameSceneSnapshotNotRepresentable)
	}
	switch {
	case g.dialogStack.HasOpenDialog() || g.timedOverlayDialog != nil:
		return fmt.Errorf("%w: open UI dialog", ErrGameSceneSnapshotNotRepresentable)
	case g.secondaryKeyState != PrimaryInput:
		return fmt.Errorf("%w: pending secondary input", ErrGameSceneSnapshotNotRepresentable)
	case g.inputBlocked:
		return fmt.Errorf("%w: scene input is blocked", ErrGameSceneSnapshotNotRepresentable)
	case len(g.transientActionFeedback) != 0:
		return fmt.Errorf("%w: transient action feedback", ErrGameSceneSnapshotNotRepresentable)
	case len(g.visualEffects) != 0 || len(g.combatImpactEffects) != 0 || len(g.combatVanishEffects) != 0 || len(g.combatSignatureEffects) != 0 ||
		g.combatMovement.Active:
		return fmt.Errorf("%w: active visual or combat animation", ErrGameSceneSnapshotNotRepresentable)
	case g.autoOpenDoorState != nil && g.autoOpenDoorState.IsActive():
		return fmt.Errorf("%w: automatic door animation", ErrGameSceneSnapshotNotRepresentable)
	case g.avatarDissolve.IsActive() || g.wandererDissolve.IsActive() || len(g.activeTileDissolves) != 0:
		return fmt.Errorf("%w: active dissolve effect", ErrGameSceneSnapshotNotRepresentable)
	case g.mapFadeLevel != 0 || g.screenFlashIntensity != 0 || g.screenShakeOffsetX != 0 ||
		g.screenShakeOffsetY != 0 || g.screenInvertIntensity != 0:
		return fmt.Errorf("%w: active screen effect", ErrGameSceneSnapshotNotRepresentable)
	case g.xRayVisionActive:
		return fmt.Errorf("%w: transient vision mode", ErrGameSceneSnapshotNotRepresentable)
	case g.activeProjectile != nil:
		return fmt.Errorf("%w: active projectile", ErrGameSceneSnapshotNotRepresentable)
	case g.miniMapScene != nil:
		return fmt.Errorf("%w: active mini-map scene", ErrGameSceneSnapshotNotRepresentable)
	case g.hideMapUnits || g.hideMoongates || g.hideQuestItems || g.hideItemStacks || g.hideAmbientSounds:
		return fmt.Errorf("%w: debug presentation overrides", ErrGameSceneSnapshotNotRepresentable)
	}
	if err := g.gameState.ValidateSaveSnapshotEligibility(); err != nil {
		return fmt.Errorf("validating game state snapshot eligibility: %w", err)
	}

	return nil
}

// CaptureSaveSnapshot captures authoritative state only when the whole gameplay
// scene is at a stable, representable boundary.
func (g *GameScene) CaptureSaveSnapshot() (save_snapshot.SaveSnapshot, error) {
	if err := g.ValidateSaveSnapshotEligibility(); err != nil {
		return save_snapshot.SaveSnapshot{}, err
	}
	snapshot, err := g.gameState.CaptureSaveSnapshot()
	if err != nil {
		return save_snapshot.SaveSnapshot{}, fmt.Errorf("capturing game state snapshot: %w", err)
	}
	snapshot.Presentation = journal.Presentation{History: g.output.JournalHistory(), Visits: g.journalVisits.Clone()}
	snapshot.LastConversation = g.lastConversation.Clone()
	if issues := save_snapshot.Validate(snapshot); len(issues) != 0 {
		return save_snapshot.SaveSnapshot{}, &save_snapshot.ValidationError{Issues: issues}
	}

	return snapshot, nil
}

// ValidateSaveSnapshotCompatibility verifies references against the active
// process's loaded game data without mutating gameplay state.
func (g *GameScene) ValidateSaveSnapshotCompatibility(snapshot save_snapshot.SaveSnapshot) error {
	if g == nil || g.gameState == nil {
		return fmt.Errorf("%w: game state is required", ErrGameSceneSnapshotNotRepresentable)
	}

	if err := g.gameState.ValidateSaveSnapshotCompatibility(snapshot); err != nil {
		return fmt.Errorf("validate save snapshot compatibility: %w", err)
	}

	return nil
}
