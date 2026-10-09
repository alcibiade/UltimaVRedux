package journal

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVisitsOpeningSnapshotAndIndependentBaselines(t *testing.T) {
	p := EmptyPresentation()
	require.NoError(t, ValidatePresentation(p))
	require.False(t, p.Visits.IsNew(ChronicleTab, 1))
	p.Visits.Record(VirtuesTab, 0)
	p.Visits.Record(ChronicleTab, 1)
	opening := p.Visits.Clone()
	p.Visits.Record(ChronicleTab, 1)
	p.Visits.Record(ShadowlordsTab, 2)
	require.Equal(t, uint64(2), opening.Revision)
	require.Equal(t, uint64(2), opening.Changes[0].Revision)
	p.Visits.Acknowledge(opening.Revision, [3]bool{true, true, false})
	require.True(t, p.Visits.IsNew(ChronicleTab, 1), "post-opening change survives dismissal")
	require.False(t, p.Visits.IsNew(VirtuesTab, 0))
	require.True(t, p.Visits.IsNew(ShadowlordsTab, 2))
	require.True(t, opening.IsNew(VirtuesTab, 0), "session stars remain frozen")
	p.Visits.Acknowledge(1, [3]bool{true, true, true})
	require.Equal(t, [3]uint64{2, 2, 1}, p.Visits.Seen, "baselines cannot regress")
	p.Visits.Acknowledge(math.MaxUint64, [3]bool{true, false, false})
	require.Equal(t, p.Visits.Revision, p.Visits.Seen[0])
	require.NoError(t, ValidatePresentation(p))
	wire, err := json.Marshal(p)
	require.NoError(t, err)
	var restored Presentation
	require.NoError(t, json.Unmarshal(wire, &restored))
	require.Equal(t, p, restored)
}

func TestVisitsFiniteSortedKeysAndValidation(t *testing.T) {
	require.Equal(t, 51, MaxChronicleEntries)
	require.Equal(t, 62, MaxChanges)
	p := EmptyPresentation()
	for entry := 2; entry >= 0; entry-- {
		p.Visits.Record(ShadowlordsTab, entry)
	}
	for entry := 7; entry >= 0; entry-- {
		p.Visits.Record(VirtuesTab, entry)
	}
	for entry := MaxChronicleEntries; entry >= 1; entry-- {
		p.Visits.Record(ChronicleTab, entry)
	}
	require.Len(t, p.Visits.Changes, 62)
	require.NoError(t, ValidatePresentation(p))
	for _, change := range p.Visits.Changes {
		require.True(t, p.Visits.IsNew(change.Tab, change.Entry))
	}
	for _, key := range [][2]int{{-1, 0}, {3, 0}, {0, 0}, {0, MaxChronicleEntries + 1}, {1, -1}, {1, 8}, {2, 3}} {
		require.Panics(t, func() { p.Visits.Record(key[0], key[1]) })
		require.False(t, p.Visits.IsNew(key[0], key[1]))
	}
	tests := map[string]func(*Presentation){
		"null changes":         func(p *Presentation) { p.Visits.Changes = nil },
		"too many":             func(p *Presentation) { p.Visits.Changes = append(p.Visits.Changes, p.Visits.Changes[0]) },
		"invalid key":          func(p *Presentation) { p.Visits.Changes[0].Entry = 0 },
		"over Chronicle bound": func(p *Presentation) { p.Visits.Changes[MaxChronicleEntries-1].Entry = MaxChronicleEntries + 1 },
		"duplicate":            func(p *Presentation) { p.Visits.Changes[1] = p.Visits.Changes[0] },
		"unsorted": func(p *Presentation) {
			p.Visits.Changes[0], p.Visits.Changes[1] = p.Visits.Changes[1], p.Visits.Changes[0]
		},
		"zero revision":   func(p *Presentation) { p.Visits.Changes[0].Revision = 0 },
		"future change":   func(p *Presentation) { p.Visits.Changes[0].Revision = p.Visits.Revision + 1 },
		"future baseline": func(p *Presentation) { p.Visits.Seen[0] = p.Visits.Revision + 1 },
		"invalid history": func(p *Presentation) { p.History = "\n" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := Presentation{History: p.History, Visits: p.Visits.Clone()}
			mutate(&candidate)
			require.Error(t, ValidatePresentation(candidate))
		})
	}
	p.Visits.Revision = math.MaxUint64
	require.Panics(t, func() { p.Visits.Record(ChronicleTab, 1) })
	require.NotNil(t, (Visits{}).Clone().Changes)
}

func TestValidateHistoryCanonicalByteBounds(t *testing.T) {
	for _, history := range []string{"", "A", " A  \n\n B ", strings.Repeat("x", 4096), strings.Repeat("\u00e9", 2048), strings.Repeat("A\n", 63) + "A"} {
		require.NoError(t, ValidateHistory(history))
	}
	for _, history := range []string{
		"\n", "\nA", "A\n", "A\n\n\nB", " ", "A\n \nB", "\u00a0", "A\tB", "A\rB", "\x00", "\x7f", "\u0085", "\u2028", "\u2029", "\xff",
		strings.Repeat("x", 4097), strings.Repeat("\u00e9", 2049), strings.Repeat("A\n", 64) + "A",
	} {
		require.Error(t, ValidateHistory(history), "%q", history)
	}
	// Sixteen lines plus fifteen LF bytes fit exactly; one byte more does not.
	exact := strings.Repeat(strings.Repeat("x", 4096)+"\n", 15) + strings.Repeat("y", 4081)
	require.Len(t, exact, MaxHistoryBytes)
	require.NoError(t, ValidateHistory(exact))
	require.Error(t, ValidateHistory(exact+"y"))
}
