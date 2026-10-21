package main

import (
	"os"
	"testing"
)

func TestRepositoryConfigLintsWholeTree(t *testing.T) {
	source, err := os.ReadFile("../../.golangci.yml")
	if err != nil {
		t.Fatal(err)
	}
	if err := requireWholeTreeLint(source); err != nil {
		t.Fatal(err)
	}
}

func TestRequireWholeTreeLintRejectsCutoffs(t *testing.T) {
	for _, source := range []string{
		"[", "{}", "issues: false", "issues: {new: true}",
		"issues: {new-from-rev: f31ba69759594face92838f0ce33855f46aca4e3}",
		"issues: {new-from-patch: changes.diff}", "issues: {new-from-merge-base: main}",
	} {
		t.Run(source, func(t *testing.T) {
			if err := requireWholeTreeLint([]byte(source)); err == nil {
				t.Fatal("accepted a config that does not lint the whole tree")
			}
		})
	}
}
