# Potions and scrolls

Eight potion colors and eight scrolls have separate Use rules; they do not inherit
prepared-casting resource gates. Original means the inspected material labelled
1.16, without immutable corpus or retail-build identification. R(a,b) records a
random request; original-generator uniformity and expression evaluation order
remain unestablished. Surface means outdoors or town/small maps, excluding
first-person dungeons and tactical combat.

### CONSUME-0001

**Item selection, targeting, and commitment**

**Kind:** rule
**Uses:** INV-0007
**Original:** Established
**Redux:** Intentional deviation

The original Use selector chooses an owned item, then the potion/scroll routine
immediately decrements its stock byte. Context checks, directions, member
selection, potion substitution, resistance, and effect validation occur afterward.
Cancellation and failure do not refund the item. Combat Use is PC-only and belongs
to the current entry; combat potions automatically use that entry's roster member.
Every noncombat potion, even White, prompts for a member first.

Redux deliberately separates review, final target confirmation, commitment, and
effect resolution so cancellation and stale/prerequisite rejection are free:
no item, gameplay time, action, or RNG is spent before final commitment. A confirmed
attempt spends one item exactly once, and ineffective targets or later failures
are not refunded. One-shot potion review retains location, context, stock, and
exact combat identity; effect resolution uses the retained entry, not a later
occupant of a reused slot. Presentation does not consume a second item.

**Implementation:** `internal/game_state/action_use_commits.go`, `internal/game_state/action_use.go`
**Tests:** `internal/game_state/action_use_commits_test.go`, `internal/game_state/cast_use_commit_test.go`

### CONSUME-0002

**Potion substitution and all eight effects**

**Kind:** rule
**Uses:** RNG-0001
**Original:** Incomplete
**Redux:** Intentional deviation

After consumption and member selection, the original plays the labeled color's
sound and requests R(0,15). Zero forces Orange; one requests another R(0,7) potion
index; 2–15 retain the label. A replacement may equal the label. No original
mislabel warning is emitted. Exact odds require an assumed uniform generator.
Substitution happens after targeting, so a replacement status potion can fail
against the already chosen member, and a replacement White still followed a prompt.

| Index | Color | Original effect and gate |
|---:|---|---|
| 0 | Blue | Sleeping roster member becomes Good. In combat, also require the acting entry to be its sleeping PC instance before waking that entry; roster status may already have changed when this check fails. |
| 1 | Yellow | Non-Dead member gains one transformed d30 HP, capped at maximum. A full-health living member still consumes the roll and succeeds. |
| 2 | Red | Poisoned becomes Good; other statuses fail. |
| 3 | Green | Good becomes Poisoned; other statuses fail. |
| 4 | Orange | Requires Good. Noncombat sets Sleep. Combat passes the roster index to a queue-indexed sleep operation, potentially affecting a different entry. |
| 5 | Purple | Combat only: replace acting object's displayed tile and shape with Rat. This does not replace class, attributes, equipment, or identity. Else no noticeable effect. |
| 6 | Black | Combat only: mark the acting entry invisible and use invisible-PC appearance. Else no noticeable effect. |
| 7 | White | Surface only: local X-ray view for 20 update ticks. Dungeon/combat have no noticeable effect. |

Redux preserves the conditional substitution requests and Yellow's transformed
d30 after them. White's outdoor/town support is a scoped match, not an extension.
Purple is an exact-entry encounter-local appearance change, including random
substitution: no replacement monster, creation-DEX roll, allegiance change, or
countdown change. Blue retains roster-first status handling and exact-entry wake.

Two deliberate changes improve understandable feedback and identity safety:
Redux announces when the resolved color differs from the label, and Orange sleeps
the exact drinking PC/clone, updating its tactical sleep and roster status instead
of reproducing the original index mismatch. A separate drinking cue accompanies
committed use, including a no-notice result.

**Implementation:** `internal/game_state/action_use_commits.go`
**Tests:** `internal/game_state/action_use_commits_test.go`

### CONSUME-0003

**All eight scroll effects and context exceptions**

**Kind:** rule
**Uses:** SPELL-0004, SPELL-0005
**Original:** Established
**Redux:** Matches

| Index | Scroll | Original effect and context |
|---:|---|---|
| 0 | Vas Lor | Any context; magical light becomes 240. |
| 1 | Rel Hur | Ask direction before context validation. Surface sets wind; passing clears it. Dungeon/combat fail after expenditure. |
| 2 | In Sanct | Any context; shared Protection for 100 duration units, subject to the original effective-armor uncertainty. |
| 3 | In An | Any context; shared Negate Magic for 20 duration units. |
| 4 | In Quas Wis | Surface local map view or dungeon symbolic view; combat has no effect. |
| 5 | Kal Xen Corp | Combat only; up to eight placement attempts for a Daemon, always loyal without the prepared spell's binding roll. |
| 6 | In Mani Corp | Noncombat only; select a member and run common resurrection. |
| 7 | An Tym | Any context except direct Stonegate or Doom use; shared Time Stop for 20 duration units. Blocked use has no effect after expenditure. |

Some context-rejection branches print failure without changing the routine's
initial success result. Inventory loss and lack of effect are established; a
uniform Boolean success contract is not. Placement failure and a living
resurrection target remain paid in the original. Duration scrolls replace or
refresh the shared slot; Light has its own counter. Their durations differ from
the corresponding prepared spells.

The scoped Redux match is effect identity, durations, town support for Rel Hur,
and guaranteed Daemon loyalty on the combat scroll route. Free cancellation is
CONSUME-0001. Wind commitment accepts a final cardinal direction, rather than
using original Pass to clear wind. Dungeon View uses its dedicated symbolic-view
route. Combat invocation uses the combat scene route and authoritative summon
operation; the generic noncombat scroll dispatcher is not evidence that summoning
is unsupported. Time Stop's direct-location no-effect check remains a paid effect.

**Implementation:** `internal/game_state/action_use.go`, `internal/game_state/combat_summon_spells.go`, `cmd/ultimav/gamescene_use_scroll.go`, `cmd/ultimav/gamescene_combat_invocation_effect.go`
**Tests:** `cmd/ultimav/gamescene_combat_use_test.go`, `internal/game_state/action_use_scroll_integration_test.go`

### CONSUME-0004

**Resurrection shared by scroll and prepared spell**

**Kind:** rule
**Uses:** PROGRESS-0001, PROGRESS-0002
**Original:** Established
**Redux:** Unverified

A Dead target becomes Good at 1 HP. MP becomes INT for Avatar/Mage and floor(INT/2)
for Bard; other classes retain their prior MP. When karma is below 98, XP is
scaled by karma/100 using integer arithmetic. Level is then recalculated as one
plus the number of right shifts needed to reduce floor(XP/100) to zero, and maximum
HP becomes level × 30. Progression rules remain canonical in the dependencies.
A non-Dead selection has no effect, but original scroll expenditure already occurred.

Redux's confirmed valid roster target spends the resurrection scroll before
applying its authoritative resurrection operation; a confirmed living target is
paid and ineffective. Complete exceptional-state and class/XP equivalence is not
claimed by this resource-boundary review.

**Implementation:** `internal/game_state/action_use.go`, `internal/game_state/effect_spells.go`
