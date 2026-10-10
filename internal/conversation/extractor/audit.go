package extractor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// AuditFormat identifies the extraction-audit grammar.
	AuditFormat = "ultima5redux.conversation-extraction-audit/v1"
	// AuditReasonUnreachableMatchKey identifies a source key that GKSTR input cannot match.
	AuditReasonUnreachableMatchKey = "unreachable_match_key"
	// AuditReasonUnrenderableControlOutput identifies source output omitted from semantic text.
	AuditReasonUnrenderableControlOutput = "unrenderable_control_output"
	// AuditReasonUnreachableOrphanAnswerKey identifies an answer key with no response that input cannot match.
	AuditReasonUnreachableOrphanAnswerKey = "unreachable_orphan_answer_key"
	auditFilename                         = "audit.yaml"
	auditKeyPrefixBytes                   = 16
	unreachableMatchKeyExplanation        = "Original GKSTR accepts input bytes 0x20..0x7f; after INSTR's " +
		"0x7f mask, this key contains a byte outside that range and can never match."
	unrenderableControlOutputExplanation = "Original output emits this control byte before transferring to the " +
		"following question; semantic v1 retains the question transfer but cannot represent the control glyph."
	unreachableOrphanAnswerKeyExplanation = "This final question answer key has no response before the terminal " +
		"marker and cannot match original input because it exceeds 15 bytes or contains a masked byte outside 0x20..0x7f."
)

// Audit records every recognized source omission outside runtime conversation YAML.
//
//nolint:tagliatelle // The versioned audit contract intentionally uses canonical snake_case YAML.
type Audit struct {
	FormatVersion    string       `yaml:"format_version"`
	ExtractorVersion string       `yaml:"extractor_version"`
	Entries          []AuditEntry `yaml:"entries"`
}

// AuditEntry accounts for one omitted source key/response pair without exposing its text.
//
//nolint:tagliatelle // The versioned audit contract intentionally uses canonical snake_case YAML.
type AuditEntry struct {
	SourceFile     string `yaml:"source_file"`
	ConversationID int    `yaml:"conversation_id"`
	KeyOffset      int    `yaml:"key_offset"`
	ResponseOffset int    `yaml:"response_offset"`
	KeyLength      int    `yaml:"key_length"`
	ResponseLength int    `yaml:"response_length"`
	KeySHA256      string `yaml:"key_sha256"`
	ResponseSHA256 string `yaml:"response_sha256"`
	KeyPrefixHex   string `yaml:"key_prefix_hex"`
	Reason         string `yaml:"reason"`
	Explanation    string `yaml:"explanation"`
}

// LoadAudit strictly loads and validates an extraction audit.
func LoadAudit(filename string) (*Audit, error) {
	return loadStrictYAML(filename, "audit", func(audit *Audit) error {
		return audit.Validate()
	})
}

// Validate checks the closed v1 audit contract and canonical entry ordering.
//
//nolint:err113 // Each versioned audit invariant has a precise diagnostic.
func (audit *Audit) Validate() error {
	if audit == nil {
		return errors.New("audit is nil")
	}
	if audit.FormatVersion != AuditFormat {
		return fmt.Errorf("format_version must equal %q", AuditFormat)
	}
	if audit.ExtractorVersion != ExtractorVersion {
		return fmt.Errorf("extractor_version must equal %q", ExtractorVersion)
	}
	if audit.Entries == nil {
		return errors.New("entries is required")
	}
	for index, entry := range audit.Entries {
		if err := validateAuditEntry(entry); err != nil {
			return fmt.Errorf("entries[%d]: %w", index, err)
		}
		if index > 0 && !auditEntryLess(audit.Entries[index-1], entry) {
			return fmt.Errorf("entries[%d] is not in canonical source order", index)
		}
	}

	return nil
}

//nolint:cyclop,err113 // Closed audit-entry fields need exact validation diagnostics.
func validateAuditEntry(entry AuditEntry) error {
	spec, exists := sourceSpecByFilename(entry.SourceFile)
	if !exists {
		return fmt.Errorf("source_file %q is not an extracted TLK source", entry.SourceFile)
	}
	if entry.ConversationID < 1 || entry.ConversationID > spec.count {
		return fmt.Errorf("conversation_id %d is outside 1..%d", entry.ConversationID, spec.count)
	}
	if entry.KeyOffset < 0 {
		return errors.New("key_offset must not be negative")
	}
	if entry.ResponseOffset <= entry.KeyOffset {
		return errors.New("response_offset must be greater than key_offset")
	}
	if entry.KeyLength < 1 {
		return errors.New("key_length must be positive")
	}
	if entry.ResponseLength < 0 {
		return errors.New("response_length must not be negative")
	}
	if !isLowerHexDigest(entry.KeySHA256) {
		return errors.New("key_sha256 must be a lowercase SHA-256 digest")
	}
	if !isLowerHexDigest(entry.ResponseSHA256) {
		return errors.New("response_sha256 must be a lowercase SHA-256 digest")
	}
	prefixLength := min(entry.KeyLength, auditKeyPrefixBytes)
	if len(entry.KeyPrefixHex) != prefixLength*2 || !isLowerHex(entry.KeyPrefixHex) {
		return fmt.Errorf("key_prefix_hex must encode the first %d key bytes in lowercase", prefixLength)
	}
	switch entry.Reason {
	case AuditReasonUnreachableMatchKey:
		if entry.Explanation != unreachableMatchKeyExplanation {
			return errors.New("explanation does not match the closed reason text")
		}
	case AuditReasonUnrenderableControlOutput:
		if entry.Explanation != unrenderableControlOutputExplanation {
			return errors.New("explanation does not match the closed reason text")
		}
		if err := validateUnrenderableControlOutput(entry); err != nil {
			return err
		}
	case AuditReasonUnreachableOrphanAnswerKey:
		if entry.Explanation != unreachableOrphanAnswerKeyExplanation {
			return errors.New("explanation does not match the closed reason text")
		}
		if err := validateUnreachableOrphanAnswerKey(entry); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown reason %q", entry.Reason)
	}

	return nil
}

//nolint:err113 // Closed reason invariants need exact validation diagnostics.
func validateUnreachableOrphanAnswerKey(entry AuditEntry) error {
	if entry.KeyLength <= maxOriginalInputLen && isPrintableAuditPrefix(entry) {
		return errors.New("unreachable_orphan_answer_key must be unreachable under original input constraints")
	}
	emptyDigest := sha256.Sum256(nil)
	if entry.ResponseLength != 0 || entry.ResponseSHA256 != hex.EncodeToString(emptyDigest[:]) {
		return errors.New("unreachable_orphan_answer_key response fields must identify an absent response")
	}
	if entry.ResponseOffset != entry.KeyOffset+entry.KeyLength+1 {
		return errors.New("unreachable_orphan_answer_key response_offset must follow the key terminator")
	}

	return nil
}

func isPrintableAuditPrefix(entry AuditEntry) bool {
	prefix, err := hex.DecodeString(entry.KeyPrefixHex)
	if err != nil {
		return false
	}

	return isPrintableMatchKey(prefix)
}

//nolint:err113 // Closed reason invariants need exact validation diagnostics.
func validateUnrenderableControlOutput(entry AuditEntry) error {
	if entry.KeyLength != 1 || entry.KeyPrefixHex != "90" {
		return errors.New("unrenderable_control_output must identify exactly one 0x90 byte")
	}
	markerDigest := sha256.Sum256([]byte{questionMarker})
	if entry.KeySHA256 != hex.EncodeToString(markerDigest[:]) {
		return errors.New("unrenderable_control_output key_sha256 must identify the 0x90 byte")
	}
	emptyDigest := sha256.Sum256(nil)
	if entry.ResponseLength != 0 || entry.ResponseSHA256 != hex.EncodeToString(emptyDigest[:]) {
		return errors.New("unrenderable_control_output response fields must identify an empty continuation")
	}
	if entry.ResponseOffset != entry.KeyOffset+1 {
		return errors.New("unrenderable_control_output response_offset must immediately follow the 0x90 byte")
	}

	return nil
}

func auditEntryLess(left, right AuditEntry) bool {
	leftSource := sourceOrder(left.SourceFile)
	rightSource := sourceOrder(right.SourceFile)
	leftKey := []int{leftSource, left.ConversationID, left.KeyOffset, left.ResponseOffset}
	rightKey := []int{rightSource, right.ConversationID, right.KeyOffset, right.ResponseOffset}

	return slices.Compare(leftKey, rightKey) < 0
}

func sourceOrder(filename string) int {
	for index, spec := range conversationSources() {
		if spec.filename == filename {
			return index
		}
	}

	return len(conversationSources())
}

func sourceSpecByFilename(filename string) (sourceSpec, bool) {
	for _, spec := range conversationSources() {
		if spec.filename == filename {
			return spec, true
		}
	}

	return sourceSpec{filename: "", directory: "", source: "", count: 0}, false
}

func isLowerHex(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character >= '0' && character <= '9' {
			continue
		}
		if character < 'a' || character > 'f' {
			return false
		}
	}

	return true
}

func auditKeyPrefix(data []byte) string {
	return hex.EncodeToString(data[:min(len(data), auditKeyPrefixBytes)])
}

func marshalAudit(audit *Audit) ([]byte, error) {
	var builder strings.Builder
	encoder := yaml.NewEncoder(&builder)
	encoder.SetIndent(manifestIndent)
	if err := encoder.Encode(audit); err != nil {
		return nil, fmt.Errorf("encoding extraction audit: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing extraction audit encoder: %w", err)
	}

	return []byte(builder.String()), nil
}
