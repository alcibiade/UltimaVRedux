package text

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
)

type nativeTextTests struct {
	tests *testing.M
	done  chan int
	code  int
}

func (g *nativeTextTests) Update() error {
	if g.done == nil {
		g.done = make(chan int, 1)
		go func() { g.done <- g.tests.Run() }()
	}
	select {
	case g.code = <-g.done:
		return ebiten.Termination
	default:
		return nil
	}
}

func (*nativeTextTests) Draw(*ebiten.Image)         {}
func (*nativeTextTests) Layout(int, int) (int, int) { return 1, 1 }

func TestMain(tests *testing.M) {
	runner := &nativeTextTests{tests: tests, done: nil, code: 1}
	if err := ebiten.RunGame(runner); err != nil {
		panic(err)
	}
	os.Exit(runner.code)
}

func TestSDKScaledGameTextReusesRaster(t *testing.T) {
	ClearTextCache()
	t.Cleanup(ClearTextCache)
	output := NewOutput(NewUltimaFont(20), 24, 2, 100)
	output.AppendLine().Header("A heading").Done()
	target := ebiten.NewImage(240, 80)
	t.Cleanup(target.Deallocate)
	before := textRenderer.Stats()
	for range 5 {
		output.DrawPanelContent(target, image.Point{}, 240, etext.AlignStart)
	}
	after := textRenderer.Stats()
	require.Equal(t, uint64(1), after.Misses-before.Misses)
	require.Equal(t, uint64(4), after.Hits-before.Hits)
	require.Equal(t, 1, after.Entries)
	retained := output.lineLayouts[0].layouts[0]
	output.RecolorCurrentLine(color.RGBA{R: 255, A: 255})
	output.DrawPanelContent(target, image.Point{}, 240, etext.AlignStart)
	require.Same(t, retained, output.lineLayouts[0].layouts[0], "paint changes retain SDK geometry")
	ClearTextCache()
	require.Zero(t, textRenderer.Stats().Entries)
	output.DrawPanelContent(target, image.Point{}, 240, etext.AlignStart)
	require.Same(t, retained, output.lineLayouts[0].layouts[0], "cache clear does not invalidate layout")
}

func TestSDKGameTextMatchesLegacyCanvasPixels(t *testing.T) {
	font := NewUltimaFont(20)
	actual, expected := ebiten.NewImage(240, 60), ebiten.NewImage(240, 60)
	t.Cleanup(actual.Deallocate)
	t.Cleanup(expected.Deallocate)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(3, 4)
	DrawCachedText(actual, "Strength: 20", font, op, color.White)
	width, height := etext.Measure("Strength: 20", font.TextFace, 0)
	legacy := ebiten.NewImage(int(width)+2, int(height)+2)
	t.Cleanup(legacy.Deallocate)
	etext.Draw(legacy, "Strength: 20", font.TextFace, &etext.DrawOptions{})
	expected.DrawImage(legacy, op)
	a, b := make([]byte, 240*60*4), make([]byte, 240*60*4)
	actual.ReadPixels(a)
	expected.ReadPixels(b)
	require.True(t, bytes.Equal(b, a), "SDK adapter must preserve the former clipped raster canvas")
}
