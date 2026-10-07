# Merchant catalogs and service transactions

This specification separates inspected DOS source behavior, provisioned-data
limits, and current Redux decisions. A source version label is not a fingerprint
of a shipped release. Tables describe active mappings and separately identified
unused entries; they do not authorize activating unused stock. Prices are gold,
coordinates are tiles, and integer division truncates toward zero unless stated
otherwise. Dialogue payloads are not reproduced.

### MERCH-0001

**Families, availability, and the pricing character**

**Kind:** rule
**Uses:** NPC-0001
**Original:** Established
**Redux:** Matches

| Talk code | Family | Active mapped rows |
| --- | --- | ---: |
| `81` hex | Arms | 9 |
| `82` | Tavern | 9 |
| `83` | Horse | 3 |
| `84` | Ship | 4 |
| `85` | Reagent | 5 |
| `86` | Guild | 3 |
| `87` | Healer | 7 |
| `88` | Inn | 6 |

A shop opens only when both last-arrived and currently desired schedule slots
are odd, which with three slots means slot 1. This prevents service while
approaching work and ends availability when departure is desired. There is no
universal shop-hours constant. Every family except Horse refuses riders.
The selected active character supplies Intelligence; absent a selection, scan
active roster order for the first Good or Poisoned character. Sleeping, Charmed,
and Dead do not qualify for that fallback. The merchant caller does not check
whether the fallback succeeded, so this does not establish a safe buyer when
none qualifies. Original Tavern salutation has an assignment defect
that clears active selection and inspects roster zero's sex rather than the
chosen character. The already-passed pricing character remains a separate input.

Redux resolves the real loaded actor by stable family/location/slot identity,
validates visibility, floor, schedule and riding state, then creates a session.
Quote/review is pure; committing uses current authoritative state. Dialogs emit
intent and refresh detached state after changes. These are current architectural
contracts, not original file-format claims. The match concerns supported family
dispatch and availability, not the original salutation defect or invalid rows.
Redux checks Good/Poisoned eligibility even for an explicit selection and falls
back when that selection is ineligible; it refuses a missing buyer. Original
dispatch trusts an explicit active index. Those exceptional selection paths are
not included in the scoped match or labelled intentional differences here.

**Implementation:** `internal/references/merchant.go`, `internal/game_state/merchant_session.go`
**Tests:** `internal/game_state/merchant_session_test.go`

### MERCH-0002

**Intelligence prices and distinct rounding boundaries**

**Kind:** rule
**Uses:** EQUIP-0001
**Original:** Established
**Redux:** Matches

For valid character Intelligence and nonnegative catalog base:

```text
adjusted_buy(base, INT) = base + trunc((100 - 3*INT) * base / 100)
arms_sale(base, INT)   = trunc(base * (3*INT) / 100) + 1
```

The buy adjustment applies to arms, reagent bundles, Guild bundles, horses,
ships/skiffs, Inn prices and Tavern provision packs. It does not price healer
services, Tavern meals, ordinary drinks, wine, paid information, or arms sales.
Base equipment values belong to EQUIP-0001 rather than another copied price list.
Pickup adjusts one monthly amount before multiplying accrued months; nightly
rest adjusts the party aggregate, not each member separately. No ordinary rule
establishes Charisma, karma markup, haggling, or replenishing finite stock.

**Implementation:** `internal/game_state/merchant_price.go`, `internal/game_state/arms_shop.go`, `internal/game_state/inn_shop.go`

### MERCH-0003

**Talk ownership, theft boundaries, and batch cadence**

**Kind:** rule
**Uses:** TIME-0002, SHADOW-0003
**Original:** Established
**Redux:** Intentional deviation

One completed player-initiated merchant Talk normally owns one town minute,
including cancellation and repeated transactions. It is not a minute per item.
Inn Rest additionally owns its overnight advances. Presentation waits are not
gameplay time; NPC-initiated Talk does not charge an extra player command.

Most paid purchases order validation/approval, payment, eligible Faulinei shop
theft, then grant. SHADOW-0003 owns the theft amount and automatic-presence gate.
Theft occurs after each original arms item, reagent/Guild bundle, horse, vessel
or loaded skiff, paid healer patient, Inn Rest or pickup, and Tavern meal/drink.
Normally completed Tavern provisions instead perform one final theft. Sales,
Leave, free/charitable care, information, cancellation, and failed transactions
do not steal. Ship delivery state is written before payment; provision partial
failure has a distinct early exit (MERCH-0008, MERCH-0014).

**REDUX DEVIATION:** Keyboard setting **Batch Merchant Theft**, persisted as
`Gameplay.Merchants.BatchTheft`, defaults on for convenient quantity purchases.
On: one payment and eligible theft per reviewed batch. Off: each original
economic unit independently pays, steals, grants, then checks the next unit's
affordability. Units are ordinary arms, reagent bundles, Guild bundles, loaded
skiffs, and paid healer patients. Ammunition remains one refill. Meals, drinks,
Inn services, and provisions retain their already-aggregate/distinct cadence.

Original-cadence theft can stop a batch early. Completed units stay granted;
receipts report actual count, payment, summed theft and final gold. Where exposed,
`GoldAfterPayment` is after the last completed payment and before its theft.
Quotes bound capacity/affordability before theft, not guaranteed completion.
Confirmation revalidates reviewed state before payment or RNG, and refreshes the
whole dialog from authority afterward. No rollback or automatic extension into
charity occurs. Sales are unaffected. Cosmetic merchant prose is deliberately
selected deterministically rather than consuming gameplay RNG as original
greetings, farewells, sale remarks, provision pitches and hint forms did.

**Implementation:** `internal/game_state/arms_shop.go`, `internal/game_state/reagent_shop.go`, `internal/game_state/guild_shop.go`, `internal/game_state/ship_shop.go`, `internal/game_state/healer_shop.go`, `cmd/ultimav/gamescene_merchant_talk.go`
**Tests:** `internal/game_state/merchant_batch_rules_test.go`

### MERCH-0004

**Arms catalogs, refills, and carried-stock sales**

**Kind:** rule
**Uses:** MERCH-0002, MERCH-0003, EQUIP-0010
**Original:** Established
**Redux:** Intentional deviation

| Location | Shop / owner | Stock in offered order |
| --- | --- | --- |
| Britain | Iolo's Bows / Gwenneth | Dagger, Sling, Bow, Arrows, Crossbow, Quarrels, Magic Bow |
| Jhelom | Naughty Nomaan's / Nomaan | Flaming Oil, Mace, Spiked Collar, Throwing Axe, Spiked Helm, Spiked Shield, Morning Star |
| Yew | Arms of Justice / Ronan | Leather Helm, Cloth, Leather Armour, Club, Spear, Silver Sword, Magic Axe |
| Minoc | Darkwatch Armoury / Shenstone | Iron Helm, Small Shield, Ring Mail, Short Sword, Long Sword, Mace, 2H Hammer |
| Trinsic | The Paladin's Protectorate! / Paul | 2H Axe, 2H Sword, Halberd, Iron Helm, Large Shield, Scale, Plate |
| Castle Britannia | North Star Armoury / Max | Chain Coif, Magic Shield, Chain, Plate, Long Sword, Silver Sword, Protection Ring |
| Buccaneer's Den | Buccaneers Booty / Kitiara | Leather Helm, Leather Armour, Dagger, Main Gauche, Short Sword, Flaming Oil, Invisibility Ring |
| Bordermarch | The Shattered Shield / Steve | Magic Shield, 2H Axe, Magic Bow, Arrows, 2H Hammer, Regeneration Ring, Turning Amulet |
| Serpent's Hold | Siege Crafters / Thol | Chain Coif, Chain, Crossbow, Quarrels, Halberd, Throwing Axe, Morning Star |

Original Talk selects Buy or Sell once. Ordinary Buy quotes and confirms one item,
then checks owned quantity 99 and affordability before paying, stealing,
refreshing statistics and granting into shared
carried stock. No recipient or equipment-eligibility choice is involved.
Arrows/Quarrels instead refill the stack to 99 for one quoted price. Capacity
refusal returns to Buy; insufficient funds ends it and suppresses farewell.

Sell lists all carried equipment, even outside the shop's stock, excluding
worn/readied slots. Each confirmation credits one offer, saturating gold at 9999,
then removes one item even when some/all value was discarded. No theft occurs.
Used ammunition is refused and ends Sell; a zero-base special item is refused
but returns to the list. Original sale inquiry, offer, follow-up and farewell
choose random prose variants.

**REDUX DEVIATION:** Buy/Sell can switch repeatedly within one Talk. Quantity
Buy uses MERCH-0003 cadence; ammunition stays one refill. Quantity Sell uses
`min(carried, floor((9999-gold)/unit_offer))`, with one atomic credit/removal,
to avoid discarding item value at the gold cap. Cosmetic prose is deterministic.

**Implementation:** `internal/game_state/arms_shop.go`, `cmd/ultimav/gamescene_merchant_talk.go`

### MERCH-0005

**Reagent bundles and paid partial capacity**

**Kind:** rule
**Uses:** MERCH-0002, MERCH-0003
**Original:** Established
**Redux:** Intentional deviation

Each entry is **base gold / units per bundle**; five active rows expose 22 offers.

| Location | Shop / owner | Bundles |
| --- | --- | --- |
| Moonglow | The Herbalist / Nilrem | Ginseng 20/10; Garlic 18/6; Spider Silk 12/4; Nightshade 12/1; Mandrake 13/1 |
| Yew | Healers Herbs / Madam Pendra | Sulfur Ash 12/12; Ginseng 16/8; Garlic 16/8; Spider Silk 8/2; Blood Moss 20/4 |
| Skara Brae | The Alchemist / Toama | Sulfur Ash 14/14; Ginseng 16/8; Blood Moss 30/6; Black Pearl 18/6 |
| Cove | Mysticism / Enlor | Spider Silk 6/2; Blood Moss 8/2; Black Pearl 8/2; Nightshade 10/1; Mandrake 15/1 |
| Lycaeum | The Sharper Mage / Virden | Blood Moss 50/4; Nightshade 30/1; Mandrake 40/1 |

Original quantity exactly 99 refuses before confirmation/payment and returns to
catalog. Each Yes buys one bundle: pay adjusted price, steal, refresh, then grant
saturated to 99. Full payment precedes saturation (95 owned plus eight purchased
receives four). Insufficient funds after Yes ends buying without mutation;
success permits another separately confirmed bundle. Greeting selects one of
four variants; ordinary polite/rude farewell independently selects one of four.
Insufficient funds suppresses farewell and its draw.

**REDUX DEVIATION:** Quote maximum is
`min(ceil((99-owned)/bundle_units), floor(gold/adjusted_price))`, with zero at
capacity. One approval can buy that many bundles using MERCH-0003 cadence. The
rounded-up capacity preserves a paid partial final bundle. Zero affordability
is blocked before confirmation without ending browsing or consuming RNG;
cosmetic variants are deterministic.

**Implementation:** `internal/game_state/reagent_shop.go`

### MERCH-0006

**Guild provisions and unused price cells**

**Kind:** rule
**Uses:** MERCH-0002, MERCH-0003
**Original:** Established
**Redux:** Intentional deviation

| Row | Mapping / shop / owner | Keys ×3 | Gems ×4 | Torches ×5 | Unexposed fourth price |
| ---: | --- | ---: | ---: | ---: | ---: |
| 0 | New Magincia / The Den / Braunam | 190 | 255 | 12 | 280 |
| 1 | Paws / The Guild / Danfits | 160 | 200 | 11 | 175 |
| 2 | Buccaneer's Den / The Nemesis / Daem | 185 | 225 | 25 | 265 |
| 3 | Unmapped | 170 | 150 | 20 | 300 |

Prices are bases. Original confirmation buys one fixed bundle with no capacity
precheck: pay, steal, refresh, then saturating grant to 99, which may grant zero.
Success returns to selection. Insufficient gold exits buying; menu D only
displays information and never buys the unexposed fourth column. Intended use
of the unmapped row and fourth column remains unknown.

**REDUX DEVIATION:** Redux blocks zero useful capacity and bounds bundles by
rounded-up remaining capacity and affordability, like MERCH-0005, then applies
MERCH-0003 cadence. This avoids knowingly paying for zero receipt while retaining
a partially useful final bundle. It does not activate unused price entries.

**Implementation:** `internal/game_state/guild_shop.go`

### MERCH-0007

**Healers, free care, and separately approved charity**

**Kind:** rule
**Uses:** MERCH-0003, KARMA-0015
**Original:** Established
**Redux:** Intentional deviation

Prices ignore Intelligence.

| Location | Shop / owner | Cure | Heal | Resurrect |
| --- | --- | ---: | ---: | ---: |
| Minoc | The Healers Mission / Regina | Free | Free | 200 |
| Trinsic | Wounds of Honour / Leila | 25 | 40 | 215 |
| Skara Brae | The Spirit Healers / Temptious | 30 | 45 | 225 |
| East Brittany | Healers' Sanctum / Milan | 35 | 50 | 237 |
| Cove | Sanctuary / Jessica | 40 | 55 | 247 |
| Lycaeum | The Shield of Truth / Faye | 15 | 60 | 249 |
| Empath Abbey | The Empath / Jessip | 10 | 65 | 262 |

Minoc's backing Cure/Heal values are 20/35, bypassed by its free-care branch.
The unused eighth row is 1/70/270. Original charity accepts an unaffordable
confirmed service only at Skara Brae with price at most 100; with active prices
this means Cure/Heal, never Resurrection. It takes no partial payment or theft.

Cure requires Poisoned and sets Good. Heal rejects Dead or full HP, restores
maximum HP and preserves Poisoned/Sleeping/Charmed status. Resurrection requires
Dead, applies KARMA-0015's XP/level recalculation, then fills recalculated HP.
One-member parties auto-select zero; others choose an active recipient before
eligibility. Decline, cancellation, no need and ordinary nonpayment return to
the service loop unchanged. Paid order is payment, theft, presentation effect,
treatment; free/charitable care skips payment/theft. Repeated services share
Talk time. Greeting/farewell each originally draw among four prose variants.

**REDUX DEVIATION:** All treats eligible active members in roster order, using
MERCH-0003 for paid patients. At Skara Brae it first offers the maximum affordable
paid subset. Only a fresh review when no additional paid patient is affordable
offers remaining Cure/Heal recipients free. Charity never silently extends a
paid confirmation. For two wounded patients, gold 0 or 44 offers both free;
45, 50 or 89 offers one paid; 90 offers two paid. Starting at 50, paying 45
without theft leaves five and a later free offer for the other patient. Theft
can truncate the paid batch and the next review recalculates. Minoc remains
free; Resurrection never charitable. Cosmetic choices are deterministic.

**Implementation:** `internal/game_state/healer_shop.go`
**Tests:** `internal/game_state/healer_shop_test.go`, `internal/game_state/merchant_batch_rules_test.go`

### MERCH-0008

**Horse placement and ship delivery are different purchases**

**Kind:** rule
**Uses:** MERCH-0002, MERCH-0003
**Original:** Established
**Redux:** Intentional deviation

| Source row | Horse location mapping | Shop / owner | Base | Availability |
| ---: | --- | --- | ---: | --- |
| 0 | Trinsic | Horse & Rider / Hettar | 100 | Active supported merchant |
| 1 | North Brittany | The Stablehouse / Theoan | 130 | Active supported merchant |
| 2 | Paws | Wishing Well Horses / Ferru | 160 | Active supported merchant |
| 3 | Lycaeum | Null shop name / Simplon | 190 | Source mapping only; no supported active merchant |

Horse dealers admit riders. Before greeting/pricing they require a free slot in
the 32-object array, then search south, north, east, west for an unoccupied
same-floor tile of Brick Floor, Metal Floor, or Grass on mutable terrain. Failure
only reports closed stables. Two affirmative decisions precede payment; either
decline, insufficient funds, and success terminate. Success pays adjusted price,
steals, creates the horse on the reviewed square and refreshes. It is a town
object, not inventory: a full object rebuild loses an unboarded horse; boarding
transfers transport into player vehicle state. The fourth row really is mapped
to Lycaeum in the source table; it is not an unmapped price row. Its null shop
name and the three active supported merchants are separate facts. Active-NPC
absence there is inherited from the baseline audit, not a new binary inspection.

| Ship location | Shop / owner | Frigate base | Skiff base | Britannia berth (hex) |
| --- | --- | ---: | ---: | --- |
| Jhelom | Island Shipwrights / Bantral | 600 | 200 | `(27,DD)` |
| Minoc | The Crow's Nest / Captain Blyth | 753 | 175 | `(97,15)` |
| East Brittany | The Oaken Oar / Master Hawkins | 650 | 125 | `(4F,6D)` |
| Buccaneer's Den | The Rusty Bucket / Jones | 700 | 100 | `(8A,9F)` |

Original pending delivery uses one byte: `0x40` means standalone skiff; high bit
marks frigate and low six bits its skiffs. A frigate writes `0x82`, including two.
On acceptance and funds check, set/increment delivery state, then deduct adjusted
price, steal, refresh and ask to continue. Another skiff increments pending
frigate encoding without a 63-skiff guard. A pending standalone skiff blocks
another without payment. The frigate no-space branch tests pending delivery bits,
not exact live berth occupancy: it displays 87,000, checks 10,000, and grants/
deducts nothing. Its intended design remains unknown.

Next outdoor entry materializes pending values at least `0x40` at the saved
berth, copying low six bits, hull 99; above `0x7F` becomes east-facing anchored
frigate, otherwise east-facing skiff. It clears the pending byte. There is no
original harbour escort or added delivery time.

**REDUX DEVIATION:** Horse final-approval cancellation returns to browsing.
Ship capacity and exact berth occupancy refuse atomically before payment/theft;
frigates cap at 12 skiffs. A delivered frigate at the canonical berth can receive
skiffs, using MERCH-0003 batch cadence. A separately approved escort can move the
party to an unoccupied cardinal Dock beside the vessel for three additional
minutes, never auto-boarding; escort failure does not undo purchase. These
choices provide bounded placement and convenient collection rather than the
inconsistent original no-space branch. Cosmetic horse prose is deterministic.

**Implementation:** `internal/game_state/horse_shop.go`, `internal/game_state/ship_shop.go`

### MERCH-0009

**Inn catalog, captured buyer, vacancy, and monthly charges**

**Kind:** rule
**Uses:** MERCH-0002, TIME-0001
**Original:** Established
**Redux:** Matches

| Location | Inn / owner | Nightly base | Guest capacity | Sleep → wake `(x,y,floor)` |
| --- | --- | ---: | ---: | --- |
| Britain | The Wayfarer Inn / Donya | 2 | 3 | `(21,10,0)` → `(22,10,0)` |
| Jhelom | The Warrior's Stead / Gremnor | 3 | 4 | `(15,7,1)` → `(16,7,1)` |
| Skara Brae | The Haunting Inn / Rogi | 2 | 3 | `(25,9,0)` → `(26,9,0)` |
| North Brittany | Hotel Brittany / Terbor | 3 | 2 | `(20,1,0)` → `(21,1,0)` |
| Paws | The Smugglers' Inn / Lorien | 2 | 2 | `(27,6,0)` → `(28,6,0)` |
| Buccaneer's Den | The King's Ransom Inn / Ransack | 3 | 2 | `(7,26,0)` → `(8,26,0)` |

Capture talking Intelligence and sex on opening; services retain these even if
roster order changes, while a new Talk captures anew. Vacancy counts all 16
records whose stored map equals this Inn, compared with capacity. Nightly price
adjusts `base * active_party_size`; monthly rate adjusts `base * 10`; pickup
multiplies that rounded rate by `max(stored_months,1)`. On each 28-day month
rollover, every roster record's month counter below 25 increments regardless
of stored location; 25 and above remain unchanged. This is calendar maintenance,
not a per-service charge.

**Implementation:** `internal/references/inn.go`, `internal/game_state/inn_shop.go`
**Tests:** `internal/game_state/inn_month_maintenance_test.go`

### MERCH-0010

**Inn Rest advances and status restoration**

**Kind:** rule
**Uses:** MERCH-0003, MERCH-0009, PROV-0002
**Original:** Established
**Redux:** Intentional deviation

Require vacancy before quote. Accepted affordable Rest pays, steals, moves to
sleep coordinates, marks Good members Sleeping, then requests twelve five-minute
advances. While hour is not 6, check regeneration rings then request nine minutes.
Restore morning state and move one tile east. No nine-minute step runs when the
fixed hour already ends in hour 6. Exact wake minute depends on starting minute;
a prose claim of 05:00 is not the executable condition. This direct advancement
does not perform ordinary hourly food consumption.

Dead members remain unchanged. Every other member first gets full HP and class
MP: Avatar/Mage = Intelligence, Bard = integer half Intelligence, Fighter MP
unchanged. Then Poisoned becomes Dead with zero HP; Sleeping becomes Good;
Charmed stays Charmed. No vacancy returns to service choice. Declining Rest,
accepted nonpayment and success end the original Inn interaction.

**REDUX DEVIATION:** Original Inn entry clears duration magic even if no Rest
is bought. Redux clears it only after paid Rest commits, preserving it for Leave,
Pickup, cancellation and refusal. Cancelling Rest approval returns to browsing.
The twelve-five/then-nine-minute sequence and restoration order remain explicit;
cosmetic greeting/farewell choices are deterministic.

**Implementation:** `internal/game_state/inn_shop.go`
**Tests:** `internal/game_state/inn_shop_test.go`

### MERCH-0011

**Leaving and collecting complete companion records**

**Kind:** rule
**Uses:** MERCH-0003, MERCH-0009, CRIME-0002
**Original:** Established
**Redux:** Intentional deviation

Leave requires vacancy and at least two active records. Slot zero and Dead
members cannot leave. Quote monthly rate, obtain confirmation, charge nothing,
set stored map to this Inn and months to zero, shift later complete records
forward, append the selected complete record at slot 15, then reduce active
size. This counts active records, not living members: the final living companion
may leave. Avatar refusal returns to selection; one-member/Dead refusal ends
the original interaction.

Pickup requires active size below six and a guest here. Multiple guests are
listed in ascending complete-record order; a sole guest is automatic. Escape
cancels multiple selection, but there is no subsequent original approval.
Nonpayment activates guards once, changes neither gold nor roster, and ends.
Affordable pickup pays, steals, stably inserts the complete record at the active
prefix end, marks it active, then restores. Poisoned guests die at zero HP
without HP/MP restoration. A Dead guest is not refused: it receives full HP/class
MP yet retains Dead. Success returns to service choice.

**REDUX DEVIATION:** Every pickup has cancelable approval, including a sole guest.
Dead guests remain visible but cannot be collected, avoiding the original
full-HP Dead result. Blocked Leave stays in browsing. Cancellation changes
nothing; no paid pickup is charged twice. These deliberate confirmation and
eligibility changes are separate from stable complete-record roster transfers.

**Implementation:** `internal/game_state/inn_shop.go`, `internal/ui/widgets/inn_shop_dialog.go`
**Tests:** `internal/game_state/inn_shop_test.go`

### MERCH-0012

**Tavern menus, bottle prices, and meals**

**Kind:** rule
**Uses:** MERCH-0003, KARMA-0006
**Original:** Established
**Redux:** Intentional deviation

| Location | Tavern / owner | Meal / per-diner price | Drink | Pack base | Information |
| --- | --- | --- | --- | --- | --- |
| Moonglow | The Honest Meal / Sam | Mutton / 3 | Ale | Rations 10 | Chat |
| Britain | The Wayfarer Tavern / Tika | Mutton / 4 | Ale | Rations 15 | Chat |
| Jhelom | The Sword and Keg / Nicole | Mutton / 5 | Ale | Rations 20 | Chat |
| Yew | The Slaughtered Lamb / Duclas | Boar / 3 | Rum | Not offered (backing 25) | Help |
| New Magincia | The Humble Palate / Felicity | Fruit / 2 | Stout | Provisions 30 | Advice |
| West Brittany | The Blue Boar Tavern / Jaymes | Cheese / 5 | Wine | Not offered (backing 25) | Talk |
| Paws | The Cat's Lair / Dr. Cat | Mutton / 3 | Ale | Rations 20 | Chat |
| Buccaneer's Den | The Fallen Virgin / Nikki | Boar / 4 | Rum | Not offered (backing 25) | Help |
| Lycaeum | The Folley Tap / Rob | Mutton / 5 | Ale | Rations 30 | Chat |

| West Brittany key | Bottle | Fixed price |
| --- | --- | ---: |
| A | Rose | 18 |
| B | Claret | 192 |
| C | Sauterne | 79 |
| D | Muscatel | 30 |
| E | Moselle | 275 |
| F | Chablis | 98 |

Meals count the packed active prefix, including every status except Dead. Charge
`meal_price * diners`, with no Intelligence adjustment; then one theft and one
saturating addition of the diner count to food. A plain table immediately north acquires its
south-facing plate; otherwise a south table gets a north-facing plate. No table
suppresses only the display, not the meal. Original Get can later collect that
plate under KARMA-0006.

**REDUX DEVIATION:** Meal plates are transient noncollectible visuals. They survive
movement and Tavern dismissal within that map visit, clear on location/floor
change or reload, and are excluded from native saves. This avoids an extra food
and theft-penalty pickup from a purchased meal while retaining its visual result.

**Implementation:** `internal/references/tavern.go`, `internal/game_state/tavern_shop.go`

### MERCH-0013

**Alcohol warning, intoxication lifetime, and command interception**

**Kind:** rule
**Uses:** MERCH-0003, KARMA-0014, CRIME-0002, RNG-0001
**Original:** Established
**Redux:** Intentional deviation

Ale/Rum/Stout cost one gold per non-Dead diner; Wine costs one bottle price,
independent of party size and Intelligence. Successful payment invokes one theft
then increments the drink counter once, regardless of diners. Warning occurs
only when that counter equals 3. Stopping buys nothing but qualifies the party
as a customer. Continuing has the pre-payment warning consequences specified in
KARMA-0014; subsequent cancellation/failure does not undo the original effects.
Successful drinks continue increasing the counter without a later warning.

Original drink count is process-local, not reset at Tavern/town entry, and absent
from `SAVED.GAM`. The separate smashed byte is saved at `0x03B1`. At each top-level
town command a nonzero smashed count makes a `0..1` draw: on nonzero,
activate guards, decrement, emit hiccup and use a `0..3` draw to replace the
command with North, South, East, West respectively; otherwise preserve both
command and counter. Fresh NPC/town
initialization clears smashed; a same-town floor load without that initialization
does not. These random calls are not proof of external-generator uniformity.

**REDUX DEVIATION:** Redux's continued fourth-drink approval is atomic: cancel,
stale review and nonpayment change nothing; success pays before warning effects,
theft and count increment. Native saves persist the nonnegative uncapped drink
counter; legacy import initializes it to zero while preserving smashed. Extra
smashed/sobered notices are presentation-only and alter neither interception,
guard RNG, counters nor time.

**Implementation:** `internal/game_state/tavern_shop.go`, `internal/game_state/smashed_interception.go`
**Tests:** `internal/game_state/tavern_shop_test.go`

### MERCH-0014

**Provision partial fulfillment, customer qualification, and paid information**

**Kind:** rule
**Uses:** MERCH-0002, MERCH-0003, MERCH-0012, MERCH-0013
**Original:** Established
**Redux:** Intentional deviation

Each provision pack grants 25 food at adjusted pack price. A two-digit requested
quantity is processed one pack at a time: check affordability, pay and grant,
stop when food reaches 9999 (including a paid partial final pack). If a later
pack is unaffordable, earlier packs remain committed and the return skips theft.
If the first is unaffordable, food below three receives a binary-selected one
or two food; otherwise receives none. Both first-pack failures eject and skip
theft. Normal completion or saturation alone reaches one final theft. Zero
requested packs buy nothing.

Session-local customer starts false. A successful meal/drink, completed/saturated/
partially fulfilled provision order, or choosing to stop at the warning makes
it true. Zero packs and Wine cancellation do not. Terminal charity/ejection
leaves no live qualifying session. Original information choices are ignored
until customer is true. Then input is limited to 15 visible characters; scan
the following keys in order, requiring offset zero or a preceding literal space,
without a trailing boundary (`honeypot` can match `hone`). Blank returns to
services; unknown repeats the subject prompt. Prices are fixed.

| Key | Price | Informant | Location |
| --- | ---: | --- | --- |
| hone | 50 | Malik | Moonglow |
| comp | 75 | Greyson | Britain |
| valo | 50 | Trian | Jhelom |
| just | 50 | Jeremy | Yew |
| sacr | 75 | Rew | Minoc |
| hono | 75 | Gruman | Trinsic |
| spir | 25 | Saul | Skara Brae |
| humi | 50 | Shirita | New Magincia |
| dece | 100 | Malifora | Moonglow |
| desp | 150 | Annon | Britain |
| dest | 75 | Trian | Jhelom |
| wron | 150 | Felespar | Yew |
| cove | 75 | Rew's mother | Minoc |
| sham | 100 | Sindar | Trinsic |
| hyth | 100 | Kaiko | New Magincia |
| crow | 200 | Terrance | Britain |
| scep | 200 | Greymarch | Yew |
| amul | 200 | Simon and Tessa | Hidden mountain keep |
| fals | 250 | Shalineth | Lycaeum |
| hatr | 250 | Daemon | Desert |
| cowa | 250 | Lord Malone | Serpent's Hold |
| astr | 100 | Zachariah | Moonglow |
| oppr | 50 | Tactus | Minoc |
| brit | 50 | Daemon | Desert |
| resi | 200 | Terrance | Britain |
| unde | 100 | Jotham | Lighthouse south of Britain |

Accepted affordable information deducts, refreshes and chooses one of four hint
forms, with no theft. Insufficient funds ejects; information can repeat within
the session. Redux preserves provision partial commitment and theft exclusions.

**REDUX DEVIATION:** Information remains visible but disabled with an explanation
until qualified; cosmetic pitch/hint choices are deterministic. Unknown-subject
feedback uses the merchant's refusal in `TavernAdviceNotice`, main-screen font,
measured speech/footer and live two-axis centering in the right workspace.
Resize reflows the same instance. Release debounce prevents the submitting Enter
from dismissing it; the Talk workflow owns removal and cleanup. Acknowledgment
returns to the same session without a transaction charge.

**Implementation:** `internal/references/tavern.go`, `internal/game_state/tavern_shop.go`, `cmd/ultimav/gamescene_merchant_tavern.go`, `internal/ui/widgets/tavern_advice_notice.go`
**Tests:** `internal/game_state/tavern_shop_test.go`
