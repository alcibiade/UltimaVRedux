package legacy

import "fmt"

// Catalog returns all shipped EGA non-tile graphics supported by the exporter.
func Catalog() []FileSpec {
	result := make([]FileSpec, 0, 32)
	result = append(result,
		shapeSpec("DNG1.16", dungeonEntryNames()),
		shapeSpec("DNG2.16", dungeonEntryNames()),
		shapeSpec("DNG3.16", dungeonEntryNames()),
		pairedSpec("ITEMS.16", 20, "item"),
	)
	for index := range 8 {
		result = append(result, pairedSpec(fmt.Sprintf("MON%d.16", index), 6, "monster"))
	}
	result = append(result,
		shapeSpec("CREATE.16", []string{"oracle", "incense", "honesty", "compassion", "valor", "justice", "sacrifice", "honor", "spirituality", "humility", "shrine"}),
		shapeSpec("TEXT.16", []string{"the", "summoning", "arrival", "story", "homecoming", "dream"}),
		shapeSpec("STORY1.16", []string{"window", "awakening", "codex"}),
		shapeSpec("STORY2.16", []string{"amulet", "ankh", "stones", "moongate"}),
		shapeSpec("STORY3.16", []string{"arrival", "picture-7b"}),
		shapeSpec("STORY4.16", []string{"picture-7c", "picture-7d"}),
		shapeSpec("STORY5.16", []string{"picture-7e", "picture-7f"}),
		shapeSpec("STORY6.16", []string{"hutward", "hut", "iolo-a-top", "iolo-a-bottom", "iolo-b-top", "iolo-b-bottom", "iolo-c-top", "iolo-c-bottom"}),
		shapeSpec("END1.16", []string{"stones-down", "robbed", "snore"}),
		shapeSpec("END2.16", []string{"the-dream", "naughty", "bye-bye"}),
		shapeSpec("ULTIMA.16", []string{"chrome", "flame-1", "flame-2", "flame-3", "flame-4"}),
		shapeSpec("STARTSC.16", []string{"left", "center", "right"}),
		shapeSpec("ENDSC.16", []string{"ending-screen"}),
		FileSpec{Name: "TITLE.BIT", Format: FormatBitmap1, Compressed: true, EntryNames: []string{"origin-0", "origin-1", "origin-2", "origin-3", "origin-4", "origin-5", "origin-6", "presents", "a", "production"}},
		FileSpec{Name: "BRITISH.BIT", Format: FormatBitmap1, Compressed: true, EntryNames: []string{"lord-british"}},
		FileSpec{Name: "WD.BIT", Format: FormatBitmap1, Compressed: false, EntryNames: []string{"flame-mask"}},
		FileSpec{Name: "PROPORT.PCS", Format: FormatFontPCS, Compressed: true, EntryNames: nil},
		FileSpec{Name: "IBM.CH", Format: FormatFontCH, Compressed: false, EntryNames: nil},
		FileSpec{Name: "RUNES.CH", Format: FormatFontCH, Compressed: false, EntryNames: nil},
		FileSpec{Name: "BRITISH.PTH", Format: FormatPath, Compressed: false, EntryNames: []string{"stroke-1", "stroke-2", "stroke-3", "stroke-4"}},
	)

	return result
}

// FindSpec returns the catalog specification for an original graphics file.
func FindSpec(name string) (FileSpec, bool) {
	for _, spec := range Catalog() {
		if spec.Name == name {
			return spec, true
		}
	}

	return FileSpec{}, false
}

func shapeSpec(name string, names []string) FileSpec {
	return FileSpec{Name: name, Format: FormatShape4, Compressed: true, EntryNames: names}
}

func pairedSpec(name string, count int, prefix string) FileSpec {
	names := make([]string, 0, count*2)
	for index := range count {
		names = append(names, fmt.Sprintf("%s-%02d", prefix, index), fmt.Sprintf("%s-%02d-mask", prefix, index))
	}

	return FileSpec{Name: name, Format: FormatPaired, Compressed: true, EntryNames: names}
}

func dungeonEntryNames() []string {
	names := make([]string, 28)
	for index := range names {
		names[index] = fmt.Sprintf("panel-%02d", index)
	}

	return names
}
