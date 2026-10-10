#!/usr/bin/env python3
"""
Map Lookup Helper Script for Ultima V Redux

This script helps explore map data, find tiles, and render ASCII maps
from the game's reference data files and original game data.

Usage:
    python3 scripts/map-lookup.py --help
    python3 scripts/map-lookup.py location --name "Britain"
    python3 scripts/map-lookup.py locations --type Towne
    python3 scripts/map-lookup.py tile --index 79
    python3 scripts/map-lookup.py tile --name "door"
    python3 scripts/map-lookup.py tiles --property BlocksLight
    python3 scripts/map-lookup.py find-tile --location Britain --index 166
    python3 scripts/map-lookup.py render --location Britain --floor 0
    python3 scripts/map-lookup.py render --overworld --x 100 --y 100 --size 16
"""

import argparse
import json
import os
import sys
from pathlib import Path

# Location ID to name mapping (from internal/references/locations.go)
LOCATIONS = {
    0: "BritanniaUnderworld",
    1: "Moonglow",
    2: "Britain",
    3: "Jhelom",
    4: "Yew",
    5: "Minoc",
    6: "Trinsic",
    7: "SkaraBrae",
    8: "NewMagincia",
    9: "Fogsbane",
    10: "Stormcrow",
    11: "Greyhaven",
    12: "Waveguide",
    13: "IolosHut",
    14: "SuteksHut",
    15: "SinVraalsHut",
    16: "GrendelsHut",
    17: "LordBritishsCastle",
    18: "PalaceOfBlackthorn",
    19: "WestBritanny",
    20: "NorthBritanny",
    21: "EastBritanny",
    22: "Paws",
    23: "Cove",
    24: "BuccaneersDen",
    25: "Ararat",
    26: "Bordermarch",
    27: "Farthing",
    28: "Windemere",
    29: "Stonegate",
    30: "Lycaeum",
    31: "EmpathAbbey",
    32: "SerpentsHold",
    33: "Deceit",
    34: "Despise",
    35: "Destard",
    36: "Wrong",
    37: "Covetous",
    38: "Shame",
    39: "Hythloth",
    40: "Doom",
}

# Reverse lookup
LOCATION_BY_NAME = {v.lower(): k for k, v in LOCATIONS.items()}

# SmallMapMasterTypes
MAP_TYPES = {
    "Castle": [
        "LordBritishsCastle",
        "PalaceOfBlackthorn",
        "WestBritanny",
        "NorthBritanny",
        "EastBritanny",
        "Paws",
        "Cove",
        "BuccaneersDen",
    ],
    "Towne": [
        "Moonglow",
        "Britain",
        "Jhelom",
        "Yew",
        "Minoc",
        "Trinsic",
        "SkaraBrae",
        "NewMagincia",
    ],
    "Dwelling": [
        "Fogsbane",
        "Stormcrow",
        "Greyhaven",
        "Waveguide",
        "IolosHut",
        "SuteksHut",
        "SinVraalsHut",
        "GrendelsHut",
    ],
    "Keep": [
        "Ararat",
        "Bordermarch",
        "Farthing",
        "Windemere",
        "Stonegate",
        "Lycaeum",
        "EmpathAbbey",
        "SerpentsHold",
    ],
    "Dungeon": [
        "Deceit",
        "Despise",
        "Destard",
        "Wrong",
        "Covetous",
        "Shame",
        "Hythloth",
        "Doom",
    ],
}

# Map files for each type
MAP_FILES = {
    "Castle": "CASTLE.DAT",
    "Towne": "TOWNE.DAT",
    "Dwelling": "DWELLING.DAT",
    "Keep": "KEEP.DAT",
    "Dungeon": "DUNGEON.DAT",
}

# Floor counts per location
FLOOR_COUNTS = {
    "LordBritishsCastle": 5,
    "PalaceOfBlackthorn": 5,
    "WestBritanny": 1,
    "NorthBritanny": 1,
    "EastBritanny": 1,
    "Paws": 1,
    "Cove": 1,
    "BuccaneersDen": 1,
    "Moonglow": 2,
    "Britain": 2,
    "Jhelom": 2,
    "Yew": 2,
    "Minoc": 2,
    "Trinsic": 2,
    "SkaraBrae": 2,
    "NewMagincia": 2,
    "Fogsbane": 3,
    "Stormcrow": 3,
    "Greyhaven": 3,
    "Waveguide": 3,
    "IolosHut": 1,
    "SuteksHut": 1,
    "SinVraalsHut": 1,
    "GrendelsHut": 1,
    "Ararat": 2,
    "Bordermarch": 2,
    "Farthing": 1,
    "Windemere": 1,
    "Stonegate": 1,
    "Lycaeum": 3,
    "EmpathAbbey": 3,
    "SerpentsHold": 3,
    "Deceit": 8,
    "Despise": 8,
    "Destard": 8,
    "Wrong": 8,
    "Covetous": 8,
    "Shame": 8,
    "Hythloth": 8,
    "Doom": 8,
}

# Locations with basements (floor -1)
HAS_BASEMENT = ["LordBritishsCastle", "PalaceOfBlackthorn", "Yew", "SerpentsHold"]

# ASCII symbols for tile rendering
TILE_SYMBOLS = {
    # Terrain
    "Water": "~",
    "Swamp": "%",
    "Grass": ".",
    "Forest": "T",
    "Desert": ":",
    "Hills": "^",
    "Mountain": "M",
    "Path": "=",
    "Bridge": "=",
    # Structures
    "Wall": "#",
    "Door": "+",
    "Floor": ".",
    "Brick": ".",
    "Window": "O",
    "Stairs": ">",
    "Ladder": "H",
    # Objects
    "Chest": "$",
    "Table": "T",
    "Chair": "c",
    "Bed": "b",
    "Fountain": "F",
    "Well": "W",
    "Sign": "!",
    "Torch": "*",
    # Special
    "Lava": "L",
    "Pit": "o",
    "Secret": "?",
    # Default
    "default": " ",
}


def find_script_dir():
    """Find the project root directory."""
    script_path = Path(__file__).resolve()
    return script_path.parent.parent


def find_data_path():
    """Use explicit ULTIMA5_DATA_PATH/DIR, otherwise repository OLD/."""
    explicit = os.environ.get("ULTIMA5_DATA_PATH") or os.environ.get("ULTIMA5_DATA_DIR")
    path = Path(explicit).expanduser() if explicit else find_script_dir() / "OLD"
    if (path / "BRIT.DAT").is_file():
        return path
    return None


def load_json(filename):
    """Load a JSON file from u5dump/json/."""
    json_path = find_script_dir() / "u5dump" / "json" / filename
    if not json_path.exists():
        print(f"Error: {json_path} not found", file=sys.stderr)
        sys.exit(1)
    with open(json_path, "r") as f:
        return json.load(f)


def load_tile_refs():
    """Load tile references."""
    return load_json("TileReferences.json")


def load_location_refs():
    """Load location references."""
    return load_json("LocationReferences.json")


def get_map_type(location_name):
    """Get the SmallMapMasterType for a location."""
    for map_type, locations in MAP_TYPES.items():
        if location_name in locations:
            return map_type
    return None


def get_location_offset(location_name):
    """Get the byte offset for a location within its .DAT file."""
    map_type = get_map_type(location_name)
    if not map_type:
        return 0

    locations_in_type = MAP_TYPES[map_type]
    index = locations_in_type.index(location_name)

    if map_type == "Dungeon":
        # Dungeons are 8x8 per floor, 8 floors
        return index * 8 * 8 * 8
    else:
        # Small maps are 32x32 per floor
        offset = 0
        for i in range(index):
            loc = locations_in_type[i]
            floors = FLOOR_COUNTS.get(loc, 1)
            offset += floors * 32 * 32
        return offset


def load_small_map(location_name, data_path):
    """Load a small map (32x32 tiles per floor)."""
    map_type = get_map_type(location_name)
    if not map_type or map_type == "Dungeon":
        return None

    filename = MAP_FILES[map_type]
    filepath = data_path / filename
    if not filepath.exists():
        print(f"Error: {filepath} not found", file=sys.stderr)
        return None

    with open(filepath, "rb") as f:
        data = f.read()

    offset = get_location_offset(location_name)
    floors = FLOOR_COUNTS.get(location_name, 1)
    has_basement = location_name in HAS_BASEMENT

    result = {}
    for floor_idx in range(floors):
        actual_floor = floor_idx - 1 if has_basement else floor_idx
        floor_offset = offset + floor_idx * 32 * 32

        tiles = [[0 for _ in range(32)] for _ in range(32)]
        for x in range(32):
            for y in range(32):
                byte_idx = floor_offset + x + y * 32
                if byte_idx < len(data):
                    tiles[x][y] = data[byte_idx]
        result[actual_floor] = tiles

    return result


def load_dungeon_map(location_name, data_path):
    """Load a dungeon map (8x8 tiles per floor, 8 floors)."""
    if location_name not in MAP_TYPES["Dungeon"]:
        return None

    filepath = data_path / "DUNGEON.DAT"
    if not filepath.exists():
        print(f"Error: {filepath} not found", file=sys.stderr)
        return None

    with open(filepath, "rb") as f:
        data = f.read()

    offset = get_location_offset(location_name)

    result = {}
    for floor in range(8):
        floor_offset = offset + floor * 8 * 8
        tiles = [[0 for _ in range(8)] for _ in range(8)]
        for x in range(8):
            for y in range(8):
                byte_idx = floor_offset + x + y * 8
                if byte_idx < len(data):
                    tiles[x][y] = data[byte_idx]
        result[floor] = tiles

    return result


def load_large_map(world, data_path):
    """Load the large overworld or underworld map (256x256)."""
    if world == "overworld":
        map_file = "BRIT.DAT"
        ovl_file = "DATA.OVL"
        ovl_offset = 0x3886
    else:
        map_file = "UNDER.DAT"
        ovl_file = None
        ovl_offset = 0

    map_path = data_path / map_file
    if not map_path.exists():
        print(f"Error: {map_path} not found", file=sys.stderr)
        return None

    with open(map_path, "rb") as f:
        map_data = f.read()

    # Load overlay for overworld
    ovl_data = None
    if ovl_file:
        ovl_path = data_path / ovl_file
        if ovl_path.exists():
            with open(ovl_path, "rb") as f:
                f.seek(ovl_offset)
                ovl_data = f.read(256)

    # Build the map (256x256 tiles in 16x16 chunks)
    tiles = [[0 for _ in range(256)] for _ in range(256)]
    map_idx = 0

    for chunk in range(256):
        col = chunk % 16
        row = chunk // 16

        # Check if chunk is water (0xFF in overlay)
        is_water = ovl_data and chunk < len(ovl_data) and ovl_data[chunk] == 0xFF

        for cy in range(16):
            for cx in range(16):
                x = col * 16 + cx
                y = row * 16 + cy

                if is_water:
                    tiles[x][y] = 1  # Water1 tile
                else:
                    if map_idx < len(map_data):
                        tiles[x][y] = map_data[map_idx]
                        map_idx += 1

    return tiles


def get_tile_symbol(tile_index, tile_refs):
    """Get ASCII symbol for a tile."""
    tile_key = str(tile_index)
    if tile_key not in tile_refs:
        return TILE_SYMBOLS["default"]

    tile = tile_refs[tile_key]
    name = tile.get("Name", "")

    # Try to match by name keywords
    name_lower = name.lower()
    for keyword, symbol in TILE_SYMBOLS.items():
        if keyword.lower() in name_lower:
            return symbol

    # Special cases by index
    if tile_index in [1, 2, 3]:  # Water tiles
        return "~"
    if tile_index == 4:  # Swamp
        return "%"
    if tile_index == 5:  # Grass
        return "."
    if tile_index in range(6, 11):  # Forest
        return "T"
    if tile_index in range(11, 16):  # Hills/Mountains
        return "^"
    if tile_index in range(32, 39):  # Paths
        return "="
    if tile_index in range(64, 74):  # Floors
        return "."
    if tile_index in range(74, 88):  # Walls
        return "#"
    if 166 <= tile_index <= 171:  # Doors
        return "+"

    return TILE_SYMBOLS["default"]


def render_ascii_map(tiles, width, height, tile_refs, start_x=0, start_y=0):
    """Render tiles as ASCII art."""
    lines = []

    # Header with column numbers
    header = "    "
    for x in range(width):
        header += str((start_x + x) % 10)
    lines.append(header)

    for y in range(height):
        row = f"{start_y + y:3} "
        for x in range(width):
            tile_x = start_x + x
            tile_y = start_y + y
            if tile_x < len(tiles) and tile_y < len(tiles[0]):
                tile_idx = tiles[tile_x][tile_y]
                row += get_tile_symbol(tile_idx, tile_refs)
            else:
                row += " "
        lines.append(row)

    return "\n".join(lines)


# ============= Commands =============


def cmd_location(args):
    """Get details about a location."""
    name = args.name

    # Find location by name (case insensitive)
    location_id = LOCATION_BY_NAME.get(name.lower())
    if location_id is None:
        # Try partial match
        matches = [n for n in LOCATIONS.values() if name.lower() in n.lower()]
        if len(matches) == 1:
            name = matches[0]
            location_id = LOCATION_BY_NAME[name.lower()]
        elif len(matches) > 1:
            print(f"Multiple matches: {', '.join(matches)}")
            return
        else:
            print(f"Location '{name}' not found")
            return

    location_name = LOCATIONS[location_id]
    map_type = get_map_type(location_name)
    floors = FLOOR_COUNTS.get(location_name, 1)
    has_basement = location_name in HAS_BASEMENT

    print(f"\nLocation: {location_name}")
    print(f"  ID: {location_id}")
    print(f"  Type: {map_type or 'Large Map'}")
    print(f"  Floors: {floors}" + (" (has basement)" if has_basement else ""))

    if map_type == "Dungeon":
        print(f"  Map Size: 8x8 tiles per floor")
    elif map_type:
        print(f"  Map Size: 32x32 tiles per floor")

    # Try to get overworld coordinates
    try:
        loc_refs = load_location_refs()
        positions = loc_refs.get("world_locations", {}).get(
            "large_map_location_positions", {}
        )
        for key, value in positions.items():
            if value.get("location") == location_id:
                pos = value.get("position", {})
                print(f"  Overworld: ({pos.get('x', '?')}, {pos.get('y', '?')})")
                break
    except:
        pass

    print()


def cmd_locations(args):
    """List locations of a specific type."""
    map_type = args.type

    if map_type not in MAP_TYPES:
        print(f"Unknown type '{map_type}'. Valid types: {', '.join(MAP_TYPES.keys())}")
        return

    print(f"\n{map_type} Locations:")
    print("-" * 40)

    for loc_name in MAP_TYPES[map_type]:
        loc_id = LOCATION_BY_NAME.get(loc_name.lower(), "?")
        floors = FLOOR_COUNTS.get(loc_name, 1)
        basement = " (basement)" if loc_name in HAS_BASEMENT else ""
        print(f"  {loc_id:2}: {loc_name} - {floors} floor(s){basement}")

    print()


def cmd_tile(args):
    """Get tile information by index or name."""
    tile_refs = load_tile_refs()

    if args.index is not None:
        tile_key = str(args.index)
        if tile_key not in tile_refs:
            print(f"Tile index {args.index} not found")
            return

        tile = tile_refs[tile_key]
        print(f"\nTile {args.index}:")
        for key, value in tile.items():
            print(f"  {key}: {value}")
        print()

    elif args.name:
        pattern = args.name.lower()
        matches = []
        for idx, tile in tile_refs.items():
            name = tile.get("Name", "")
            if pattern in name.lower():
                matches.append((idx, tile))

        if not matches:
            print(f"No tiles matching '{args.name}'")
            return

        print(f"\nTiles matching '{args.name}':")
        print("-" * 50)
        for idx, tile in sorted(matches, key=lambda x: int(x[0])):
            name = tile.get("Name", "")
            desc = tile.get("Description", "")
            print(f"  {idx:3}: {name}" + (f" - {desc}" if desc else ""))
        print(f"\nTotal: {len(matches)} tiles")
        print()


def cmd_tiles(args):
    """List tiles with a specific property."""
    tile_refs = load_tile_refs()
    prop = args.property

    matches = []
    for idx, tile in tile_refs.items():
        if prop in tile and tile[prop]:
            matches.append((idx, tile))

    if not matches:
        print(f"No tiles with property '{prop}' = true")
        print(f"Available properties: {', '.join(list(tile_refs.get('1', {}).keys()))}")
        return

    print(f"\nTiles with {prop} = true:")
    print("-" * 50)
    for idx, tile in sorted(matches, key=lambda x: int(x[0])):
        name = tile.get("Name", "")
        print(f"  {idx:3}: {name}")
    print(f"\nTotal: {len(matches)} tiles")
    print()


def cmd_find_tile(args):
    """Find all positions of a tile in a map."""
    data_path = find_data_path()
    if not data_path:
        print("Error: Game data not found. Set ULTIMA5_DATA_PATH/ULTIMA5_DATA_DIR or provide repository OLD/.")
        return

    location_name = None
    for name in LOCATIONS.values():
        if name.lower() == args.location.lower():
            location_name = name
            break

    if not location_name:
        # Try partial match
        for name in LOCATIONS.values():
            if args.location.lower() in name.lower():
                location_name = name
                break

    if not location_name:
        print(f"Location '{args.location}' not found")
        return

    map_type = get_map_type(location_name)

    if map_type == "Dungeon":
        map_data = load_dungeon_map(location_name, data_path)
        map_size = 8
    elif map_type:
        map_data = load_small_map(location_name, data_path)
        map_size = 32
    else:
        print(f"Cannot search large maps yet")
        return

    if not map_data:
        return

    tile_index = args.index
    target_floor = args.floor

    print(f"\nSearching for tile {tile_index} in {location_name}:")
    print("-" * 50)

    total_found = 0
    for floor, tiles in sorted(map_data.items()):
        if target_floor is not None and floor != target_floor:
            continue

        positions = []
        for x in range(map_size):
            for y in range(map_size):
                if tiles[x][y] == tile_index:
                    positions.append((x, y))

        if positions:
            print(f"  Floor {floor}: {len(positions)} found")
            for x, y in positions:
                print(f"    ({x}, {y})")
            total_found += len(positions)

    print(f"\nTotal: {total_found} tiles found")
    print()


def cmd_render(args):
    """Render ASCII map."""
    tile_refs = load_tile_refs()
    data_path = find_data_path()

    if not data_path:
        print("Error: Game data not found. Set ULTIMA5_DATA_PATH/ULTIMA5_DATA_DIR or provide repository OLD/.")
        return

    if args.overworld:
        tiles = load_large_map("overworld", data_path)
        if not tiles:
            return

        x = args.x or 0
        y = args.y or 0
        size = args.size or 32

        print(f"\nOverworld ({x},{y}) to ({x + size},{y + size}):")
        print(render_ascii_map(tiles, size, size, tile_refs, x, y))
        print_legend()

    elif args.location:
        location_name = None
        for name in LOCATIONS.values():
            if name.lower() == args.location.lower():
                location_name = name
                break

        if not location_name:
            for name in LOCATIONS.values():
                if args.location.lower() in name.lower():
                    location_name = name
                    break

        if not location_name:
            print(f"Location '{args.location}' not found")
            return

        map_type = get_map_type(location_name)
        floor = args.floor if args.floor is not None else 0

        if map_type == "Dungeon":
            map_data = load_dungeon_map(location_name, data_path)
            map_size = 8
        elif map_type:
            map_data = load_small_map(location_name, data_path)
            map_size = 32
        else:
            print(f"Use --overworld for large maps")
            return

        if not map_data:
            return

        if floor not in map_data:
            print(f"Floor {floor} not found. Available: {sorted(map_data.keys())}")
            return

        tiles = map_data[floor]
        print(f"\n{location_name} Floor {floor} ({map_size}x{map_size}):")
        print(render_ascii_map(tiles, map_size, map_size, tile_refs))
        print_legend()


def print_legend():
    """Print ASCII rendering legend."""
    print("\nLegend: ~=Water .=Floor/Grass #=Wall +=Door T=Tree/Forest")
    print("        ^=Hills M=Mountain %=Swamp :=Desert ==Path H=Ladder")
    print()


def main():
    parser = argparse.ArgumentParser(
        description="Map Lookup Helper for Ultima V Redux",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    subparsers = parser.add_subparsers(dest="command", help="Command to run")

    # location command
    loc_parser = subparsers.add_parser("location", help="Get location details")
    loc_parser.add_argument("--name", required=True, help="Location name")

    # locations command
    locs_parser = subparsers.add_parser("locations", help="List locations by type")
    locs_parser.add_argument(
        "--type",
        required=True,
        choices=["Castle", "Towne", "Dwelling", "Keep", "Dungeon"],
        help="Location type",
    )

    # tile command
    tile_parser = subparsers.add_parser("tile", help="Get tile information")
    tile_parser.add_argument("--index", type=int, help="Tile sprite index")
    tile_parser.add_argument("--name", help="Search tiles by name")

    # tiles command
    tiles_parser = subparsers.add_parser("tiles", help="List tiles by property")
    tiles_parser.add_argument("--property", required=True, help="Property to filter by")

    # find-tile command
    find_parser = subparsers.add_parser(
        "find-tile", help="Find tile positions in a map"
    )
    find_parser.add_argument("--location", required=True, help="Location name")
    find_parser.add_argument(
        "--index", type=int, required=True, help="Tile sprite index"
    )
    find_parser.add_argument("--floor", type=int, help="Floor number (optional)")

    # render command
    render_parser = subparsers.add_parser("render", help="Render ASCII map")
    render_parser.add_argument("--location", help="Location name")
    render_parser.add_argument("--floor", type=int, help="Floor number")
    render_parser.add_argument(
        "--overworld", action="store_true", help="Render overworld"
    )
    render_parser.add_argument("--x", type=int, help="X coordinate for overworld")
    render_parser.add_argument("--y", type=int, help="Y coordinate for overworld")
    render_parser.add_argument(
        "--size", type=int, default=32, help="Size of area to render"
    )

    args = parser.parse_args()

    if args.command == "location":
        cmd_location(args)
    elif args.command == "locations":
        cmd_locations(args)
    elif args.command == "tile":
        cmd_tile(args)
    elif args.command == "tiles":
        cmd_tiles(args)
    elif args.command == "find-tile":
        cmd_find_tile(args)
    elif args.command == "render":
        cmd_render(args)
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
