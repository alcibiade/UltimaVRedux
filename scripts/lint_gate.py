#!/usr/bin/env python3
"""Lint the exact requested inventory with host concurrency or serial fallback."""

import argparse
import os
from pathlib import Path
import subprocess
import sys

BATCH_SIZE = 4


def host_concurrency(environment):
    # Otherwise leave the pinned linter's quota-aware Go CPU auto-detection on.
    try:
        cpus = int(environment.get("GOMAXPROCS", ""))
    except ValueError:
        return []
    return [f"--concurrency={cpus}"] if cpus > 0 else []


def run_inventory(linter, packages, all_findings=False, go="go", runner=subprocess.run,
                  profile="host", environment=None):
    if profile not in ("host", "constrained", "throughput"):
        raise ValueError(f"Unknown lint resource profile: {profile}")
    environment = os.environ if environment is None else environment
    listed = runner([go, "list", "-f", "{{.Dir}}", *packages], check=True, capture_output=True, text=True)
    inventory = listed.stdout.splitlines()
    if not inventory or len(inventory) != len(set(inventory)) or any(not name.strip() for name in inventory):
        raise ValueError("Go package inventory must be nonempty and unique")
    flags = host_concurrency(environment) if profile == "host" else ["--concurrency=1"]
    if all_findings:
        flags += ["--new=false", "--new-from-rev=", "--new-from-patch=", "--new-from-merge-base=",
                  "--max-issues-per-linter=0", "--max-same-issues=0", "--uniq-by-line=false"]
    result = 0
    visited = []
    batch_size = len(inventory) if profile == "host" else BATCH_SIZE
    batch_flags = [] if profile == "host" else ["--disable=dupl"]
    concurrency = flags[0].split("=", 1)[1] if flags and flags[0].startswith("--concurrency=") else "auto (available Go CPUs)"
    print(f"Lint profile {profile}: concurrency={concurrency}; "
          f"GOMAXPROCS={environment.get('GOMAXPROCS') or 'auto'!r} "
          f"GOMEMLIMIT={environment.get('GOMEMLIMIT') or 'off (Go default)'!r} "
          f"GOGC={environment.get('GOGC') or '100 (Go default)'!r}", flush=True)
    for offset in range(0, len(inventory), batch_size):
        batch = inventory[offset:offset + batch_size]
        print(f"Lint packages {offset + 1}-{offset + len(batch)}/{len(inventory)}", flush=True)
        completed = runner([str(Path(linter).resolve()), "run", *flags, *batch_flags, *batch], check=False)
        if completed.returncode < 0:
            return 128 - completed.returncode
        result = max(result, completed.returncode)
        visited.extend(batch)
    if visited != inventory:
        raise ValueError("lint workload did not cover the exact inventory")
    # Host already checks cross-package duplicates in its single full run.
    # Serial fallback defers dupl from batches to exactly one global run.
    if profile != "host":
        global_result = runner([str(Path(linter).resolve()), "run", *flags,
                                "--enable-only=dupl", *inventory], check=False)
        if global_result.returncode < 0:
            return 128 - global_result.returncode
        result = max(result, global_result.returncode)
    print(f"Lint inventory: {len(visited)} packages, each package analysis once; "
          f"global duplicate analysis once; exit {result}", flush=True)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--linter", required=True)
    parser.add_argument("--go", default="go")
    parser.add_argument("--all", action="store_true")
    parser.add_argument("--profile", choices=("host", "constrained", "throughput"),
                        default=os.environ.get("TEST_PROFILE", "host"))
    parser.add_argument("packages", nargs="+")
    args = parser.parse_args()
    try:
        return run_inventory(args.linter, args.packages, args.all, args.go, profile=args.profile)
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"Lint inventory failed: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
