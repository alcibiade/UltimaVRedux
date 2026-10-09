package text

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
)

func TestJournalHistoryAssembly(t *testing.T) {
	tests := []struct {
		name  string
		write func(*Output)
		want  string
	}{
		{"plain terminal LF", func(o *Output) { o.AddText("A\n"); o.AppendText("B") }, "AB"},
		{"plain blank", func(o *Output) { o.AddText("A\n\n"); o.AddText("B") }, "A\n\nB"},
		{"styled target filled", func(o *Output) { o.AppendLine().White("A\n").Done(); o.AppendText("B") }, "A\nB"},
		{"styled target bypassed", func(o *Output) { o.AppendLine().White("A\n").Done(); o.AddText("B") }, "A\n\nB"},
		{"append terminal LF", func(o *Output) { o.AddText("A"); o.AppendText("\n"); o.AppendText("B") }, "AB"},
		{"marker prompt", func(o *Output) { o.AddCommandPrompt(""); o.AppendText("Open") }, "Open"},
		{"marker target after message", func(o *Output) { o.AddText("A"); o.AddCommandPrompt(""); o.AppendText("B") }, "A\nB"},
		{"marker not separator", func(o *Output) { o.AddText("A"); o.AddCommandPrompt(""); o.AddCommandPrompt(""); o.AddText("B") }, "A\nB"},
		{"empty append preserves marker", func(o *Output) { o.AddText("A"); o.AddCommandPrompt(""); o.AppendText(""); o.AddText("B") }, "A\nB"},
		{"empty styled segment preserves marker", func(o *Output) {
			o.AddText("A")
			o.AppendLine().CommandPromptMarker(ucolor.EGABlue).White("").Done()
			o.AddText("B")
		}, "A\nB"},
		{"filled blank target", func(o *Output) { o.AddText("A\n\n"); o.AppendText("B") }, "A\nB"},
		{"earlier pending blank", func(o *Output) { o.AddText("A\n\n\n"); o.AppendText("B") }, "A\n\nB"},
		{"pending blank then marker", func(o *Output) { o.AddText("A\n\n"); o.AddCommandPrompt(""); o.AppendText("B") }, "A\n\nB"},
		{"append embedded LF", func(o *Output) { o.AddText("A"); o.AppendText("B\nC\n\nD\n"); o.AppendText("E") }, "AB\nC\n\nDE"},
		{"styled segments contiguous", func(o *Output) { o.AppendLine().White("A").Runic("B").Green(" C ").Done() }, "AB C "},
		{"UTF8 split between styles", func(o *Output) { o.AppendLine().White("\xe2").Green("\x82").Runic("\xac").Done() }, "\u20ac"},
		{"malformed split styles", func(o *Output) { o.AppendLine().White("\xe2").Green("\xc3").Runic("\xa9\xe2\x82").Done() }, "\ufffd\u00e9\ufffd\ufffd"},
		{"styled split LF", func(o *Output) { o.AppendLine().White("A\n").Green("B").Done(); o.AppendText("C") }, "A\nBC"},
		{"builder blank", func(o *Output) { o.AppendLine().White("A").NewLine().White("B").Done() }, "A\n\nB"},
		{"builder marker", func(o *Output) { o.AppendLine().CommandPromptMarker(ucolor.EGABlue).Runic("IN LOR").Done() }, "IN LOR"},
		{"empty add no-op", func(o *Output) {
			o.AddText("A")
			o.AddText("")
			o.AddStyledText(nil, etext.AlignStart)
			o.AppendText("B")
		}, "AB"},
		{"blank only", func(o *Output) { o.AddText(" \n\n\t\n"); o.AppendLine().White("").Done() }, ""},
		{"preserve authored spaces", func(o *Output) { o.AddText("  A  "); o.AddTextRaw(" B "); o.AppendText("  ") }, "  A  \n B   "},
		{"normalize live only", func(o *Output) { o.AddText("A\x00\t\r\x7f\u0085\u2028\u2029B\xff") }, "A       B\ufffd"},
		{"overwrite rail", func(o *Output) {
			for range 20 {
				o.AddText("repeat")
			}
		}, strings.TrimSuffix(strings.Repeat("repeat\n", 20), "\n")},
		{"backspace", func(o *Output) { o.AddCommandPrompt("Op\u00e9"); o.RemoveRightSideCharacter(); o.AppendText("en") }, "Open"},
		{"clear append isolation", func(o *Output) { o.AddText("A"); o.Clear(); o.AppendText("B") }, "A\nB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := NewOutput(nil, 1, 4, 16)
			output.EnableJournalHistory()
			tt.write(output)
			require.Equal(t, tt.want, output.JournalHistory())
			require.NoError(t, journal.ValidateHistory(output.JournalHistory()))
			output.EnableJournalHistory()
			require.Equal(t, tt.want, output.JournalHistory(), "enable is idempotent")
		})
	}
}

func TestJournalHistoryOptInDualOutputAndTalkIsolation(t *testing.T) {
	main := NewOutput(nil, 1, 4, 16)
	main.AddText("not retained")
	require.Empty(t, main.JournalHistory())
	main.EnableJournalHistory()
	modal := NewOutput(nil, 1, 4, 16)
	modal.RetainLogicalHistory(256, 256*1024)
	builder := modal.AppendLine().Runic("SUMM").Green(" clue")
	builder.SetPlainTextCallback(main.AddText)
	builder.Done()
	require.Equal(t, "SUMM clue", main.JournalHistory())
	require.Empty(t, modal.JournalHistory())
	require.Len(t, modal.history.entries, 1)
	require.Equal(t, FontTypeRunic, modal.history.entries[0].segments[0].FontType)
	modal.ReflowRetained(nil, 1, 10)
	require.Len(t, modal.history.entries, 1)
	require.Equal(t, "SUMM clue", main.JournalHistory())
	// Even if both optional collectors are enabled, a style replay is not a new emission.
	main.RetainLogicalHistory(256, 256*1024)
	main.AddText("A\n\n")
	before := main.JournalHistory()
	main.ReflowRetained(nil, 1, 10)
	require.Equal(t, before, main.JournalHistory())
	main.AppendText("B")
	require.Equal(t, "SUMM clue\nA\nB", main.JournalHistory())
	copy := main.WrappedCopy(20)
	copy.AddText("copy only")
	require.Equal(t, "SUMM clue\nA\nB", main.JournalHistory())
}

func TestJournalHistoryRetentionBounds(t *testing.T) {
	output := NewOutput(nil, 1, 4, 16)
	output.EnableJournalHistory()
	for index := range 65 {
		output.AddText(fmt.Sprintf("entry %02d\n\n", index))
	}
	history := output.JournalHistory()
	require.True(t, strings.HasPrefix(history, "entry 01\n\n"))
	require.True(t, strings.HasSuffix(history, "entry 64"))
	require.NoError(t, journal.ValidateHistory(history))
	for _, content := range []string{strings.Repeat("x", 4096), strings.Repeat("\u00e9", 2048), strings.Repeat("\u754c", 1365) + "x"} {
		require.NoError(t, output.RestoreJournalHistory(""))
		output.AddText(content)
		require.Equal(t, content, output.JournalHistory())
		output.AppendText("overflow")
		shortened := output.JournalHistory()
		require.LessOrEqual(t, len(shortened), journal.MaxHistoryLineBytes)
		require.True(t, utf8.ValidString(shortened))
		require.True(t, strings.HasSuffix(shortened, journal.HistoryShortened))
		for range 100 {
			output.AppendText("discarded")
		}
		require.Equal(t, shortened, output.JournalHistory())
		output.AppendText("discarded\nnext\n")
		output.AppendText("!")
		require.Equal(t, shortened+"\nnext!", output.JournalHistory())
	}
	require.NoError(t, output.RestoreJournalHistory(""))
	for range 32 {
		output.AddText(strings.Repeat("x", 4096) + "\n\n")
	}
	history = output.JournalHistory()
	require.Len(t, strings.Split(history, "\n\n"), 15)
	require.NoError(t, journal.ValidateHistory(history))
	require.LessOrEqual(t, len(history), journal.MaxHistoryBytes)
}

func TestJournalHistoryHugeInputsRemainBounded(t *testing.T) {
	// Collector-only tests avoid benchmarking the intentionally unchanged live rail.
	h := &journalHistory{}
	h.retain(outputInsertion{plain: true, raw: strings.Repeat(" ", 1024*1024)})
	require.False(t, h.nonblank)
	require.LessOrEqual(t, len(h.target), journal.MaxHistoryLineBytes)
	h.retain(outputInsertion{append: true, segments: []ColorSegment{{Text: "\nnext"}}})
	require.Equal(t, "next", string(h.target))
	h.retain(outputInsertion{plain: true, raw: strings.Repeat("A\n\n", 10000)})
	require.LessOrEqual(t, len(h.entries), journal.MaxHistoryEntries)
	output := &Output{journalHistory: h}
	require.NoError(t, journal.ValidateHistory(output.JournalHistory()))
	h.retain(outputInsertion{plain: true, raw: strings.Repeat("x", 1024*1024) + "\nlast"})
	require.LessOrEqual(t, cap(h.target), 2*journal.MaxHistoryLineBytes)
	require.Equal(t, "last", string(h.target))
	require.NoError(t, journal.ValidateHistory(output.JournalHistory()))
}

func TestJournalHistoryAggregateBoundaryCountsSeparators(t *testing.T) {
	output := NewOutput(nil, 1, 4, 16)
	output.EnableJournalHistory()
	for _, separator := range []string{"\n", "\n\n"} {
		prefix := strings.Repeat(strings.Repeat("x", 4096)+separator, 15)
		exact := prefix + strings.Repeat("y", journal.MaxHistoryBytes-len(prefix))
		require.Len(t, exact, journal.MaxHistoryBytes)
		require.NoError(t, output.RestoreJournalHistory(exact))
		require.Equal(t, exact, output.JournalHistory())
		output.AppendText("z")
		want := exact[4096+len(separator):] + "z"
		require.Equal(t, want, output.JournalHistory(), "eviction includes the newly exposed separator")
		require.NoError(t, journal.ValidateHistory(output.JournalHistory()))
	}
}

func TestRestoreJournalHistoryAtomicPlainAndDetached(t *testing.T) {
	output := NewOutput(nil, 1, 4, 16)
	output.EnableJournalHistory()
	output.AddCommandPrompt("old")
	for _, history := range []string{"", "A\n\n B  \nrepeat\nrepeat", strings.Repeat("\u00e9", 2048), strings.Repeat("x", 4080) + journal.HistoryShortened} {
		require.NoError(t, output.RestoreJournalHistory(history))
		require.Equal(t, history, output.JournalHistory())
		for _, line := range output.lines {
			require.Empty(t, line.Segments)
			require.True(t, utf8.ValidString(line.Text))
		}
		rail := output.GetOutputStr(true)
		require.Error(t, output.RestoreJournalHistory("invalid\n"))
		require.Equal(t, history, output.JournalHistory())
		require.Equal(t, rail, output.GetOutputStr(true))
	}
	snapshot := output.JournalHistory()
	output.AppendText("cannot regrow discarded suffix")
	require.Equal(t, snapshot, output.JournalHistory())
	output.AddText("new message")
	require.NotEqual(t, snapshot, output.JournalHistory())
	require.NotContains(t, snapshot, "new message")
	require.NoError(t, output.RestoreJournalHistory("authored [text shortened]"))
	output.AppendText(" continues")
	require.Equal(t, "authored [text shortened] continues", output.JournalHistory())
	disabled := NewOutput(nil, 1, 4, 16)
	require.NoError(t, disabled.RestoreJournalHistory("rail only"))
	require.Nil(t, disabled.journalHistory)
	require.Empty(t, disabled.JournalHistory())
	require.Contains(t, disabled.GetOutputStr(false), "rail only")
}

func FuzzJournalHistoryCanonicalRoundTrip(f *testing.F) {
	f.Add("A\n\n", "B\nC")
	f.Add(" \xff\u2028\n", "\xe2\x82\xac")
	f.Add(strings.Repeat("\u754c", 2000), "discarded\nnext")
	f.Fuzz(func(t *testing.T, raw, appended string) {
		if len(raw)+len(appended) > 1024*1024 {
			t.Skip("bound fuzz input processing work")
		}
		history := new(journalHistory)
		history.retain(outputInsertion{plain: true, raw: raw})
		middle := len(appended) / 2
		history.retain(outputInsertion{append: true, segments: []ColorSegment{{Text: appended[:middle]}, {Text: appended[middle:]}}})
		output := &Output{journalHistory: history}
		canonical := output.JournalHistory()
		require.NoError(t, journal.ValidateHistory(canonical))
		require.NoError(t, output.RestoreJournalHistory(canonical))
		require.Equal(t, canonical, output.JournalHistory())
	})
}
