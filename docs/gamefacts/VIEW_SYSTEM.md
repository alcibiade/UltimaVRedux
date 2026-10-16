# View, scrying, and the traveller's map

Gem View, crystal balls, prepared/scroll View, astronomical instruments, and the
Redux atlas have different resource and knowledge contracts. Original RNG bounds
do not certify uniformity; original data formats do not establish modern UI policy.

### VIEW-0001

**Entry points and resource commitment**

**Kind:** rule
**Uses:** PROV-0005, INV-0006, INV-0007
**Original:** Established
**Redux:** Intentional deviation

Exploration Gem View consumes before rendering: map below 33 chooses surface,
otherwise dungeon; that branch is not comprehensive map validation. Combat rejects
before gem consumption and re-prompts the same actor. Crystal Look uses actor/INT
selection without gems. In Quas Wis is prepared spell 39/circle seven, legal
outdoors/town/dungeon, not combat. Context and Stonegate/uncrowned Palace absorption
precede ordinary prepared commitment. Its scroll consumes before combat rejection.

**REDUX DEVIATION: VIEW-0001:** Redux rejects an illegal combat View scroll before
consumption under its validated-attempt policy. Gem View remains a separately paid
32×32 observation; the free atlas does not silently replace it.

**Implementation:** `internal/game_state/action_view.go`, `internal/game_state/cast_use_commit.go`

### VIEW-0002

**Surface sample geometry and tile-class mapping**

**Kind:** rule
**Uses:** TILE-0001
**Original:** Established
**Redux:** Unverified

Original View samples 32×32 terrain cells from current map offsets, drawing 4×4
symbols starting at pixel (32,32). Party position is byte-wrapped relative to
offsets; its XOR marker toggles every fourth unsuccessful input scan. Dynamic
object-list drawing is absent from this sampling loop.

The compatibility matrix maps a tile byte (row=high nibble, column=low nibble)
to a plot class; it is not tactical-room metadata:

| High / low | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | A | B | C | D | E | F |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 0 | 12 | 11 | 10 | 13 | 1 | 9 | 3 | 9 | 2 | 2 | 8 | 7 | 6 | 8 | 8 |
| 1 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 3 | 4 | 3 | 3 |
| 2 | 16 | 16 | 16 | 16 | 16 | 16 | 16 | 7 | 7 | 5 | 5 | 5 | 9 | 2 | 5 | 5 |
| 3 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 4 | 7 | 7 | 7 | 7 | 7 | 7 | 7 |
| 4 | 3 | 5 | 5 | 5 | 3 | 6 | 7 | 4 | 3 | 3 | 6 | 6 | 5 | 7 | 7 | 7 |
| 5 | 7 | 7 | 7 | 7 | 7 | 7 | 7 | 7 | 5 | 5 | 4 | 5 | 4 | 4 | 5 | 5 |
| 6 | 10 | 10 | 10 | 10 | 10 | 10 | 10 | 10 | 10 | 10 | 3 | 3 | 10 | 10 | 10 | 10 |
| 7 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 | 3 |
| 8 | 5 | 5 | 5 | 5 | 5 | 5 | 6 | 3 | 5 | 5 | 5 | 5 | 3 | 5 | 5 | 3 |
| 9 | 5 | 5 | 5 | 5 | 4 | 4 | 4 | 6 | 6 | 6 | 4 | 4 | 4 | 5 | 5 | 5 |
| A | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 3 | 4 | 4 | 5 | 5 | 5 |
| B | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 6 | 6 | 6 | 6 | 3 | 5 | 4 | 5 |
| C | 0 | 0 | 0 | 0 | 6 | 6 | 6 | 6 | 5 | 5 | 5 | 5 | 0 | 0 | 0 | 0 |
| D | 7 | 7 | 7 | 7 | 11 | 11 | 11 | 11 | 15 | 15 | 15 | 15 | 15 | 3 | 5 | 5 |
| E | 14 | 14 | 14 | 14 | 10 | 10 | 10 | 10 | 5 | 5 | 5 | 5 | 6 | 6 | 6 | 6 |
| F | 6 | 6 | 6 | 6 | 6 | 6 | 6 | 6 | 6 | 6 | 5 | 5 | 5 | 5 | 7 | 0 |

| Class | Symbol semantics |
| --- | --- |
| 0 / 1 / 2 / 3 | Blank / four bright-green points / solid bright green / solid red |
| 4 / 5 / 6 / 7 | White top+bottom / white central 2×2 / white outline / solid white |
| 8 / 9 | Two diagonal yellow 2×2 blocks / green horizontal strokes and points |
| 10 / 11 / 12 / 13 | River-mask green/water points / two diagonal water points / one water point / mixed green/water points |
| 14 / 15 / 16 | Two central white columns / solid bright blue / vegetation base plus red road center/arms and tile-specific black cutout |

Class 90 also draws solid white but is not selected by this matrix. CGA/Hercules
purple water substitution applies to classes 10–13. Exact pixel masks remain a
narrower rendering-verification boundary. Redux scales the plot language into a
responsive dialog; no pixel-identical Match is asserted.

The current Redux surface View still derives its sample origin from the party's
position; it does not consume the retained outdoor window used by
[Blink](MAGIC_SPELLS.md#spell-0002). Opening View preserves that runtime window.
The Blink correction does not establish surface View window-history parity or
change its existing sample geometry; that comparison remains unverified.

### VIEW-0003

**Dungeon symbolic flood**

**Kind:** rule
**Uses:** DNG-0002, SIGN-0005
**Original:** Established
**Redux:** Matches

Display X/Y range 0–21 with party at (11,11). Source coordinate is display
displacement plus party position modulo eight. Visit by display coordinate,
not unique dungeon cell; draw once. Neighbor order is NW,N,NE,W,E,SW,S,SE.
Wall/special-wall/hidden-door families draw then stop propagation; other families,
including revealed doors, propagate. No orthogonal-clearance test forbids diagonals.
The original clears 736 mark bytes but has a 22×22 effective domain and two
256-byte circular coordinate queues. No light gate applies. After input, restore
first-person rendering. Glyph semantics are SIGN-0005.

Redux's detached `BuildDungeonViewModel` preserves the reviewed wrapped flood,
neighbor order, and blocker classes, validating coordinates first. It still emits
the 22×22 display-space observation from live authoritative current-floor state;
the widget alone collapses visible occurrences into one north-up 8×8 floor. This
Match is geometric, not visual, and does not claim that revealed adjacency is safe
or reachable.

**REDUX DEVIATION: VIEW-0003 (presentation):** Solid filled walls, dotted passages,
hatched Not revealed space and shared map/key artwork replace the original glyph
presentation for navigation clarity. Stalactites, skeletons and cave-ins retain
ordinary wall rendering without separate decorative markers. A peripheral party
frame/facing arrow preserves the central feature. The compact key keeps terrain
and party entries, adding only features in the revealed collapsed plan, including
ceiling openings and open/hidden pit-with-ceiling variants. Plain paired west/east
and north/south edge labels describe only observed wrapped continuations on the
same level; unknown endpoints supply neither feature artwork, key entries nor labels.
The [Dungeon View contract](../ui/WIDGET_COMPONENTS.md#dungeon-view) defines the
presentation and documents optional inspection as a considered, unimplemented
alternative. These choices do not change flood geometry, source state authority,
hidden-feature disclosure, or gem/prepared-spell/scroll commitment and lifecycle.

**Implementation:** `internal/game_state/dungeon_view.go`, `internal/ui/widgets/dungeon_view_dialog.go`, `internal/ui/widgets/dungeon_view_plan.go`, `internal/ui/widgets/dungeon_view_layout.go`, `internal/ui/widgets/dungeon_view_symbols.go`
**Tests:** `internal/game_state/dungeon_view_test.go`, `internal/ui/widgets/dungeon_view_dialog_test.go`

### VIEW-0004

**Crystal risk and coordinate instruments**

**Kind:** rule
**Uses:** RNG-0001, ASTRO-0003, SIGN-0007
**Original:** Established
**Redux:** Intentional deviation

Crystal terrain precedes ordinary object-overlay dispatch. Original selection
trusts active designation; otherwise Good/Poisoned fallback auto-selects a sole
eligible member or prompts among several. Cancellation precedes RNG. R(1,30)>=INT
fails with one HP damage; success opens surface View without a gem. This direct
request is not transformed d30; `(31-INT)/30` is only a conditional uniform-input
probability, not an independently verified generator result.

**REDUX DEVIATION: VIEW-0004:** Redux requires Good/Poisoned even for an active
viewer and retains a fallback dialog for one member, explicitly confirming vision
risk instead of permitting disabled active actors to bypass selection. The chosen
viewer owns name, Intelligence, and damage. Instruments reuse ASTRO-0003/SIGN-0007;
In Wis has outdoor/map-zero context but no separate Sextant night/level test.
No Peer command was found in the bounded original search; P is Push, not evidence
for a Peer mechanic on every release/platform.

### VIEW-0005

**Redux supplied atlas versus earned plotting**

**Kind:** rule
**Uses:** ASTRO-0006, MOON-0006, MOON-0009
**Original:** Incomplete
**Redux:** Unverified

The maintained Redux atlas contract is distinct from original explicit coordinate
Use; a full original printed-map/knowledge comparison is not established.
Shift-M opens a free paused Britannia atlas; paid V remains separate. The supplied
directory has 34 entries: eight virtue towns, Buccaneer's Den, three principle
strongholds, Lord British's castle, unnamed castle, four generic lighthouses,
seven physical shrines, Guardians checkpoint (233,235), eight moon origins.
Catalog membership grants no precise point. Generic names become learned through
successful Enter/accepted introductions; Lord British's Castle retains its supplied
name. Huts/hidden keeps require real discovery. Spirituality's sentinel, Ararat,
Doom, and Underworld are excluded; eight region labels are decoration.

The separate 27-symbol artwork allowlist covers eight virtue towns, three principle
strongholds, Lord British's Castle, seven shrines, and eight moon origins. It
excludes Buccaneer's Den, Blackthorn's Castle, lighthouses, and Guardians. Static
symbols grant no numeric coordinates, interactive leaders/brackets/focus, mapped
count, coordinate badge, or earned discovery color. Earned coincident points
coalesce with matching artwork; retired moonstone authority does not erase artwork.

`PlotPosition` accepts measured landmarks/current moonstone placements or an
authorized personal-note/registered-vessel point. Detached readers contain no
unmeasured internal sites and never acquire knowledge or fill gaps from live state.

**Implementation:** `internal/game_state/atlas.go`, `internal/references/atlas.go`

### VIEW-0006

**Redux observation, receipt exception, and notices**

**Kind:** rule
**Uses:** VIEW-0005, ASTRO-0006, MOON-0007, MOON-0008
**Original:** Established
**Redux:** Intentional deviation

**REDUX DEVIATION: VIEW-0006:** Redux remembers exact locally identified targets
and automatically surveys qualifying sites for navigation beyond explicit Avatar
coordinate Use. Look earns only the feature actually described after overlay/
occupant precedence. Successful Enter learns its displayed name using captured
exterior; accepted shrine introduction learns virtue, rejected ruins entry does not.
Unusable instruments preserve genuine old surveys; acquisition elsewhere never
backfills them. Local room/burial coordinates never become world coordinates.

Live first Sextant receipt in Greyhaven is the bounded exception: a real absent-
to-owned grant on a valid floor can measure its canonical exterior (104,216) at
any hour/policy only with legitimate prior Enter/name evidence. Naming provenance
and an older genuine survey stay intact; repeated grants/debug/setup/hydration
cannot manufacture the event. It does not enable a strict indoor live fix.

Guardians site 108 merges into supplied chart-25 without a naming event. Only
successful exterior Look at installed angels (232,235)/(234,235), after precedence,
can observe/measure checkpoint (233,235). Walking, checkpoint terrain, and quest
progress do not survey it. Mutations publish first discovery/name/measurement;
the scene coalesces ordinary Output notices after command/modal/workflow ownership,
map before journal, with styled hints and History. Readers/load do not replay them.

**Implementation:** `internal/game_state/atlas.go`, `internal/game_state/atlas_sextant_receipt.go`

### VIEW-0007

**Redux map reader, vessels, and personal notes**

**Kind:** rule
**Uses:** VIEW-0005, VIEW-0006
**Original:** Incomplete
**Redux:** Unverified

These are current Redux facilities, not a reconstructed original atlas interface.
Specific categories have an All mapped overview of authorized entries; All has
no synthetic row. Counts/badges/labels share point authority. Vessels registers
boats used with a carried Sextant, without pre-ownership backfill, and projects
currently parked surface vessels. My Notes captures an eligible fix and user text
through core-validated drafts.

Exact Ctrl-N from safe surface-overworld primary input opens the atlas and existing
note editor at the current fix, independent of camera. Existing notes are edited,
not duplicated. Without a fix, atlas explains availability and opens no editor.
Save/cancel returns to atlas; final close releases pause. No time, RNG, or resource
cost is added. Avatar crosshair is current or absent, never last-known fallback;
flexible settlement fixes explain Within. A centers only a valid current fix;
first opening shows the whole world. Remembered measurements remain readable.

**Implementation:** `internal/game_state/atlas_fleet.go`, `internal/game_state/atlas_notes.go`
