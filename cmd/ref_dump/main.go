// Package main provides JSON snapshots used by developer lookup scripts.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	conversationcatalog "github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// dumpReferences writes the reference snapshots consumed by developer lookup scripts.
func dumpReferences(outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	// Create output directory if it doesn't exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	if err := removeRetiredDumpFiles(outputDir); err != nil {
		return err
	}

	// Build references from the same configuration used by the game.
	gameConfig, err := config.NewUltimaVConfiguration()
	if err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}
	gameReferences, err := references.NewGameReferences(gameConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize game references: %w", err)
	}

	conversations, err := conversationSnapshot(gameReferences.ConversationCatalog)
	if err != nil {
		return err
	}
	// Keep this set limited to snapshots with active repository consumers.
	fields := map[string]interface{}{
		"LocationReferences":  gameReferences.LocationReferences,
		"TileReferences":      tileSnapshot(gameReferences.TileReferences),
		"NPCReferences":       gameReferences.NPCReferences,
		"ConversationCatalog": conversations,
	}

	for name, field := range fields {
		filePath := filepath.Join(outputDir, name+".json")
		output, err := json.MarshalIndent(field, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode data for %s: %w", name, err)
		}
		if err := os.WriteFile(filePath, output, 0600); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}

	return nil
}

// conversationSnapshot exports the exact post-patch semantic documents as JSON
// values for npc-lookup.py. It preserves the grammar and one-based identities;
// it does not introduce another decoder or execution model.
func conversationSnapshot(catalog *conversationcatalog.Catalog) ([]map[string]any, error) {
	documents := make([]map[string]any, 0, catalog.Count())
	for _, source := range []semantic.OriginalSource{
		semantic.OriginalSourceCastle, semantic.OriginalSourceKeep,
		semantic.OriginalSourceTowne, semantic.OriginalSourceDwelling,
	} {
		for _, key := range catalog.OriginalKeys(source) {
			program, exists := catalog.Original(key.Source, key.Dialog)
			if !exists {
				panic("enumerated semantic conversation is missing from its immutable catalog")
			}
			var document map[string]any
			if err := yaml.Unmarshal(program.CanonicalYAML(), &document); err != nil {
				return nil, fmt.Errorf("decoding semantic snapshot %s: %w", program.ID(), err)
			}
			documents = append(documents, document)
		}
	}

	return documents, nil
}

// tileDumpEntry combines runtime properties with developer-owned annotations for
// map-lookup.py. Core tile definitions do not depend on the debugging package.
type tileDumpEntry struct {
	references.Tile
	debugging.TileAnnotation
}

func tileSnapshot(tiles *references.Tiles) map[indexes.SpriteIndex]tileDumpEntry {
	entries := make(map[indexes.SpriteIndex]tileDumpEntry, len(*tiles))
	for index, tile := range *tiles {
		entries[index] = tileDumpEntry{Tile: *tile, TileAnnotation: debugging.TileAtlasAnnotation(index)}
	}

	return entries
}

func removeRetiredDumpFiles(outputDir string) error {
	retired := []string{
		"DataOvl.json", "DataOvl.yaml",
		"DockReferences.json", "DockReferences.yaml",
		"EnemyReferences.json", "EnemyReferences.yaml",
		"InventoryItemReferences.json", "InventoryItemReferences.yaml",
		"LocationReferences.yaml",
		"LookReferences.json", "LookReferences.yaml",
		"NPCReferences.yaml",
		"TalkReferences.json", "TalkReferences.yaml",
		"TileReferences.yaml",
		"dump_log.txt",
		"npc_data_index.json",
	}
	for _, name := range retired {
		path := filepath.Join(outputDir, name)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove retired dump file %s: %w", path, err)
		}
	}

	return nil
}

func main() {
	var outputDir string

	rootCmd := &cobra.Command{
		Use:   "ref_dump",
		Short: "Generate JSON snapshots for developer lookup scripts",
		RunE: func(_ *cobra.Command, _ []string) error {
			return dumpReferences(outputDir)
		},
	}

	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Directory to dump the files into (required)")
	if err := rootCmd.MarkFlagRequired("output"); err != nil {
		logging.Error(logging.CategorySystem, "Error marking flag as required: %v", err)
	}

	if err := rootCmd.Execute(); err != nil {
		logging.Error(logging.CategorySystem, "Command execution failed: %v", err)
		os.Exit(1)
	}
}
