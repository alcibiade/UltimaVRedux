package main

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDiagnosticConfigPreservesPolicy(t *testing.T) {
	source, err := os.ReadFile("../../.golangci.yml")
	if err != nil {
		t.Fatal(err)
	}
	result, err := diagnosticConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	var original, diagnostic map[string]any
	if err := yaml.Unmarshal(source, &original); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(result, &diagnostic); err != nil {
		t.Fatal(err)
	}
	issues, ok := diagnostic["issues"].(map[string]any)
	if !ok {
		t.Fatal("diagnostic issues mapping missing")
	}
	if _, exists := issues["new-from-rev"]; exists {
		t.Fatal("diagnostic copy retains new-from-rev")
	}
	issues["new-from-rev"] = original["issues"].(map[string]any)["new-from-rev"]
	if !reflect.DeepEqual(original, diagnostic) {
		t.Fatal("diagnostic copy changed policy beyond removing new-from-rev")
	}
}

func TestDiagnosticConfigRejectsUnexpectedPolicy(t *testing.T) {
	for _, source := range []string{
		"[", "{}", "issues: false", "issues: {new-from-rev: HEAD}",
		"issues: {new-from-rev: origin/main}", "issues: {new-from-rev: f31ba69}",
		"issues: {new-from-rev: g31ba69759594face92838f0ce33855f46aca4e3}",
	} {
		t.Run(source, func(t *testing.T) {
			if _, err := diagnosticConfig([]byte(source)); err == nil {
				t.Fatal("accepted invalid or unexpected policy")
			}
		})
	}
}
