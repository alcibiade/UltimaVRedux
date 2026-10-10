# Character creation

Original DOS runtime loads a template; generator definitions are not verification
of shipped INIT files. Original name-editor implementation and RNG distribution
remain unresolved. Creation, illustrated introduction, and Journey are separate.

### CREATE-0001

**Template workflow, input, and save order**

**Kind:** rule
**Uses:** SAVE-0001, SAVE-0002
**Original:** Established
**Redux:** Intentional deviation

Load PROPORT.PCS/CREATE.16 and the saved-state region from INIT.GAM, then read name
and sex, present framing/seven questions, and patch final Avatar attributes.
Name requests at most eight characters into nine-byte storage; empty submission
skips the creation/save body. Sex repeats until case-folded M/F and stores 0B/0C
hex, not ASCII. Exact editing/Escape behavior of this linked name editor is unknown.

Read 256 INIT.OOL bytes into the second half of a 512-byte buffer and zero the
first half. Write **SAVED.OOL before SAVED.GAM**, with no overwrite confirmation,
then return to the title menu; Journey is another selection. INIT.OOL is an
Underworld large-map object seed, not small-map NPC data. Journey splits the saved
halves into BRIT.OOL and UNDER.OOL.

**REDUX DEVIATION: CREATE-0001:** Redux uses a fresh deterministic seed per attempt,
supports cancellation and native-slot overwrite confirmation, and enters gameplay
after saving, providing a modern explicit save workflow. Its questionnaire validates
one to eight ASCII letters; that is not proof of the unavailable original editor.

**Implementation:** `internal/character_creation/questionnaire.go`, `internal/new_game/initial_state.go`

### CREATE-0002

**Virtue tournament and attribute result**

**Kind:** rule
**Uses:** CREATE-0001
**Original:** Established
**Redux:** Matches

Run rounds with four, two, one questions. Clear per-round used flags; retain
eliminated flags. Draw two indexes with R(0,7), rejecting used/eliminated virtues.
Present lower index as A, higher as B; accept only case-insensitive A/B. Mark loser
eliminated and immediately add winner bonuses, so surviving virtues can contribute
again. The symmetric question selection covers all 28 unordered pairs.

| Index / virtue | STR | DEX | INT |
| --- | ---: | ---: | ---: |
| 0 Honesty | 0 | 0 | 2 |
| 1 Compassion | 0 | 2 | 0 |
| 2 Valour | 2 | 0 | 0 |
| 3 Justice | 0 | 1 | 1 |
| 4 Sacrifice | 1 | 1 | 0 |
| 5 Honor | 1 | 0 | 1 |
| 6 Spirituality | 1 | 1 | 1 |
| 7 Humility | 0 | 0 | 0 |

Add bonuses to **loaded template** attributes, then floor STR at 20 and set MP to
final INT. Generator 15/15/15 is not hard-coded runtime input. No class selection,
final-virtue field, or answer history is written; only name, sex, STR/DEX/INT/MP
are direct Avatar patches. Creation does not reseed the original process RNG;
title startup seeded it earlier. Redux matches tournament/modifier structure,
not that random stream; its seed provenance is independently validated.

**Implementation:** `internal/character_creation/questionnaire.go`
**Tests:** `internal/character_creation/questionnaire_test.go`

### CREATE-0003

**Generator-defined initial party and provisions**

**Kind:** rule
**Uses:** CREATE-0002, MOON-0001
**Original:** Established
**Redux:** Unverified

These are generator values, not a fresh shipped INIT.GAM measurement:

| Member | Class/status | STR/DEX/INT | MP | HP/max | XP/level |
| --- | --- | --- | ---: | --- | --- |
| Avatar | Avatar/Good | 15/15/15 before answers | 0 before patch | 60/60 | 150/2 |
| Shamino | Fighter/Good | 20/22/16 | 0 | 5/60 | 167/2 |
| Iolo | Bard/Good | 19/21/17 | 8 | 90/90 | 249/3 |

Avatar starts with Chain Coif, Chain Armour, Long Sword, Ankh; second hand/ring
absent. Remaining roster order: Mariah, Geoffrey, Jaana, Julia, Dupre, Katrina,
Sentri, Gwenno, Johne, Gorn, Maxwell, Toshi, Saduj.

| State/resource | Generated value |
| --- | --- |
| Calendar | 139-04-05 08:35 |
| Location | Iolo's Hut/map 13, floor 0, (15,15), three active members |
| Karma / form / sight / opening flag | 75 / 1C hex walking / 5 / enabled |
| Shadowlord assignments / Watch | All zero / held FF hex |
| Food / gold; Keys / Gems / Torches / Skull Keys | 63 / 150; 2 / 0 / 4 / 0 |
| Small Shield / Cloth Armour / Dagger / Club / Flaming Oil | 1 / 2 / 6 / 1 / 3 |
| In Lor / Grav Por / An Zu / An Nox | 6 / 5 / 3 / 10 |
| Mani / An Xen Cor / Vas Lor / Vas Flam | 8 / 4 / 2 / 2 |
| Protection scroll / Yellow potion | 2 / 3 |
| Ash / Ginseng / Garlic / Silk | 4 / 6 / 7 / 6 |
| Moss / Pearl / Nightshade / Mandrake | 0 / 3 / 0 / 0 |

Other listed special items start zero; moonstones use MOON-0001. The generator does
not assign every allocated byte. Redux consumes player-supplied INIT.GAM/INIT.OOL
and converts after limited patches; it does not manufacture missing binary proof.

### CREATE-0004

**Creation artwork and opening population boundary**

**Kind:** rule
**Uses:** CREATE-0001
**Original:** Incomplete
**Redux:** Unverified

CREATE.16 shape 0 is oracle framing, 1 incense at (16,0)/(200,0), shapes 2–9 are
virtues in index order, and 10 the shrine conclusion. Left virtue positions are:
Honesty (40,5), Compassion (48,7), Valour (48,4), Justice (40,10), Sacrifice (40,8),
Honor (48,0), Spirituality (40,5), Humility (48,6). Right symbols are 184 pixels
east. QUESTION.DAT framing offsets are 0 and 802. No explicit creation music/tone/
noise call was found; pixels and narrative payload remain unverified here.

First Journey into a saved small map retains its active-object snapshot and skips
normal NPC loading/scheduling/placement. Normal re-entry rebuilds authored actors.
The old precise opening Rats/Smith counts/coordinates require shipped INIT.GAM and
NPC bytes; this control flow alone proves neither. Redux's hydrated opening and
normal re-entry have provisioned regression coverage, not universal release proof.

**Tests:** `internal/new_game/initial_state_original_test.go`
