# Astronomy and navigational instruments

These rules describe the reviewed DOS sky and instrument paths, with Redux
presentation and navigation policies identified separately. Original random bounds
do not establish distribution or compiler-dependent argument evaluation order.
Authored font-pixel equivalence and every retail edition are not assumed.

### ASTRO-0001

**Border sky glyphs and eligibility**

**Kind:** rule
**Uses:** SIGN-0006, MOON-0002
**Original:** Established
**Redux:** Matches

The original sky occupies twelve character cells: window 0, row 0, columns 6–17,
bracketed by inward blue/white arrows at columns 5 and 18. These are alternate
font glyphs, not world tiles. The sun uses glyph `2A` hex in yellow; moons use
`30 + phase` hex in gray, including phase zero. Non-Hercules startup loads this
charset from `RUNES.CH`; Hercules uses `RUNES.HCS`.

Hourly positioning and overlap order are canonical in
[SIGN-0006](SIGNS_AND_FONTS.md#sign-0006); phases use the independent
[28-day schedule](MOONSTONE_LOCATIONS.md#moon-0002). The resulting visible windows
are sun 06:00–17:59, Trammel 21:00–08:59, and Felucca 15:00–02:59. On day 8 at
22:00, Trammel phase 4 occupies cell 10 and Felucca phase 6 cell 4. Daytime moons
are expected; this strip is not a moongate-materialization indicator.

Map IDs at least 33 return without drawing. Ararat (map 25) and encoded levels
at least 128 restore a blank border. Other eligible surface maps do not test
whether the party is under a roof.

Redux's `projectSky` preserves the reviewed glyph selection, ordering, phase
schedule, and hourly geometry. The scene uses the runic bitmap font in the static
top-border band, with nearest-neighbor scaling and inward end caps. It is HUD
presentation beneath dialogs, independent of camera motion and screen shake;
projection changes neither game time nor RNG nor gate state. The glyph regression
compares supplied original font pixels when provisioned; a historical comparison
is not blanket certification of all fonts/releases.

**Implementation:** `cmd/ultimav/gamescene_sky.go`
**Tests:** `cmd/ultimav/gamescene_sky_test.go`

### ASTRO-0002

**Inherited sky and refresh ownership**

**Kind:** rule
**Uses:** ASTRO-0001, TIME-0001
**Original:** Established
**Redux:** Intentional deviation

Ordinary original combat and shrine presentation use temporary map ID 255, so
normal sky drawing does not refresh their inherited border. Dungeon framing owns
its level-information header. Surface camping temporarily restores its originating
map identity and refreshes the sky after each five-minute advancement; underground
camping stays blank and dungeon camping retains its header.

Town-combat return reloads the town level and explicitly redraws. Shrine return
restores the map before advancing time. Outdoor-combat return requests zero
minutes, which does not unconditionally refresh the sky: a later nonzero request
can overwrite the previous-hour baseline after a crossing. Stale inherited sky is
a control-flow consequence; it has not been reproduced here in a fingerprinted
retail executable. The same original refresh also assigns stored moon phases.

Redux owns a scene presentation snapshot. Normal draws read the authoritative date,
including debug edits and rollovers; replacing GameState resets snapshot ownership.
Synchronization before input/workflows preserves inheritance even without an
intervening draw. Shrine visits freeze it; combat retains only an inherited sun,
suppressing both moons, including camping ambushes and first draws without a prior
snapshot. If no sun remains, the sky backing is omitted as well. Camping outside
combat uses its origin and live date. Death, non-shrine scripted scenes, and finale layouts suppress a
surface sky; dungeon contexts retain their header.

**REDUX DEVIATION: ASTRO-0002:** outdoor-combat return refreshes immediately from
the current date rather than retaining the original stale-border possibility,
so navigation information is reliable. Original combat inherits the complete sky;
Redux deliberately hides both moon indicators during combat to avoid visual clutter
in the tactical view. It preserves the inherited sun and dungeon level headers.
This is presentation-only and changes neither time, phases, RNG nor moongates.

**Implementation:** `cmd/ultimav/gamescene_sky.go`
**Tests:** `cmd/ultimav/gamescene_sky_test.go`

### ASTRO-0003

**Instrument gates and telescope sun damage**

**Kind:** rule
**Uses:** SIGN-0007, TIME-0002
**Original:** Established
**Redux:** Matches

| Viewer | Original location gate | Time gate |
| --- | --- | --- |
| Telescope | Look resolves telescope terrain | Sun at hours 6–17; night branch otherwise. |
| Spyglass | Map below 33 and encoded level below 128 | Night at 0–5 or 19–23. |
| Sextant | Map 0 and encoded level below 128 | Position at 0–5 or 19–23. |

At 18:00 the telescope already has a night view, but Spyglass/Sextant wait until
19:00. Looking through a daytime telescope applies one HP of damage; daytime
Spyglass rejection does not. The telescope uses the existing active designation
directly. Only designation 255 triggers its fallback, which looks for a Good or
Poisoned member. This is not general validation of every selected character.
Sextant output uses [Y-before-X nibble encoding](SIGNS_AND_FONTS.md#sign-0007).

Redux preserves the reviewed normal hour/location gates and one-point sun damage,
with command completion owned by the surrounding workflow. The damage selector
preserves an explicit designation without invoking active-player cleanup; absent
selection falls back to the first active Good/Poisoned member. Sleeping/charmed
members cannot become fallback targets. Flexible Sextant
policy is a separate deliberate choice below.

**Implementation:** `cmd/ultimav/gamescene_astronomy.go`, `internal/game_state/astronomy_target.go`, `internal/game_state/action_use.go`, `internal/party_state/party_state.go`
**Tests:** `cmd/ultimav/gamescene_astronomy_use_test.go`, `internal/game_state/findings_closeout_test.go`

### ASTRO-0004

**Stars, planets, and Shadowlord comets**

**Kind:** rule
**Uses:** ASTRO-0005
**Original:** Established
**Redux:** Intentional deviation

Prepare a dark 11×11 original view with telescope artwork at bottom center.
Plot 80 bright-blue stars, requesting coordinates with bounds 9–182 and 9–172:
160 coordinate draws total. The two requests occur as arguments of one plotting
operation; which coordinate consumes the earlier random result is not established
without the compiler/runtime. Do not infer independent uniform distributions.

For each planet, draw its white marker and then its associated red comet checks.
A comet is drawn for each Shadowlord whose stored town equals that planet's
one-based index. Planet indexes 0–7 correspond to Moonglow, Britain, Jhelom, Yew,
Minoc, Trinsic, Skara Brae, and New Magincia. The display does not identify which
Shadowlord a comet represents.

Redux creates 80 stars from 160 controlled game-RNG calls once per dialog and
retains that field across redraws. Reopening produces fresh deterministic fields.
**REDUX DEVIATION: ASTRO-0004:** its bounds depend on dialog dimensions, it draws all planets before
all comets and the telescope afterward, and its town projection supplies one
entry per occupied town. Exact original coordinates, compositing, and unusual
multiple-lord multiplicity are not established by the call count. These differences
are explicit rather than a broad exact-rendering Match. Responsive bounds and
layering fit the modern dialog rather than reproducing original pixels. The viewer adds no town,
planet, or virtue identification labels.

**Implementation:** `internal/ui/widgets/astronomy_dialog.go`, `internal/game_state/astronomy.go`
**Tests:** `internal/ui/widgets/astronomy_dialog_test.go`

### ASTRO-0005

**Planet tracks and date walk**

**Kind:** rule
**Uses:** TIME-0001
**Original:** Established
**Redux:** Matches

Use `year modulo 100` and reference date 39-04-05. At or before that date, retain
the initial index. After it, each elapsed calendar day moves to the preceding
enabled index on a circular track of 22 possible positions, borrowing calendar
days/months with 28 days and 13 months. Disabled positions do not participate.

| Planet index | Initial index | Fixed vertical value | Enabled indices 0–21 | Derived return period |
| ---: | ---: | ---: | --- | ---: |
| 0 | 18 | 18 | 4, 11, 18 | 3 days |
| 1 | 2 | 17 | 2, 7, 11, 15, 20 | 5 days |
| 2 | 8 | 15 | 2, 5, 8, 11, 14, 17, 20 | 7 days |
| 3 | 15 | 13 | Odd indexes 1–21 | 11 days |
| 4 | 11 | 11 | 0, 2, 4, 6, 8, 9, 11, 13, 14, 16, 18, 19, 21 | 13 days |
| 5 | 6 | 8 | All except 0, 4, 8, 14, 18 | 17 days |
| 6 | 4 | 5 | All except 0, 7, 15 | 19 days |
| 7 | 2 | 1 | All | 22 days |

The original planet renderer receives index plus one; the comet renderer receives
the index itself. Larger vertical values appear lower, giving top-to-bottom order
7, 6, 5, 4, 3, 2, 1, 0. Return periods are derived within the post-reference walk,
not a continuity claim across modulo-100-year boundaries.

Redux's `calculatePlanetXPosition` preserves the reviewed date walk and enabled
positions. Its resized rendering preserves vertical order and places associated
comets on the same row, with margins for whole markers. Tests cover the reference
boundary, 13-month rollover, vertical order, and lower-planet/telescope clearance.
This Match concerns track calculation/order, not pixel-identical drawing.

**Implementation:** `internal/ui/widgets/astronomy_dialog.go`
**Tests:** `internal/ui/widgets/astronomy_dialog_test.go`

### ASTRO-0006

**Redux flexible Sextant and earned navigation**

**Kind:** rule
**Uses:** ASTRO-0003
**Original:** Established
**Redux:** Intentional deviation

Original Sextant Use explicitly reports the Avatar's position under the instrument
gates above. Redux's default eligibility is an ownership-inclusive, callback-free
query over map/floor/hour. It includes 05:59, excludes 06:00 and 18:59, and includes
19:00. Town outdoor tiles, Underworld, dungeons, combat, and virtual maps are not
default eligible. Sky rendering, ambient tint, weather, vehicles, and wall time
are not eligibility authority.

**REDUX DEVIATION: ASTRO-0006:** optional `FlexibleSextantUse` permits every valid
hour on Britannia's surface and reference-backed floors of surface settlements,
including rooms, upper levels, and cellars. Ownership remains mandatory. Ararat,
actual dungeons, Underworld, combat/camping, and virtual contexts remain excluded.
An interior fix reports the settlement's canonical surface exterior, never local
room coordinates or a stale last world position. The map labels this as
`Within <learned name>` or `Within this settlement`.

This navigation convenience preserves earned knowledge: enabling the preference
or opening a map does not grant a visit/name/survey. Explicit interior Use can
measure only the already genuinely entered settlement. Acquired flexible survey
provenance survives disabling the option. Observation and measurement dates remain
distinct; absent historical timestamps stay absent.

Redux additionally remembers encountered dungeon targets and automatically surveys
them without extra time, RNG, prompts, or resources. Look uses its pre-command
sample; Enter retains accepted source eligibility across its fade; qualifying
Yell uses its action context. A survey earned at 05:59 stays known after dawn.
Moonstone-specific target/arrival rules are canonical in
[MOON-0008](MOONSTONE_LOCATIONS.md#moon-0008). Broader acquisition/disclosure rules
remain owned by their navigation topics, not an automatic survey of every landmark.

**Implementation:** `internal/game_state/sextant.go`, `internal/game_state/journal_dungeons.go`
**Tests:** `internal/game_state/sextant_test.go`, `internal/game_state/sextant_flexible_test.go`
