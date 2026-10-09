package save_snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestNativeSaveSchemasCompileIndependentlyWithLocalReferences(t *testing.T) {
	t.Parallel()
	for _, version := range []int{12, 13, 14, 15} {
		t.Run(stringVersion(version), func(t *testing.T) {
			t.Parallel()
			_, document := compileNativeSaveSchema(t, version)
			walkJSONSchema(t, document, func(reference string) {
				if !strings.HasPrefix(reference, "#/$defs/") {
					t.Errorf("version %d schema contains non-local $ref %q", version, reference)
				}
			})
		})
	}
}

func TestNativeSaveSchemasAcceptTheirCanonicalWireShapes(t *testing.T) {
	t.Parallel()
	v12, _ := compileNativeSaveSchema(t, 12)
	v13, _ := compileNativeSaveSchema(t, 13)
	v14, _ := compileNativeSaveSchema(t, 14)
	v15, _ := compileNativeSaveSchema(t, 15)

	snapshot := validSnapshot()
	snapshot.Redux.PlayerKnowledge = append(validLoreKnowledge(), validPlayerKnowledge()...)
	v12Data := versionTwelveEnvelopeData(t, snapshot, nil)
	v13Data := versionThirteenEnvelopeData(t, snapshot, nil)
	v14Data := versionFourteenEnvelopeData(t, snapshot, nil)
	v15Data := versionFifteenEnvelopeData(t, snapshot, nil)
	v12Instance := archivedEnvelopeOneSchemaInstance(t, v12Data)
	v13Instance := archivedEnvelopeOneSchemaInstance(t, v13Data)
	v14Instance := archivedEnvelopeOneSchemaInstance(t, v14Data)
	v15Instance := archivedEnvelopeOneSchemaInstance(t, v15Data)
	if err := v12.Validate(v12Instance); err != nil {
		t.Fatalf("standalone version 12 schema rejected canonical v12: %v", err)
	}
	if err := v13.Validate(v13Instance); err != nil {
		t.Fatalf("standalone version 13 schema rejected canonical v13: %v", err)
	}
	if err := v14.Validate(v14Instance); err != nil {
		t.Fatalf("standalone version 14 schema rejected canonical v14: %v", err)
	}
	if err := v15.Validate(v15Instance); err != nil {
		t.Fatalf("standalone version 15 schema rejected canonical v15: %v", err)
	}
	if err := v14.Validate(v15Instance); err == nil {
		t.Fatal("version 14 schema accepted version 15 wire JSON")
	}
	if err := v15.Validate(v14Instance); err == nil {
		t.Fatal("version 15 schema accepted version 14 wire JSON")
	}
	if err := v12.Validate(v13Instance); err == nil {
		t.Fatal("version 12 schema accepted version 13 wire JSON")
	}
	if err := v13.Validate(v12Instance); err == nil {
		t.Fatal("version 13 schema accepted version 12 wire JSON")
	}
}

func TestNativeSaveV14SchemaRequiresSemanticChestState(t *testing.T) {
	t.Parallel()
	compiled, _ := compileNativeSaveSchema(t, 14)
	data := versionFourteenEnvelopeData(t, validSnapshot(), nil)
	instance := archivedEnvelopeOneSchemaInstance(t, data)
	unit := instance["snapshot"].(map[string]any)["mapUnits"].([]any)[0].(map[string]any)
	unit["kind"] = "chest"
	unit["referenceId"] = float64(0)
	unit["enemyVariant"] = ""
	unit["enemyCurrentHp"] = float64(0)
	unit["enemyFacing"] = float64(0)
	unit["shadowlord"] = nil
	unit["surfaceChestNumber"] = float64(0x9e)
	if err := compiled.Validate(instance); err != nil {
		t.Fatalf("version 14 schema rejected semantic chest: %v", err)
	}
	delete(unit, "surfaceChestNumber")
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("version 14 schema accepted semantic chest without state")
	}
}

func TestNativeSaveV12SchemaPreservesHistoricalReconstructionSemantics(t *testing.T) {
	t.Parallel()
	compiled, _ := compileNativeSaveSchema(t, 12)
	withUnits := archivedEnvelopeOneSchemaInstance(t, versionTwelveEnvelopeData(t, validSnapshot(), func(snapshot map[string]any) {
		snapshot["redux"].(map[string]any)["reconstructMapUnits"] = true
	}))
	if err := compiled.Validate(withUnits); err == nil {
		t.Fatal("version 12 schema accepted reconstruction with semantic map units")
	}

	withoutUnits := cloneJSONSchemaInstance(t, withUnits)
	withoutUnits["snapshot"].(map[string]any)["mapUnits"] = nil
	withoutUnits["snapshot"].(map[string]any)["itemStacks"] = nil
	withoutUnits["snapshot"].(map[string]any)["mapMutations"] = []any{}
	if err := compiled.Validate(withoutUnits); err != nil {
		t.Fatalf("version 12 schema rejected valid reconstruction and collection nilness: %v", err)
	}

	missing := cloneJSONSchemaInstance(t, withoutUnits)
	delete(missing["snapshot"].(map[string]any)["redux"].(map[string]any), "reconstructMapUnits")
	if err := compiled.Validate(missing); err == nil {
		t.Fatal("version 12 schema accepted missing reconstructMapUnits")
	}

	withContinuation := cloneJSONSchemaInstance(t, withoutUnits)
	withContinuation["snapshot"].(map[string]any)["continuation"] = map[string]any{
		"activePlayerIndex": -1, "isCriminal": false,
		"hmsCapeAutomaticWaitActorSkip": false, "beggarKarmaCounter": 100,
	}
	if err := compiled.Validate(withContinuation); err == nil {
		t.Fatal("version 12 schema accepted a version 13 continuation member")
	}
}

func TestNativeSaveV13SchemaRequiresContinuationMembersIncludingBooleans(t *testing.T) {
	t.Parallel()
	compiled, _ := compileNativeSaveSchema(t, 13)
	data := versionThirteenEnvelopeData(t, validSnapshot(), nil)
	base := archivedEnvelopeOneSchemaInstance(t, data)

	for _, member := range []string{"activePlayerIndex", "isCriminal", "hmsCapeAutomaticWaitActorSkip", "beggarKarmaCounter"} {
		t.Run("missing "+member, func(t *testing.T) {
			instance := cloneJSONSchemaInstance(t, base)
			delete(instance["snapshot"].(map[string]any)["continuation"].(map[string]any), member)
			if err := compiled.Validate(instance); err == nil {
				t.Fatalf("version 13 schema accepted continuation without %s", member)
			}
		})
	}
	for _, member := range []string{"isCriminal", "hmsCapeAutomaticWaitActorSkip"} {
		t.Run("non-boolean "+member, func(t *testing.T) {
			instance := cloneJSONSchemaInstance(t, base)
			instance["snapshot"].(map[string]any)["continuation"].(map[string]any)[member] = 0
			if err := compiled.Validate(instance); err == nil {
				t.Fatalf("version 13 schema accepted non-boolean %s", member)
			}
		})
	}

	missing := cloneJSONSchemaInstance(t, base)
	delete(missing["snapshot"].(map[string]any), "continuation")
	if err := compiled.Validate(missing); err == nil {
		t.Fatal("version 13 schema accepted a snapshot without continuation")
	}
}

func TestNativeSaveV13SchemaValidatesContinuationRanges(t *testing.T) {
	t.Parallel()
	compiled, _ := compileNativeSaveSchema(t, 13)
	data := versionThirteenEnvelopeData(t, validSnapshot(), nil)
	base := archivedEnvelopeOneSchemaInstance(t, data)

	for _, value := range []int{-1, 0, 5} {
		instance := cloneJSONSchemaInstance(t, base)
		instance["snapshot"].(map[string]any)["continuation"].(map[string]any)["activePlayerIndex"] = value
		if err := compiled.Validate(instance); err != nil {
			t.Fatalf("version 13 schema rejected activePlayerIndex=%d: %v", value, err)
		}
	}
	for _, value := range []int{-2, 6} {
		instance := cloneJSONSchemaInstance(t, base)
		instance["snapshot"].(map[string]any)["continuation"].(map[string]any)["activePlayerIndex"] = value
		if err := compiled.Validate(instance); err == nil {
			t.Fatalf("version 13 schema accepted activePlayerIndex=%d", value)
		}
	}
	for _, value := range []int{0, 255} {
		instance := cloneJSONSchemaInstance(t, base)
		instance["snapshot"].(map[string]any)["continuation"].(map[string]any)["beggarKarmaCounter"] = value
		if err := compiled.Validate(instance); err != nil {
			t.Fatalf("version 13 schema rejected beggarKarmaCounter=%d: %v", value, err)
		}
	}
	for _, value := range []int{-1, 256} {
		instance := cloneJSONSchemaInstance(t, base)
		instance["snapshot"].(map[string]any)["continuation"].(map[string]any)["beggarKarmaCounter"] = value
		if err := compiled.Validate(instance); err == nil {
			t.Fatalf("version 13 schema accepted beggarKarmaCounter=%d", value)
		}
	}
}

func TestNativeSaveV13SchemaRejectsHistoricalReconstructionProperty(t *testing.T) {
	t.Parallel()
	compiled, _ := compileNativeSaveSchema(t, 13)
	data := versionThirteenEnvelopeData(t, validSnapshot(), nil)
	instance := archivedEnvelopeOneSchemaInstance(t, data)
	instance["snapshot"].(map[string]any)["redux"].(map[string]any)["reconstructMapUnits"] = false
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("version 13 schema accepted historical reconstructMapUnits")
	}
}

func compileNativeSaveSchema(t *testing.T, version int) (*jsonschema.Schema, any) {
	t.Helper()
	return compileNativeSaveSchemaFile(t, "native-save-v"+strings.TrimPrefix(stringVersion(version), "version ")+".schema.json")
}

func compileCurrentNativeSaveSchema(t *testing.T) (*jsonschema.Schema, any) {
	t.Helper()
	return compileNativeSaveSchemaFile(t, fmt.Sprintf("native-save-e2-s%d.schema.json", CurrentVersion))
}

func compileNativeSaveSchemaFile(t *testing.T, filename string) (*jsonschema.Schema, any) {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate json_schema_test.go")
	}
	path := filepath.Join(filepath.Dir(testFile), "..", "..", "docs", "schemas", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read JSON schema %q: %v", path, err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse JSON schema %q: %v", path, err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		t.Fatalf("JSON schema %q is not an object", path)
	}
	resource, ok := object["$id"].(string)
	if !ok || resource == "" {
		t.Fatalf("JSON schema %q has no $id", path)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	formatVersion := CurrentVersion
	switch filename {
	case "native-save-e2-s16.schema.json":
		formatVersion = 16
	case "native-save-e2-s17.schema.json":
		formatVersion = 17
	case "native-save-e2-s18.schema.json":
		formatVersion = 18
	case "native-save-e2-s19.schema.json":
		formatVersion = 19
	case "native-save-e2-s20.schema.json":
		formatVersion = 20
	case "native-save-e2-s21.schema.json":
		formatVersion = 21
	case "native-save-e2-s22.schema.json":
		formatVersion = 22
	case "native-save-e2-s23.schema.json":
		formatVersion = 23
	}
	registerJournalSchemaFormats(compiler, formatVersion)
	var dependencies []string
	if formatVersion >= 17 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s16.schema.json")
	}
	if formatVersion >= 18 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s17.schema.json")
	}
	if formatVersion >= 19 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s18.schema.json")
	}
	if formatVersion >= 20 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s19.schema.json")
	}
	if formatVersion >= 21 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s20.schema.json")
	}
	if formatVersion >= 22 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s21.schema.json")
	}
	if formatVersion >= 23 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s22.schema.json")
	}
	if formatVersion >= 24 && strings.HasPrefix(filename, "native-save-e2-") {
		dependencies = append(dependencies, "native-save-e2-s23.schema.json")
	}
	for _, dependency := range dependencies {
		frozenData, readErr := os.ReadFile(filepath.Join(filepath.Dir(path), dependency))
		if readErr != nil {
			t.Fatal(readErr)
		}
		frozen, parseErr := jsonschema.UnmarshalJSON(bytes.NewReader(frozenData))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		if addErr := compiler.AddResource(frozen.(map[string]any)["$id"].(string), frozen); addErr != nil {
			t.Fatal(addErr)
		}
	}
	if err = compiler.AddResource(resource, document); err != nil {
		t.Fatalf("add standalone JSON schema %q: %v", path, err)
	}
	compiled, err := compiler.Compile(resource)
	if err != nil {
		t.Fatalf("compile standalone JSON schema %q: %v", path, err)
	}

	return compiled, document
}

func walkJSONSchema(t *testing.T, value any, visitReference func(string)) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "$ref" {
				reference, ok := child.(string)
				if !ok {
					t.Fatalf("JSON schema contains non-string $ref %#v", child)
				}
				visitReference(reference)
			}
			walkJSONSchema(t, child, visitReference)
		}
	case []any:
		for _, child := range typed {
			walkJSONSchema(t, child, visitReference)
		}
	}
}

func decodeJSONSchemaInstance(t *testing.T, data []byte) any {
	t.Helper()
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse JSON schema instance: %v", err)
	}

	return instance
}

// Archived schemas describe envelope 1, independently of current snapshot promotion fixtures.
func archivedEnvelopeOneSchemaInstance(t *testing.T, data []byte) map[string]any {
	t.Helper()
	instance := decodeJSONSchemaInstance(t, data).(map[string]any)
	instance["envelopeVersion"] = 1
	return instance
}

func cloneJSONSchemaInstance(t *testing.T, source map[string]any) map[string]any {
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
