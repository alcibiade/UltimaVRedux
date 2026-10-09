package references

import (
	"bytes"
	"errors"
	"strings"
)

// talkOffsetAdjust matches Origin’s 0x80 rule for phrase bytes.
const talkOffsetAdjust = 0x80

// WordDict holds the compressed‑word table and a fast byte‑lookup map.
type WordDict struct {
	words       []string // index → word
	byteToIndex map[byte]int
}

// NewWordDict builds the lookup from the string list already extracted from
// DATA.OVL.
//
// Dictionary token gaps are handled once during construction, allowing O(1)
// lookups afterward; see the token-domain distinction in ref:TALK-0002.
func NewWordDict(raw []string) *WordDict {
	d := &WordDict{
		words:       raw,
		byteToIndex: make(map[byte]int, len(raw)),
	}

	off := 0
	off-- // -1
	d.addRange(1, 7, off)

	off-- // -2
	d.addRange(9, 27, off)

	off-- // -3
	d.addRange(29, 49, off)

	off-- // -4
	d.addRange(51, 64, off)

	off-- // -5
	d.addRange(66, 66, off)

	off-- // -6
	d.addRange(68, 69, off)

	off-- // -7
	d.addRange(71, 71, off)

	off -= 4 // -11 after the four-entry gap.
	d.addRange(76, 129, off)

	return d
}

func (d *WordDict) addRange(start, stop, offset int) {
	for b := start; b <= stop; b++ {
		d.byteToIndex[byte(b)] = b + offset
	}
}

// Word returns the expanded word for a TLK code byte.
func (d *WordDict) Word(b byte) (string, error) {
	idx, ok := d.byteToIndex[b]
	if !ok || idx < 0 || idx >= len(d.words) {
		return "", errors.New("no compressed word for code byte")
	}

	return d.words[idx], nil
}

// ReplaceMerchantString expands every compressed‑word byte found in a raw shop
// dialog line (leaving “variable” placeholders like %, $, etc. intact).
//
// High-half bytes select dictionary words; literal placeholders remain available
// for later merchant substitutions.
func (d *WordDict) ReplaceMerchantString(raw string) (string, error) {
	var out bytes.Buffer
	useWord := false

	for i := 0; i < len(raw); i++ {
		currentChar := raw[i]

		switch {
		case currentChar >= talkOffsetAdjust:
			wordCode := currentChar - talkOffsetAdjust + 1
			word, err := d.Word(wordCode)
			if err != nil {
				return "", err
			}
			if out.Len() > 0 && !strings.HasSuffix(out.String(), " ") {
				out.WriteByte(' ')
			}
			out.WriteString(word)
			useWord = true

		default:
			if useWord && currentChar != ' ' {
				out.WriteByte(' ')
			}
			out.WriteByte(currentChar)
			useWord = false
		}
	}

	return out.String(), nil
}
