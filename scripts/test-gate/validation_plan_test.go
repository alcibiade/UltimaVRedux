package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationPlanRunsGoChecksOnce(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	command := exec.Command("make", "--no-print-directory", "-n", "validate", "GO=go", "PYTHON=python3")
	command.Dir = root
	command.Env = cleanMakeEnvironment()
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	plan := string(output)
	for _, step := range []string{
		"go run -p=1 ./scripts/test-gate", "go vet -p=1 ./...", "go build -p=1 ./...",
		"go run -p=1 ./scripts/check-lint-config", "go run -p=1 ./scripts/check-documentation",
		"./scripts/pre-commit-checks.sh --policy-only", "python3 scripts/check-gamefacts.py",
		"python3 -m unittest discover -s scripts",
	} {
		require.Equal(t, 1, strings.Count(plan, step), "validation must schedule %q exactly once", step)
	}
	require.NotContains(t, plan, "go test -short")
	require.NotContains(t, plan, "opencode")
	require.NotContains(t, plan, "go generate")
	require.NotContains(t, plan, "npm ")
	require.Contains(t, plan, "-suite=logic")
	require.NotContains(t, plan, "-suite=render")
	require.NotContains(t, plan, "-suite=all")

	command = exec.Command("make", "--no-print-directory", "-n", "pre-commit")
	command.Dir = root
	command.Env = cleanMakeEnvironment()
	output, err = command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.Equal(t, "./scripts/pre-commit-checks.sh", strings.TrimSpace(string(output)),
		"standalone pre-commit must retain its complete check path")
}

func TestBuildUsesShippedAssets(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-n", "build", "GO=go")
	command.Dir = root
	command.Env = cleanMakeEnvironment()
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	plan := string(output)
	require.Contains(t, plan, "go build -o bin/ultimav ./cmd/ultimav")
	for _, generation := range []string{"go generate", "npm", "web-build", "python", "opencode"} {
		require.NotContains(t, plan, generation)
	}
}

func TestQualityGateSchedulesRequiredScopesWithoutAggregateRepeat(t *testing.T) {
	contents, err := os.ReadFile("../quality-gate.sh")
	require.NoError(t, err)
	plan := string(contents)
	for _, step := range []string{"make validate\n", "make web-build\n", "make web-test-browser\n"} {
		require.Equal(t, 1, strings.Count(plan, step), "quality gate must schedule %q once", step)
	}
	// Each platform branch executes one native pass and one debug race pass.
	require.Equal(t, 1, strings.Count(plan, "xvfb-run -a make test-render\n"))
	require.Equal(t, 1, strings.Count(plan, "    make test-render\n"))
	require.Equal(t, 2, strings.Count(plan, "-parallel=1 -race ./internal/debugging/..."))
	require.Less(t, strings.Index(plan, "make validate"), strings.Index(plan, "make test-render"))
	require.Less(t, strings.Index(plan, "make test-render"), strings.Index(plan, "make web-build"))
	for _, duplicate := range []string{"make test-all", "make test-provisioned", "go test ./...", "make check"} {
		require.NotContains(t, plan, duplicate)
	}
}

func TestFormatCheckPreservesToolFailure(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "format-check", "GOIMPORTS=false")
	command.Dir = root
	command.Env = cleanMakeEnvironment()
	output, err := command.CombinedOutput()
	require.Error(t, err, "a failed formatting tool must fail validation: %s", output)
}

func cleanMakeEnvironment() []string {
	var environment []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "MAKEFLAGS", "MFLAGS", "MAKEOVERRIDES", "GOFLAGS", "GOENV", "TEST_PROFILE", "TEST_GO_BUILD_FLAGS",
			"GOMAXPROCS", "GOMEMLIMIT", "ULTIMA5_REQUIRE_DATA":
			continue
		}
		environment = append(environment, entry)
	}
	return environment
}

func TestMakeForwardsProfilesAndStrictRaceFlags(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	for _, profile := range []string{profileConstrained, profileThroughput} {
		for _, target := range []string{"test", "test-render", "test-all", "test-race", "test-provisioned", "test-race-provisioned"} {
			t.Run(profile+"/"+target, func(t *testing.T) {
				command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-n", target, "GO=go", "TEST_PROFILE="+profile)
				command.Dir = root
				command.Env = append(cleanMakeEnvironment(), "MAKEFLAGS=-j8", "ULTIMA5_REQUIRE_DATA=1")
				output, err := command.CombinedOutput()
				require.NoError(t, err, "%s", output)
				plan := string(output)
				require.Equal(t, 1, strings.Count(plan, "go run -p=1 ./scripts/test-gate"))
				require.Contains(t, plan, "go run -p=1 ./scripts/test-gate -profile="+profile+" -strict")
				suite := "all"
				if target == "test" {
					suite = "logic"
				} else if target == "test-render" {
					suite = "render"
				}
				require.Contains(t, plan, "-suite="+suite)
				if strings.Contains(target, "race") {
					require.Contains(t, plan, " -race")
				} else {
					require.NotContains(t, plan, " -race")
				}
			})
		}
	}
}

func TestMakeProfileEnvironmentAndPreflight(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	probe := filepath.Join(t.TempDir(), "probe.mk")
	// Replace only the test recipe with a tiny shell probe. No Go command runs.
	require.NoError(t, os.WriteFile(probe, []byte(`test:
	@printf 'PROFILE=%s CPU=%s MEMORY=%s\nGOFLAGS=%s\nBUILD_FLAGS=%s\n' "$$TEST_PROFILE" "$$GOMAXPROCS" "$$GOMEMLIMIT" "$$GOFLAGS" "$(TEST_GO_BUILD_FLAGS)"
`), 0o600))
	const callerFlags = "-tags='one two' -mod=readonly -p=9 -parallel=9"
	for _, test := range []struct {
		name        string
		arguments   []string
		environment []string
		want        string
		fail        bool
	}{
		{name: "default", want: "PROFILE=constrained CPU=2 MEMORY=256MiB"},
		{name: "throughput", arguments: []string{"TEST_PROFILE=throughput"}, want: "PROFILE=throughput CPU=4 MEMORY=2GiB"},
		{name: "exported selection", environment: []string{"TEST_PROFILE=throughput"}, want: "PROFILE=throughput CPU=4 MEMORY=2GiB"},
		{name: "command-line build flags", arguments: []string{"GOFLAGS=" + callerFlags}, want: "PROFILE=constrained CPU=2 MEMORY=256MiB"},
		{name: "matching explicit", environment: []string{"GOMAXPROCS=2", "GOMEMLIMIT=256MiB"}, want: "PROFILE=constrained CPU=2 MEMORY=256MiB"},
		{name: "lower CPU", environment: []string{"GOMAXPROCS=1"}, want: "requires GOMAXPROCS=2", fail: true},
		{name: "higher memory", environment: []string{"GOMEMLIMIT=512MiB"}, want: "requires GOMEMLIMIT=256MiB", fail: true},
		{name: "higher CPU", arguments: []string{"GOMAXPROCS=8"}, want: "requires GOMAXPROCS=2", fail: true},
		{name: "empty memory", environment: []string{"GOMEMLIMIT="}, want: "requires GOMEMLIMIT=256MiB", fail: true},
		{name: "Windows lowercase memory", environment: []string{"OS=Windows_NT", "gomemlimit=512MiB"}, want: "Windows resource variables must use uppercase names", fail: true},
		{name: "Windows mixed CPU", environment: []string{"OS=Windows_NT", "GoMaxProcs=1"}, want: "Windows resource variables must use uppercase names", fail: true},
		{name: "wrong profile", arguments: []string{"TEST_PROFILE=typo"}, want: "Unknown TEST_PROFILE", fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			arguments := append([]string{"--no-print-directory", "-f", "Makefile", "-f", probe, "test"}, test.arguments...)
			command := exec.CommandContext(t.Context(), "make", arguments...)
			command.Dir = root
			command.Env = append(cleanMakeEnvironment(), "MAKEFLAGS=-j8", "GOFLAGS="+callerFlags)
			command.Env = append(command.Env, test.environment...)
			output, err := command.CombinedOutput()
			if test.fail {
				require.Error(t, err)
				require.NotContains(t, string(output), " CPU=", "the workload must not start before preflight succeeds")
			} else {
				require.NoError(t, err, "%s", output)
				require.Contains(t, string(output), "GOFLAGS="+callerFlags+"\nBUILD_FLAGS=-p=1\n")
			}
			require.Contains(t, string(output), test.want)
		})
	}
}

func TestMakeProfileDoesNotChangeGameEnvironment(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	probe := filepath.Join(t.TempDir(), "probe.mk")
	require.NoError(t, os.WriteFile(probe, []byte(`run:
	@printf 'CPU=%s MEMORY=%s\nGOFLAGS=%s\nBUILD_FLAGS=%s\n' "$$GOMAXPROCS" "$$GOMEMLIMIT" "$$GOFLAGS" "$(TEST_GO_BUILD_FLAGS)"
`), 0o600))
	const callerFlags = "-tags='one two' -mod=readonly -p=9 -parallel=9"
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-f", "Makefile", "-f", probe,
		"run", "TEST_PROFILE=throughput", "GOFLAGS="+callerFlags)
	command.Dir = root
	command.Env = append(cleanMakeEnvironment(), "MAKEFLAGS=-j8", "GOMAXPROCS=1", "GOMEMLIMIT=512MiB")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.Contains(t, string(output), "CPU=1 MEMORY=512MiB\nGOFLAGS="+callerFlags+"\nBUILD_FLAGS=\n")
}

func TestMakeDoesNotShadowPersistedGoFlags(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	directory := t.TempDir()
	goenv := filepath.Join(directory, "goenv")
	require.NoError(t, os.WriteFile(goenv, []byte("GOFLAGS=-tags=profilefixture\n"), 0o600))
	probe := filepath.Join(directory, "probe.mk")
	require.NoError(t, os.WriteFile(probe, []byte(`test:
	@printf 'GOENV=%s\nGOFLAGS_SET=%s\nBUILD_FLAGS=%s\n' "$$GOENV" "$${GOFLAGS+x}" "$(TEST_GO_BUILD_FLAGS)"
`), 0o600))
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-f", "Makefile", "-f", probe, "test")
	command.Dir = root
	command.Env = append(cleanMakeEnvironment(), "MAKEFLAGS=-j8", "GOENV="+goenv)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.Contains(t, string(output), "GOENV="+goenv+"\nGOFLAGS_SET=\nBUILD_FLAGS=-p=1\n",
		"Make must not synthesize GOFLAGS and hide the selected Go environment file")
}

func TestMakeSerializesPrerequisitesDespiteInheritedJobs(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	probe := filepath.Join(t.TempDir(), "probe.mk")
	require.NoError(t, os.WriteFile(probe, []byte(`.PHONY: profile-order profile-first profile-second
profile-order: profile-first profile-second
profile-first:
	@printf 'first-start\n'; sleep 0.02; printf 'first-end\n'
profile-second:
	@printf 'second-start\n'; sleep 0.02; printf 'second-end\n'
`), 0o600))
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-f", "Makefile", "-f", probe, "profile-order")
	command.Dir = root
	command.Env = append(cleanMakeEnvironment(), "MAKEFLAGS=-j8")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.Equal(t, "first-start\nfirst-end\nsecond-start\nsecond-end\n", string(output))
}
