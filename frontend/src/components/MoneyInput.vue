<script setup>
import { computed, useAttrs } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { type: [Number, String], default: 0 },
  allowNegative: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])
const attrs = useAttrs()
const formatter = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 })
const numericValue = computed(() => {
  const value = Number(props.modelValue)
  return Number.isFinite(value) ? Math.trunc(value) : 0
})
const displayValue = computed(() => formatter.format(numericValue.value))

function update(event) {
  const raw = event.target.value
  const negative = props.allowNegative && raw.trimStart().startsWith('-')
  const digits = raw.replace(/\D/g, '')
  const value = digits ? Number(digits) * (negative ? -1 : 1) : 0
  emit('update:modelValue', value)
  event.target.value = formatter.format(value)
}
function normalize(event) {
  event.target.value = formatter.format(numericValue.value)
}
</script>

<template>
  <div class="money-input">
    <span>Rp</span>
    <input
      v-bind="attrs"
      :value="displayValue"
      type="text"
      inputmode="numeric"
      autocomplete="off"
      @input="update"
      @blur="normalize"
    >
  </div>
</template>
