package extractor

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestRoundTripIsStrictAndDeterministic(t *testing.T) {
	t.Parallel()

	manifest := validTestManifest()
	first, err := marshalManifest(manifest)
	require.NoError(t, err)
	second, err := marshalManifest(manifest)
	require.NoError(t, err)
	assert.Equal(t, first, second)

	path := filepath.Join(t.TempDir(), manifestFilename)
	require.NoError(t, os.WriteFile(path, first, filePermissions))
	loaded, err := LoadManifest(path)
	require.NoError(t, err)
	assert.Equal(t, manifest, loaded)
}

func TestLoadManifestRejectsUnknownFieldsAndExtraDocuments(t *testing.T) {
	t.Parallel()

	data, err := marshalManifest(validTestManifest())
	require.NoError(t, err)
	testCases := []struct {
		name    string
		data    []byte
		message string
	}{
		{
			name:    "unknown field",
			data:    []byte(strings.Replace(string(data), "format_version:", "unknown:", 1)),
			message: "field unknown not found",
		},
		{name: "extra document", data: append(data, []byte("---\n{}\n")...), message: "multiple YAML documents"},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), manifestFilename)
			require.NoError(t, os.WriteFile(path, testCase.data, filePermissions))
			_, err := LoadManifest(path)
			require.ErrorContains(t, err, testCase.message)
		})
	}
}

func TestManifestValidateRejectsContractViolations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mutate  func(*Manifest)
		message string
	}{
		{name: "format", mutate: func(value *Manifest) { value.FormatVersion = "v2" }, message: "format_version"},
		{name: "audit file", mutate: func(value *Manifest) { value.AuditFile = "other.yaml" }, message: "audit_file"},
		{name: "audit digest", mutate: func(value *Manifest) { value.AuditSHA256 = "ABC" }, message: "audit_sha256"},
		{name: "audit count", mutate: func(value *Manifest) { value.AuditEntryCount = -1 }, message: "audit_entry_count"},
		{name: "source order", mutate: func(value *Manifest) { value.Sources[0].File = "OTHER" }, message: "sources[0].file"},
		{name: "digest", mutate: func(value *Manifest) { value.ContentSHA256 = "ABC" }, message: "content_sha256"},
		{name: "file mismatch", mutate: func(value *Manifest) { value.ActualFiles = []string{} }, message: "exactly match"},
		{name: "unsafe file", mutate: func(value *Manifest) {
			value.ExpectedFiles[0] = "../escape.yaml"
			value.ActualFiles[0] = "../escape.yaml"
		}, message: "clean relative slash path"},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			manifest := validTestManifest()
			testCase.mutate(manifest)
			require.ErrorContains(t, manifest.Validate(), testCase.message)
		})
	}
}

func TestValidateManifestAuditRejectsDigestAndCountMismatch(t *testing.T) {
	t.Parallel()

	audit := &Audit{FormatVersion: AuditFormat, ExtractorVersion: ExtractorVersion, Entries: []AuditEntry{}}
	data, err := marshalAudit(audit)
	require.NoError(t, err)
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, auditFilename), data, filePermissions))
	digest := sha256.Sum256(data)
	manifest := validTestManifest()
	manifest.AuditSHA256 = hex.EncodeToString(digest[:])
	require.NoError(t, validateManifestAudit(directory, manifest))

	manifest.AuditSHA256 = strings.Repeat("a", sha256HexLength)
	require.ErrorContains(t, validateManifestAudit(directory, manifest), "audit_sha256")
	manifest.AuditSHA256 = hex.EncodeToString(digest[:])
	manifest.AuditEntryCount = 1
	require.ErrorContains(t, validateManifestAudit(directory, manifest), "audit_entry_count")
}

func TestPrepareDestinationRefusesUnownedAndModifiedTrees(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	unowned := filepath.Join(root, "unowned")
	require.NoError(t, os.Mkdir(unowned, directoryPermissions))
	require.ErrorContains(t, prepareDestination(unowned, true), "not owned by this extractor")

	owned := filepath.Join(root, "owned")
	require.NoError(t, os.Mkdir(owned, directoryPermissions))
	manifest := validTestManifest()
	data, err := marshalManifest(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(owned, manifestFilename), data, filePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(owned, "surprise.txt"), []byte("user data"), filePermissions))
	require.ErrorContains(t, prepareDestination(owned, true), "unrecognized path")
}

func validTestManifest() *Manifest {
	digest := strings.Repeat("a", sha256HexLength)
	sources := make([]SourceDigest, 0, len(sourceFilenames()))
	for _, filename := range sourceFilenames() {
		sources = append(sources, SourceDigest{File: filename, SHA256: digest})
	}

	return &Manifest{
		FormatVersion: ManifestFormat, SchemaVersion: semantic.SchemaVersion,
		ExtractorVersion: ExtractorVersion, SchemaFile: schemaFilename,
		AuditFile: auditFilename, AuditSHA256: digest, AuditEntryCount: 0,
		Sources: sources, ExpectedFiles: expectedConversationFiles(),
		ActualFiles: expectedConversationFiles(), ContentSHA256: digest,
	}
}
