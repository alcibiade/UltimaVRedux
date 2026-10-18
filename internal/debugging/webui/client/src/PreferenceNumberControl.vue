<script setup lang="ts">
import { computed, watch } from "vue"
import type { CommandDefinition } from "./protocol"
import { validateParameter } from "./command_forms"
import { useInstrumentDraft } from "./instrument_session"

const props = defineProps<{ commandId: string; command?: CommandDefinition; value: number; disabled: boolean }>()
const emit = defineEmits<{ apply: [argumentsByID: Record<string, string>] }>()
// Number inputs implicitly coerce Vue v-model values. Keep raw DOM strings so
// validation and retained drafts distinguish an empty field from explicit zero.
const draft = useInstrumentDraft<string>(`preference.${props.commandId}`, "")
const previous = useInstrumentDraft<number | undefined>(`preference.${props.commandId}.previous`, undefined)
const parameter = computed(() => props.command?.parameters?.[0])
const error = computed(() => parameter.value ? validateParameter(parameter.value, draft.value) : "Preference metadata unavailable.")
watch(() => props.value, (value) => {
  if (previous.value === undefined || draft.value === String(previous.value)) draft.value = String(value)
  previous.value = value
}, { immediate: true })
function apply(): void {
  if (props.disabled || error.value || !parameter.value || Number(draft.value) === props.value) return
  emit("apply", { [parameter.value.id]: draft.value })
}
</script>

<template>
  <section class="instrument-setting" :data-preference="commandId">
    <div><strong>{{ command?.name }}</strong><p>{{ command?.description }}</p></div>
    <input :value="draft" @input="draft = ($event.target as HTMLInputElement).value" :aria-label="command?.name" type="number" :min="parameter?.minimum" :max="parameter?.maximum" :step="parameter?.step ?? 1" :disabled="disabled || !parameter" />
    <button type="button" :disabled="disabled || Boolean(error) || Number(draft) === value" @click="apply">APPLY {{ draft }}</button>
    <small v-if="error" role="status">{{ error }}</small>
  </section>
</template>
