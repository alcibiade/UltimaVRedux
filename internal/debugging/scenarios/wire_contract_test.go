package scenarios

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This checks the actual TypeScript AST, rather than a second hand-maintained
// fixture. Go-only installations can still test history without web dependencies.
func TestRunRecordMatchesTypeScriptWireContract(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("ULTIMA5_REQUIRE_WEB_CONTRACT") == "1" {
			t.Fatalf("required scenario TypeScript contract: Node unavailable: %v", err)
		}
		t.Skip("scenario TypeScript contract requires Node and installed Workbench dependencies")
	}
	client, err := filepath.Abs("../webui/client")
	require.NoError(t, err)
	if _, err = os.Stat(filepath.Join(client, "node_modules/typescript/lib/typescript.js")); os.IsNotExist(err) {
		if os.Getenv("ULTIMA5_REQUIRE_WEB_CONTRACT") == "1" {
			t.Fatalf("required scenario TypeScript contract: install Workbench dependencies: %v", err)
		}
		t.Skip("scenario TypeScript contract requires installed Workbench dependencies")
	}
	require.NoError(t, err)

	unknown := testRunRecord("unknown", time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	known := unknown
	known.RunID = "known"
	known.State = RunCancelled
	known.Launch = &RunLaunch{
		ParameterValues: map[string]json.RawMessage{"flag": json.RawMessage(`false`), "count": json.RawMessage(`0`)},
		BaselineApplied: true, RNG: &RunRNG{Seed: "18446744073709551615", SourceCalls: "9007199254740993"},
	}
	empty := unknown
	empty.RunID = "empty"
	empty.Launch = &RunLaunch{ParameterValues: map[string]json.RawMessage{}}
	wire, err := json.Marshal([]RunRecord{unknown, known, empty})
	require.NoError(t, err)

	command := exec.Command(node, "-e", scenarioTypeScriptContractProbe)
	command.Dir = client
	command.Stdin = bytes.NewReader(wire)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	var actual struct {
		Record map[string]bool `json:"record"`
		Launch map[string]bool `json:"launch"`
		RNG    map[string]bool `json:"rng"`
		Types  []string        `json:"types"`
		States []string        `json:"states"`
		Runs   json.RawMessage `json:"runs"`
	}
	require.NoError(t, json.Unmarshal(output, &actual))
	assert.Equal(t, scenarioWireOptionalFields(RunRecord{}), actual.Record)
	assert.Equal(t, scenarioWireOptionalFields(RunLaunch{}), actual.Launch)
	assert.Equal(t, scenarioWireOptionalFields(RunRNG{}), actual.RNG)
	assert.Equal(t, []string{"Record<string,unknown>", "boolean", "string", "string", "ScenarioRunState"}, actual.Types)
	states := []string{string(RunQueued), string(RunRunning), string(RunSucceeded), string(RunFailed), string(RunCancelled)}
	sort.Strings(states)
	assert.Equal(t, states, actual.States)
	assert.JSONEq(t, string(wire), string(actual.Runs), "browser JSON must retain decimal strings, false, zero, empty object, and absent launch")
}

func scenarioWireOptionalFields(value any) map[string]bool {
	fields := make(map[string]bool)
	typeOf := reflect.TypeOf(value)
	for index := range typeOf.NumField() {
		parts := strings.Split(typeOf.Field(index).Tag.Get("json"), ",")
		fields[parts[0]] = len(parts) > 1 && parts[1] == "omitempty"
	}

	return fields
}

const scenarioTypeScriptContractProbe = `
const fs = require("node:fs");
const ts = require("typescript");
const source = ts.createSourceFile("scenarios.ts", fs.readFileSync("src/scenarios.ts", "utf8"), ts.ScriptTarget.Latest, true);
const record = source.statements.find(node => ts.isInterfaceDeclaration(node) && node.name.text === "ScenarioRunStatus");
const member = (node, name) => node.members.find(field => field.name.getText(source) === name);
const fields = node => Object.fromEntries(node.members.map(field => [field.name.getText(source), Boolean(field.questionToken)]));
const launch = member(record, "launch").type;
const rng = member(launch, "rng").type;
const type = (node, name) => member(node, name).type.getText(source).replace(/\s/g, "");
const states = source.statements.find(node => ts.isTypeAliasDeclaration(node) && node.name.text === "ScenarioRunState");
process.stdout.write(JSON.stringify({
  record: fields(record), launch: fields(launch), rng: fields(rng),
  types: [type(launch, "parameterValues"), type(launch, "baselineApplied"), type(rng, "seed"), type(rng, "sourceCalls"), type(record, "state")],
  states: states.type.types.map(node => node.literal.text).sort(),
  runs: JSON.parse(fs.readFileSync(0, "utf8")),
}));
`
