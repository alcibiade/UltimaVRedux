package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/legacy_save"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	playerSaveGameplayDescription = "Saved during gameplay."
	playerSaveLegacyDescription   = "Imported from an original Ultima V save."
	playerSaveNewGameDescription  = "Created as a new Ultima V game."
)

var (
	// ErrPlayerSaveUnavailable indicates that player-slot persistence or native
	// scene construction has not been configured.
	ErrPlayerSaveUnavailable = errors.New("player save services are unavailable")
	// ErrPlayerSaveSceneIneligible indicates that the active or candidate scene
	// does not support the required player-save operation.
	ErrPlayerSaveSceneIneligible = errors.New("scene does not support player saves")
	// ErrNoActivePlayerSaveSlot indicates that quick-save has no loaded or saved slot.
	ErrNoActivePlayerSaveSlot = errors.New("no active player save slot")
)

// NoActivePlayerSaveSlotError is the typed quick-save failure for a gameplay
// scene that has not been loaded from or committed to a player slot.
type NoActivePlayerSaveSlotError struct{}

func (*NoActivePlayerSaveSlotError) Error() string { return ErrNoActivePlayerSaveSlot.Error() }

func (*NoActivePlayerSaveSlotError) Unwrap() error { return ErrNoActivePlayerSaveSlot }

type playerSaveRepository interface {
	List() []player_save.SlotSummary
	Load(slot player_save.SlotID) (player_save.Record, error)
	Save(slot player_save.SlotID, envelope save_snapshot.Envelope, expectedRevision string) (player_save.Record, error)
	Restore(record player_save.Record, expectedRevision string) error
	Delete(slot player_save.SlotID, expectedRevision string) error
	Latest() (player_save.Record, error)
	ValidateCompatibility(snapshot save_snapshot.SaveSnapshot) error
}

type playerSaveProvenanceOwner interface {
	playthroughMetadataOwner
	setPlayerSaveProvenance(slot player_save.SlotID, revision string)
	playerSaveProvenance() (player_save.SlotID, string, bool)
}

type playerSaveCaptureScene interface {
	playerSaveProvenanceOwner
	CaptureSaveSnapshot() (save_snapshot.SaveSnapshot, error)
}

type doomRunAssociationOwner interface {
	setDoomRunAssociation(runID string)
	doomRunAssociation() string
	doomRunAssociationForPlayerSave() string
}

type preparedLegacyPlayerSave struct {
	snapshot        save_snapshot.SaveSnapshot
	warnings        []legacy_save.ConversionWarning
	playthrough     playthroughMetadata
	draft           playerSaveDescriptionDraft
	sourceDirectory string
}

type preparedPlayerSaveCandidate struct {
	scene      Scene
	provenance playerSaveProvenanceOwner
	snapshot   save_snapshot.SaveSnapshot
}

func (g *GameScene) setPlayerSaveProvenance(slot player_save.SlotID, revision string) {
	if !slot.Valid() || strings.TrimSpace(revision) == "" {
		panic("GameScene.setPlayerSaveProvenance: valid slot and revision are required")
	}
	g.activePlayerSaveSlot = slot
	g.activePlayerSaveRevision = revision
}

func (g *GameScene) playerSaveProvenance() (player_save.SlotID, string, bool) {
	if g == nil || !g.activePlayerSaveSlot.Valid() || g.activePlayerSaveRevision == "" {
		return "", "", false
	}

	return g.activePlayerSaveSlot, g.activePlayerSaveRevision, true
}

func (g *Game) listPlayerSaveSlots() ([]player_save.SlotSummary, error) {
	if g.playerSaveRepository == nil {
		return nil, fmt.Errorf("listing player save slots: %w", ErrPlayerSaveUnavailable)
	}

	summaries := g.playerSaveRepository.List()
	if observer, ok := g.currentScene.(playerSaveCatalogObserver); ok {
		observer.applyPlayerSaveCatalog(summaries)
	}

	return summaries, nil
}

func (g *Game) loadLatestPlayerSave() error {
	if err := g.requirePlayerSaveLoading(); err != nil {
		return err
	}
	record, err := g.playerSaveRepository.Latest()
	if err != nil {
		return fmt.Errorf("loading latest player save: %w", err)
	}

	return g.loadPlayerSaveRecord(record)
}

func (g *Game) loadPlayerSaveSlot(slot player_save.SlotID, expectedRevision string) error {
	if err := g.requirePlayerSaveLoading(); err != nil {
		return err
	}
	record, err := g.playerSaveRepository.Load(slot)
	if err != nil {
		return fmt.Errorf("loading player save %s: %w", slot, err)
	}
	if record.Revision != expectedRevision {
		return fmt.Errorf("loading reviewed player save %s: %w", slot, &player_save.StaleRevisionError{
			Slot: slot, Expected: expectedRevision, Actual: record.Revision,
		})
	}

	return g.loadPlayerSaveRecord(record)
}

func (g *Game) loadPlayerSaveRecord(record player_save.Record) error {
	metadata := playthroughMetadataFromEnvelope(record.Envelope.Metadata)
	if err := metadata.validatePlayerSave(); err != nil {
		return fmt.Errorf("loading player save %s: %w", record.Slot, err)
	}
	if !record.Slot.Valid() || record.Revision == "" {
		return fmt.Errorf("loading player save record: %w: repository returned invalid provenance", ErrPlayerSaveUnavailable)
	}
	configuration := g.currentScene.GetUltimaConfiguration()
	snapshot, err := g.prepareExcerptMigration(record.Envelope)
	if err != nil {
		return fmt.Errorf("preparing player save %s: %w", record.Slot, err)
	}
	candidate, err := g.playerSaveSceneBuilder(configuration, snapshot)
	if err != nil {
		return fmt.Errorf("constructing player save %s scene: %w", record.Slot, err)
	}
	if candidate == nil {
		return fmt.Errorf("constructing player save %s scene: %w", record.Slot, ErrPlayerSaveSceneIneligible)
	}
	provenance, ok := candidate.(playerSaveProvenanceOwner)
	if !ok {
		if disposer, disposable := candidate.(sceneDisposer); disposable {
			disposer.Dispose()
		}

		return fmt.Errorf("constructing player save %s scene: %w", record.Slot, ErrPlayerSaveSceneIneligible)
	}
	provenance.setPlaythroughMetadata(metadata)
	provenance.setPlayerSaveProvenance(record.Slot, record.Revision)
	if association, associated := candidate.(doomRunAssociationOwner); associated {
		runID := ""
		if record.Envelope.Snapshot.World.Position.Location == int(references.Doom) {
			runID = record.Envelope.Metadata.DoomRunID
		}
		association.setDoomRunAssociation(runID)
	} else if record.Envelope.Metadata.DoomRunID != "" {
		if disposer, disposable := candidate.(sceneDisposer); disposable {
			disposer.Dispose()
		}

		return fmt.Errorf("constructing player save %s scene: %w", record.Slot, ErrPlayerSaveSceneIneligible)
	}
	if err = g.commitSceneReplacement(candidate); err != nil {
		return fmt.Errorf("activating player save %s scene: %w", record.Slot, err)
	}

	return nil
}

func (g *Game) savePlayerSaveSlot(
	slot player_save.SlotID,
	expectedRevision string,
	userDescription string,
) (player_save.Record, error) {
	if g.playerSaveRepository == nil {
		return player_save.Record{}, fmt.Errorf("saving player save %s: %w", slot, ErrPlayerSaveUnavailable)
	}
	if !slot.Valid() {
		return player_save.Record{}, fmt.Errorf("saving player save: %w: %q", player_save.ErrInvalidSlotID, slot)
	}
	scene, ok := g.currentScene.(playerSaveCaptureScene)
	if !ok {
		return player_save.Record{}, fmt.Errorf("saving player save %s: %w", slot, ErrPlayerSaveSceneIneligible)
	}
	metadata := scene.playthroughMetadata()
	metadata.userDescription = userDescription
	if err := metadata.validatePlayerSave(); err != nil {
		return player_save.Record{}, err
	}
	snapshot, err := scene.CaptureSaveSnapshot()
	if err != nil {
		return player_save.Record{}, fmt.Errorf("capturing player save %s: %w", slot, err)
	}
	runID := ""
	if association, associated := scene.(doomRunAssociationOwner); associated {
		runID = association.doomRunAssociationForPlayerSave()
	}
	record, err := g.persistPlayerSaveSnapshot(
		slot, expectedRevision, snapshot, playerSaveGameplayDescription, runID, metadata,
	)
	if err != nil {
		return player_save.Record{}, err
	}
	scene.setPlayerSaveProvenance(slot, record.Revision)
	scene.setPlaythroughMetadata(metadata)

	return record, nil
}

func (g *Game) quickSavePlayerSave() (player_save.Record, error) {
	scene, ok := g.currentScene.(playerSaveCaptureScene)
	if !ok {
		return player_save.Record{}, fmt.Errorf("quick-saving player save: %w", ErrPlayerSaveSceneIneligible)
	}
	slot, revision, active := scene.playerSaveProvenance()
	if !active {
		return player_save.Record{}, fmt.Errorf("quick-saving player save: %w", &NoActivePlayerSaveSlotError{})
	}

	return g.savePlayerSaveSlot(slot, revision, scene.playthroughMetadata().userDescription)
}

func (g *Game) prepareLegacyPlayerSaveContext(
	ctx context.Context,
	selectedSavedGamPath string,
) (preparedLegacyPlayerSave, error) {
	bundle, err := legacy_save.ReadBundleContext(ctx, selectedSavedGamPath)
	if err != nil {
		return preparedLegacyPlayerSave{}, fmt.Errorf("reading original save: %w", err)
	}
	conversion, err := legacy_save.Convert(bundle)
	if err != nil {
		return preparedLegacyPlayerSave{}, fmt.Errorf("converting original save: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return preparedLegacyPlayerSave{}, fmt.Errorf("preparing original save: %w", err)
	}

	metadata, err := freshPlaythroughMetadata(g.playthroughIDProvider)
	if err != nil {
		return preparedLegacyPlayerSave{}, err
	}
	if err := ctx.Err(); err != nil {
		return preparedLegacyPlayerSave{}, fmt.Errorf("preparing original save: %w", err)
	}

	return preparedLegacyPlayerSave{
		snapshot: conversion.Snapshot, warnings: conversion.Warnings, playthrough: metadata,
		draft:           playerSaveDescriptionDraft{slot: "", text: "", set: false},
		sourceDirectory: filepath.Dir(selectedSavedGamPath),
	}, nil
}

func (g *Game) importPreparedLegacyPlayerSave(
	slot player_save.SlotID,
	expectedRevision string,
	prepared preparedLegacyPlayerSave,
) error {
	return g.commitPreparedPlayerSave(
		slot,
		expectedRevision,
		prepared.snapshot,
		playerSaveLegacyDescription,
		prepared.playthrough,
	)
}

func (g *Game) commitPreparedPlayerSave(
	slot player_save.SlotID,
	expectedRevision string,
	snapshot save_snapshot.SaveSnapshot,
	description string,
	playthrough playthroughMetadata,
) error {
	if err := playthrough.validatePlayerSave(); err != nil {
		return err
	}
	if g.playerSaveRepository == nil || g.playerSaveSceneBuilder == nil {
		return fmt.Errorf("committing prepared player save into %s: %w", slot, ErrPlayerSaveUnavailable)
	}
	if err := g.playerSaveRepository.ValidateCompatibility(snapshot); err != nil {
		return fmt.Errorf("validating prepared player save for %s: %w", slot, err)
	}

	candidate, err := g.buildPreparedPlayerSaveCandidate(slot, snapshot)
	if err != nil {
		return err
	}
	candidateOwned := true
	defer func() {
		if !candidateOwned {
			return
		}
		if disposer, ok := candidate.scene.(sceneDisposer); ok {
			disposer.Dispose()
		}
	}()
	previous, occupied, err := g.reviewPreparedPlayerSaveDestination(slot, expectedRevision)
	if err != nil {
		return err
	}
	record, err := g.persistPlayerSaveSnapshot(slot, expectedRevision, candidate.snapshot, description, "", playthrough)
	if err != nil {
		return err
	}
	candidate.provenance.setPlaythroughMetadata(playthrough)
	candidate.provenance.setPlayerSaveProvenance(slot, record.Revision)
	candidateOwned = false
	if err = g.activatePreparedPlayerSave(candidate.scene, slot, record.Revision, previous, occupied); err != nil {
		return err
	}

	return nil
}

func (g *Game) reviewPreparedPlayerSaveDestination(
	slot player_save.SlotID,
	expectedRevision string,
) (player_save.Record, bool, error) {
	var emptyRecord player_save.Record
	if expectedRevision == "" {
		return emptyRecord, false, nil
	}
	record, err := g.playerSaveRepository.Load(slot)
	if err != nil && !errors.Is(err, player_save.ErrIncompatible) &&
		!errors.Is(err, save_snapshot.ErrInvalidEnvelope) && !errors.Is(err, save_snapshot.ErrInvalidSnapshot) {
		return emptyRecord, false, fmt.Errorf("reviewing player save %s before commit: %w", slot, err)
	}
	if record.Revision != expectedRevision {
		staleRevision := &player_save.StaleRevisionError{
			Slot: slot, Expected: expectedRevision, Actual: record.Revision,
		}

		return emptyRecord, false, fmt.Errorf("reviewing player save %s before commit: %w", slot, staleRevision)
	}

	return record, true, nil
}

func (g *Game) activatePreparedPlayerSave(
	scene Scene,
	slot player_save.SlotID,
	importedRevision string,
	previous player_save.Record,
	occupied bool,
) error {
	if err := g.commitSceneReplacement(scene); err != nil {
		activationErr := fmt.Errorf("activating prepared player save %s scene: %w", slot, err)
		rollbackErr := g.rollbackPreparedPlayerSaveDestination(slot, importedRevision, previous, occupied)
		if rollbackErr != nil {
			return errors.Join(
				activationErr,
				fmt.Errorf("rolling back prepared player save %s: %w", slot, rollbackErr),
			)
		}

		return activationErr
	}

	return nil
}

func (g *Game) rollbackPreparedPlayerSaveDestination(
	slot player_save.SlotID,
	importedRevision string,
	previous player_save.Record,
	occupied bool,
) error {
	if !occupied {
		if err := g.playerSaveRepository.Delete(slot, importedRevision); err != nil {
			return fmt.Errorf("deleting prepared player save %s: %w", slot, err)
		}

		return nil
	}
	if err := g.playerSaveRepository.Restore(previous, importedRevision); err != nil {
		return fmt.Errorf("restoring previous player save %s: %w", slot, err)
	}

	return nil
}

func (g *Game) buildPreparedPlayerSaveCandidate(
	slot player_save.SlotID,
	snapshot save_snapshot.SaveSnapshot,
) (preparedPlayerSaveCandidate, error) {
	var emptyCandidate preparedPlayerSaveCandidate
	candidate, err := g.playerSaveSceneBuilder(g.currentScene.GetUltimaConfiguration(), snapshot)
	if err != nil {
		return emptyCandidate, fmt.Errorf("constructing prepared player save %s scene: %w", slot, err)
	}
	if candidate == nil {
		return emptyCandidate, fmt.Errorf("constructing prepared player save %s scene: %w",
			slot, ErrPlayerSaveSceneIneligible)
	}
	owned := true
	defer func() {
		if disposer, disposable := candidate.(sceneDisposer); owned && disposable {
			disposer.Dispose()
		}
	}()
	provenance, hasProvenance := candidate.(playerSaveProvenanceOwner)
	if !hasProvenance {
		return emptyCandidate, fmt.Errorf(
			"constructing prepared player save %s scene: %w",
			slot,
			ErrPlayerSaveSceneIneligible,
		)
	}
	if preflight, ok := candidate.(sceneActivationPreflighter); ok {
		if err = preflight.PreflightActivation(); err != nil {
			return emptyCandidate, fmt.Errorf("preflighting prepared player save %s scene: %w", slot, err)
		}
	}
	capture, ok := candidate.(playerSaveCaptureScene)
	if !ok {
		return emptyCandidate, fmt.Errorf(
			"capturing prepared player save %s scene: %w",
			slot,
			ErrPlayerSaveSceneIneligible,
		)
	}
	canonicalSnapshot, err := capture.CaptureSaveSnapshot()
	if err != nil {
		return emptyCandidate, fmt.Errorf("capturing prepared player save %s scene: %w", slot, err)
	}
	if canonicalSnapshot.NeedsMapUnitReconstruction() || canonicalSnapshot.MapUnits == nil {
		return emptyCandidate, fmt.Errorf(
			"capturing prepared player save %s scene: %w: capture did not contain concrete map units",
			slot,
			ErrPlayerSaveSceneIneligible,
		)
	}

	owned = false

	return preparedPlayerSaveCandidate{
		scene:      candidate,
		provenance: provenance,
		snapshot:   canonicalSnapshot,
	}, nil
}

func (g *Game) persistPlayerSaveSnapshot(
	slot player_save.SlotID,
	expectedRevision string,
	snapshot save_snapshot.SaveSnapshot,
	description string,
	doomRunID string,
	playthrough playthroughMetadata,
) (player_save.Record, error) {
	if err := playthrough.validatePlayerSave(); err != nil {
		return player_save.Record{}, err
	}
	if !slot.Valid() {
		return player_save.Record{}, fmt.Errorf("persisting player save: %w: %q", player_save.ErrInvalidSlotID, slot)
	}
	if snapshot.Version != save_snapshot.CurrentVersion {
		return player_save.Record{}, fmt.Errorf(
			"persisting player save %s: %w",
			slot,
			&save_snapshot.UnsupportedVersionError{Version: snapshot.Version},
		)
	}
	if g.playerSaveNow == nil {
		return player_save.Record{}, fmt.Errorf(
			"persisting player save %s: %w: time source is missing",
			slot,
			ErrPlayerSaveUnavailable,
		)
	}
	envelope := save_snapshot.Envelope{
		Format:          save_snapshot.EnvelopeFormat,
		EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{
			Name:            playerSaveSlotName(slot),
			Description:     description,
			CreatedAt:       g.playerSaveNow().UTC(),
			GameVersion:     g.playerSaveGameVersion,
			DoomRunID:       doomRunID,
			PlaythroughID:   playthrough.id,
			UserDescription: playthrough.userDescription,
		},
		Snapshot: snapshot,
	}
	record, err := g.playerSaveRepository.Save(slot, envelope, expectedRevision)
	if err != nil {
		return player_save.Record{}, fmt.Errorf("persisting player save %s: %w", slot, err)
	}
	if record.Revision == "" {
		return player_save.Record{}, fmt.Errorf(
			"persisting player save %s: %w: repository returned an empty revision",
			slot,
			ErrPlayerSaveUnavailable,
		)
	}

	return record, nil
}

func (g *Game) requirePlayerSaveLoading() error {
	if g.playerSaveRepository == nil || g.playerSaveSceneBuilder == nil {
		return fmt.Errorf("loading player save: %w", ErrPlayerSaveUnavailable)
	}

	return nil
}

func playerSaveSlotName(slot player_save.SlotID) string {
	return "Slot " + playerSaveSlotNumber(slot)
}

func playerSaveSlotNumber(slot player_save.SlotID) string {
	for index, candidate := range player_save.AllSlotIDs() {
		if candidate == slot {
			return strconv.Itoa(index + 1)
		}
	}

	panic("playerSaveSlotNumber: invalid slot")
}
