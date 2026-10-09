package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/display"
)

type debugDisplayManager interface {
	Mode() config.Mode
	RequestedMode() config.Mode
	IsTransitionPending() bool
	LastNormalWindowSize() (int, int)
	WindowSize() (int, int)
	MonitorSize() (int, int)
	ContainerSize() (int, int)
	LogicalSize() (int, int)
	RequestMode(mode config.Mode, width, height int) error
}

type displayResolutionResult struct {
	Index  int `json:"index"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type displayStatusResult struct {
	RequestedMode      config.Mode               `json:"requestedMode"`
	ActualMode         config.Mode               `json:"actualMode"`
	TransitionPending  bool                      `json:"transitionPending"`
	NormalWindowWidth  int                       `json:"normalWindowWidth"`
	NormalWindowHeight int                       `json:"normalWindowHeight"`
	ActualWindowWidth  int                       `json:"actualWindowWidth"`
	ActualWindowHeight int                       `json:"actualWindowHeight"`
	MonitorWidth       int                       `json:"monitorWidth"`
	MonitorHeight      int                       `json:"monitorHeight"`
	ContainerWidth     int                       `json:"containerWidth"`
	ContainerHeight    int                       `json:"containerHeight"`
	LogicalWidth       int                       `json:"logicalWidth"`
	LogicalHeight      int                       `json:"logicalHeight"`
	PresetResolutions  []displayResolutionResult `json:"presetResolutions"`
}

func executeDisplayDebugCommand(
	request debugging.Request,
	configuration *config.UltimaVConfiguration,
) (debugging.Result, bool) {
	if configuration == nil {
		panic("display debug commands require configuration")
	}

	return executeDisplayDebugCommandWithManager(request, display.GetManager())
}

func executeDisplayDebugCommandWithManager(
	request debugging.Request,
	manager debugDisplayManager,
) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandDisplayStatus:
		return debugSuccessResult(
			request,
			"Runtime game-window display status loaded.",
			newDisplayStatusResult(manager),
		), true
	case debugging.CommandDisplayModeSet:
		mode, err := config.ParseMode(mustDebugStringArgument(request, "mode"))
		if err != nil {
			panic("validated display mode was rejected: " + err.Error())
		}

		return requestRuntimeDisplayMode(request, manager, mode), true
	case debugging.CommandDisplayResolutionSet:
		return requestRuntimeDisplayResolution(request, manager), true
	case debugging.CommandDisplayFullscreenSet:
		mode := config.ModeWindowed
		if mustDebugBooleanArgument(request, "enabled") {
			mode = config.ModeFullscreen
		}

		return requestRuntimeDisplayMode(request, manager, mode), true
	default:
		var empty debugging.Result

		return empty, false
	}
}

func newDisplayStatusResult(manager debugDisplayManager) displayStatusResult {
	normalWidth, normalHeight := manager.LastNormalWindowSize()
	windowWidth, windowHeight := manager.WindowSize()
	monitorWidth, monitorHeight := manager.MonitorSize()
	containerWidth, containerHeight := manager.ContainerSize()
	logicalWidth, logicalHeight := manager.LogicalSize()
	presets := config.SupportedWindowResolutions()
	presetResults := make([]displayResolutionResult, 0, len(presets))
	for index, preset := range presets {
		presetResults = append(presetResults, displayResolutionResult{
			Index: index, Width: preset.X, Height: preset.Y,
		})
	}

	return displayStatusResult{
		RequestedMode: manager.RequestedMode(), ActualMode: manager.Mode(),
		TransitionPending:  manager.IsTransitionPending(),
		NormalWindowWidth:  normalWidth,
		NormalWindowHeight: normalHeight,
		ActualWindowWidth:  windowWidth,
		ActualWindowHeight: windowHeight,
		MonitorWidth:       monitorWidth,
		MonitorHeight:      monitorHeight,
		ContainerWidth:     containerWidth,
		ContainerHeight:    containerHeight,
		LogicalWidth:       logicalWidth,
		LogicalHeight:      logicalHeight,
		PresetResolutions:  presetResults,
	}
}

func requestRuntimeDisplayMode(
	request debugging.Request,
	manager debugDisplayManager,
	mode config.Mode,
) debugging.Result {
	before := manager.RequestedMode()
	width, height := manager.LastNormalWindowSize()
	if err := manager.RequestMode(mode, width, height); err != nil {
		return debugging.FailureResult(
			request,
			"display_request_failed",
			fmt.Sprintf("requesting runtime display mode: %v", err),
		)
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionPresentation,
		"Runtime game-window mode requested; player settings unchanged.",
		newDisplayStatusResult(manager),
	)
	appendSemanticChange(&result.Changes, "presentation.display.requestedMode", before, mode)

	return result
}

func requestRuntimeDisplayResolution(
	request debugging.Request,
	manager debugDisplayManager,
) debugging.Result {
	index := mustDebugIntegerArgument(request, "index")
	presets := config.SupportedWindowResolutions()
	if index < 0 || index >= len(presets) {
		panic("validated display resolution index was rejected")
	}
	preset := presets[index]
	beforeWidth, beforeHeight := manager.LastNormalWindowSize()
	if err := manager.RequestMode(manager.RequestedMode(), preset.X, preset.Y); err != nil {
		return debugging.FailureResult(
			request,
			"display_request_failed",
			fmt.Sprintf("requesting runtime window size: %v", err),
		)
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionPresentation,
		"Runtime game-window size requested; player settings unchanged.",
		newDisplayStatusResult(manager),
	)
	appendSemanticChange(&result.Changes, "presentation.display.normalWindowWidth", beforeWidth, preset.X)
	appendSemanticChange(&result.Changes, "presentation.display.normalWindowHeight", beforeHeight, preset.Y)

	return result
}
