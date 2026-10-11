//go:build !settingsproof || !ebitenginevmguest

package main

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type dungeonRendererTestGame struct {
	tests *testing.M
	code  int
	done  chan int
}

func (g *dungeonRendererTestGame) Update() error {
	if g.done == nil {
		g.done = make(chan int, 1)
		// Finish real frames while tests run, releasing frame-local uploads.
		// Input advances too: tests must supply detached keyboard/text sources.
		go func() { g.done <- g.tests.Run() }()
	}
	select {
	case g.code = <-g.done:
		return ebiten.Termination
	default:
		return nil
	}
}

func (*dungeonRendererTestGame) Draw(*ebiten.Image) {}

func (*dungeonRendererTestGame) Layout(_, _ int) (int, int) {
	return dungeonCanvasWidth, dungeonCanvasHeight
}

func TestMain(tests *testing.M) {
	// Automated native tests must not steal focus and ingest the maintainer's
	// desktop typing. Real graphics frames still run while fixtures inject input.
	ebiten.SetInitFocused(false)
	ebiten.SetRunnableOnUnfocused(true)
	// RunGame initializes native services on the main thread before Update
	// launches test goroutines (Ebitengine 2.10 uses lazy monitor initialization).
	runner := &dungeonRendererTestGame{tests: tests, code: 1, done: nil}
	if err := ebiten.RunGame(runner); err != nil {
		panic(err)
	}
	if runner.code != 0 {
		os.Exit(runner.code)
	}
}
