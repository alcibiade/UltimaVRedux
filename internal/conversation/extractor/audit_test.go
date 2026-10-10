package extractor

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditRoundTripIsStrictAndDeterministic(t *testing.T) {
	t.Parallel()

	audit := &Audit{
		FormatVersion: AuditFormat, ExtractorVersion: ExtractorVersion,
		Entries: []AuditEntry{validAuditEntry()},
	}
	first, err := marshalAudit(audit)
	require.NoError(t, err)
	second, err := marshalAudit(audit)
	require.NoError(t, err)
	assert.Equal(t, first, second)

	path := filepath.Join(t.TempDir(), auditFilename)
	require.NoError(t, os.WriteFile(path, first, filePermissions))
	loaded, err := LoadAudit(path)
	require.NoError(t, err)
	assert.Equal(t, audit, loaded)
}

func TestAuditValidateRejectsUnknownAndNonCanonicalEntries(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mutate  func(*Audit)
		message string
	}{
		{name: "format", mutate: func(value *Audit) { value.FormatVersion = "v2" }, message: "format_version"},
		{name: "reason", mutate: func(value *Audit) { value.Entries[0].Reason = "other" }, message: "unknown reason"},
		{name: "explanation", mutate: func(value *Audit) { value.Entries[0].Explanation = "other" }, message: "explanation"},
		{name: "digest", mutate: func(value *Audit) { value.Entries[0].KeySHA256 = "ABC" }, message: "key_sha256"},
		{name: "prefix", mutate: func(value *Audit) { value.Entries[0].KeyPrefixHex = "ABC" }, message: "key_prefix_hex"},
		{name: "source", mutate: func(value *Audit) { value.Entries[0].SourceFile = "OTHER.TLK" }, message: "source_file"},
		{name: "order", mutate: func(value *Audit) {
			second := value.Entries[0]
			second.KeyOffset--
			value.Entries = append(value.Entries, second)
		}, message: "canonical source order"},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			audit := &Audit{
				FormatVersion: AuditFormat, ExtractorVersion: ExtractorVersion,
				Entries: []AuditEntry{validAuditEntry()},
			}
			testCase.mutate(audit)
			require.ErrorContains(t, audit.Validate(), testCase.message)
		})
	}
}

func TestLoadAuditRejectsUnknownFieldsAndExtraDocuments(t *testing.T) {
	t.Parallel()

	audit := &Audit{FormatVersion: AuditFormat, ExtractorVersion: ExtractorVersion, Entries: []AuditEntry{}}
	data, err := marshalAudit(audit)
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

			path := filepath.Join(t.TempDir(), auditFilename)
			require.NoError(t, os.WriteFile(path, testCase.data, filePermissions))
			_, err := LoadAudit(path)
			require.ErrorContains(t, err, testCase.message)
		})
	}
}

func TestUnreachableMatchKeyAuditAccountsForRawPair(t *testing.T) {
	t.Parallel()

	pair := rawPair{
		key: rawField{
			data: []byte{opIncludeNext}, offset: 0x1b39,
			where: sourceLocation{filename: "CASTLE.TLK", conversationID: 14},
		},
		response: rawField{
			data: upper("Invented response"), offset: 0x1b3b,
			where: sourceLocation{filename: "CASTLE.TLK", conversationID: 14},
		},
	}
	entry := newUnreachableMatchKeyAudit(pair)
	require.NoError(t, validateAuditEntry(entry))
	assert.Equal(t, "87", entry.KeyPrefixHex)
	assert.Equal(t, 0x1b39, entry.KeyOffset)
	assert.Equal(t, 0x1b3b, entry.ResponseOffset)
	assert.Equal(t, shaHex(pair.key.data), entry.KeySHA256)
	assert.Equal(t, shaHex(pair.response.data), entry.ResponseSHA256)
}

func TestUnrenderableControlOutputAuditAccountsForMarkerOnly(t *testing.T) {
	t.Parallel()

	marker := rawField{
		data: []byte{questionMarker}, offset: 0x367f,
		where: sourceLocation{filename: "TOWNE.TLK", conversationID: 23},
	}
	entry := newUnrenderableControlOutputAudit(marker)
	require.NoError(t, validateAuditEntry(entry))
	assert.Equal(t, AuditReasonUnrenderableControlOutput, entry.Reason)
	assert.Equal(t, "90", entry.KeyPrefixHex)
	assert.Equal(t, 0x367f, entry.KeyOffset)
	assert.Equal(t, 0x3680, entry.ResponseOffset)
	assert.Equal(t, 1, entry.KeyLength)
	assert.Zero(t, entry.ResponseLength)
	assert.Equal(t, shaHex(marker.data), entry.KeySHA256)
	assert.Equal(t, shaHex(nil), entry.ResponseSHA256)
}

func TestUnreachableOrphanAnswerKeyAuditAccountsForAbsentResponse(t *testing.T) {
	t.Parallel()

	key := rawField{
		data: append(upper("Long orphan answer key"), opAskName), offset: 0x1f0d,
		where: sourceLocation{filename: "DWELLING.TLK", conversationID: 15},
	}
	entry := newUnreachableOrphanAnswerKeyAudit(key)
	require.NoError(t, validateAuditEntry(entry))
	assert.Equal(t, AuditReasonUnreachableOrphanAnswerKey, entry.Reason)
	assert.Equal(t, 0x1f0d, entry.KeyOffset)
	assert.Equal(t, 0x1f0d+len(key.data)+1, entry.ResponseOffset)
	assert.Equal(t, len(key.data), entry.KeyLength)
	assert.Zero(t, entry.ResponseLength)
	assert.Equal(t, shaHex(key.data), entry.KeySHA256)
	assert.Equal(t, shaHex(nil), entry.ResponseSHA256)
}

func TestUnreachableOrphanAnswerKeyAuditRejectsReachableKey(t *testing.T) {
	t.Parallel()

	key := rawField{
		data: upper("reachable"), offset: 0x1f0d,
		where: sourceLocation{filename: "DWELLING.TLK", conversationID: 15},
	}
	entry := newUnreachableOrphanAnswerKeyAudit(key)
	require.ErrorContains(t, validateAuditEntry(entry), "must be unreachable")
}

func validAuditEntry() AuditEntry {
	digest := strings.Repeat("a", sha256HexLength)

	return AuditEntry{
		SourceFile: "CASTLE.TLK", ConversationID: 1,
		KeyOffset: 100, ResponseOffset: 102, KeyLength: 1, ResponseLength: 5,
		KeySHA256: digest, ResponseSHA256: digest, KeyPrefixHex: "87",
		Reason: AuditReasonUnreachableMatchKey, Explanation: unreachableMatchKeyExplanation,
	}
}

func shaHex(data []byte) string {
	digest := sha256.Sum256(data)

	return hex.EncodeToString(digest[:])
}
