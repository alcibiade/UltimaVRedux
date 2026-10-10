<script setup lang="ts">
import { computed } from "vue"

import type { OptionSets } from "./command_forms"
import type { ParameterDefinition } from "./protocol"
import {
  scenarioLiteralDisplay,
  scenarioLiteralValue,
  scenarioParameterChoices,
  scenarioPlanningParameter,
  scenarioValueIssue,
} from "./scenarios"

const props = defineProps<{
  parameter: ParameterDefinition
  value: unknown
  source: "literal" | string
  launchParameters: ParameterDefinition[]
  optionSets: OptionSets
}>()

const emit = defineEmits<{
  updateSource: [source: string]
  updateLiteral: [value: unknown]
}>()

const planningParameter = computed(() => scenarioPlanningParameter(props.parameter))
const choices = computed(() => scenarioParameterChoices(planningParameter.value, props.optionSets))
const usesChoiceControl = computed(() => choices.value.length > 0 || planningParameter.value.kind === "enum" || Boolean(planningParameter.value.optionsProvider))
const issue = computed(() => props.source === "literal" ? scenarioValueIssue(planningParameter.value, props.value, props.optionSets) : undefined)

function updateLiteral(raw: string): void {
  emit("updateLiteral", scenarioLiteralValue(planningParameter.value, raw))
}
</script>

<template>
  <div v-if="!parameter.scenarioRuntime" class="scenario-argument" :data-scenario-argument="parameter.id">
    <div class="scenario-argument-label">
      <strong>{{ parameter.name }}</strong>
      <code>{{ parameter.id }}</code>
      <small>{{ parameter.required ? 'REQUIRED' : 'OPTIONAL' }} · {{ parameter.kind }}</small>
      <small v-if="parameter.description">{{ parameter.description }}</small>
	  <small v-if="parameter.scenarioValueResolver" class="scenario-resolution-help">Stored as a generic target. At runtime, {{ parameter.scenarioValueResolver }} resolves it against the active party.</small>
    </div>
    <label>SOURCE
      <select :aria-label="`Source for ${parameter.id}`" :value="source" @change="emit('updateSource', ($event.target as HTMLSelectElement).value)">
        <option value="literal">Literal Value</option>
        <option v-if="source !== 'literal' && !launchParameters.some((candidate) => candidate.id === source)" :value="source">Incompatible Parameter: {{ source }}</option>
        <option v-for="candidate in launchParameters" :key="candidate.id" :value="candidate.id">Launch Parameter: {{ candidate.name }} ({{ candidate.id }})</option>
      </select>
    </label>
    <label v-if="source === 'literal'">VALUE
      <select v-if="parameter.kind === 'boolean'" :aria-label="`Literal value for ${parameter.id}`" :value="scenarioLiteralDisplay(value)" @change="updateLiteral(($event.target as HTMLSelectElement).value)">
        <option value="">UNSET</option><option value="true">TRUE</option><option value="false">FALSE</option>
      </select>
      <select v-else-if="usesChoiceControl" :aria-label="`Literal value for ${parameter.id}`" :value="scenarioLiteralDisplay(value)" @change="updateLiteral(($event.target as HTMLSelectElement).value)">
        <option value="">UNSET</option><option v-for="choice in choices" :key="choice.value" :value="choice.value">{{ choice.label }}</option>
      </select>
      <input v-else :aria-label="`Literal value for ${parameter.id}`" :type="parameter.kind === 'integer' ? 'number' : 'text'" :min="parameter.minimum" :max="parameter.maximum" :pattern="parameter.pattern" :placeholder="parameter.kind === 'integer_list' ? '1,2,3' : undefined" :value="scenarioLiteralDisplay(value)" @input="updateLiteral(($event.target as HTMLInputElement).value)" />
      <small class="scenario-literal-preview" :class="{ unset: !scenarioLiteralDisplay(value) }">
        <template v-if="scenarioLiteralDisplay(value)"><span>STORED AS</span><code>{{ scenarioLiteralDisplay(value) }}</code></template>
        <span v-else>NO VALUE SET</span>
      </small>
    </label>
    <div v-else class="scenario-argument-reference"><span>VALUE PROVIDED AT LAUNCH</span><code>{{ source }}</code></div>
    <p v-if="issue" class="error-line" :data-scenario-argument-issue="parameter.id">{{ issue }}</p>
  </div>
</template>
