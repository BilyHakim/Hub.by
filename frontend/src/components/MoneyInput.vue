<script setup>
import { computed, nextTick, ref, useAttrs } from 'vue'
import { Calculator } from '@lucide/vue'
import { calculate } from '../utils/calculator'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { type: [Number, String], default: 0 },
  allowNegative: { type: Boolean, default: false },
  calculator: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])
const attrs = useAttrs()
const calculatorOpen = ref(false)
const expression = ref('')
const expressionInput = ref(null)
const amountInput = ref(null)
const keys = ['7', '8', '9', '÷', '4', '5', '6', '×', '1', '2', '3', '−', '0', ',', '=', '+']
const calculation = computed(() => {
  try {
    const value = Math.round(calculate(expression.value))
    if (!props.allowNegative && value < 0) return { error: 'Nominal tidak boleh negatif.' }
    return { value }
  } catch (error) {
    return { error: error.message }
  }
})
async function openCalculator() {
  expression.value = String(numericValue.value)
  calculatorOpen.value = true
  await nextTick()
  expressionInput.value?.focus()
  expressionInput.value?.select()
}
async function closeCalculator() {
  calculatorOpen.value = false
  await nextTick()
  amountInput.value?.focus()
}
function press(key) {
  if (key === '=') {
    if (!calculation.value.error) expression.value = String(calculation.value.value)
  } else {
    expression.value = expression.value === '0' && /^\d$/.test(key) ? key : expression.value + key
  }
}
function applyResult() {
  if (calculation.value.error) return
  emit('update:modelValue', calculation.value.value)
  closeCalculator()
}
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
  <div class="money-field">
  <div class="money-input">
    <span>Rp</span>
    <input
      ref="amountInput"
      v-bind="attrs"
      :value="displayValue"
      type="text"
      inputmode="numeric"
      autocomplete="off"
      @input="update"
      @blur="normalize"
    >
    <button v-if="calculator" type="button" class="calculator-toggle" aria-label="Buka kalkulator nominal" :aria-expanded="calculatorOpen" @click="calculatorOpen ? closeCalculator() : openCalculator()">
      <Calculator :size="19" />
    </button>
  </div>
  <section v-if="calculatorOpen" class="amount-calculator" aria-label="Kalkulator nominal" @keydown.esc.stop.prevent="closeCalculator">
    <div class="calculator-heading"><strong>Kalkulator</strong><button type="button" @click="closeCalculator">Tutup</button></div>
    <input ref="expressionInput" v-model="expression" aria-label="Perhitungan nominal" type="text" inputmode="text" autocomplete="off" spellcheck="false" @keydown.enter.stop.prevent="applyResult">
    <p class="calculator-result" aria-live="polite">{{ calculation.error || `= Rp ${formatter.format(calculation.value)}` }}</p>
    <div class="calculator-actions">
      <button type="button" @click="expression = '0'">C</button>
      <button type="button" aria-label="Hapus angka terakhir" @click="expression = expression.slice(0, -1) || '0'">Hapus</button>
    </div>
    <div class="calculator-keys">
      <button v-for="key in keys" :key="key" type="button" :class="{ operator: ['+', '−', '×', '÷', '='].includes(key) }" @click="press(key)">{{ key }}</button>
    </div>
    <small>Hasil dibulatkan ke rupiah terdekat.</small>
    <button type="button" class="primary-button full-button" :disabled="!!calculation.error" @click="applyResult">Gunakan hasil</button>
  </section>
  </div>
</template>

<style scoped>
.money-field { min-width: 0; }
.money-input input { min-width: 0; width: 100%; }
.calculator-toggle { display: grid; place-items: center; flex-shrink: 0; width: 40px; height: 40px; border: 0; background: transparent; color: var(--primary, #49685c); cursor: pointer; }
.amount-calculator { margin-top: 8px; padding: 12px; border: 1px solid var(--line); border-radius: 12px; background: var(--surface); }
.calculator-heading { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.calculator-heading button { border: 0; background: transparent; color: var(--muted); cursor: pointer; }
.calculator-result { margin: 8px 0; min-height: 18px; overflow-wrap: anywhere; }
.calculator-actions, .calculator-keys { display: grid; gap: 6px; margin-bottom: 6px; }
.calculator-actions { grid-template-columns: repeat(2, 1fr); }
.calculator-keys { grid-template-columns: repeat(4, 1fr); }
.calculator-actions button, .calculator-keys button { min-height: 40px; border: 1px solid var(--line); border-radius: 7px; background: white; color: var(--ink); font: inherit; font-size: 16px; cursor: pointer; }
.calculator-keys .operator { background: #e8eeea; color: #38574b; }
.amount-calculator button:focus-visible { outline: 2px solid #49685c; outline-offset: 2px; }
.amount-calculator small { display: block; margin: 8px 0; color: var(--muted); }
</style>
