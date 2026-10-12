package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/legacy_save"
	"github.com/bradhannah/UltimaVRedux/internal/originaldata"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	viewportWidth       = 19
	viewportHeight      = 13
	playthroughIDBytes  = 16
	importedDescription = "Imported from an original Ultima V save."
)

func newPlaythroughID() (string, error) {
	var identity [playthroughIDBytes]byte
	if _, err := rand.Read(identity[:]); err != nil {
		return "", fmt.Errorf("generate playthrough identity: %w", err)
	}

	return hex.EncodeToString(identity[:]), nil
}

func convert(
	ctx context.Context, source string, parsed options, now func() time.Time, identity func() (string, error),
) (save_snapshot.Envelope, []legacy_save.ConversionWarning, error) {
	var envelope save_snapshot.Envelope
	bundle, err := legacy_save.ReadBundleContext(ctx, source)
	if err != nil {
		return envelope, nil, fmt.Errorf("read original save: %w", err)
	}
	converted, err := legacy_save.Convert(bundle)
	if err != nil {
		return envelope, nil, fmt.Errorf("convert original save: %w", err)
	}
	snapshot, err := canonicalize(ctx, converted.Snapshot, parsed.dataDirectory)
	if err != nil {
		return envelope, nil, err
	}
	playthroughID, err := identity()
	if err != nil {
		return envelope, nil, fmt.Errorf("create import identity: %w", err)
	}
	envelope = save_snapshot.Envelope{
		Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{
			Name: importName(parsed.slot), Description: importedDescription,
			CreatedAt: now().UTC(), GameVersion: buildVersion(), DoomRunID: "",
			PlaythroughID: playthroughID, UserDescription: parsed.description,
		},
		Snapshot: snapshot,
	}

	return envelope, converted.Warnings, nil
}

func importName(slot string) string {
	if slot == "" {
		return "Imported original save"
	}
	for index, candidate := range player_save.AllSlotIDs() {
		if string(candidate) == slot {
			return fmt.Sprintf("Slot %d", index+1)
		}
	}

	panic("importName: slot must be validated before conversion")
}

func canonicalize(
	ctx context.Context, raw save_snapshot.SaveSnapshot, directory string,
) (save_snapshot.SaveSnapshot, error) {
	var empty save_snapshot.SaveSnapshot
	if err := originaldata.ValidateInstallation(ctx, directory); err != nil {
		return empty, fmt.Errorf("validate --data-dir: %w", err)
	}
	configuration, err := config.NewUltimaVConfigurationWithPath(directory)
	if err != nil {
		return empty, fmt.Errorf("configure original data: %w", err)
	}
	gameReferences, err := references.NewGameReferences(configuration)
	if err != nil {
		return empty, fmt.Errorf("load original references: %w", err)
	}
	state, err := game_state.NewGameStateFromSaveSnapshot(
		raw, configuration, gameReferences, viewportWidth, viewportHeight,
	)
	if err != nil {
		return empty, fmt.Errorf("hydrate converted save: %w", err)
	}
	snapshot, err := state.CaptureSaveSnapshot()
	if err != nil {
		return empty, fmt.Errorf("capture converted save: %w", err)
	}
	// Presentation normally belongs to the scene, which this command never creates.
	snapshot.Presentation = raw.Presentation
	snapshot.Presentation.Visits = raw.Presentation.Visits.Clone()
	snapshot.LastConversation = raw.LastConversation.Clone()
	if snapshot.MapUnits == nil || snapshot.NeedsMapUnitReconstruction() {
		return empty, fmt.Errorf("canonical map units required: %w", save_snapshot.ErrInvalidSnapshot)
	}
	if err := game_state.ValidateSaveSnapshotCompatibility(snapshot, gameReferences); err != nil {
		return empty, fmt.Errorf("validate converted save compatibility: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return empty, fmt.Errorf("finish conversion: %w", err)
	}

	return snapshot, nil
}
