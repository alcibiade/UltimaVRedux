"""Stable location IDs shared by the standalone map and NPC lookup commands."""

# Matches internal/references/locations.go, including world and dungeon identities.
LOCATIONS: dict[int, str] = {
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

# NPC dialogs exist only for the world and settlement range used by the old CLI.
NPC_LOCATIONS: dict[int, str] = {
    identifier: name for identifier, name in LOCATIONS.items() if identifier <= 32
}
