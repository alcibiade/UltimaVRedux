// Package references provides game data constants, structures, and loading for Ultima 5 Redux.
//
// This package serves as the central repository for all game reference data including:
//   - NPCs and their schedules
//   - Map locations and tiles
//   - Items, equipment, and inventory
//   - Dialog scripts and conversation data
//   - Game constants and enumerations
//
// Runtime tile and item metadata is defined in typed Go catalogs alongside its
// owning domain. Original maps, dialogue, and other binary references are loaded
// from the configured original-data directory.
//
// Key components:
//   - GameReferences: Central struct holding all loaded game data
//   - SmallLocationReference: Town and dungeon definitions
//   - NPCReference: Character data and schedules
//   - TileReferences: Sprite and terrain definitions
//   - DialogScripts: Conversation trees and responses
package references

import (
	"fmt"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/config"
	conversationcatalog "github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/files"
)

// GameReferences is every reference table loaded from the original game files.
type GameReferences struct {
	OverworldLargeMapReference  *LargeMapReference `json:"overworldLargeMapReference"  yaml:"overworldLargeMapReference"`
	UnderworldLargeMapReference *LargeMapReference `json:"underworldLargeMapReference" yaml:"underworldLargeMapReference"`

	LocationReferences      *LocationReferences      `json:"locationReferences"      yaml:"locationReferences"`
	DataOvl                 *DataOvl                 `json:"dataOvl"                 yaml:"dataOvl"`
	TileReferences          *Tiles                   `json:"tileReferences"          yaml:"tileReferences"`
	InventoryItemReferences *InventoryItemReferences `json:"inventoryItemReferences" yaml:"inventoryItemReferences"`
	LookReferences          *LookReferences          `json:"lookReferences"          yaml:"lookReferences"`
	NPCReferences           *NPCReferences           `json:"npcReferences"           yaml:"npcReferences"`
	DockReferences          *DockReferences          `json:"dockReferences"          yaml:"dockReferences"`
	EnemyReferences         *EnemyReferences         `json:"enemyReferences"         yaml:"enemyReferences"`
	MerchantReferences      *MerchantReferences      `json:"merchantReferences"      yaml:"merchantReferences"`
	SignReferences          *SignReferences          `json:"signReferences"          yaml:"signReferences"`
	KarmaReports            *KarmaReports            `json:"-"                       yaml:"-"`

	ConversationCatalog *conversationcatalog.Catalog `json:"-" yaml:"-"`
}

// NewGameReferences loads all reference tables from the configured original game files.
func NewGameReferences(gameConfig *config.UltimaVConfiguration) (*GameReferences, error) {
	gameRefs := &GameReferences{} //nolint:exhaustruct

	var err error
	gameRefs.OverworldLargeMapReference, err = NewLargeMapReference(gameConfig, OVERWORLD)
	if err != nil {
		return nil, fmt.Errorf("loading overworld large map reference: %w", err)
	}

	gameRefs.UnderworldLargeMapReference, err = NewLargeMapReference(gameConfig, UNDERWORLD)
	if err != nil {
		return nil, fmt.Errorf("loading underworld large map reference: %w", err)
	}

	gameRefs.DataOvl, err = NewDataOvl(gameConfig)
	if err != nil {
		return nil, fmt.Errorf("loading DATA.OVL references: %w", err)
	}
	gameRefs.KarmaReports, err = NewKarmaReports(gameConfig.GetFileWithFullPath(files.KarmaDAT))
	if err != nil {
		return nil, fmt.Errorf("loading karma reports: %w", err)
	}
	gameRefs.LocationReferences, err = NewSmallMapReferences(gameConfig, gameRefs.DataOvl)
	if err != nil {
		return nil, fmt.Errorf("loading location references: %w", err)
	}

	gameRefs.TileReferences = NewTileReferences()
	gameRefs.InventoryItemReferences = NewInventoryItemsReferences()
	gameRefs.LookReferences, err = NewLookReferences(gameConfig)
	if err != nil {
		return nil, fmt.Errorf("loading look references: %w", err)
	}

	gameRefs.NPCReferences, err = NewNPCReferences(gameConfig)
	if err != nil {
		return nil, fmt.Errorf("loading NPC references: %w", err)
	}
	gameRefs.DockReferences, err = NewDocks(gameConfig)
	if err != nil {
		return nil, fmt.Errorf("loading dock references: %w", err)
	}

	gameRefs.EnemyReferences = NewAllEnemyReferences(gameRefs.TileReferences)

	gameRefs.MerchantReferences, err = NewMerchantReferences(
		gameConfig.GetFileWithFullPath(files.ShoppeDat),
		NewWordDict(gameRefs.DataOvl.CompressedWords),
	)
	if err != nil {
		return nil, fmt.Errorf("loading merchant references: %w", err)
	}

	gameRefs.ConversationCatalog, err = conversationcatalog.LoadOriginal(
		gameConfig.SavedConfigData.DataFilePath,
		conversationpatchdata.Files,
	)
	if err != nil {
		return nil, fmt.Errorf("loading semantic conversation catalog: %w", err)
	}

	gameRefs.SignReferences, err = NewSignReferences(gameConfig.GetSignsDataFilePath())
	if err != nil {
		return nil, fmt.Errorf("loading sign references: %w", err)
	}

	return gameRefs, nil
}
