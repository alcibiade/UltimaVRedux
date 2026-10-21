// Package patch generates and strictly applies conversation unified diffs.
package patch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

const (
	// FormatVersion identifies the conversation patch metadata contract.
	FormatVersion = "ultima5redux.conversation-patch/v1"
	contextLines  = 3
	metadataLines = 5
	sha256Length  = 64
)

// Metadata binds a patch to one exact extracted conversation.
type Metadata struct {
	Target           string
	SchemaVersion    string
	ExtractorVersion string
	BaseSHA256       string
}

// Expectation supplies the trusted metadata required while applying a patch.
type Expectation struct {
	Target           string
	SchemaVersion    string
	ExtractorVersion string
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []string
}

type document struct {
	metadata Metadata
	hunks    []hunk
}

// Generate returns one deterministic v1 patch with three lines of context.
//
//nolint:err113 // Strict patch diagnostics identify the selected target and violated contract.
func Generate(target, schemaVersion, extractorVersion string, base, edited []byte) ([]byte, error) {
	if err := validateIdentity(target, schemaVersion, extractorVersion); err != nil {
		return nil, err
	}
	if bytes.Equal(base, edited) {
		return nil, errors.New("conversation patch requires changed content")
	}
	if err := validateContent("base", base); err != nil {
		return nil, err
	}
	if err := validateContent("edited", edited); err != nil {
		return nil, err
	}

	digest := sha256.Sum256(base)
	metadata := fmt.Sprintf(
		"# format: %s\n# target: %s\n# schema: %s\n# extractor: %s\n# base-sha256: %s\n",
		FormatVersion,
		target,
		schemaVersion,
		extractorVersion,
		hex.EncodeToString(digest[:]),
	)
	var unified strings.Builder
	err := difflib.WriteUnifiedDiff(&unified, difflib.UnifiedDiff{
		A: splitLines(base), B: splitLines(edited),
		FromFile: "a/" + target, ToFile: "b/" + target, FromDate: "", ToDate: "",
		Eol: "\n", Context: contextLines,
	})
	if err != nil {
		return nil, fmt.Errorf("generating unified diff for %q: %w", target, err)
	}
	result := []byte(metadata + unified.String())
	if _, err := parse(result); err != nil {
		return nil, fmt.Errorf("validating generated patch for %q: %w", target, err)
	}

	return result, nil
}

// ParseMetadata strictly parses a patch and returns its binding metadata.
func ParseMetadata(data []byte) (Metadata, error) {
	document, err := parse(data)
	if err != nil {
		return Metadata{}, err
	}

	return document.metadata, nil
}

// Apply validates all metadata and hunks, then applies the patch without fuzzy matching.
//
//nolint:err113 // Exact metadata checks are the security boundary for patch application.
func Apply(base, patchData []byte, expectation Expectation) ([]byte, error) {
	if err := validateIdentity(expectation.Target, expectation.SchemaVersion, expectation.ExtractorVersion); err != nil {
		return nil, fmt.Errorf("validating patch expectation: %w", err)
	}
	if err := validateContent("base", base); err != nil {
		return nil, err
	}
	document, err := parse(patchData)
	if err != nil {
		return nil, err
	}
	if document.metadata.Target != expectation.Target {
		return nil, fmt.Errorf("patch target is %q, want %q", document.metadata.Target, expectation.Target)
	}
	if document.metadata.SchemaVersion != expectation.SchemaVersion {
		return nil, fmt.Errorf(
			"patch schema is %q, want %q",
			document.metadata.SchemaVersion,
			expectation.SchemaVersion,
		)
	}
	if document.metadata.ExtractorVersion != expectation.ExtractorVersion {
		return nil, fmt.Errorf(
			"patch extractor is %q, want %q",
			document.metadata.ExtractorVersion,
			expectation.ExtractorVersion,
		)
	}
	digest := sha256.Sum256(base)
	if actual := hex.EncodeToString(digest[:]); document.metadata.BaseSHA256 != actual {
		return nil, fmt.Errorf("patch base SHA-256 is %q, want %q", document.metadata.BaseSHA256, actual)
	}

	result, err := applyHunks(splitLines(base), document.hunks)
	if err != nil {
		return nil, err
	}
	joined := []byte(strings.Join(result, ""))
	if err := validateContent("patched", joined); err != nil {
		return nil, err
	}
	if bytes.Equal(base, joined) {
		return nil, errors.New("patch does not change base content")
	}

	return joined, nil
}

//nolint:cyclop,funlen,gocognit,err113 // The parser deliberately recognizes a closed unified-diff subset.
func parse(data []byte) (*document, error) {
	if len(data) == 0 {
		return nil, errors.New("patch is empty")
	}
	if data[len(data)-1] != '\n' {
		return nil, errors.New("patch must end with a newline")
	}
	lines := splitLines(data)
	if len(lines) < metadataLines+3 {
		return nil, errors.New("patch is missing metadata, file headers, or hunks")
	}
	format, err := metadataValue(lines[0], "format")
	if err != nil {
		return nil, err
	}
	if format != FormatVersion {
		return nil, fmt.Errorf("patch format is %q, want %q", format, FormatVersion)
	}
	target, err := metadataValue(lines[1], "target")
	if err != nil {
		return nil, err
	}
	schemaVersion, err := metadataValue(lines[2], "schema")
	if err != nil {
		return nil, err
	}
	extractorVersion, err := metadataValue(lines[3], "extractor")
	if err != nil {
		return nil, err
	}
	baseSHA256, err := metadataValue(lines[4], "base-sha256")
	if err != nil {
		return nil, err
	}
	if err := validateIdentity(target, schemaVersion, extractorVersion); err != nil {
		return nil, err
	}
	if !isLowerHexDigest(baseSHA256) {
		return nil, errors.New("patch base-sha256 must be a lowercase SHA-256 digest")
	}
	if lines[5] != "--- a/"+target+"\n" {
		return nil, fmt.Errorf("patch old file header is %q, want %q", trimLine(lines[5]), "--- a/"+target)
	}
	if lines[6] != "+++ b/"+target+"\n" {
		return nil, fmt.Errorf("patch new file header is %q, want %q", trimLine(lines[6]), "+++ b/"+target)
	}

	hunks := make([]hunk, 0, 1)
	for index := 7; index < len(lines); {
		current, parseErr := parseHunkHeader(lines[index])
		if parseErr != nil {
			return nil, fmt.Errorf("parsing patch line %d: %w", index+1, parseErr)
		}
		index++
		oldLines := 0
		newLines := 0
		changes := 0
		for index < len(lines) && !strings.HasPrefix(lines[index], "@@ ") {
			line := lines[index]
			if len(line) < 2 || line[len(line)-1] != '\n' {
				return nil, fmt.Errorf("patch line %d is not a complete unified-diff line", index+1)
			}
			switch line[0] {
			case ' ':
				oldLines++
				newLines++
			case '-':
				oldLines++
				changes++
			case '+':
				newLines++
				changes++
			default:
				return nil, fmt.Errorf("patch line %d uses unsupported unified-diff syntax", index+1)
			}
			current.lines = append(current.lines, line)
			index++
		}
		if len(current.lines) == 0 {
			return nil, errors.New("patch hunk has no lines")
		}
		if changes == 0 {
			return nil, errors.New("patch hunk has no changes")
		}
		if oldLines != current.oldCount || newLines != current.newCount {
			return nil, fmt.Errorf(
				"patch hunk line counts are old=%d new=%d, header declares old=%d new=%d",
				oldLines,
				newLines,
				current.oldCount,
				current.newCount,
			)
		}
		hunks = append(hunks, current)
	}
	if len(hunks) == 0 {
		return nil, errors.New("patch has no hunks")
	}

	return &document{
		metadata: Metadata{
			Target: target, SchemaVersion: schemaVersion,
			ExtractorVersion: extractorVersion, BaseSHA256: baseSHA256,
		},
		hunks: hunks,
	}, nil
}

//nolint:err113 // Exact old/new cursor checks are the core no-fuzz application contract.
func applyHunks(base []string, hunks []hunk) ([]string, error) {
	result := make([]string, 0, len(base))
	basePosition := 0
	for index, current := range hunks {
		oldIndex, err := rangeIndex(current.oldStart, current.oldCount)
		if err != nil {
			return nil, fmt.Errorf("hunk %d old range: %w", index+1, err)
		}
		newIndex, err := rangeIndex(current.newStart, current.newCount)
		if err != nil {
			return nil, fmt.Errorf("hunk %d new range: %w", index+1, err)
		}
		if oldIndex < basePosition || oldIndex > len(base) {
			return nil, fmt.Errorf("hunk %d old range starts outside the next exact base position", index+1)
		}
		result = append(result, base[basePosition:oldIndex]...)
		basePosition = oldIndex
		if newIndex != len(result) {
			return nil, fmt.Errorf("hunk %d new range starts at %d, want %d", index+1, newIndex, len(result))
		}

		for _, line := range current.lines {
			content := line[1:]
			switch line[0] {
			case ' ':
				if basePosition >= len(base) || base[basePosition] != content {
					return nil, fmt.Errorf("hunk %d context does not exactly match base line %d", index+1, basePosition+1)
				}
				result = append(result, content)
				basePosition++
			case '-':
				if basePosition >= len(base) || base[basePosition] != content {
					return nil, fmt.Errorf("hunk %d deletion does not exactly match base line %d", index+1, basePosition+1)
				}
				basePosition++
			case '+':
				result = append(result, content)
			}
		}
	}
	result = append(result, base[basePosition:]...)

	return result, nil
}

//nolint:err113 // Strict parser diagnostics include rejected header values.
func parseHunkHeader(line string) (hunk, error) {
	expression := regexp.MustCompile(`^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@\n$`)
	matches := expression.FindStringSubmatch(line)
	if matches == nil {
		return hunk{}, fmt.Errorf("unsupported or malformed hunk header %q", trimLine(line))
	}
	oldStart, err := strconv.Atoi(matches[1])
	if err != nil {
		return hunk{}, fmt.Errorf("decoding old start: %w", err)
	}
	oldCount, err := parseRangeCount(matches[2])
	if err != nil {
		return hunk{}, fmt.Errorf("decoding old count: %w", err)
	}
	newStart, err := strconv.Atoi(matches[3])
	if err != nil {
		return hunk{}, fmt.Errorf("decoding new start: %w", err)
	}
	newCount, err := parseRangeCount(matches[4])
	if err != nil {
		return hunk{}, fmt.Errorf("decoding new count: %w", err)
	}
	if _, err := rangeIndex(oldStart, oldCount); err != nil {
		return hunk{}, fmt.Errorf("validating old range: %w", err)
	}
	if _, err := rangeIndex(newStart, newCount); err != nil {
		return hunk{}, fmt.Errorf("validating new range: %w", err)
	}

	return hunk{
		oldStart: oldStart, oldCount: oldCount, newStart: newStart, newCount: newCount,
		lines: nil,
	}, nil
}

func parseRangeCount(value string) (int, error) {
	if value == "" {
		return 1, nil
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("decoding range count: %w", err)
	}

	return count, nil
}

//nolint:err113 // Strict parser diagnostics distinguish invalid unified-diff ranges.
func rangeIndex(start, count int) (int, error) {
	if start < 0 || count < 0 {
		return 0, errors.New("range values must not be negative")
	}
	if count > 0 {
		if start == 0 {
			return 0, errors.New("non-empty range must start at line one or later")
		}

		return start - 1, nil
	}

	return start, nil
}

//nolint:err113 // Strict parser diagnostics identify the malformed metadata field.
func metadataValue(line, name string) (string, error) {
	prefix := "# " + name + ": "
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, "\n") {
		return "", fmt.Errorf("patch metadata line must use %q", trimLine(prefix))
	}
	value := strings.TrimSuffix(strings.TrimPrefix(line, prefix), "\n")
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("patch metadata %q must have one non-empty line value", name)
	}

	return value, nil
}

//nolint:err113 // Closed path/version validation is intentionally explicit.
func validateIdentity(target, schemaVersion, extractorVersion string) error {
	if target == "" || filepath.IsAbs(target) || filepath.ToSlash(filepath.Clean(target)) != target ||
		target == ".." || strings.HasPrefix(target, "../") || strings.ContainsRune(target, '\\') ||
		strings.ContainsAny(target, "\x00\r\n") || filepath.Ext(target) != ".yaml" {
		return fmt.Errorf("patch target %q must be a clean relative slash path ending in .yaml", target)
	}
	if schemaVersion == "" || strings.ContainsAny(schemaVersion, "\r\n") {
		return errors.New("patch schema version must be one non-empty line")
	}
	if extractorVersion == "" || strings.ContainsAny(extractorVersion, "\r\n") {
		return errors.New("patch extractor version must be one non-empty line")
	}

	return nil
}

//nolint:err113 // Diagnostic identifies which content side violates the canonical newline rule.
func validateContent(kind string, data []byte) error {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return fmt.Errorf("%s conversation content must end with a newline", kind)
	}

	return nil
}

func splitLines(data []byte) []string {
	parts := bytes.SplitAfter(data, []byte{'\n'})
	if len(parts) > 0 && len(parts[len(parts)-1]) == 0 {
		parts = parts[:len(parts)-1]
	}
	lines := make([]string, len(parts))
	for index, part := range parts {
		lines[index] = string(part)
	}

	return lines
}

func isLowerHexDigest(value string) bool {
	if len(value) != sha256Length {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}

	return true
}

func trimLine(line string) string {
	return strings.TrimSuffix(line, "\n")
}
