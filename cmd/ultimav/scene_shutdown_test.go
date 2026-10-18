package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type applicationDisposalScene struct {
	Scene
	dispose func()
}

func (scene *applicationDisposalScene) Dispose() { scene.dispose() }

func TestApplicationShutdownDisposesCurrentOwnerExactlyOnceOnEveryExit(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "startup or loop error"}[failure], func(t *testing.T) {
			firstCount, lastCount := 0, 0
			first := &applicationDisposalScene{dispose: func() { firstCount++ }}
			game := &Game{currentScene: first}
			last := &applicationDisposalScene{}
			last.dispose = func() {
				require.Same(t, last, game.currentScene, "finalizers must still resolve the active owner")
				lastCount++
			}
			result := func() error {
				defer game.disposeActiveScene()
				game.currentScene = last
				if failure {
					return errors.New("application failed")
				}
				game.disposeActiveScene()
				return nil
			}()
			require.Equal(t, failure, result != nil)
			require.Equal(t, 0, firstCount)
			require.Equal(t, 1, lastCount)
			game.disposeActiveScene()
			require.Equal(t, 1, lastCount)
		})
	}
}
