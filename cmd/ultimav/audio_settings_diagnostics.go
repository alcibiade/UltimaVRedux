package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

type configurationAudioReport struct {
	lines         []string
	output        *text.Output
	font          *text.UltimaFont
	width, offset int
}

func (d *ConfigurationDialog) openAudioDiagnostics() {
	lines := []string{"Captured audio state. R refreshes; playback continues.", "Audio diagnostics are unavailable."}
	if provider, ok := d.audioManager.(interface{ PlaybackSnapshot() audio.PlaybackSnapshot }); ok {
		lines = append(lines[:1], provider.PlaybackSnapshot().Lines()...)
	}
	d.audioReport = &configurationAudioReport{lines: lines, output: nil, font: nil, width: 0, offset: 0}
}

func (d *ConfigurationDialog) updateAudioDiagnostics() bool {
	if d.audioReport == nil {
		return false
	}
	keys := []ebiten.Key{ebiten.KeyD, ebiten.KeyEscape, ebiten.KeyR, ebiten.KeyUp, ebiten.KeyDown,
		ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyHome, ebiten.KeyEnd}
	key := d.keyboard.GetBoundKeyPressed(&keys)
	if key == nil || !d.keyboard.TryToRegisterKeyPress(*key) {
		return true
	}
	report := d.audioReport
	page, maxOffset := 1, 0
	if report.output != nil {
		page = max(1, report.output.GetLineCapacity(d.layout.bodyRect.Dy()))
		maxOffset = max(0, report.output.GetPanelContentLineCount()-page)
	}
	//nolint:exhaustive // Only the report's explicit keyboard bindings reach this switch.
	switch *key {
	case ebiten.KeyD, ebiten.KeyEscape:
		d.audioReport = nil
	case ebiten.KeyR:
		d.openAudioDiagnostics()
	case ebiten.KeyUp:
		report.offset = max(0, report.offset-1)
	case ebiten.KeyDown:
		report.offset = min(maxOffset, report.offset+1)
	case ebiten.KeyPageUp:
		report.offset = max(0, report.offset-page)
	case ebiten.KeyPageDown:
		report.offset = min(maxOffset, report.offset+page)
	case ebiten.KeyHome:
		report.offset = 0
	case ebiten.KeyEnd:
		report.offset = maxOffset
	}

	return true
}

func (d *ConfigurationDialog) drawAudioDiagnostics(screen *ebiten.Image) {
	report := d.audioReport
	width := max(1, d.layout.bodyRect.Dx())
	if report.output == nil || report.width != width || report.font != d.detailOutput.Font {
		source := text.NewOutput(d.detailOutput.Font, d.detailOutput.GetLineSpacing(), len(report.lines)+1, 1)
		for _, line := range report.lines {
			source.AppendLine().Text(line, widgets.DialogValueColor).Done()
		}
		report.output = source.WrappedCopyForWidth(width)
		report.width, report.font = width, d.detailOutput.Font
	}
	capacity := max(1, report.output.GetLineCapacity(d.layout.bodyRect.Dy()))
	report.offset = min(report.offset, max(0, report.output.GetPanelContentLineCount()-capacity))
	report.output.DrawPanelContentRange(screen, d.layout.bodyRect.Min, width, etext.AlignStart, report.offset, capacity)
}
