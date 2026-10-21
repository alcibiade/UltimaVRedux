package save_snapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCanonicalCodecIsDeterministicAndStrict(t *testing.T) {
	t.Parallel()
	envelope := validEnvelope(validSnapshot())
	first, err := EncodeCanonical(envelope)
	if err != nil {
		t.Fatalf("EncodeCanonical() error = %v", err)
	}
	second, err := EncodeCanonical(envelope)
	if err != nil {
		t.Fatalf("EncodeCanonical(second) error = %v", err)
	}
	if !bytes.Equal(first, second) || first[len(first)-1] != '\n' {
		t.Fatal("canonical encoding is not deterministic or lacks final newline")
	}
	decoded, err := Decode(first)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	reencoded, err := EncodeCanonical(decoded)
	if err != nil {
		t.Fatalf("EncodeCanonical(decoded) error = %v", err)
	}
	if !bytes.Equal(first, reencoded) {
		t.Fatal("decode/encode did not preserve canonical bytes")
	}

	unknown := bytes.Replace(first, []byte(`"format":`), []byte(`"unknown":true,"format":`), 1)
	if _, err = Decode(unknown); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("Decode(unknown field) error = %v, want ErrInvalidEnvelope", err)
	}
	if _, err = Decode(append(first, []byte(` {}`)...)); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("Decode(trailing JSON) error = %v, want ErrInvalidEnvelope", err)
	}
}

func TestEnvelopeMetadataOptionallyRoundTripsDoomRunAssociation(t *testing.T) {
	t.Parallel()

	legacy := validEnvelope(validSnapshot())
	data, err := EncodeCanonical(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("doomRunId")) {
		t.Fatal("unassociated envelope encoded optional Doom run metadata")
	}
	decoded, err := Decode(data)
	if err != nil || decoded.Metadata.DoomRunID != "" {
		t.Fatalf("Decode(unassociated) = %#v, %v", decoded.Metadata, err)
	}

	associated := validEnvelope(validSnapshot())
	associated.Metadata.DoomRunID = "opaque-run-id"
	data, err = EncodeCanonical(associated)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = Decode(data)
	if err != nil || decoded.Metadata.DoomRunID != associated.Metadata.DoomRunID {
		t.Fatalf("Decode(associated) = %#v, %v", decoded.Metadata, err)
	}
}

func TestTransientReconstructionHintSurvivesPlainGoCopy(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.MapUnits = nil
	snapshot.MarkMapUnitsForReconstruction()

	copyOfSnapshot := snapshot
	if !copyOfSnapshot.NeedsMapUnitReconstruction() {
		t.Fatal("plain SaveSnapshot copy lost reconstruction metadata")
	}
	data, err := json.Marshal(copyOfSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("reconstructMapUnits")) {
		t.Fatalf("transient reconstruction metadata was encoded: %s", data)
	}
}

func TestCurrentCodecRequiresContinuationAndEveryMember(t *testing.T) {
	t.Parallel()
	data := currentEnvelopeDocument(t, validSnapshot())
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range []string{"isCriminal", "hmsCapeAutomaticWaitActorSkip", "openingCampaignEncounterReduction"} {
		if !bytes.Contains(encoded, []byte(`"`+member+`":false`)) {
			t.Fatalf("canonical continuation omitted required false boolean %s", member)
		}
	}

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "continuation", mutate: func(snapshot map[string]any) { delete(snapshot, "continuation") }},
		{name: "null continuation", mutate: func(snapshot map[string]any) { snapshot["continuation"] = nil }},
		{name: "active player", mutate: func(snapshot map[string]any) { delete(snapshot["continuation"].(map[string]any), "activePlayerIndex") }},
		{name: "criminal boolean", mutate: func(snapshot map[string]any) { delete(snapshot["continuation"].(map[string]any), "isCriminal") }},
		{name: "HMS cadence boolean", mutate: func(snapshot map[string]any) {
			delete(snapshot["continuation"].(map[string]any), "hmsCapeAutomaticWaitActorSkip")
		}},
		{name: "beggar counter", mutate: func(snapshot map[string]any) { delete(snapshot["continuation"].(map[string]any), "beggarKarmaCounter") }},
		{name: "opening campaign reduction", mutate: func(snapshot map[string]any) {
			delete(snapshot["continuation"].(map[string]any), "openingCampaignEncounterReduction")
		}},
		{name: "combat action counter", mutate: func(snapshot map[string]any) {
			delete(snapshot["continuation"].(map[string]any), "combatActionTimeCounter")
		}},
		{name: "null criminal boolean", mutate: func(snapshot map[string]any) { snapshot["continuation"].(map[string]any)["isCriminal"] = nil }},
		{name: "null HMS cadence boolean", mutate: func(snapshot map[string]any) {
			snapshot["continuation"].(map[string]any)["hmsCapeAutomaticWaitActorSkip"] = nil
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := cloneJSONDocument(t, data)
			test.mutate(document["snapshot"].(map[string]any))
			malformed, marshalErr := json.Marshal(document)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if _, decodeErr := Decode(malformed); !errors.Is(decodeErr, ErrInvalidEnvelope) {
				t.Fatalf("Decode() error = %v, want ErrInvalidEnvelope", decodeErr)
			}
		})
	}
}

func TestCurrentCodecRoundTripsContinuationState(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Continuation = ContinuationSnapshot{
		ActivePlayerIndex:                 5,
		IsCriminal:                        true,
		HMSCapeAutomaticWaitActorSkip:     true,
		BeggarKarmaCounter:                255,
		OpeningCampaignEncounterReduction: true,
		CombatActionTimeCounter:           9,
	}
	data, err := EncodeCanonical(validEnvelope(snapshot))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Snapshot.Continuation != snapshot.Continuation {
		t.Fatalf("continuation round trip = %#v, want %#v", decoded.Snapshot.Continuation, snapshot.Continuation)
	}
}

func TestCurrentCodecRejectsHistoricalReconstructionJSON(t *testing.T) {
	t.Parallel()
	document := currentEnvelopeDocument(t, validSnapshot())
	document["snapshot"].(map[string]any)["redux"].(map[string]any)["reconstructMapUnits"] = false
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Decode(data); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("Decode(reconstructMapUnits) error = %v, want ErrInvalidEnvelope", err)
	}
}

func TestCodecRequiresCurrentReduxArrays(t *testing.T) {
	t.Parallel()
	base := currentEnvelopeDocument(t, validSnapshot())
	for _, member := range []string{"dungeonDiscoveries", "playerKnowledge", "questJournal"} {
		for _, null := range []bool{false, true} {
			name := "missing " + member
			if null {
				name = "null " + member
			}
			t.Run(name, func(t *testing.T) {
				document := cloneJSONDocument(t, base)
				redux := document["snapshot"].(map[string]any)["redux"].(map[string]any)
				if null {
					redux[member] = nil
				} else {
					delete(redux, member)
				}
				data, marshalErr := json.Marshal(document)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				if _, decodeErr := Decode(data); !errors.Is(decodeErr, ErrInvalidEnvelope) {
					t.Fatalf("Decode() error = %v, want ErrInvalidEnvelope", decodeErr)
				}
			})
		}
	}
}

func TestDecodeReturnsTypedUnsupportedVersionForOlderAndFuture(t *testing.T) {
	t.Parallel()
	base := currentEnvelopeDocument(t, validSnapshot())
	for _, version := range []int{MinimumVersion - 1, 12, 11, 4, 1, 0, -1, CurrentVersion + 1, 99} {
		t.Run(stringVersion(version), func(t *testing.T) {
			document := cloneJSONDocument(t, base)
			document["snapshot"].(map[string]any)["version"] = version
			data, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Decode(data)
			var unsupported *UnsupportedVersionError
			if !errors.As(err, &unsupported) || unsupported.Version != version || !errors.Is(err, ErrUnsupportedSnapshotVersion) {
				t.Fatalf("Decode(version %d) error = %v, want typed UnsupportedVersionError", version, err)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("supported versions are %d through %d", MinimumVersion, CurrentVersion)) {
				t.Fatalf("Decode(version %d) guidance = %q", version, err)
			}
		})
	}
}

func TestDecodeUnsupportedVersionRejectsEverySupportedVersion(t *testing.T) {
	t.Parallel()
	base := currentEnvelopeDocument(t, validSnapshot())
	for version := MinimumVersion; version <= CurrentVersion; version++ {
		t.Run(stringVersion(version), func(t *testing.T) {
			document := cloneJSONDocument(t, base)
			// Only the version is relevant to this guard; supported schemas must
			// never enter the opaque-body path, even with otherwise invalid bodies.
			document["snapshot"] = map[string]any{"version": version, "futureField": true}
			data, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			if _, decodeErr := DecodeUnsupportedVersion(data); !errors.Is(decodeErr, ErrInvalidEnvelope) {
				t.Fatalf("DecodeUnsupportedVersion(version %d) error = %v, want ErrInvalidEnvelope", version, decodeErr)
			}
		})
	}

	for _, version := range []int{MinimumVersion - 1, 12, 11, 4, CurrentVersion + 1, 99} {
		document := cloneJSONDocument(t, base)
		document["snapshot"].(map[string]any)["version"] = version
		document["snapshot"].(map[string]any)["futureField"] = map[string]any{"opaque": true}
		data, marshalErr := json.Marshal(document)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		decoded, decodeErr := DecodeUnsupportedVersion(data)
		if decodeErr != nil || decoded.Snapshot.Version != version {
			t.Fatalf("DecodeUnsupportedVersion(version %d) = %#v, %v", version, decoded, decodeErr)
		}
	}
}

func TestDecodersRejectNullSnapshotVersion(t *testing.T) {
	t.Parallel()
	document := currentEnvelopeDocument(t, validSnapshot())
	document["snapshot"].(map[string]any)["version"] = nil
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Decode(data); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("Decode(null version) error = %v, want ErrInvalidEnvelope", err)
	}
	if _, err = DecodeUnsupportedVersion(data); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("DecodeUnsupportedVersion(null version) error = %v, want ErrInvalidEnvelope", err)
	}
}

func TestDecodeRejectsDuplicateObjectMembers(t *testing.T) {
	t.Parallel()
	data, err := EncodeCanonical(validEnvelope(validSnapshot()))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		duplicate []byte
		member    []byte
	}{
		{name: "format", duplicate: []byte(`"format":"other",`), member: []byte(`"format":`)},
		{name: "metadata", duplicate: []byte(`"metadata":{},`), member: []byte(`"metadata":`)},
		{name: "snapshot", duplicate: []byte(`"snapshot":{},`), member: []byte(`"snapshot":`)},
		{name: "snapshot version", duplicate: []byte(`"version":99,`), member: []byte(`"version":`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			duplicated := bytes.Replace(data, test.member, append(test.duplicate, test.member...), 1)
			if _, decodeErr := Decode(duplicated); !errors.Is(decodeErr, ErrInvalidEnvelope) {
				t.Fatalf("Decode() error = %v, want ErrInvalidEnvelope", decodeErr)
			}
		})
	}
}

func TestCodecRejectsUnsupportedEnvelopeAndMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*Envelope)
		want   error
	}{
		{name: "format", mutate: func(envelope *Envelope) { envelope.Format = "other" }, want: ErrInvalidEnvelope},
		{name: "envelope version", mutate: func(envelope *Envelope) { envelope.EnvelopeVersion++ }, want: ErrInvalidEnvelope},
		{name: "snapshot version", mutate: func(envelope *Envelope) { envelope.Snapshot.Version++ }, want: ErrInvalidSnapshot},
		{name: "name", mutate: func(envelope *Envelope) { envelope.Metadata.Name = " " }, want: ErrInvalidEnvelope},
		{name: "time", mutate: func(envelope *Envelope) { envelope.Metadata.CreatedAt = time.Time{} }, want: ErrInvalidEnvelope},
		{name: "non UTC", mutate: func(envelope *Envelope) {
			envelope.Metadata.CreatedAt = envelope.Metadata.CreatedAt.In(time.FixedZone("offset", 3600))
		}, want: ErrInvalidEnvelope},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envelope := validEnvelope(validSnapshot())
			test.mutate(&envelope)
			_, err := EncodeCanonical(envelope)
			if !errors.Is(err, test.want) {
				t.Fatalf("EncodeCanonical() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestDecodeRejectsWrongFixedArrayLength(t *testing.T) {
	t.Parallel()
	document := currentEnvelopeDocument(t, validSnapshot())
	shadowlords := document["snapshot"].(map[string]any)["progression"].(map[string]any)["shadowlords"].(map[string]any)
	shadowlords["townAssignments"] = []any{1, 2}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Decode(data); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("Decode(short fixed array) error = %v, want ErrInvalidEnvelope", err)
	}
}

func currentEnvelopeDocument(t *testing.T, snapshot SaveSnapshot) map[string]any {
	t.Helper()
	data, err := EncodeCanonical(validEnvelope(snapshot))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	return document
}

func cloneJSONDocument(t *testing.T, source map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var clone map[string]any
	if err = json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}

	return clone
}

func stringVersion(version int) string {
	return "version " + strconv.Itoa(version)
}
