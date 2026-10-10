package audio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const catalogPlaybackRequestFound = "playback_request_found"

// AttachSourceEvidence reads a checkout, not game state. Evidence is static and
// explicitly weaker than observed playback. Strings, comments, declarations,
// tests, and the catalog itself cannot manufacture apparent trigger coverage.
func (catalog *Catalog) AttachSourceEvidence(root string) error {
	wanted := make(map[string]bool)
	for _, moment := range catalog.Moments {
		for _, symbol := range moment.SourceSymbols {
			wanted[symbol] = true
		}
	}
	found := make(map[string][]CatalogEvidence)
	var fingerprint strings.Builder
	for _, sourceRoot := range []string{"cmd/ultimav", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, sourceRoot), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return fmt.Errorf("visit source: %w", walkErr)
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("relative source path: %w", err)
			}
			relative = filepath.ToSlash(relative)
			//nolint:gosec // Read-only inspection of Go files in the explicitly selected developer checkout.
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read audio evidence source: %w", err)
			}
			digest := sha256.Sum256(data)
			fingerprint.WriteString(relative + "\x00" + hex.EncodeToString(digest[:]) + "\n")
			if strings.HasPrefix(relative, "internal/audio/catalog") {
				return nil
			}

			return scanCatalogSource(data, relative, wanted, found)
		})
		if err != nil {
			return fmt.Errorf("scan audio source evidence: %w", err)
		}
	}
	digest := sha256.Sum256([]byte(fingerprint.String()))
	catalog.SourceTreeSHA256 = hex.EncodeToString(digest[:])
	catalog.attachSourceReferences(found)

	return nil
}

func (catalog *Catalog) attachSourceReferences(found map[string][]CatalogEvidence) {
	for i := range catalog.Moments {
		moment := &catalog.Moments[i]
		if len(moment.SourceSymbols) == 0 {
			continue
		}
		moment.Evidence = []CatalogEvidence{}
		if moment.TriggerCoverage == catalogDeclaredOnly {
			continue // Unconfigured ambient type: registration does not install a source.
		}
		moment.TriggerCoverage = catalogNoReference
		for _, symbol := range moment.SourceSymbols {
			moment.Evidence = append(moment.Evidence, found[symbol]...)
		}
		if len(moment.Evidence) > 0 {
			moment.TriggerCoverage = "source_reference_found"
		}
		if slices.ContainsFunc(moment.Evidence, func(evidence CatalogEvidence) bool {
			return evidence.Kind == "playback_request_in_code"
		}) {
			moment.TriggerCoverage = catalogPlaybackRequestFound
		}
		slices.SortFunc(moment.Evidence, compareCatalogEvidence)
	}
}

func scanCatalogSource(
	source []byte, relative string, wanted map[string]bool, found map[string][]CatalogEvidence,
) error {
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, relative, source, 0)
	if err != nil {
		return fmt.Errorf("parse audio source %s: %w", relative, err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		playbackArguments := catalogPlaybackArguments(function.Body)
		ast.Inspect(function.Body, func(node ast.Node) bool {
			symbol, ok := node.(*ast.Ident)
			if !ok || !wanted[symbol.Name] {
				return true
			}
			kind := "static_reference_not_runtime_acceptance"
			if playbackArguments[symbol.Pos()] {
				kind = "playback_request_in_code"
			}
			found[symbol.Name] = append(found[symbol.Name], CatalogEvidence{
				Path: relative, Line: positions.Position(symbol.Pos()).Line,
				Function: function.Name.Name, Symbol: symbol.Name, Kind: kind,
			})

			return true
		})
	}

	return nil
}

// Identify literal cue arguments to known audio request/building methods. This
// deliberately does not infer indirect data flow or mark a request as executed.
func catalogPlaybackArguments(body *ast.BlockStmt) map[token.Pos]bool {
	arguments := make(map[token.Pos]bool)
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		method, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !slices.Contains([]string{"PlaySoundEffect", "WithSound", "Play"}, method.Sel.Name) {
			return true
		}
		for _, argument := range call.Args {
			switch value := argument.(type) {
			case *ast.Ident:
				arguments[value.Pos()] = true
			case *ast.SelectorExpr:
				arguments[value.Sel.Pos()] = true
			}
		}

		return true
	})

	return arguments
}

func compareCatalogEvidence(left, right CatalogEvidence) int {
	if comparison := strings.Compare(left.Path, right.Path); comparison != 0 {
		return comparison
	}
	if left.Line != right.Line {
		return left.Line - right.Line
	}

	return strings.Compare(left.Symbol, right.Symbol)
}
