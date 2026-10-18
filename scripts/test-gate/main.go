// Command test-gate inventories every repository package and runs the selected
// required logic/render scope uncached. Raw JSON goes to stdout; diagnostics and
// the package/skip inventory go to stderr. It writes no test-output artifacts.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

var errGate = errors.New("test gate failed")

const (
	goJSONFlag       = "-json"
	allPackages      = "./..."
	goRaceFlagPrefix = "-race="
	goSerialFlag     = "-p=1"
	suiteLogic       = "logic"
	suiteRender      = "render"
	suiteAll         = "all"
)

type options struct {
	race    bool
	short   bool
	strict  bool
	profile string
	suite   string
}

type goRunner func(context.Context, []string, []string, io.Writer, io.Writer) error

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, runGo)
	cancel()
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return
	}
	if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
		os.Exit(1)
	}
	os.Exit(1)
}

func run(ctx context.Context, arguments []string, stdout, stderr io.Writer, runner goRunner) error {
	var settings options
	flags := flag.NewFlagSet("test-gate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.BoolVar(&settings.race, "race", false, "enable the Go race detector")
	flags.BoolVar(&settings.short, "short", false, "run tests in short mode")
	flags.BoolVar(&settings.strict, "strict", false, "require explicit original data and reject every test skip")
	flags.StringVar(&settings.profile, "profile", profileHost, "resource profile: host, constrained or throughput")
	flags.StringVar(&settings.suite, "suite", suiteAll, "required scope: logic, render, or all")
	if err := flags.Parse(arguments); err != nil {
		return fmt.Errorf("parsing test-gate flags: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("%w: package arguments are not accepted; the gate inventories ./... from repository root", errGate)
	}
	if settings.suite != suiteLogic && settings.suite != suiteRender && settings.suite != suiteAll {
		return fmt.Errorf("%w: unknown suite %q; use logic, render, or all", errGate, settings.suite)
	}
	if settings.strict {
		if os.Getenv("ULTIMA5_DATA_PATH") == "" && os.Getenv("ULTIMA5_DATA_DIR") == "" {
			return fmt.Errorf("%w: -strict requires ULTIMA5_DATA_PATH or ULTIMA5_DATA_DIR", errGate)
		}
		if _, err := originaldata.ResolveDirectory(); err != nil {
			return fmt.Errorf("selecting strict original data: %w", err)
		}
	}

	// Last environment entry wins for exec.Cmd, even if the caller inherited
	// provisioned mode. The CLI flag alone selects the lane's skip policy.
	requireData := "ULTIMA5_REQUIRE_DATA=0"
	if settings.strict {
		requireData = "ULTIMA5_REQUIRE_DATA=1"
	}
	environment, err := profileEnvironment(settings.profile, os.Environ())
	if err != nil {
		return err
	}
	environment = append(environment, requireData)
	if err := writeProfile(stderr, settings.profile, environment); err != nil {
		return err
	}
	var listed bytes.Buffer
	if err := runner(ctx, inventoryArguments(settings, false), environment, &listed, stderr); err != nil {
		return fmt.Errorf("collecting expected package inventory: %w", err)
	}
	expected, err := readPackages(&listed)
	if err != nil {
		return err
	}

	return runTests(ctx, settings, environment, expected, stdout, stderr, runner)
}

func runGo(ctx context.Context, arguments, environment []string, stdout, stderr io.Writer) error {
	//nolint:gosec // The executable and all arguments are constructed by this command, not a shell.
	command := exec.CommandContext(ctx, "go", arguments...)
	command.Env = environment
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("go %v: %w", arguments, err)
	}

	return nil
}

func runTests(ctx context.Context, settings options, environment []string, expected map[string]*packageResult,
	stdout, stderr io.Writer, runner goRunner,
) error {
	var listed bytes.Buffer
	arguments := inventoryArguments(settings, true)
	if err := runner(ctx, arguments, environment, &listed, stderr); err != nil {
		return fmt.Errorf("collecting test dependency closures: %w", err)
	}
	lanes, err := partitionPackages(&listed, expected)
	if err != nil {
		return err
	}
	selected := make(map[string]*packageResult)
	for index, lane := range lanes {
		scope := []string{suiteLogic, suiteRender}[index]
		for _, name := range slices.Sorted(maps.Keys(lane)) {
			result := lane[name]
			result.scope = scope
			result.selected = settings.suite == suiteAll || settings.suite == scope
			if result.selected {
				selected[name] = result
			}
		}
	}
	var runErr error
	for index, lane := range lanes {
		if len(lane) == 0 || (settings.suite != suiteAll && settings.suite != []string{suiteLogic, suiteRender}[index]) {
			continue
		}
		if runErr = ctx.Err(); runErr != nil {
			break
		}
		ordinary := make(map[string]*packageResult)
		for _, name := range slices.Sorted(maps.Keys(lane)) {
			result := lane[name]
			if index == 1 && result.cohorts {
				runErr = runNativeCohorts(ctx, settings, environment, name, result, stdout, stderr, runner)
				if runErr != nil {
					break
				}
			} else {
				ordinary[name] = result
			}
		}
		if runErr != nil {
			break
		}
		if len(ordinary) == 0 {
			continue
		}
		parallelism := "1"
		targets := slices.Sorted(maps.Keys(ordinary))
		if index == 0 {
			parallelism = logicParallelism(settings.profile, environment)
		}
		// Explicit flags prevent inherited GOFLAGS from selecting a subset,
		// returning cached results, or raising graphics concurrency.
		arguments := []string{
			"test", goJSONFlag, "-count=1", "-run=.", "-skip=^$", "-list=", "-bench=", "-fuzz=",
			goRaceFlagPrefix + strconv.FormatBool(settings.race), "-short=" + strconv.FormatBool(settings.short),
			"-p=" + parallelism, "-parallel=" + parallelism,
		}
		arguments = append(arguments, targets...)
		runErr = runTestPackages(ctx, arguments, environment, ordinary, stdout, stderr, runner)
		runErr = errors.Join(runErr, validateResults(ordinary, settings.strict))
		if runErr != nil {
			break
		}
	}

	return errors.Join(runErr, validateResults(selected, settings.strict), writeSummary(stderr, expected))
}

func runTestPackages(ctx context.Context, arguments, environment []string, expected map[string]*packageResult,
	stdout, stderr io.Writer, runner goRunner,
) error {
	reader, writer := io.Pipe()
	completed := make(chan error, 1)
	go func() {
		err := runner(ctx, arguments, environment, io.MultiWriter(stdout, writer), stderr)
		completed <- errors.Join(err, writer.Close())
	}()
	decodeErr := readEvents(reader, expected)
	// A malformed event must not truncate later compiler/test diagnostics or
	// strand go test on a blocked pipe. Keep streaming and always reap the child.
	_, drainErr := io.Copy(io.Discard, reader)
	closeErr := reader.Close()
	commandErr := <-completed

	return errors.Join(decodeErr, drainErr, closeErr, commandErr)
}
