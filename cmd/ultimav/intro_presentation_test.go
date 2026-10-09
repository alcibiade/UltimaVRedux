package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntroPresentationWorkflowRunsScriptAndFinalizes(t *testing.T) {
	now := int64(0)
	manager := event_queue.NewEventQueueManager(func() int64 { return now }, nil)
	driver := newIntroPresentationDriver(nil)
	finalized := false
	startIntroPresentationWorkflow(manager, driver, func() { finalized = true })

	for updates := 0; updates < 1000 && manager.HasActiveWorkflows(); updates++ {
		now += introUpdateInterval
		manager.Update()
	}

	assert.True(t, driver.IsComplete())
	assert.True(t, finalized)
	assert.False(t, manager.HasActiveWorkflows())
}

func TestIntroPresentationWorkflowSkipStillFinalizesOnce(t *testing.T) {
	now := int64(0)
	manager := event_queue.NewEventQueueManager(func() int64 { return now }, nil)
	driver := newIntroPresentationDriver(nil)
	finalizeCount := 0
	startIntroPresentationWorkflow(manager, driver, func() { finalizeCount++ })
	manager.Update()
	driver.Skip()

	for updates := 0; updates < 10 && manager.HasActiveWorkflows(); updates++ {
		now += introUpdateInterval
		manager.Update()
	}

	assert.Equal(t, 1, finalizeCount)
	assert.False(t, manager.HasActiveWorkflows())
}

func TestIntroPresentationDriverCompletesAllPhases(t *testing.T) {
	driver := newIntroPresentationDriver(nil)

	for updates := 0; updates < 1000 && !driver.IsComplete(); updates++ {
		driver.Update(introUpdateInterval)
	}

	assert.True(t, driver.IsComplete())
	assert.Equal(t, introPresentationComplete, driver.phase)
}

func TestIntroOriginFrameOrderCompletesEndOverEndTurn(t *testing.T) {
	require.Len(t, introOriginFrameOrder, 25)
	assert.Equal(t, introOriginFrame{imageIndex: 6, flipped: false}, introOriginFrameOrder[0])
	assert.Equal(t, introOriginFrame{imageIndex: 0, flipped: false}, introOriginFrameOrder[6])
	assert.Equal(t, introOriginFrame{imageIndex: 6, flipped: true}, introOriginFrameOrder[12])
	assert.Equal(t, introOriginFrame{imageIndex: 0, flipped: true}, introOriginFrameOrder[18])
	assert.Equal(t, introOriginFrame{imageIndex: 6, flipped: false}, introOriginFrameOrder[24])
}

func TestIntroOriginUsesDeterministicElapsedSteps(t *testing.T) {
	driver := newIntroPresentationDriver(nil)
	assert.Equal(t, introOriginFrameOrder[0], driver.currentOriginFrame())

	driver.Update(introOriginFrameMs)
	assert.Equal(t, introOriginFrameOrder[1], driver.currentOriginFrame())
	assert.Equal(t, introOrigin, driver.phase)

	driver.Update(introOriginFrameMs * (len(introOriginFrameOrder) - 1))
	assert.Equal(t, introPresents, driver.phase)
}

func TestIntroPresentationRunsTitleThenFlameFizz(t *testing.T) {
	driver := newIntroPresentationDriver(nil)
	driver.phase = introProduction

	driver.Update(introProductionDuration)
	assert.Equal(t, introTitleFizz, driver.phase)
	driver.Update(introTitleFizzDuration / 2)
	assert.InDelta(t, 0.5, driver.phaseProgress(introTitleFizzDuration), 0.001)
	driver.Update(introTitleFizzDuration / 2)
	assert.Equal(t, introFlameFizz, driver.phase)
	driver.Update(introFlameFizzDuration)
	assert.True(t, driver.IsComplete())
}

func TestIntroPresentationDriverSkipCompletesImmediately(t *testing.T) {
	driver := newIntroPresentationDriver(nil)
	driver.Update(introUpdateInterval)

	driver.Skip()

	assert.True(t, driver.IsComplete())
	assert.Equal(t, introPresentationComplete, driver.phase)
}

func TestBuildBritishSignaturePointsPreservesPenUpMoves(t *testing.T) {
	paths := [][]legacy.PathCommand{{
		{DX: 1, DY: 0, PlotNext: true},
		{DX: 4, DY: 0, PlotNext: false},
		{DX: 1, DY: 0, PlotNext: true},
	}}

	points := buildBritishSignaturePoints(paths)

	require.Len(t, points, 3)
	assert.Equal(t, introPoint{x: 44, y: 68}, points[0])
	assert.Equal(t, introPoint{x: 45, y: 68}, points[1])
	assert.Equal(t, introPoint{x: 50, y: 68}, points[2])
}
