// Package catalog loads immutable semantic conversation programs.
package catalog

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	conversationpatch "github.com/bradhannah/UltimaVRedux/internal/conversation/patch"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
)

var (
	// ErrProgramNil indicates a caller requested a definition from a nil program.
	ErrProgramNil           = errors.New("conversation program is nil")
	errOriginalOrigin       = errors.New("conversation must retain an original origin")
	errConversationIdentity = errors.New("conversation identity mismatch")
	errDuplicateOriginal    = errors.New("duplicate original conversation")
	errDuplicateID          = errors.New("duplicate conversation ID")
	errUnknownPatchTarget   = errors.New("conversation patch targets unknown document")
	errIncompatiblePatch    = errors.New("conversation patch uses incompatible metadata")
	errPatchPathMismatch    = errors.New("conversation patch path does not match target")
	errDuplicatePatchTarget = errors.New("duplicate conversation patch target")
)

// OriginalKey identifies one one-based original TLK conversation.
type OriginalKey struct {
	Source semantic.OriginalSource
	Dialog int
}

// Program is an immutable validated semantic conversation document.
type Program struct {
	id      string
	data    []byte
	patched bool
}

// CanonicalYAML returns an independent copy of the exact semantic document
// loaded by the runtime after project patches are applied.
func (program *Program) CanonicalYAML() []byte {
	if program == nil {
		return nil
	}

	return append([]byte(nil), program.data...)
}

// Patched reports whether an embedded project patch was applied to this program.
func (program *Program) Patched() bool {
	return program != nil && program.patched
}

// ID returns the stable conversation ID.
func (program *Program) ID() string {
	if program == nil {
		return ""
	}

	return program.id
}

// Definition returns a detached validated semantic definition for one session.
func (program *Program) Definition() (*semantic.Conversation, error) {
	if program == nil {
		return nil, ErrProgramNil
	}
	conversation, err := semantic.Load(program.id, strings.NewReader(string(program.data)))
	if err != nil {
		return nil, fmt.Errorf("loading catalog program %q: %w", program.id, err)
	}

	return conversation, nil
}

// Catalog is an immutable index of semantic conversations.
type Catalog struct {
	originals map[OriginalKey]*Program
	byID      map[string]*Program
}

// New builds an immutable catalog from semantic definitions. Original-origin
// definitions are also indexed by source and dialog.
func New(definitions ...*semantic.Conversation) (*Catalog, error) {
	catalog := &Catalog{
		originals: make(map[OriginalKey]*Program, len(definitions)),
		byID:      make(map[string]*Program, len(definitions)),
	}
	for _, definition := range definitions {
		data, err := semantic.Marshal(definition)
		if err != nil {
			return nil, fmt.Errorf("serializing semantic conversation: %w", err)
		}
		if _, exists := catalog.byID[definition.ID]; exists {
			return nil, fmt.Errorf("%w: %q", errDuplicateID, definition.ID)
		}
		program := &Program{id: definition.ID, data: data, patched: false}
		catalog.byID[definition.ID] = program
		origin, original := definition.Origin.(*semantic.OriginalOrigin)
		if !original {
			continue
		}
		key := OriginalKey{Source: origin.Source, Dialog: origin.Dialog}
		if _, exists := catalog.originals[key]; exists {
			return nil, fmt.Errorf("%w: %s/%d", errDuplicateOriginal, key.Source, key.Dialog)
		}
		catalog.originals[key] = program
	}

	return catalog, nil
}

// LoadOriginal imports original data in memory and applies exact project patches.
func LoadOriginal(dataDir string, patches fs.FS) (*Catalog, error) {
	imported, err := extractor.Import(dataDir)
	if err != nil {
		return nil, fmt.Errorf("importing original conversations: %w", err)
	}
	patchData, err := loadPatches(patches)
	if err != nil {
		return nil, err
	}

	catalog := &Catalog{
		originals: make(map[OriginalKey]*Program, len(imported.Conversations)),
		byID:      make(map[string]*Program, len(imported.Conversations)),
	}
	knownTargets := make(map[string]struct{}, len(imported.Conversations))
	for _, importedConversation := range imported.Conversations {
		knownTargets[importedConversation.Path] = struct{}{}
		if addErr := catalog.addOriginal(importedConversation, patchData[importedConversation.Path]); addErr != nil {
			return nil, addErr
		}
	}
	for target := range patchData {
		if _, exists := knownTargets[target]; !exists {
			return nil, fmt.Errorf("%w: %q", errUnknownPatchTarget, target)
		}
	}

	return catalog, nil
}

func (catalog *Catalog) addOriginal(imported extractor.ImportedConversation, patchData []byte) error {
	data := imported.Data
	var err error
	if patchData != nil {
		data, err = conversationpatch.Apply(data, patchData, conversationpatch.Expectation{
			Target: imported.Path, SchemaVersion: semantic.SchemaVersion,
			ExtractorVersion: extractor.ExtractorVersion,
		})
		if err != nil {
			return fmt.Errorf("applying conversation patch for %q: %w", imported.Path, err)
		}
	}
	conversation, err := semantic.Load(imported.Path, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("loading semantic conversation %q: %w", imported.Path, err)
	}
	origin, ok := conversation.Origin.(*semantic.OriginalOrigin)
	if !ok {
		return fmt.Errorf("%w: %q", errOriginalOrigin, imported.Path)
	}
	wantPath := fmt.Sprintf("%s/%03d.yaml", origin.Source, origin.Dialog)
	wantID := fmt.Sprintf("original.%s.%03d", origin.Source, origin.Dialog)
	if imported.Path != wantPath || conversation.ID != wantID {
		return fmt.Errorf(
			"%w: path=%q id=%q origin=%s/%d",
			errConversationIdentity,
			imported.Path,
			conversation.ID,
			origin.Source,
			origin.Dialog,
		)
	}
	key := OriginalKey{Source: origin.Source, Dialog: origin.Dialog}
	if _, exists := catalog.originals[key]; exists {
		return fmt.Errorf("%w: %s/%d", errDuplicateOriginal, key.Source, key.Dialog)
	}
	if _, exists := catalog.byID[conversation.ID]; exists {
		return fmt.Errorf("%w: %q", errDuplicateID, conversation.ID)
	}
	program := &Program{id: conversation.ID, data: append([]byte(nil), data...), patched: patchData != nil}
	catalog.originals[key] = program
	catalog.byID[conversation.ID] = program

	return nil
}

// Original returns one original conversation program.
func (catalog *Catalog) Original(source semantic.OriginalSource, dialog int) (*Program, bool) {
	if catalog == nil {
		return nil, false
	}
	program, exists := catalog.originals[OriginalKey{Source: source, Dialog: dialog}]

	return program, exists
}

// OriginalKeys returns detached original identities for a source in dialog order.
// It uses actual catalog keys, including sparse synthetic catalogs, rather than
// assuming original-file counts or exposing the catalog's backing maps.
func (catalog *Catalog) OriginalKeys(source semantic.OriginalSource) []OriginalKey {
	keys := make([]OriginalKey, 0)
	if catalog == nil {
		return keys
	}
	for key := range catalog.originals {
		if key.Source == source {
			keys = append(keys, key)
		}
	}
	slices.SortFunc(keys, func(a, b OriginalKey) int { return cmp.Compare(a.Dialog, b.Dialog) })

	return keys
}

// Count returns the number of loaded programs.
func (catalog *Catalog) Count() int {
	if catalog == nil {
		return 0
	}

	return len(catalog.byID)
}

func loadPatches(patches fs.FS) (map[string][]byte, error) {
	loaded := make(map[string][]byte)
	if patches == nil {
		return loaded, nil
	}
	err := fs.WalkDir(patches, ".", func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path.Ext(filename) != ".patch" {
			return nil
		}

		return loadPatch(patches, filename, loaded)
	})
	if err != nil {
		return nil, fmt.Errorf("loading embedded conversation patches: %w", err)
	}

	return loaded, nil
}

func loadPatch(patches fs.FS, filename string, loaded map[string][]byte) error {
	data, err := fs.ReadFile(patches, filename)
	if err != nil {
		return fmt.Errorf("reading embedded conversation patch %q: %w", filename, err)
	}
	metadata, err := conversationpatch.ParseMetadata(data)
	if err != nil {
		return fmt.Errorf("parsing embedded conversation patch %q: %w", filename, err)
	}
	if metadata.SchemaVersion != semantic.SchemaVersion || metadata.ExtractorVersion != extractor.ExtractorVersion {
		return fmt.Errorf("%w: %q", errIncompatiblePatch, filename)
	}
	wantFilename := strings.TrimSuffix(metadata.Target, ".yaml") + ".patch"
	if filename != wantFilename {
		return fmt.Errorf("%w: path=%q target=%q", errPatchPathMismatch, filename, metadata.Target)
	}
	if _, exists := loaded[metadata.Target]; exists {
		return fmt.Errorf("%w: %q", errDuplicatePatchTarget, metadata.Target)
	}
	loaded[metadata.Target] = data

	return nil
}
