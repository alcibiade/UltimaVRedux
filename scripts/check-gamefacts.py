#!/usr/bin/env python3
"""Validate public Game Facts without external repositories or network access."""

import argparse
import sys
from pathlib import Path

from gamefacts_public import validate_repository


def main() -> int:
    """Validate the selected standalone checkout and report a concise result."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[1]
    )
    args = parser.parse_args()
    try:
        documents, rules = validate_repository(args.root)
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"Game Facts validation failed: {error}", file=sys.stderr)
        return 1
    print(f"PASS public Game Facts: {len(documents)} topics, {len(rules)} references")
    return 0


if __name__ == "__main__":
    sys.exit(main())
