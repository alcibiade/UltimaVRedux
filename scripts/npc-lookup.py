#!/usr/bin/env python3
"""
NPC Lookup Helper Script for Ultima V Redux

This script helps find NPC locations, schedules, and conversation details
from the game's reference data files.
Generate fresh snapshots with `go run ./cmd/ref_dump --output u5dump/json`.
Conversation searches use the patched semantic catalog, with one-based dialogs.

Usage:
    python3 scripts/npc-lookup.py --help
    python3 scripts/npc-lookup.py find-npc --name "Annon"
    python3 scripts/npc-lookup.py find-npc --dialog 12
    python3 scripts/npc-lookup.py list-npcs --location Britain
    python3 scripts/npc-lookup.py show-schedule --npc-index 45
    python3 scripts/npc-lookup.py find-dialog --keyword "unlock"
"""

import argparse
from collections.abc import Iterator
import json
import sys
from pathlib import Path
from typing import NotRequired, TypedDict, cast

from lookup_locations import NPC_LOCATIONS as LOCATIONS

type JSONValue = None | bool | int | float | str | list[JSONValue] | dict[
    str, JSONValue
]
type Schedule = dict[str, list[int]]


class NPCReference(TypedDict):
    """Fields consumed from a ref_dump NPC record."""

    location: int
    dialog_number: int
    npc_type: int
    schedule: Schedule


class NPCReferences(TypedDict):
    """The exported NPC snapshot container."""

    npcs: list[NPCReference]


class Origin(TypedDict):
    """Conversation family and one-based dialog identity."""

    source: str
    dialog: int
    kind: NotRequired[str]


class Conversation(TypedDict):
    """Named semantic program, including recursive authored/control content."""

    id: str
    origin: Origin
    name: JSONValue
    topics: NotRequired[JSONValue]
    questions: NotRequired[JSONValue]


class KeywordGroup(TypedDict):
    """An authored topic or question answer with keyword aliases."""

    keywords: list[str]
    response: JSONValue


class NameMatch(TypedDict):
    """A semantic name matched to a real NPC slot."""

    semantic_id: str
    dialog_number: int
    npc_index: int
    location_name: str
    matched_text: str


class NPCSummary(TypedDict):
    """Fields displayed in the location NPC listing."""

    npc_index: int
    dialog_number: int
    npc_type: int
    schedule: Schedule


class NPCDetails(NPCSummary):
    """Full slot lookup with the display location and conversation file."""

    location: int
    location_name: str
    tlk_file: str


class DialogMatch(TypedDict):
    """Keyword search hit and its readable response preview."""

    semantic_id: str
    dialog_number: int
    keyword: str
    all_keywords: list[str]
    response_preview: str


# Location to SmallMapMasterType mapping
LOCATION_TO_MAP_TYPE = {
    # Castle type (uses CASTLE.TLK)
    17: "Castle",
    18: "Castle",
    19: "Castle",
    20: "Castle",
    21: "Castle",
    22: "Castle",
    23: "Castle",
    24: "Castle",
    # Towne type (uses TOWNE.TLK)
    1: "Towne",
    2: "Towne",
    3: "Towne",
    4: "Towne",
    5: "Towne",
    6: "Towne",
    7: "Towne",
    8: "Towne",
    # Dwelling type (uses DWELLING.TLK)
    9: "Dwelling",
    10: "Dwelling",
    11: "Dwelling",
    12: "Dwelling",
    13: "Dwelling",
    14: "Dwelling",
    15: "Dwelling",
    16: "Dwelling",
    # Keep type (uses KEEP.TLK)
    25: "Keep",
    26: "Keep",
    27: "Keep",
    28: "Keep",
    29: "Keep",
    30: "Keep",
    31: "Keep",
    32: "Keep",
}

# Map type to TLK file
MAP_TYPE_TO_TLK = {
    "Castle": "CASTLE.TLK",
    "Towne": "TOWNE.TLK",
    "Dwelling": "DWELLING.TLK",
    "Keep": "KEEP.TLK",
}


def get_project_root() -> Path:
    """Find the project root directory."""
    script_dir = Path(__file__).parent
    return script_dir.parent


def load_npc_references() -> NPCReferences:
    """Load NPCReferences.json."""
    path = get_project_root() / "u5dump" / "json" / "NPCReferences.json"
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def load_conversations() -> list[Conversation]:
    """Load the post-patch semantic documents exported by ref_dump."""
    path = get_project_root() / "u5dump" / "json" / "ConversationCatalog.json"
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def display_text(value: JSONValue) -> str:
    """Project authored speech/narration without keywords or control identities."""
    if isinstance(value, str):
        return value
    if isinstance(value, list):
        # Scalar list members are control actions or explicit line breaks, not
        # authored prose. Shorthand prose is a scalar body/content, handled above.
        return "".join(
            (
                "\n"
                if part == "line_break"
                else display_text(part) if isinstance(part, dict) else ""
            )
            for part in value
        )
    if isinstance(value, dict):
        return "".join(
            display_text(part)
            for key, part in value.items()
            if key in ("say", "narrate", "text", "runic", "branch", "then", "otherwise")
        )
    return ""


def keyword_groups(value: JSONValue) -> Iterator[KeywordGroup]:
    """Yield semantic topic/answer groups, including named questions."""
    if isinstance(value, dict):
        if "keywords" in value and "response" in value:
            yield cast(KeywordGroup, value)
        for part in value.values():
            yield from keyword_groups(part)
    elif isinstance(value, list):
        for part in value:
            yield from keyword_groups(part)


def get_location_name(location_id: int) -> str:
    """Get location name from ID."""
    return LOCATIONS.get(location_id, f"Unknown({location_id})")


def get_tlk_file(location_id: int) -> str:
    """Get TLK file for a location."""
    map_type = LOCATION_TO_MAP_TYPE.get(location_id)
    if map_type:
        return MAP_TYPE_TO_TLK.get(map_type, "Unknown")
    return "Unknown"


def format_schedule(schedule: Schedule) -> str:
    """Format NPC schedule for display."""
    times = schedule.get("time", [0, 0, 0, 0])
    xs = schedule.get("x", [0, 0, 0])
    ys = schedule.get("y", [0, 0, 0])
    floors = schedule.get("floor", [0, 0, 0])
    ais = schedule.get("ai", [0, 0, 0])

    result = []
    # Times are: [end1, start2, end2, start3] or similar
    # Positions correspond to 3 schedule slots
    for i in range(3):
        floor_str = f"F{floors[i]}" if floors[i] != 255 else "Any"
        ai_str = f"AI={ais[i]}" if ais[i] != 0 else ""
        result.append(
            f"  Slot {i + 1}: ({xs[i]}, {ys[i]}) {floor_str} {ai_str}".strip()
        )

    result.append(f"  Times: {times}")
    return "\n".join(result)


def find_npc_by_name(name: str, location_id: int | None = None) -> list[NameMatch]:
    """Find named semantic programs and their real NPC reference slots."""
    conversations = load_conversations()
    npc_refs = load_npc_references()

    results: list[NameMatch] = []
    name_lower = name.lower()

    for conversation in conversations:
        text = display_text(conversation["name"])
        if name_lower not in text.lower():
            continue
        origin = conversation["origin"]
        for npc_index, npc in enumerate(npc_refs["npcs"]):
            location = npc["location"]
            source = LOCATION_TO_MAP_TYPE.get(location, "").lower()
            if npc["dialog_number"] != origin["dialog"] or source != origin["source"]:
                continue
            if location_id is not None and location != location_id:
                continue
            results.append(
                {
                    "semantic_id": conversation["id"],
                    "dialog_number": origin["dialog"],
                    "npc_index": npc_index,
                    "location_name": get_location_name(location),
                    "matched_text": text,
                }
            )

    return results


def find_npc_by_dialog(
    dialog_num: int, location_id: int | None = None
) -> list[NPCDetails]:
    """Find NPCs by dialog number."""
    npc_refs = load_npc_references()

    results: list[NPCDetails] = []
    for i, npc in enumerate(npc_refs["npcs"]):
        if npc.get("dialog_number") == dialog_num:
            if location_id is None or npc.get("location") == location_id:
                results.append(
                    {
                        "npc_index": i,
                        "location": npc.get("location"),
                        "location_name": get_location_name(npc.get("location")),
                        "dialog_number": dialog_num,
                        "npc_type": npc.get("npc_type"),
                        "schedule": npc.get("schedule"),
                        "tlk_file": get_tlk_file(npc.get("location")),
                    }
                )

    return results


def list_npcs_in_location(location_name: str) -> list[NPCSummary]:
    """List all NPCs in a location."""
    npc_refs = load_npc_references()

    # Find location ID
    location_id = None
    for loc_id, loc_name in LOCATIONS.items():
        if loc_name.lower() == location_name.lower():
            location_id = loc_id
            break

    if location_id is None:
        print(f"Unknown location: {location_name}")
        print(f"Available locations: {', '.join(LOCATIONS.values())}")
        return []

    results: list[NPCSummary] = []
    for i, npc in enumerate(npc_refs["npcs"]):
        if npc.get("location") == location_id:
            dn = npc.get("dialog_number", 0)
            # Skip empty/guard NPCs for cleaner output
            if 0 < dn < 128:
                results.append(
                    {
                        "npc_index": i,
                        "dialog_number": dn,
                        "npc_type": npc.get("npc_type"),
                        "schedule": npc.get("schedule"),
                    }
                )

    return sorted(results, key=lambda x: x["dialog_number"])


def find_dialog_by_keyword(keyword: str) -> list[DialogMatch]:
    """Find topic/answer keywords in the patched semantic documents."""
    conversations = load_conversations()

    results: list[DialogMatch] = []
    keyword_lower = keyword.lower()[:4]  # Keywords are 4-char prefixes

    for conversation in conversations:
        for group in keyword_groups(cast(JSONValue, conversation)):
            options = group["keywords"]
            for opt in options:
                if keyword_lower in opt.lower():
                    results.append(
                        {
                            "semantic_id": conversation["id"],
                            "dialog_number": conversation["origin"]["dialog"],
                            "keyword": opt,
                            "all_keywords": options,
                            "response_preview": display_text(group["response"])[
                                :150
                            ].strip(),
                        }
                    )

    return results


def show_npc_schedule(npc_index: int) -> NPCDetails | None:
    """Show detailed schedule for an NPC."""
    npc_refs = load_npc_references()

    if npc_index < 0 or npc_index >= len(npc_refs["npcs"]):
        print(f"Invalid NPC index: {npc_index}")
        return None

    npc = npc_refs["npcs"][npc_index]
    return {
        "npc_index": npc_index,
        "location": npc.get("location"),
        "location_name": get_location_name(npc.get("location")),
        "dialog_number": npc.get("dialog_number"),
        "npc_type": npc.get("npc_type"),
        "schedule": npc.get("schedule"),
        "tlk_file": get_tlk_file(npc.get("location")),
    }


def main() -> None:
    """Dispatch the public NPC lookup commands and retain their output formats."""
    # The CLI is one dispatch table with independent legacy text layouts per command.
    # pylint: disable=too-many-branches,too-many-statements
    parser = argparse.ArgumentParser(description="NPC Lookup Helper for Ultima V Redux")
    subparsers = parser.add_subparsers(dest="command", help="Commands")

    # find-npc command
    find_parser = subparsers.add_parser(
        "find-npc", help="Find NPC by name or dialog number"
    )
    find_parser.add_argument("--name", help="NPC name to search for")
    find_parser.add_argument("--dialog", type=int, help="Dialog number to search for")
    find_parser.add_argument("--location", help="Filter by location name")

    # list-npcs command
    list_parser = subparsers.add_parser("list-npcs", help="List NPCs in a location")
    list_parser.add_argument("--location", required=True, help="Location name")

    # show-schedule command
    schedule_parser = subparsers.add_parser("show-schedule", help="Show NPC schedule")
    schedule_parser.add_argument(
        "--npc-index", type=int, required=True, help="NPC index"
    )

    # find-dialog command
    dialog_parser = subparsers.add_parser("find-dialog", help="Find dialogs by keyword")
    dialog_parser.add_argument("--keyword", required=True, help="Keyword to search for")

    args = parser.parse_args()

    if args.command == "find-npc":
        location_id = None
        if args.location:
            location_id = next(
                (
                    loc_id
                    for loc_id, loc_name in LOCATIONS.items()
                    if loc_name.lower() == args.location.lower()
                ),
                None,
            )
            if location_id is None:
                parser.error(f"Unknown location: {args.location}")
        if args.name:
            results = find_npc_by_name(args.name, location_id)
            print(f"\nFound {len(results)} matches for '{args.name}':\n")
            for r in results:
                print(f"  Program: {r['semantic_id']}, Dialog: {r['dialog_number']}")
                print(f"  NPC Index: {r['npc_index']}, Location: {r['location_name']}")
                print(f'  Name: "{r["matched_text"]}"')
                print()
        elif args.dialog is not None:
            results = find_npc_by_dialog(args.dialog, location_id)
            print(f"\nFound {len(results)} NPCs with dialog {args.dialog}:\n")
            for r in results:
                print(f"  NPC Index: {r['npc_index']}")
                print(f"  Location: {r['location_name']} (ID: {r['location']})")
                print(f"  TLK File: {r['tlk_file']}")
                print(f"  NPC Type: {r['npc_type']}")
                print("  Schedule:")
                print(format_schedule(r["schedule"]))
                print()
        else:
            print("Please specify --name or --dialog")

    elif args.command == "list-npcs":
        results = list_npcs_in_location(args.location)
        print(f"\nNPCs in {args.location}:\n")
        for r in results:
            print(
                f"  Index: {r['npc_index']}, Dialog: {r['dialog_number']}, "
                f"Type: {r['npc_type']}"
            )

    elif args.command == "show-schedule":
        result = show_npc_schedule(args.npc_index)
        if result:
            print(f"\nNPC {result['npc_index']}:")
            print(f"  Location: {result['location_name']}")
            print(
                f"  Dialog: {result['dialog_number']} "
                f"(TLK index: {result['dialog_number'] - 1})"
            )
            print(f"  TLK File: {result['tlk_file']}")
            print(f"  NPC Type: {result['npc_type']}")
            print("  Schedule:")
            print(format_schedule(result["schedule"]))

    elif args.command == "find-dialog":
        results = find_dialog_by_keyword(args.keyword)
        print(f"\nFound {len(results)} dialogs with keyword '{args.keyword}':\n")
        for r in results:
            print(f"  Program: {r['semantic_id']}, Dialog: {r['dialog_number']}")
            print(f"  Keywords: {r['all_keywords']}")
            print(f'  Response: "{r["response_preview"]}..."')
            print()

    else:
        parser.print_help()


if __name__ == "__main__":
    try:
        main()
    except (OSError, json.JSONDecodeError) as error:
        print(
            f"NPC lookup failed: {error}. Regenerate snapshots with "
            "go run ./cmd/ref_dump --output u5dump/json",
            file=sys.stderr,
        )
        sys.exit(1)
