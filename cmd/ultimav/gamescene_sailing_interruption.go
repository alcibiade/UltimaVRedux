package main

import "github.com/hajimehoshi/ebiten/v2"

// syncAutomaticSailingInterruption is sampled before input and after queues on
// every scene frame, including preempted projectile presentation. Only physical
// movement holds are rejected; Fire, Yell and other fresh commands remain usable.
func (g *GameScene) syncAutomaticSailingInterruption() {
	epoch := g.gameState.AutomaticSailingStopEpoch()
	if epoch == g.observedSailingStopEpoch {
		return
	}
	g.observedSailingStopEpoch = epoch
	g.surfaceCamera = surfaceCameraProjection{}
	for _, key := range [...]ebiten.Key{
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight,
		ebiten.KeyNumpad8, ebiten.KeyNumpad2, ebiten.KeyNumpad4, ebiten.KeyNumpad6,
	} {
		g.keyboard.SuppressUntilRelease(key)
	}
}

func (g *GameScene) sailingMovementHeldBeforeInterruption(key ebiten.Key) bool {
	for _, physical := range [...]ebiten.Key{
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight,
		ebiten.KeyNumpad8, ebiten.KeyNumpad2, ebiten.KeyNumpad4, ebiten.KeyNumpad6,
	} {
		if primaryMovementKey(physical) == key && g.keyboard.IsSuppressedUntilRelease(physical) {
			return true
		}
	}

	return false
}

// Ordinary dispatch can see a short command press between 55 ms phases. End
// sailing here rather than depending on that press still being held next phase.
func (g *GameScene) interruptAutomaticSailingForCommand(key ebiten.Key) bool {
	if !g.gameState.IsSailing() {
		return false
	}
	normalized := primaryMovementKey(key)
	if normalized == ebiten.KeyUp || normalized == ebiten.KeyDown ||
		normalized == ebiten.KeyLeft || normalized == ebiten.KeyRight {
		return false
	}
	g.gameState.CompleteAutomaticSailingInterruption()
	g.syncAutomaticSailingInterruption()
	if key == ebiten.KeySpace {
		g.gameState.SystemCallbacks.Message.AddRowStr("Sheets in irons!")
		g.keyboard.PrepareForTransition(0)

		return true
	}

	return false
}
