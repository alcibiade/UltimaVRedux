#!/usr/bin/env python3
"""Validate Berad Audio in isolation, including a separate consumer module."""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parent.parent
MODULE = "github.com/bradhannah/UltimaVRedux/modules/berad-audio"


def run(arguments, cwd, environment, capture=False):
    print(f"Berad Audio: {arguments!r}", flush=True)
    try:
        return subprocess.run(arguments, cwd=cwd, env=environment, check=True,
                              text=True, stdout=subprocess.PIPE if capture else None).stdout
    except subprocess.CalledProcessError as error:
        if capture and error.stdout:
            print(error.stdout, flush=True)
        raise


def json_objects(text):
    decoder = json.JSONDecoder()
    while text.strip():
        value, end = decoder.raw_decode(text.lstrip())
        yield value
        text = text.lstrip()[end:]


def validate(lint=False, race=False):
    source = ROOT / "modules" / "berad-audio"
    expected_modules = {source / "go.mod", source / "testdata/consumer/go.mod"}
    if set((ROOT / "modules").rglob("go.mod")) != expected_modules:
        raise ValueError("nested module inventory changed; register its validation explicitly")
    environment = dict(os.environ, GOWORK="off", GOPROXY="off", GOSUMDB="off",
                       GOTOOLCHAIN="local")
    with tempfile.TemporaryDirectory(prefix="berad-audio-") as directory:
        root = Path(directory)
        library = root / "library"
        # Only module contents enter the isolated build, never sibling game code.
        shutil.copytree(source, library, ignore=shutil.ignore_patterns("testdata"))
        metadata = json.loads(run(["go", "mod", "edit", "-json"], library, environment, True))
        if metadata["Module"]["Path"] != MODULE or metadata.get("Replace"):
            raise ValueError("Berad Audio must declare its independent module without replacements")
        dependencies = run(["go", "list", "-mod=readonly", "-deps", "-test", "-json", "./..."],
                           library, environment, True)
        for dependency in json_objects(dependencies):
            path = dependency["ImportPath"]
            owner = dependency.get("Module", {})
            if path.startswith("github.com/bradhannah/UltimaVRedux/") and owner.get("Path") != MODULE:
                raise ValueError(f"Berad Audio imports game code: {path}")
            if owner.get("Replace"):
                raise ValueError(f"Berad Audio relies on a replacement: {path}")
        if lint:
            concurrency = []
            if environment.get("TEST_PROFILE", "host") != "host":
                concurrency = ["--concurrency=1"]
            elif environment.get("GOMAXPROCS", "").isdigit() and int(environment["GOMAXPROCS"]) > 0:
                concurrency = ["--concurrency=" + environment["GOMAXPROCS"]]
            run([str(ROOT / "bin/tools/golangci-lint"), "run", *concurrency,
                 "--config", str(ROOT / ".golangci.yml"),
                 "--new=false", "--new-from-rev=", "--max-issues-per-linter=0", "--max-same-issues=0",
                 "--uniq-by-line=false", "./..."], library, environment)
            return
        expected = {item["ImportPath"] for item in json_objects(dependencies)
                    if item.get("Module", {}).get("Path") == MODULE and not item.get("ForTest")
                    and not item["ImportPath"].endswith(".test")}
        check_tests(library, environment, expected, race)
        run(["go", "vet", "-mod=readonly", "./..."], library, environment)
        run(["go", "build", "-mod=readonly", "./..."], library, environment)
        consumer = root / "consumer"
        shutil.copytree(source / "testdata" / "consumer", consumer)
        # This throwaway harness alone replaces the dependency with the isolated
        # library. It has no Redux tree and cannot mask undeclared game imports.
        check_tests(consumer, environment, {"example.com/berad-audio-consumer"}, race, "mod")
    print(f"Berad Audio: isolated module test/vet/build and external consumer passed; race={race}", flush=True)


def check_tests(directory, environment, expected, race, mode="readonly"):
    output = run(["go", "test", f"-mod={mode}", "-json", "-count=1", f"-race={str(race).lower()}",
                  "-run=.", "-skip=^$", "-list=", "-bench=", "-fuzz=", "-short=false", "./..."],
                 directory, environment, True)
    events = [json.loads(line) for line in output.splitlines() if line.strip()]
    passed = {item["Package"] for item in events if item.get("Action") == "pass" and not item.get("Test")}
    skipped = [item for item in events if item.get("Action") == "skip"]
    if passed != expected or skipped:
        raise ValueError(f"incomplete module tests: expected={expected}, passed={passed}, skipped={skipped}")
    tests = sum(item.get("Action") == "pass" and bool(item.get("Test")) for item in events)
    print(f"Berad Audio: {len(passed)} packages, {tests} tests/subtests, zero skips", flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    modes = parser.add_mutually_exclusive_group()
    modes.add_argument("--lint", action="store_true")
    modes.add_argument("--race", action="store_true")
    options = parser.parse_args()
    validate(lint=options.lint, race=options.race)
