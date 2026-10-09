package minimap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestStepType_Constants(t *testing.T) {
	assert.Equal(t, StepType(0), StepWalk)
	assert.Equal(t, StepType(1), StepWait)
	assert.Equal(t, StepType(2), StepFace)
	assert.Equal(t, StepType(3), StepSetVisible)
	assert.Equal(t, StepType(4), StepSetSprite)
	assert.Equal(t, StepType(5), StepSetFrame)
	assert.Equal(t, StepType(6), StepAwaitAdvance)
}

func TestMiniMapScriptValidateRejectsMalformedSteps(t *testing.T) {
	tests := []struct {
		name   string
		script *MiniMapScript
	}{
		{name: "nil", script: nil},
		{name: "walk", script: &MiniMapScript{Steps: []ScriptStep{{Type: StepWalk}}}},
		{name: "wait", script: &MiniMapScript{Steps: []ScriptStep{{Type: StepWait, Duration: -1}}}},
		{name: "actor", script: &MiniMapScript{Steps: []ScriptStep{{Type: StepFace}}}},
		{name: "frame", script: &MiniMapScript{Steps: []ScriptStep{{Type: StepSetFrame}}}},
		{name: "unknown", script: &MiniMapScript{Steps: []ScriptStep{{Type: StepType(99)}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Error(t, test.script.Validate())
		})
	}
}

func TestScriptStep_WalkStep(t *testing.T) {
	target := references.Position{X: 5, Y: 5}
	step := ScriptStep{
		Type:     StepWalk,
		ActorIDs: []ActorID{ActorAvatar},
		Target:   &target,
		DelayMs:  300,
	}

	assert.Equal(t, StepWalk, step.Type)
	assert.Equal(t, []ActorID{ActorAvatar}, step.ActorIDs)
	assert.Equal(t, references.Position{X: 5, Y: 5}, *step.Target)
	assert.Equal(t, 300, step.DelayMs)
}

func TestScriptStep_WaitStep(t *testing.T) {
	step := ScriptStep{
		Type:     StepWait,
		Duration: 500,
	}

	assert.Equal(t, StepWait, step.Type)
	assert.Equal(t, 500, step.Duration)
}

func TestScriptStep_FaceStep(t *testing.T) {
	step := ScriptStep{
		Type:      StepFace,
		ActorIDs:  []ActorID{ActorAvatar},
		Direction: references.Up,
	}

	assert.Equal(t, StepFace, step.Type)
	assert.Equal(t, references.Up, step.Direction)
}

func TestMiniMapScript_HasSteps(t *testing.T) {
	target := references.Position{X: 5, Y: 5}
	script := &MiniMapScript{
		Name: "test",
		Steps: []ScriptStep{
			{Type: StepWalk, ActorIDs: []ActorID{ActorAvatar}, Target: &target},
		},
	}

	assert.Equal(t, "test", script.Name)
	assert.Len(t, script.Steps, 1)
}

func TestMiniMapScript_MultipleSteps(t *testing.T) {
	target := references.Position{X: 5, Y: 5}
	script := &MiniMapScript{
		Name: "multi_step",
		Steps: []ScriptStep{
			{Type: StepWalk, ActorIDs: []ActorID{ActorAvatar}, Target: &target, DelayMs: 300},
			{Type: StepWait, Duration: 200},
			{Type: StepFace, ActorIDs: []ActorID{ActorAvatar}, Direction: references.Up},
		},
	}

	assert.Len(t, script.Steps, 3)
	assert.Equal(t, StepWalk, script.Steps[0].Type)
	assert.Equal(t, StepWait, script.Steps[1].Type)
	assert.Equal(t, StepFace, script.Steps[2].Type)
}
