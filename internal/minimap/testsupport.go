package minimap

// GetCurrentStep returns the active step for presentation rendering.
func (e *ScriptExecutor) GetCurrentStep() *ScriptStep {
	if e.IsComplete() {
		return nil
	}

	return &e.script.Steps[e.currentStepIndex]
}
