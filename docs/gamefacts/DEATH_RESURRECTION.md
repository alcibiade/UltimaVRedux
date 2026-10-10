# Death and resurrection

Karma consequences remain canonical in the Karma rules. Recovery entry points
have different resources and presentation, even when they share a resurrection helper.

### RECOVERY-0001

**Individual revival and caller commitment**

**Kind:** rule
**Uses:** KARMA-0015, PROGRESS-0001, INV-0006, INV-0007
**Original:** Established
**Redux:** Unverified

Negative original target returns failure; non-Dead target returns no revival,
with feedback depending on mode. Dead success sets Good/one HP, restores class MP,
applies KARMA-0015, recalculates level and maximum HP=30*level, selects mode-specific
sound, then marks stats dirty. No stat-growth award occurs. Prepared spell and
scroll modes have different sound; healer/recovery mode suppresses it.

The non-CHEAT prepared path commits mixture before MP, MP before level and later
targeting. Scroll consumes before context/target checks and dispatches revival
only below map ID 128. Healer validates Dead, takes payment, revives, then fills
new maximum HP. Party recovery calls the helper per active slot then fills HP.
Redux shares individual resurrection mechanics and healer topping-up, while its
validated Cast/Use commitments differ; normal final-state agreement is narrower
than every original caller/resource path.

**Implementation:** `internal/game_state/effect_spells.go`

### RECOVERY-0002

**Full-party non-karma restoration**

**Kind:** rule
**Uses:** KARMA-0016, KARMA-0017, RECOVERY-0001, TIME-0001
**Original:** Established
**Redux:** Unverified

After pre-recovery report/resurrection/karma order in KARMA-0016, return to Castle
Britannia map 17, floor 1,(10,10), walking; clear duration effect and duration;
request nine minutes repeatedly until hour equals 6; clear torch/magic light;
replace zero food with 63, otherwise preserve it. If already in hour 6 the loop
does not advance. It neither fixes minute to zero nor preserves the starting
minute unchanged; rollover side effects remain active. When recovery began in
a small map, the final town initialization is the **current destination castle**,
not the departed town.

Redux's shared consequence mutator and guardian restoration retain the main
destination, low-food, class-MP, and karma policies. Its penalty helper visits
every active member rather than relying on an individual Dead guard; ordinary
wipe eligibility and other recovery callers must therefore stay distinguished.
Wanderer and inspected shrine paths do not revive Dead members; no original
shrine resurrection is established.

**Implementation:** `internal/game_state/death_resurrection.go`
