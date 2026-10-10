package patch_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	conversationpatch "github.com/bradhannah/UltimaVRedux/internal/conversation/patch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTarget    = "keep/010.yaml"
	testSchema    = "ultima5redux.conversation/v1"
	testExtractor = extractor.ExtractorVersion
)

func TestGenerateAndApplyEditingForms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		base   string
		edited string
	}{
		{name: "replace", base: "one\ntwo\nthree\n", edited: "one\nchanged\nthree\n"},
		{name: "delete", base: "one\ntwo\nthree\n", edited: "one\nthree\n"},
		{name: "insert", base: "one\nthree\n", edited: "one\ntwo\nthree\n"},
		{name: "prepend", base: "two\nthree\n", edited: "one\ntwo\nthree\n"},
		{name: "append", base: "one\ntwo\n", edited: "one\ntwo\nthree\n"},
		{
			name: "multiple hunks",
			base: strings.Join([]string{
				"01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12", "13", "14", "15",
			}, "\n") + "\n",
			edited: strings.Join([]string{
				"01", "first", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12", "13", "second", "15",
			}, "\n") + "\n",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			generated, err := conversationpatch.Generate(
				testTarget,
				testSchema,
				testExtractor,
				[]byte(testCase.base),
				[]byte(testCase.edited),
			)
			require.NoError(t, err)
			assert.Contains(t, string(generated), "# format: "+conversationpatch.FormatVersion+"\n")
			assert.Contains(t, string(generated), "--- a/"+testTarget+"\n+++ b/"+testTarget+"\n")

			actual, err := conversationpatch.Apply([]byte(testCase.base), generated, testExpectation())
			require.NoError(t, err)
			assert.Equal(t, testCase.edited, string(actual))
		})
	}
}

func TestGenerateIsDeterministicAndUsesOrdinaryContext(t *testing.T) {
	t.Parallel()

	base := []byte("one\ntwo\nthree\nfour\nfive\nsix\nseven\n")
	edited := []byte("one\ntwo\nthree\nchanged\nfive\nsix\nseven\n")
	first, err := conversationpatch.Generate(testTarget, testSchema, testExtractor, base, edited)
	require.NoError(t, err)
	second, err := conversationpatch.Generate(testTarget, testSchema, testExtractor, base, edited)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Contains(t, string(first), "@@ -1,7 +1,7 @@\n")
}

func TestApplyRejectsMetadataAndHashMismatch(t *testing.T) {
	t.Parallel()

	base := []byte("one\ntwo\n")
	generated, err := conversationpatch.Generate(testTarget, testSchema, testExtractor, base, []byte("one\nchanged\n"))
	require.NoError(t, err)

	testCases := []struct {
		name        string
		patch       []byte
		expectation conversationpatch.Expectation
		base        []byte
		message     string
	}{
		{
			name: "target metadata", patch: bytes.Replace(generated, []byte("# target: "+testTarget), []byte("# target: keep/011.yaml"), 1),
			expectation: testExpectation(), base: base, message: "old file header",
		},
		{
			name: "expected target", patch: generated,
			expectation: conversationpatch.Expectation{Target: "keep/011.yaml", SchemaVersion: testSchema, ExtractorVersion: testExtractor},
			base:        base, message: "patch target",
		},
		{
			name: "schema", patch: generated,
			expectation: conversationpatch.Expectation{Target: testTarget, SchemaVersion: "schema/v2", ExtractorVersion: testExtractor},
			base:        base, message: "patch schema",
		},
		{
			name: "extractor", patch: generated,
			expectation: conversationpatch.Expectation{Target: testTarget, SchemaVersion: testSchema, ExtractorVersion: "extractor/v2"},
			base:        base, message: "patch extractor",
		},
		{
			name: "base hash", patch: generated, expectation: testExpectation(),
			base: []byte("different\nbase\n"), message: "base SHA-256",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, applyErr := conversationpatch.Apply(testCase.base, testCase.patch, testCase.expectation)
			require.ErrorContains(t, applyErr, testCase.message)
		})
	}
}

func TestApplyRejectsFuzzyContext(t *testing.T) {
	t.Parallel()

	base := []byte("one\ntwo\nthree\nfour\n")
	generated, err := conversationpatch.Generate(testTarget, testSchema, testExtractor, base, []byte("one\ntwo\nchanged\nfour\n"))
	require.NoError(t, err)
	tampered := bytes.Replace(generated, []byte(" two\n"), []byte(" other\n"), 1)

	_, err = conversationpatch.Apply(base, tampered, testExpectation())
	require.ErrorContains(t, err, "context does not exactly match")
}

func TestApplyRejectsMalformedAndTraversalPatches(t *testing.T) {
	t.Parallel()

	base := []byte("one\ntwo\n")
	generated, err := conversationpatch.Generate(testTarget, testSchema, testExtractor, base, []byte("one\nchanged\n"))
	require.NoError(t, err)

	testCases := []struct {
		name    string
		patch   []byte
		message string
	}{
		{name: "traversal", patch: bytes.Replace(generated, []byte(testTarget), []byte("../010.yaml"), 1), message: "clean relative slash path"},
		{name: "wrong old header", patch: bytes.Replace(generated, []byte("--- a/"+testTarget), []byte("--- a/keep/011.yaml"), 1), message: "old file header"},
		{name: "wrong new header", patch: bytes.Replace(generated, []byte("+++ b/"+testTarget), []byte("+++ b/keep/011.yaml"), 1), message: "new file header"},
		{name: "unsupported no-newline marker", patch: append(generated, []byte("\\ No newline at end of file\n")...), message: "unsupported unified-diff syntax"},
		{name: "bad hunk", patch: bytes.Replace(generated, []byte("@@ -"), []byte("@@ malformed "), 1), message: "hunk header"},
		{name: "missing terminal newline", patch: generated[:len(generated)-1], message: "must end with a newline"},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, applyErr := conversationpatch.Apply(base, testCase.patch, testExpectation())
			require.ErrorContains(t, applyErr, testCase.message)
		})
	}
}

func testExpectation() conversationpatch.Expectation {
	return conversationpatch.Expectation{
		Target: testTarget, SchemaVersion: testSchema, ExtractorVersion: testExtractor,
	}
}
