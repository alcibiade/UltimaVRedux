//nolint:mnd // Verified original screen coordinates, frame indexes, and phase counts.
package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
)

type introPresentationPhase uint8

const (
	introOrigin introPresentationPhase = iota
	introPresents
	introArticle
	introSignature
	introProduction
	introTitleFizz
	introFlameFizz
	introPresentationComplete
)

const (
	introOriginFrameMs      = 56
	introPresentsDurationMs = 1500
	introArticleDurationMs  = 900
	introSignatureDuration  = 2800
	introProductionDuration = 1100
	introTitleFizzDuration  = 1200
	introFlameFizzDuration  = 900
)

type introOriginFrame struct {
	imageIndex int
	flipped    bool
}

//nolint:gochecknoglobals // Original bitmap stages played as one end-over-end turn.
var introOriginFrameOrder = [...]introOriginFrame{
	{imageIndex: 6, flipped: false}, {imageIndex: 5, flipped: false},
	{imageIndex: 4, flipped: false}, {imageIndex: 3, flipped: false},
	{imageIndex: 2, flipped: false}, {imageIndex: 1, flipped: false},
	{imageIndex: 0, flipped: false},
	{imageIndex: 1, flipped: true}, {imageIndex: 2, flipped: true}, {imageIndex: 3, flipped: true},
	{imageIndex: 4, flipped: true}, {imageIndex: 5, flipped: true}, {imageIndex: 6, flipped: true},
	{imageIndex: 5, flipped: true}, {imageIndex: 4, flipped: true}, {imageIndex: 3, flipped: true},
	{imageIndex: 2, flipped: true}, {imageIndex: 1, flipped: true}, {imageIndex: 0, flipped: true},
	{imageIndex: 1, flipped: false}, {imageIndex: 2, flipped: false},
	{imageIndex: 3, flipped: false}, {imageIndex: 4, flipped: false},
	{imageIndex: 5, flipped: false}, {imageIndex: 6, flipped: false},
}

var britishStrokeStarts = []introPoint{ //nolint:gochecknoglobals // Verified original screen coordinates.
	{x: 44, y: 68},
	{x: 64, y: 94},
	{x: 143, y: 78},
	{x: 167, y: 105},
}

type introPoint struct {
	x int
	y int
}

type introPresentationDriver struct {
	phase              introPresentationPhase
	phaseElapsedMs     int
	originStep         int
	signaturePoints    []introPoint
	allSignaturePoints []introPoint
	complete           bool
}

func newIntroPresentationDriver(paths [][]legacy.PathCommand) *introPresentationDriver {
	return &introPresentationDriver{
		phase:              introOrigin,
		phaseElapsedMs:     0,
		originStep:         0,
		signaturePoints:    nil,
		allSignaturePoints: buildBritishSignaturePoints(paths),
		complete:           false,
	}
}

func startIntroPresentationWorkflow(
	manager *event_queue.EventQueueManager,
	driver *introPresentationDriver,
	onComplete func(),
) {
	manager.Workflow(introWorkflowName).
		Priority(event_queue.PriorityLow).
		Sequential().
		First(func() {}).
		RunScript(driver).
		Finally(onComplete).
		Start()
}

//nolint:cyclop // Explicit presentation phases map directly to the original startup sequence.
func (d *introPresentationDriver) Update(deltaMs int) {
	if d.complete {
		return
	}
	d.phaseElapsedMs += deltaMs
	switch d.phase {
	case introOrigin:
		d.originStep = min(d.phaseElapsedMs/introOriginFrameMs, len(introOriginFrameOrder)-1)
		if d.phaseElapsedMs >= introOriginFrameMs*len(introOriginFrameOrder) {
			d.advance(introPresents)
		}
	case introPresents:
		if d.phaseElapsedMs >= introPresentsDurationMs {
			d.advance(introArticle)
		}
	case introArticle:
		if d.phaseElapsedMs >= introArticleDurationMs {
			d.advance(introSignature)
		}
	case introSignature:
		visible := len(d.allSignaturePoints) * d.phaseElapsedMs / introSignatureDuration
		visible = min(visible, len(d.allSignaturePoints))
		d.signaturePoints = d.allSignaturePoints[:visible]
		if d.phaseElapsedMs >= introSignatureDuration {
			d.advance(introProduction)
		}
	case introProduction:
		if d.phaseElapsedMs >= introProductionDuration {
			d.advance(introTitleFizz)
		}
	case introTitleFizz:
		if d.phaseElapsedMs >= introTitleFizzDuration {
			d.advance(introFlameFizz)
		}
	case introFlameFizz:
		if d.phaseElapsedMs >= introFlameFizzDuration {
			d.Skip()
		}
	case introPresentationComplete:
		d.complete = true
	}
}

func (d *introPresentationDriver) currentOriginFrame() introOriginFrame {
	return introOriginFrameOrder[d.originStep]
}

func (d *introPresentationDriver) phaseProgress(durationMs int) float64 {
	if durationMs <= 0 {
		return 1
	}

	return min(float64(d.phaseElapsedMs)/float64(durationMs), 1)
}

func (d *introPresentationDriver) IsComplete() bool {
	return d.complete
}

func (d *introPresentationDriver) Skip() {
	d.phase = introPresentationComplete
	d.complete = true
}

func (d *introPresentationDriver) advance(phase introPresentationPhase) {
	d.phase = phase
	d.phaseElapsedMs = 0
}

func buildBritishSignaturePoints(paths [][]legacy.PathCommand) []introPoint {
	points := make([]introPoint, 0)
	for strokeIndex, commands := range paths {
		if strokeIndex >= len(britishStrokeStarts) {
			break
		}
		position := britishStrokeStarts[strokeIndex]
		shouldPlot := true
		for _, command := range commands {
			if shouldPlot {
				points = append(points, position)
			}
			shouldPlot = command.PlotNext
			position.x += command.DX
			position.y += command.DY
		}
		if shouldPlot {
			points = append(points, position)
		}
	}

	return points
}
