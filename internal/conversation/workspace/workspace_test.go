package workspace

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	conversationpatch "github.com/bradhannah/UltimaVRedux/internal/conversation/patch"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWorkspaceLifecycle(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	result, err := initWorkspace(options, syntheticExtract)
	require.NoError(t, err)
	assert.Equal(t, 135, result.ConversationCount)
	assert.Zero(t, result.AppliedPatches)
	require.NoError(t, Check(options))

	target := "keep/010.yaml"
	editedPath := filepath.Join(options.Workspace, editedDirectory, filepath.FromSlash(target))
	baseData := readTestFile(t, filepath.Join(options.Workspace, baseDirectory, filepath.FromSlash(target)))
	editedData := bytes.Replace(baseData, []byte("Greeting 10."), []byte("Changed greeting 10."), 1)
	writeTestFile(t, editedPath, editedData)

	var diff bytes.Buffer
	diffResult, err := Diff(options, &diff)
	require.NoError(t, err)
	assert.Equal(t, 1, diffResult.ChangedConversations)
	assert.Positive(t, diffResult.DialogueLinesTouched)
	assert.NotContains(t, diff.String(), "# format:")
	assert.Contains(t, diff.String(), "--- a/keep/010.yaml\n+++ b/keep/010.yaml\n")
	assert.Contains(t, diff.String(), "-greeting: Greeting 10.\n+greeting: Changed greeting 10.\n")

	patchResult, err := Patches(options)
	require.NoError(t, err)
	assert.Equal(t, 1, patchResult.Added)
	assert.Zero(t, patchResult.Updated)
	assert.Positive(t, patchResult.DialogueLinesTouched)
	patchPath := filepath.Join(options.PatchDir, "keep", "010.patch")
	firstPatch := readTestFile(t, patchPath)
	assert.Contains(t, string(firstPatch), "# target: keep/010.yaml\n")
	assert.Contains(t, string(firstPatch), "# schema: "+semantic.SchemaVersion+"\n")
	assert.Contains(t, string(firstPatch), "# extractor: "+extractor.ExtractorVersion+"\n")
	baseDigest := sha256.Sum256(baseData)
	assert.Contains(t, string(firstPatch), "# base-sha256: "+hex.EncodeToString(baseDigest[:])+"\n")

	patchResult, err = Patches(options)
	require.NoError(t, err)
	assert.Equal(t, 1, patchResult.Unchanged)
	assert.Equal(t, firstPatch, readTestFile(t, patchPath))

	verifyResult, err := Verify(options)
	require.NoError(t, err)
	assert.Equal(t, 135, verifyResult.ConversationCount)
	assert.Equal(t, 1, verifyResult.AppliedPatches)

	updatedData := bytes.Replace(editedData, []byte("Changed greeting 10."), []byte("Updated greeting 10."), 1)
	writeTestFile(t, editedPath, updatedData)
	patchResult, err = Patches(options)
	require.NoError(t, err)
	assert.Equal(t, 1, patchResult.Updated)
	assert.NotEqual(t, firstPatch, readTestFile(t, patchPath))

	writeTestFile(t, editedPath, baseData)
	patchResult, err = Patches(options)
	require.NoError(t, err)
	assert.Equal(t, 1, patchResult.Removed)
	_, err = os.Lstat(patchPath)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = Verify(options)
	require.NoError(t, err)
}

func TestInitAppliesExistingPatchesAndRefusesOverwrite(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	base := syntheticConversationYAML("keep", 10)
	edited := bytes.Replace(base, []byte("Greeting 10."), []byte("Patched greeting 10."), 1)
	patchData, err := conversationpatch.Generate(
		"keep/010.yaml",
		semantic.SchemaVersion,
		extractor.ExtractorVersion,
		base,
		edited,
	)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(options.PatchDir, "keep"), projectDirectory))
	require.NoError(t, os.WriteFile(filepath.Join(options.PatchDir, "keep", "010.patch"), patchData, projectFile))

	result, err := initWorkspace(options, syntheticExtract)
	require.NoError(t, err)
	assert.Equal(t, 1, result.AppliedPatches)
	assert.Equal(t, edited, readTestFile(t, filepath.Join(options.Workspace, editedDirectory, "keep", "010.yaml")))

	_, err = initWorkspace(options, syntheticExtract)
	require.ErrorContains(t, err, "already exists")
}

func TestPatchRootAcceptsManagedEmbeddedPackageFile(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	require.NoError(t, os.MkdirAll(options.PatchDir, projectDirectory))
	require.NoError(t, os.WriteFile(
		filepath.Join(options.PatchDir, patchPackageFile),
		[]byte(patchPackageSource),
		projectFile,
	))

	_, err := initWorkspace(options, syntheticExtract)
	require.NoError(t, err)
}

func TestInitFailureRemovesProtectedStagingTree(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	require.NoError(t, os.MkdirAll(options.PatchDir, projectDirectory))
	require.NoError(t, os.WriteFile(filepath.Join(options.PatchDir, "unknown.txt"), []byte("unknown\n"), projectFile))

	_, err := initWorkspace(options, syntheticExtract)
	require.ErrorContains(t, err, "unrecognized file")
	_, err = os.Lstat(options.Workspace)
	require.ErrorIs(t, err, os.ErrNotExist)
	stagingTrees, err := filepath.Glob(filepath.Join(filepath.Dir(options.Workspace), ".conversations.staging-*"))
	require.NoError(t, err)
	assert.Empty(t, stagingTrees)
}

func TestCheckRejectsBaseTamperAndEditedExtras(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		tamper func(*testing.T, Options)
		text   string
	}{
		{
			name: "base tamper",
			tamper: func(t *testing.T, options Options) {
				t.Helper()
				path := filepath.Join(options.Workspace, baseDirectory, "castle", "001.yaml")
				require.NoError(t, os.Chmod(path, privateFile))
				writeTestFile(t, path, bytes.Replace(readTestFile(t, path), []byte("Greeting 1."), []byte("Tampered."), 1))
			},
			text: "protected base path",
		},
		{
			name: "edited extra",
			tamper: func(t *testing.T, options Options) {
				t.Helper()
				writeTestFile(t, filepath.Join(options.Workspace, editedDirectory, "extra.yaml"), []byte("extra\n"))
			},
			text: "unrecognized file",
		},
		{
			name: "edited symlink",
			tamper: func(t *testing.T, options Options) {
				t.Helper()
				root := filepath.Join(options.Workspace, editedDirectory)
				require.NoError(t, os.Symlink(filepath.Join(root, "castle", "001.yaml"), filepath.Join(root, "link.yaml")))
			},
			text: "symlink",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			options := testOptions(t)
			_, err := initWorkspace(options, syntheticExtract)
			require.NoError(t, err)
			testCase.tamper(t, options)
			require.ErrorContains(t, Check(options), testCase.text)
		})
	}
}

func TestCheckRejectsEditedIdentityMismatch(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	_, err := initWorkspace(options, syntheticExtract)
	require.NoError(t, err)
	path := filepath.Join(options.Workspace, editedDirectory, "keep", "010.yaml")
	data := bytes.Replace(
		readTestFile(t, path),
		[]byte("id: original.keep.010"),
		[]byte("id: original.keep.011"),
		1,
	)
	writeTestFile(t, path, data)

	require.ErrorContains(t, Check(options), `has id "original.keep.011", want "original.keep.010"`)
}

func TestPatchesRejectsUnknownFilesWithoutChangingPatchRoot(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	_, err := initWorkspace(options, syntheticExtract)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(options.PatchDir, projectDirectory))
	unknownPath := filepath.Join(options.PatchDir, "notes.txt")
	require.NoError(t, os.WriteFile(unknownPath, []byte("keep me\n"), projectFile))
	before := snapshotTree(t, options.PatchDir)

	_, err = Patches(options)
	require.ErrorContains(t, err, "unrecognized file")
	assert.Equal(t, before, snapshotTree(t, options.PatchDir))
}

func TestPatchesValidationFailureIsAtomic(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	_, err := initWorkspace(options, syntheticExtract)
	require.NoError(t, err)
	target := filepath.Join(options.Workspace, editedDirectory, "keep", "010.yaml")
	valid := bytes.Replace(readTestFile(t, target), []byte("Greeting 10."), []byte("First edit."), 1)
	writeTestFile(t, target, valid)
	_, err = Patches(options)
	require.NoError(t, err)
	before := snapshotTree(t, options.PatchDir)

	writeTestFile(t, target, []byte("not valid semantic YAML\n"))
	_, err = Patches(options)
	require.ErrorContains(t, err, "validating edited conversation")
	assert.Equal(t, before, snapshotTree(t, options.PatchDir))
}

func TestVerifyDetectsPrivateOnlyEdit(t *testing.T) {
	t.Parallel()

	options := testOptions(t)
	_, err := initWorkspace(options, syntheticExtract)
	require.NoError(t, err)
	target := filepath.Join(options.Workspace, editedDirectory, "keep", "010.yaml")
	writeTestFile(t, target, bytes.Replace(readTestFile(t, target), []byte("Greeting 10."), []byte("Private edit."), 1))

	_, err = Verify(options)
	require.ErrorContains(t, err, "not reproduced byte-for-byte")
}

func syntheticExtract(options extractor.Options) (*extractor.Result, error) {
	if err := os.Mkdir(options.Output, privateDirectory); err != nil {
		return nil, fmt.Errorf("creating synthetic extraction output: %w", err)
	}
	expected := syntheticExpectedFiles()
	for _, directory := range []string{"castle", "towne", "dwelling", "keep"} {
		if err := os.Mkdir(filepath.Join(options.Output, directory), privateDirectory); err != nil {
			return nil, fmt.Errorf("creating synthetic source directory: %w", err)
		}
	}
	schemaData := semantic.JSONSchema()
	if err := writeSyntheticFile(filepath.Join(options.Output, "conversation-v1.schema.json"), schemaData); err != nil {
		return nil, err
	}
	hasher := sha256.New()
	hashSyntheticContent(hasher, "conversation-v1.schema.json", schemaData)
	for _, target := range expected {
		parts := strings.Split(target, "/")
		dialog, err := strconvAtoi(strings.TrimSuffix(parts[1], ".yaml"))
		if err != nil {
			return nil, err
		}
		data := syntheticConversationYAML(parts[0], dialog)
		if err := writeSyntheticFile(filepath.Join(options.Output, filepath.FromSlash(target)), data); err != nil {
			return nil, err
		}
		hashSyntheticContent(hasher, target, data)
	}
	audit := &extractor.Audit{
		FormatVersion:    extractor.AuditFormat,
		ExtractorVersion: extractor.ExtractorVersion,
		Entries:          []extractor.AuditEntry{},
	}
	auditData, err := marshalSyntheticYAML(audit)
	if err != nil {
		return nil, err
	}
	if err := writeSyntheticFile(filepath.Join(options.Output, "audit.yaml"), auditData); err != nil {
		return nil, err
	}
	hashSyntheticContent(hasher, "audit.yaml", auditData)
	auditDigest := sha256.Sum256(auditData)
	manifest := &extractor.Manifest{
		FormatVersion:    extractor.ManifestFormat,
		SchemaVersion:    semantic.SchemaVersion,
		ExtractorVersion: extractor.ExtractorVersion,
		SchemaFile:       "conversation-v1.schema.json",
		AuditFile:        "audit.yaml",
		AuditSHA256:      hex.EncodeToString(auditDigest[:]),
		AuditEntryCount:  0,
		Sources:          syntheticSources(),
		ExpectedFiles:    expected,
		ActualFiles:      append([]string(nil), expected...),
		ContentSHA256:    hex.EncodeToString(hasher.Sum(nil)),
	}
	manifestData, err := marshalSyntheticYAML(manifest)
	if err != nil {
		return nil, err
	}
	if err := writeSyntheticFile(filepath.Join(options.Output, manifestFilename), manifestData); err != nil {
		return nil, err
	}

	return &extractor.Result{
		ConversationCount: len(expected),
		AuditEntryCount:   0,
		Output:            options.Output,
		ContentSHA256:     manifest.ContentSHA256,
	}, nil
}

func syntheticExpectedFiles() []string {
	counts := []struct {
		source string
		count  int
	}{
		{source: "castle", count: 40},
		{source: "towne", count: 48},
		{source: "dwelling", count: 15},
		{source: "keep", count: 32},
	}
	files := make([]string, 0, 135)
	for _, source := range counts {
		for dialog := 1; dialog <= source.count; dialog++ {
			files = append(files, fmt.Sprintf("%s/%03d.yaml", source.source, dialog))
		}
	}

	return files
}

func syntheticSources() []extractor.SourceDigest {
	files := []string{"DATA.OVL", "CASTLE.TLK", "TOWNE.TLK", "DWELLING.TLK", "KEEP.TLK"}
	sources := make([]extractor.SourceDigest, 0, len(files))
	for _, filename := range files {
		digest := sha256.Sum256([]byte(filename))
		sources = append(sources, extractor.SourceDigest{
			File:   filename,
			SHA256: hex.EncodeToString(digest[:]),
		})
	}

	return sources
}

func syntheticConversationYAML(source string, dialog int) []byte {
	conversation := &semantic.Conversation{
		Schema:      semantic.SchemaVersion,
		Protocol:    semantic.ProtocolUltima5,
		ID:          fmt.Sprintf("original.%s.%03d", source, dialog),
		Origin:      &semantic.OriginalOrigin{Source: semantic.OriginalSource(source), Dialog: dialog},
		Name:        semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: fmt.Sprintf("Keeper %d", dialog)}}}},
		Description: semantic.Body{&semantic.NarrateAction{Content: semantic.RichText{&semantic.TextPart{Text: "A synthetic keeper waits."}}}},
		Greeting:    semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: fmt.Sprintf("Greeting %d.", dialog)}}}},
		Job:         semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "I keep synthetic data."}}}},
		Farewell:    semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Farewell."}}}},
		Topics:      []semantic.Topic{},
		Questions:   map[string]semantic.Question{},
	}
	data, err := semantic.Marshal(conversation)
	if err != nil {
		panic(err)
	}

	return data
}

func testOptions(t *testing.T) Options {
	t.Helper()
	root := t.TempDir()
	privateParent := filepath.Join(root, "private")
	require.NoError(t, os.Mkdir(privateParent, privateDirectory))

	options := Options{
		DataDir:   filepath.Join(root, "unused-data"),
		Workspace: filepath.Join(privateParent, "conversations"),
		PatchDir:  filepath.Join(root, "patches"),
	}
	t.Cleanup(func() {
		_ = makeTreeWritable(filepath.Join(options.Workspace, baseDirectory))
	})

	return options
}

func makeTreeWritable(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return os.Chmod(path, privateDirectory)
		}

		return os.Chmod(path, privateFile)
	})
}

func marshalSyntheticYAML(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encoding synthetic extraction YAML: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing synthetic extraction YAML encoder: %w", err)
	}

	return buffer.Bytes(), nil
}

func writeSyntheticFile(filename string, data []byte) error {
	if err := os.WriteFile(filename, data, privateFile); err != nil {
		return fmt.Errorf("writing synthetic extraction file: %w", err)
	}

	return nil
}

func hashSyntheticContent(writer io.Writer, name string, data []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(name)))
	_, _ = writer.Write(length[:])
	_, _ = writer.Write([]byte(name))
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	_, _ = writer.Write(length[:])
	_, _ = writer.Write(data)
}

func readTestFile(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	return data
}

func writeTestFile(t *testing.T, filename string, data []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(filename, data, privateFile))
}

func snapshotTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	snapshot := make(map[string][]byte)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[relative] = data

		return nil
	}))

	return snapshot
}

func strconvAtoi(value string) (int, error) {
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("parsing synthetic dialog %q: %w", value, err)
	}

	return result, nil
}
