package widgets

import (
	stdcolor "image/color"
	"strings"
)

// detailLine is one line of a paged detail pane: a labelled value, or a header
// drawn in the heading colour.
type detailLine struct {
	label  string
	value  string
	color  stdcolor.Color
	header bool
}

// splitDetailLine keeps the words of line that fit lineCapacity rows, as measured
// by count, and returns the rest as an unlabelled continuation, or nil.
func splitDetailLine(line detailLine, lineCapacity int, count func([]detailLine) int) (detailLine, *detailLine) {
	words := strings.Fields(line.value)
	if len(words) <= 1 {
		return line, nil
	}
	bestEnd := 1
	for end := 1; end <= len(words); end++ {
		candidate := line
		candidate.value = strings.Join(words[:end], " ")
		if count([]detailLine{candidate}) > lineCapacity {
			break
		}
		bestEnd = end
	}
	first := line
	first.value = strings.Join(words[:bestEnd], " ")
	if bestEnd == len(words) {
		return first, nil
	}
	remainder := line
	remainder.label = ""
	remainder.header = false
	remainder.value = strings.Join(words[bestEnd:], " ")

	return first, &remainder
}
