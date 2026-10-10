# Signs, text encodings, and symbolic displays

This reference separates stream semantics, data structure, glyph artwork, and
Redux presentation. Original inscriptions, graffiti prose, and artwork remain
external assets. Numeric glyph identities do not independently establish their
visual design in every font variant.

### SIGN-0001

**Character-set selection**

**Kind:** rule
**Uses:** GFX-0005
**Original:** Established
**Redux:** Unverified

For ordinary sign bytes, a clear high bit selects Runic and a set high bit selects
IBM. The emitted index is the low seven bits. Thus 41 hex selects Runic glyph 41,
while C1 selects IBM glyph 41. Raw control recognition occurs separately; a
high-bit version of a control is not automatically the same command.

This establishes selection, not fixed-font file packing or the meaning of every
compound glyph. Redux's font sheets are derived assets, not original CH files.

### SIGN-0002

**Sign container, traversal, and aliases**

**Kind:** rule
**Uses:** SIGN-0001
**Original:** Established
**Redux:** Unverified

`SIGNS.DAT` begins with 33 little-endian offset words, a 66-byte header. `0x42`
is the header end, not independent proof of the first block's actual offset.
Select the offset by current location. Zero chooses fallback presentation.

A block contains four-byte record headers (location, level, X, Y) followed by
NUL-terminated text. Lookup compares level/X/Y because block selection already
identifies location. After an unmatched record, a following FF byte ends traversal.
The original working buffer is prefilled with FF; this is not a modern bounds check.

A leading raw newline aliases the next record's text by repeatedly advancing six
bytes: newline, terminator, next four-byte header. Aliases retain their own
coordinates. This convention has no separate stored pointer and does not itself
validate the assumed layout or chain bounds.

### SIGN-0003

**Readable terrain and generated displays**

**Kind:** rule
**Uses:** SIGN-0002
**Original:** Established
**Redux:** Unverified

After dynamic-object precedence, direct inscription lookup applies to terrain
89, 8A, A0, A4, and F8 hex: cross, headstone, ordinary sign, small sign, wall sign.
Structural E0/E1/E2 pieces follow north/east/west respectively, then describe the
terminal terrain normally. A terminal inscription-shaped tile reached this way
is not automatically redispatched as an inscription. Flagpoles, flags, and service
symbols use ordinary descriptions.

Missing blocks or unmatched coordinates generate a fallback display. Location 4,
level 0, coordinate (11,15) hex has a generated wanted display incorporating up to
three active-party names, bypassing stored sign lookup. Exact prose and issuer
attribution are not established by those dispatch facts alone.

### SIGN-0004

**Text controls and framing**

**Kind:** rule
**Uses:** SIGN-0001, SIGN-0002
**Original:** Established
**Redux:** Unverified

| Raw byte | Operation |
| --- | --- |
| 00 | End text |
| 0A | Runic newline, except leading alias convention |
| 8A | IBM newline |
| 0D | Wait for a key, emit no glyph |
| 26, 27 | Emit the horizontal Runic frame glyph |
| 29-31 | Expand one of nine sixteen-character frame sequences |
| Other | Select charset by high bit; output low-seven-bit glyph |

The message window is sixteen columns wide. A sixteen-character macro starts at
the current cursor; it is not guaranteed to begin or occupy exactly one aligned
row. Newlines and wrapping still apply. Framing comes from the stream, not from
an automatic border chosen by terrain. Compound-letter and visual-space designs
are font artwork, not multibyte expansion performed by this interpreter.

### SIGN-0005

**Dungeon symbols and separate graffiti**

**Kind:** rule
**Uses:** SIGN-0001
**Original:** Established
**Redux:** Unverified

The symbolic dungeon view is 22 x 22, centered locally at (11,11), using eight-pixel
cell spacing. The one-cell screen margin places the player at text coordinate
(12,12). Ladder glyphs distinguish up/down/both; chests and room glyphs are yellow;
wall, plaque, special-wall, and hidden-door families stop propagation. Energy
fields and fountains use drawing primitives.

| Cell family/value | Glyph identity |
| --- | --- |
| Passage with opening bit | IBM 18 hex |
| 1x, 2x, 3x | Runic 2E, 2D, 2F |
| 4x | Runic 70 |
| 60 | IBM 19 |
| 61, 69 | Runic 71 |
| 68 | IBM 12 |
| Other 6x | Runic 72 |
| Ax, Fx | Runic 73 |
| B0 | IBM 7F |
| Other Bx | Runic 74 |
| Cx, Dx, Ex | Runic 75, 76, 77 |

Graffiti is a separate collection of eleven messages, selected by dungeon-specific
first/count metadata and placement coordinates rather than SIGNS.DAT. Its exact
prose and every dungeon-to-array association were not reverified in this pass.
View cell spacing does not prove fixed-font binary packing.

### SIGN-0006

**Other Runic consumers and sky positions**

**Kind:** rule
**Uses:** SIGN-0001
**Original:** Established
**Redux:** Unverified

Conversation control 8E hex toggles a high-bit mask. Inventory uses Runic icons,
scroll names, and moonstone phase glyphs, while potion names use IBM. Shrine
offering/final Codex displays, ending text, and ordinary scroll pickup have Runic
paths; special plans pickup is a separate case.

The sky has twelve display cells plus a terminator. Sun position is `17-hour`.
Moon positions begin at `8-hour` and `2-hour`; add 24 only if below -12. Write a
body only for a position in 0-11, in sun/Trammel/Felucca order. The last write wins
on overlap. Dungeon and underground/special-map presentation have separate gates.
Omitting the moon-position wrap changes nighttime behavior.

### SIGN-0007

**Locate coordinate encoding**

**Kind:** rule
**Uses:** SIGN-0001
**Original:** Established
**Redux:** Unverified

Emit Y before X, high nibble before low nibble, mapping 0-15 to Runic A-P. The
punctuation is also emitted in that charset. In Wis has outdoor context admission;
Sextant additionally requires map zero, a level below 128, and hour 19-23 or 0-5.
Do not transfer those item restrictions to every caller of coordinate formatting.

### SIGN-0008

**Redux inscription presentation**

**Kind:** rule
**Uses:** SIGN-0002, SIGN-0004
**Original:** Established
**Redux:** Intentional deviation

**REDUX DEVIATION: SIGN-0008** — The original emits encoded text into the message
window and pauses at each wait control. Redux presents a modal inscription with
semantic colors and omits those intermediate waits for a coherent reading view.
It resolves aliases into bounded independent buffers rather than trusting sentinel
traversal. Its dungeon view also uses modern topology/icons instead of every
original font symbol.

An optional per-reading readable rendering is another presentation aid: understood
Runic spans become English bitmap lettering while structural artwork, IBM cells,
row count, and bounds remain intact. Compound expansions borrow padding, then
compress only their own span up to 2:1; they do not omit artwork or rewrite source
prose. Translation state is local to the reading and does not grant journal
knowledge, alter the stored inscription, or consume game time. Maintainer-only
activation instructions remain separate from player Help.

**Implementation:** `internal/text/sign_renderer.go`, `internal/text/sign_translation.go`
**Tests:** none
