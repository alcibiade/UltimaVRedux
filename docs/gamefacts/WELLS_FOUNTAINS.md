# Wells and fountains

These procedural Look paths have distinct selection and commitment rules.
Original binary placement and external random distribution are separate evidence.

### WELL-0001

**Wishing-well payment, matching, and placement**

**Kind:** rule
**Uses:** TIME-0002
**Original:** Established
**Redux:** Intentional deviation

Negative confirmation spends nothing. Affirmative with zero gold returns before
wish input; otherwise consume one gold and read up to 12 bytes. Return submits;
empty submission still spends. Escape clears text rather than submitting/cancelling.
Search each complete accepted term inside the entered response, case-insensitively:
Horse, Corvette, Ferrari, Lamborghini, Lotus, Porsche. Partial aliases are not
admitted. The original matcher can skip overlapping starts after partial mismatch,
so arbitrary decorated containing text is not guaranteed to match.

After matching/payment, only Paws/map 22 and Empath Abbey/map 31 succeed. Create
a horse one tile east of the **party**, on the supplied level, without boarding:
the Look caller supplies party coordinates, not well coordinates. Placement has
no terrain/occupancy/town-width check and shared allocation can replace an object.

**REDUX DEVIATION: WELL-0001:** Redux revalidates a submitted wish before spending,
allowing free Escape cancellation; every accepted submission still spends one.
It retains Mustang as an extra alias and directly boards an unmounted party on
eligible maps, avoiding a temporary map unit. Wrong-map/mounted/no-match submissions
have no horse effect. Modern normalized substring matching is not exact emulation
of the old overlapping-match quirk.

**Implementation:** `internal/game_state/wishing_well_service.go`
**Tests:** `internal/game_state/wishing_well_location_test.go`

### WELL-0002

**Surface fountain is mechanically cosmetic**

**Kind:** rule
**Uses:** TIME-0002
**Original:** Established
**Redux:** Unverified

Explicitly select a character; cancellation stops. Dead/Sleeping selections refuse;
other selections receive refreshing feedback with no HP, MP, status, or attribute
mutation. Look owns the surrounding time. Redux adds presentation/audio around
this cosmetic effect; full selector/exceptional-state equivalence is not asserted.

**Implementation:** `internal/game_state/action_look_fountain.go`

### WELL-0003

**Dungeon fountain exact-byte effects**

**Kind:** rule
**Uses:** LIGHT-0005, TIME-0002
**Original:** Established
**Redux:** Unverified

Resolve the common actor selector, then light, direction, drinking confirmation,
and exact-cell effect. Active designation is trusted by the original selector;
fallback enforces Good/Poisoned. Cell 50 hex sets Good, 51 fills HP without a
separate poison cure, 52 sets Poisoned, and 53–5F damages with R(0,7), including
zero. Cells remain repeatable and unchanged. Helpers add no intrinsic minutes.
Redux's dungeon interaction authority represents these effects; action ownership
and selector differences must remain separate from numeric effect agreement.

**Implementation:** `internal/game_state/dungeon_interactions.go`
**Tests:** `internal/game_state/dungeon_interactions_test.go`
