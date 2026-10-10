<script setup lang="ts">
import type { OptionSets } from "./command_forms"
import type { CommandDefinition, ParameterDefinition } from "./protocol"
import { scenarioStepDisplay, type ScenarioStep } from "./scenarios"

const props = defineProps<{
  steps: ScenarioStep[]
  parameters?: ParameterDefinition[]
  commands: CommandDefinition[]
  optionSets: OptionSets
}>()
</script>

<template>
  <ol class="scenario-execution-plan scenario-step-summary">
    <li v-for="(step, index) in props.steps" :key="index" :class="[`scenario-summary-step-${step.type}`, { 'has-issues': scenarioStepDisplay(step, props.parameters ?? [], props.commands, props.optionSets).unknownCommand }]" :data-scenario-summary-step="index + 1" :data-summary-step-kind="step.type">
      <span>{{ String(index + 1).padStart(2, '0') }}</span>
      <div>
        <strong :class="{ 'scenario-unknown-command': scenarioStepDisplay(step, props.parameters ?? [], props.commands, props.optionSets).unknownCommand }">{{ scenarioStepDisplay(step, props.parameters ?? [], props.commands, props.optionSets).commandName }}</strong>
        <code v-if="scenarioStepDisplay(step, props.parameters ?? [], props.commands, props.optionSets).commandId">{{ scenarioStepDisplay(step, props.parameters ?? [], props.commands, props.optionSets).commandId }}</code>
      </div>
      <small :title="scenarioStepDisplay(step, props.parameters ?? [], props.commands, props.optionSets).fullDetail">{{ scenarioStepDisplay(step, props.parameters ?? [], props.commands, props.optionSets).detail }}</small>
    </li>
  </ol>
</template>
