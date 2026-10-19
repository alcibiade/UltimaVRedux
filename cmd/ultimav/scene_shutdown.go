package main

// disposeActiveScene runs once after normal/error exit (and on startup errors
// after Game allocation). Finalizers still resolve their original scene while
// its workflow queues and audio are retired. This runs on the application thread.
func (g *Game) disposeActiveScene() {
	if g.shutdown {
		return
	}
	g.shutdown = true
	if disposer, ok := g.currentScene.(sceneDisposer); ok {
		disposer.Dispose()
	}
	g.atlasTexture.dispose()
}
