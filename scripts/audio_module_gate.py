#!/usr/bin/env python3
"""Validate the game-owned audio bridge against the application's published SDK pin."""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parent.parent
MODULE = "github.com/bradhannah/berad-sdk/modules/berad-audio"
TOOLS = "github.com/bradhannah/berad-sdk/tools/berad-audio"


def run(arguments, cwd, environment, capture=False):
    print(f"Berad Audio: {arguments!r}", flush=True)
    try:
        return subprocess.run(arguments, cwd=cwd, env=environment, check=True,
                              text=True, stdout=subprocess.PIPE if capture else None).stdout
    except subprocess.CalledProcessError as error:
        if capture and error.stdout:
            print(error.stdout, flush=True)
        raise


def validate(lint=False, race=False):
    if (ROOT / "modules/berad-audio/go.mod").exists():
        raise ValueError("Berad Audio runtime belongs in SDK, not a duplicate game module")
    environment = dict(os.environ, GOWORK="off", GOPROXY="off", GOSUMDB="off",
                       GOTOOLCHAIN="local")
    application = json.loads(run(["go", "mod", "edit", "-json"], ROOT, environment, True))
    if application.get("Replace"):
        raise ValueError("published game must not rely on local module replacements")
    versions = {module:sdk_version(application,module) for module in (MODULE,TOOLS)}
    with tempfile.TemporaryDirectory(prefix="berad-audio-") as directory:
        root = Path(directory)
        bridge = root / "redux-consumer"
        shutil.copytree(ROOT / "testdata/berad-audio-redux", bridge)
        # This game-owned fixture substitutes the candidate game alone. The SDK
        # resolves normally; its independent tests are owned by its repository.
        metadata = json.loads(run(["go", "mod", "edit", "-json"], bridge, environment, True))
        if metadata.get("Replace") or any(sdk_version(metadata,module)!=version for module,version in versions.items()):
            raise ValueError("bridge must declare the game SDK pin without checked-in replacements")
        run(["go", "mod", "edit", "-replace=github.com/bradhannah/UltimaVRedux=" + str(ROOT)], bridge, environment)
        bridge_environment = dict(environment, BERAD_AUDIO_REDUX_ROOT=str(ROOT))
        run(["go", "list", "-mod=mod", "-deps", "-test", "./..."], bridge, bridge_environment, True)
        bridge_metadata = json.loads(run(["go", "mod", "edit", "-json"], bridge, bridge_environment, True))
        verify_bridge_replacements(bridge_metadata)
        for module,version in versions.items():
            selected = json.loads(run(["go", "list", "-mod=readonly", "-m", "-json", module],
                                      bridge, bridge_environment, True))
            verify_selected_sdk(selected, version,module)
        if lint:
            lint_module(bridge, bridge_environment)
            return
        check_tests(bridge, bridge_environment,
                    {"github.com/bradhannah/UltimaVRedux/testdata/berad-audio-redux"}, race)
    print(f"Berad Audio: game bridge passed against {versions}; race={race}", flush=True)


def sdk_version(metadata,module=MODULE):
    versions = [item["Version"] for item in metadata.get("Require", []) if item["Path"] == module]
    if len(versions) != 1 or not versions[0] or versions[0] == "v0.0.0":
        raise ValueError("Berad Audio requires one concrete published SDK version")
    return versions[0]


def verify_selected_sdk(selected, version,module=MODULE):
    if selected.get("Path") != module or selected.get("Version") != version or selected.get("Replace"):
        raise ValueError("bridge must resolve exactly the published application SDK pin")


def verify_bridge_replacements(metadata):
    replacements = {(item["Old"]["Path"], item["New"]["Path"],
                     item["Old"].get("Version", ""), item["New"].get("Version", ""))
                    for item in metadata.get("Replace", [])}
    expected = {("github.com/bradhannah/UltimaVRedux", str(ROOT), "", "")}
    if replacements != expected or len(metadata.get("Replace", [])) != 1:
        raise ValueError("game bridge must substitute only the reviewed candidate game")


def lint_module(directory, environment):
    # Analyzer caches contain absolute filenames. A cache from a prior temporary
    # copy can otherwise emit findings for deleted paths and miss nolint context.
    environment = dict(environment, GOLANGCI_LINT_CACHE=str(directory.parent / "lint-cache" / directory.name))
    concurrency = []
    if environment.get("TEST_PROFILE", "host") != "host":
        concurrency = ["--concurrency=1"]
    elif environment.get("GOMAXPROCS", "").isdigit() and int(environment["GOMAXPROCS"]) > 0:
        concurrency = ["--concurrency=" + environment["GOMAXPROCS"]]
    # The copied fixture substitutes the reviewed game candidate, checked above.
    # Published game metadata retains the blanket replacement prohibition.
    run([str(ROOT / "bin/tools/golangci-lint"), "run", *concurrency, "--disable=gomoddirectives",
         "--config", str(ROOT / ".golangci.yml"),
         "--new=false", "--new-from-rev=", "--max-issues-per-linter=0", "--max-same-issues=0",
         "--uniq-by-line=false", "./..."], directory, environment)


def check_tests(directory, environment, expected, race):
    output = run(["go", "test", "-mod=readonly", "-json", "-count=1", f"-race={str(race).lower()}",
                  "-run=.", "-skip=^$", "-list=", "-bench=", "-fuzz=", "-short=false", "./..."],
                 directory, environment, True)
    events = [json.loads(line) for line in output.splitlines() if line.strip()]
    passed = {item["Package"] for item in events if item.get("Action") == "pass" and not item.get("Test")}
    skipped = [item for item in events if item.get("Action") == "skip"]
    if passed != expected or skipped:
        raise ValueError(f"incomplete module tests: expected={expected}, passed={passed}, skipped={skipped}")
    tests = sum(item.get("Action") == "pass" and bool(item.get("Test")) for item in events)
    if not tests:
        raise ValueError("game bridge ran no assertions")
    print(f"Berad Audio: {len(passed)} packages, {tests} tests/subtests, zero skips", flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    modes = parser.add_mutually_exclusive_group()
    modes.add_argument("--lint", action="store_true")
    modes.add_argument("--race", action="store_true")
    options = parser.parse_args()
    validate(lint=options.lint, race=options.race)
