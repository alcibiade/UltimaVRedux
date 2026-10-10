# Tribute, guard reactions, and prison

Rules are scoped to the inspected DOS source model. Its version label alone is
not retail-build proof. Authored NPC observations retain their separate data
limitations. Karma arithmetic is owned by the referenced Karma rules.

### CRIME-0001

**Procedural tribute and Palace credentials**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Intentional deviation

Tribute is a specialized interaction, not an ordinary TLK script.

| Context | Demand and result |
| --- | --- |
| Ordinary small map | 10 gold for each active non-Dead member. Refusal or insufficient gold requests arrest; accepted affordable tribute deducts the total. |
| Minoc, map 5 | Refusal requests arrest. Acceptance sets gold to `floor(gold/2)`; odd totals leave the smaller half. |
| Palace of Blackthorn, map 18 | Require active Badge state, not merely inventory ownership. Compare only first four entered characters with IMPE, case-insensitively; missing Badge or mismatch requests arrest. |

Original text-byte comparison also masks high bits. Redux's Unicode string
comparison is reviewed for ordinary entered ASCII credentials, not every raw
high-bit byte. Redux review/commit guards the quoted location and gold state;
the tribute action does not own Talk timing.

**REDUX DEVIATION:** Insufficient tribute adds an explicit unpaid-tribute
explanation beside the surrender prompt. Original behavior requests arrest
without explaining lack of funds; the added feedback changes neither arrest
nor completion ownership.

**Implementation:** `internal/game_state/guard_extortion.go`, `cmd/ultimav/gamescene_talk_workflow.go`
**Tests:** `internal/game_state/guard_extortion_test.go`, `cmd/ultimav/gamescene_guard_tribute_test.go`

### CRIME-0002

**Ordered guard activation and shape-classified contact**

**Kind:** rule
**Uses:** RNG-0001
**Original:** Established
**Redux:** Matches

Visit active NPC slots 0–31 in ascending order. Guard, Devil/Daemon, and
Shadowlord shapes become attacking. Every other active slot consumes one
`0..255` draw first; below 128 attempts fleeing. Only afterward does flee
eligibility test ordinary-NPC-through-before-Wanderer shape and either a nonzero
schedule marker or Astaroth Talk marker. Accepted fleeing installs fearful Talk
state and all three flee actions. Rejected candidates still consumed their draw.
The external random bounds do not independently prove a 50% distribution.

NPC phase contact retains the last eligible command/source during ascending
scan. Astaroth's special Talk marker is handled first; otherwise only Guard
shape routes to arrest. Other eligible shapes, including Devils, enter combat
without tribute or surrender dialogue. Redux keeps criminal bookkeeping
caller-owned; activating guards itself is not an extra karma or criminal-state
mutation.

**Implementation:** `internal/game_state/guard_activation.go`, `internal/ai/npc_ai_controller_small_map.go`
**Tests:** `internal/game_state/guard_activation_test.go`, `internal/ai/npc_ai_controller_small_map_contact_test.go`

### CRIME-0003

**Ordinary surrender, Time Stop, and Palace routing**

**Kind:** rule
**Uses:** CRIME-0002, TIME-0001, KARMA-0013
**Original:** Established
**Redux:** Intentional deviation

Outside the Palace, surrender changes location to Yew, map 4, coordinate
`(25,4)`, then repeatedly requests 20-minute advances until hour equals 8.
Minutes are not reset. It then sets floor 0, confiscates all ordinary Keys,
and reloads the town. Refusal activates guards and requests combat.
Under original Time Stop those time requests do not advance, so this loop
does not terminate unless hour already equals 8.

**REDUX DEVIATION:** Redux advances sentence time despite Time Stop while
preserving the effect, avoiding a hang. It exposes Yew to calendar systems
during sentencing, installs the destination after the elapsed loop, confiscates
Keys, and clears its criminal status. This is not permission for ordinary
commands to bypass Time Stop.

Palace arrest bypasses ordinary surrender and enters capture/interrogation,
then reloads if anyone survives. The old documentation's statement that this
sequence is absent from Redux is obsolete: current guard and failed-credential
workflows route to its implemented interrogation. The current implementation
selects the first unruined shrine, installs the prison mini-map if a question
remains, and on completion returns to Palace basement `(10,7)` with Keys zero.
KARMA-0013 owns correct-answer/shrine consequences. This correction establishes
the live route, not complete frame-for-frame capture fidelity or all torture
timing under every temporal modifier.

**Implementation:** `internal/game_state/guard_arrest.go`, `internal/game_state/blackthorn_interrogation.go`, `cmd/ultimav/gamescene_talk_workflow.go`
**Tests:** `internal/game_state/guard_jail_test.go`, `internal/game_state/blackthorn_interrogation_test.go`

### CRIME-0004

**Town attack consequences and persistent actor absence**

**Kind:** rule
**Uses:** KARMA-0003, CRIME-0002
**Original:** Established
**Redux:** Unverified

An intact mirror is resolved before object lookup and breaks without crime
penalty. Accepted ordinary-NPC attacks apply KARMA-0003's first deduction and
activate guards while the victim is still loaded. Defenseless terrain then
selects murder or Blackthorn's miss; otherwise combat begins. The karma rule
owns precise tile gates and separately floored deductions.

Persistent death for eligible non-Guard ordinary NPCs is recorded on combat
entry, whether initiated by party or NPC, irrespective of eventual victory.
Guards and monster shapes do not receive that bit; the triggering actor is
removed from the current visit and may return on a later load. A Crawler-shaped
quest-object persistence exception exists, but town Attack excludes that shape.

Explicit Devil attacks activate guards without the civilian attack deduction.
Defeat does not persist the Devil death bit. Other eligible monster attacks
have neither that deduction nor guard activation. Hostile non-Guard contact
itself has neither consequence. Combat group composition remains independent.

The baseline data audit identifies two Windemere Devils, three Iolo's Hut Rats,
and five Skara Brae Bats; the latter two sets use single encounters and return
on later location loading. These exact authored populations were not independently
reopened here. Redux routes eligibility by shape instead of a location allowlist
and persists ordinary civilian absence on entry; all malformed object mappings
and uninspected authored editions remain outside the match scope.

**Implementation:** `internal/game_state/action_attack.go`, `internal/game_state/combat_small_map_encounter.go`
**Tests:** `internal/game_state/action_attack_murder_integration_test.go`, `internal/game_state/action_attack_mirror_test.go`

### CRIME-0005

**Jimmy rescue and failed physical unlocking**

**Kind:** rule
**Uses:** PROV-0004, KARMA-0008
**Original:** Established
**Redux:** Matches

Original Jimmy requires a Key before direction and character selection. On
occupied stocks/manacles, select an actor and apply PROV-0004's physical Dexterity
check. Failure consumes one Key and returns. Success keeps the Key, resolves
the NPC, and clears nonzero Talk state. Only when its persistent absence bit
is not already set does it change all three actions to follow, acknowledge
rescue, and award KARMA-0008's rescue benefit. Finally it records persistent
absence. Thus success, repeated rescue eligibility, and persistence are separate.

Ordinary physical doors share the failure-only Key rule. Magical locks consume
one Key without unlocking. These are not chest-disarm rules; chest knowledge and
safe refusal have their own command policy. Redux authoritative prisoner actions
retain the reviewed success/failure order and reward guard.

**Implementation:** `internal/game_state/action_jimmy.go`
**Tests:** `internal/game_state/action_jimmy_prisoner_test.go`
