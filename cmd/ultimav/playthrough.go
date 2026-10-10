package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

// Checkpoint metadata is independent of both player-slot authority and Doom runs.
type playthroughMetadata struct {
	id              string
	userDescription string
}

type playthroughMetadataOwner interface {
	setPlaythroughMetadata(metadata playthroughMetadata)
	playthroughMetadata() playthroughMetadata
}

func newPlaythroughID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generating playthrough identity: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}

func freshPlaythroughMetadata(provider func() (string, error)) (playthroughMetadata, error) {
	if provider == nil {
		panic("freshPlaythroughMetadata: playthrough identity provider is required")
	}
	id, err := provider()
	if err != nil {
		return playthroughMetadata{}, fmt.Errorf("preparing playthrough: %w", err)
	}
	metadata := playthroughMetadata{id: id, userDescription: ""}
	if err = metadata.validatePlayerSave(); err != nil {
		return playthroughMetadata{}, err
	}

	return metadata, nil
}

func (metadata playthroughMetadata) validatePlayerSave() error {
	if !save_snapshot.ValidPlaythroughID(metadata.id) {
		return fmt.Errorf("player save requires a valid playthrough identity: %w", save_snapshot.ErrInvalidEnvelope)
	}
	if err := save_snapshot.ValidateUserDescription(metadata.userDescription); err != nil {
		return fmt.Errorf("invalid player description: %w", err)
	}

	return nil
}

func playthroughMetadataFromEnvelope(metadata save_snapshot.Metadata) playthroughMetadata {
	return playthroughMetadata{id: metadata.PlaythroughID, userDescription: metadata.UserDescription}
}

func (g *GameScene) setPlaythroughMetadata(metadata playthroughMetadata) {
	g.activePlaythrough = metadata
}

func (g *GameScene) playthroughMetadata() playthroughMetadata {
	return g.activePlaythrough
}

// This transfers reviewed metadata, never the departing scene's slot authority.
// The caller still owns disposal until it hands the candidate to scene replacement.
func transferScenePlaythrough(candidate Scene, metadata playthroughMetadata) error {
	owner, ok := candidate.(playthroughMetadataOwner)
	if !ok {
		return ErrPlayerSaveSceneIneligible
	}
	if provenance, ok := candidate.(playerSaveProvenanceOwner); ok {
		if _, _, active := provenance.playerSaveProvenance(); active {
			return fmt.Errorf("checkpoint candidate must not own a player slot: %w", ErrPlayerSaveSceneIneligible)
		}
	}
	if metadata.id != "" && !save_snapshot.ValidPlaythroughID(metadata.id) {
		return fmt.Errorf("invalid checkpoint playthrough identity: %w", save_snapshot.ErrInvalidEnvelope)
	}
	if err := save_snapshot.ValidateUserDescription(metadata.userDescription); err != nil {
		return fmt.Errorf("invalid checkpoint player description: %w", err)
	}
	owner.setPlaythroughMetadata(metadata)

	return nil
}
