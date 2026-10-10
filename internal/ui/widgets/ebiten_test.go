package widgets

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type widgetTestGame struct {
	tests *testing.M
	code  int
	done  chan int
}

func (g *widgetTestGame) Update() error {
	if g.done == nil {
		g.done = make(chan int, 1)
		// Let Ebitengine finish frames while the graphics suite runs. Executing
		// every test inside this Update retains frame-local GPU resources across
		// the entire package, eventually exhausting memory in responsive draws.
		// OS input also advances: fixtures must use newWidgetTestKeyboard or an
		// explicit held-key/text source, never the live keyboard constructor.
		go func() { g.done <- g.tests.Run() }()
	}
	select {
	case g.code = <-g.done:
		return ebiten.Termination
	default:
		return nil
	}
}

func (*widgetTestGame) Draw(*ebiten.Image) {}

func (*widgetTestGame) Layout(_, _ int) (int, int) {
	return 1, 1
}

func TestMain(tests *testing.M) {
	// Keep lazy native/monitor initialization on the main thread before tests.
	runner := &widgetTestGame{tests: tests, code: 1, done: nil}
	if err := ebiten.RunGame(runner); err != nil {
		panic(err)
	}
	if runner.code != 0 {
		os.Exit(runner.code)
	}
}
