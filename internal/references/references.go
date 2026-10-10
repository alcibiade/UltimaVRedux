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

type GameReferences struct {
	OverworldLargeMapReference  *LargeMapReference `json:"overworld_large_map_reference" yaml:"overworld_large_map_reference"`
	UnderworldLargeMapReference *LargeMapReference `json:"underworld_large_map_reference" yaml:"underworld_large_map_reference"`

	LocationReferences      *LocationReferences      `json:"location_references"       yaml:"location_references"`
	DataOvl                 *DataOvl                 `json:"data_ovl"                  yaml:"data_ovl"`
	TileReferences          *Tiles                   `json:"tile_references"           yaml:"tile_references"`
	InventoryItemReferences *InventoryItemReferences `json:"inventory_item_references" yaml:"inventory_item_references"`
	LookReferences          *LookReferences          `json:"look_references"           yaml:"look_references"`
	NPCReferences           *NPCReferences           `json:"npc_references"            yaml:"npc_references"`
	DockReferences          *DockReferences          `json:"dock_references"           yaml:"dock_references"`
	EnemyReferences         *EnemyReferences         `json:"enemy_references"          yaml:"enemy_references"`
	MerchantReferences      *MerchantReferences      `json:"merchantReferences"        yaml:"merchantReferences"`
	SignReferences          *SignReferences          `json:"sign_references"           yaml:"sign_references"`
	KarmaReports            *KarmaReports            `json:"-"                         yaml:"-"`

	ConversationCatalog *conversationcatalog.Catalog `json:"-" yaml:"-"`
}

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
		gameConfig.GetFileWithFullPath(files.SHOPPE_DAT),
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
