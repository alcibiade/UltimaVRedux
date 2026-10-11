#!/usr/bin/env python3
"""Run the complete requested Go package inventory through serial lint batches."""

import argparse
from pathlib import Path
import subprocess
import sys

BATCH_SIZE = 4


def run_inventory(linter, packages, all_findings=False, go="go", runner=subprocess.run):
    listed = runner([go, "list", "-f", "{{.Dir}}", *packages], check=True, capture_output=True, text=True)
    inventory = listed.stdout.splitlines()
    if not inventory or len(inventory) != len(set(inventory)) or any(not name.strip() for name in inventory):
        raise ValueError("Go package inventory must be nonempty and unique")
    flags = ["--concurrency=1"]
    if all_findings:
        flags += ["--new=false", "--new-from-rev=", "--new-from-patch=", "--new-from-merge-base=",
                  "--max-issues-per-linter=0", "--max-same-issues=0", "--uniq-by-line=false"]
    result = 0
    visited = []
    for offset in range(0, len(inventory), BATCH_SIZE):
        batch = inventory[offset:offset + BATCH_SIZE]
        print(f"Lint packages {offset + 1}-{offset + len(batch)}/{len(inventory)}", flush=True)
        completed = runner([str(Path(linter).resolve()), "run", *flags, *batch], check=False)
        result = max(result, completed.returncode if completed.returncode >= 0 else 1)
        visited.extend(batch)
    if visited != inventory:
        raise ValueError("lint workload did not cover the exact inventory")
    # Duplicate detection compares source across package roots. Repeat this
    # inexpensive analyzer on the whole inventory so batching does not hide
    # duplicates spanning two batches. Other package analyses load dependencies.
    global_result = runner([str(Path(linter).resolve()), "run", *flags,
                            "--enable-only=dupl", *inventory], check=False)
    result = max(result, global_result.returncode if global_result.returncode >= 0 else 1)
    print(f"Lint inventory: {len(visited)} packages, each checked once; exit {result}", flush=True)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--linter", required=True)
    parser.add_argument("--go", default="go")
    parser.add_argument("--all", action="store_true")
    parser.add_argument("packages", nargs="+")
    args = parser.parse_args()
    try:
        return run_inventory(args.linter, args.packages, args.all, args.go)
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"Lint inventory failed: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
