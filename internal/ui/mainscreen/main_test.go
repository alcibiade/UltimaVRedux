package mainscreen

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type mainScreenTestGame struct {
	tests *testing.M
	code  int
	done  chan int
}

func (g *mainScreenTestGame) Update() error {
	if g.done == nil {
		g.done = make(chan int, 1)
		// Allow actual frame ends to release native upload buffers. Fixtures
		// must not read live input concurrently with the advancing native loop.
		go func() { g.done <- g.tests.Run() }()
	}
	select {
	case g.code = <-g.done:
		return ebiten.Termination
	default:
		return nil
	}
}

func (*mainScreenTestGame) Draw(*ebiten.Image) {}

func (*mainScreenTestGame) Layout(_, _ int) (int, int) { return 1, 1 }

func TestMain(tests *testing.M) {
	// Native initialization stays on the main thread; tests start after the
	// first Update and can inspect real pixels while the loop finishes frames.
	runner := &mainScreenTestGame{tests: tests, code: 1, done: nil}
	if err := ebiten.RunGame(runner); err != nil {
		panic(err)
	}
	if runner.code != 0 {
		os.Exit(runner.code)
	}
}
