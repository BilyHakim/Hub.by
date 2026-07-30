<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ArrowUpRight, ArrowDownRight, Info, Minus } from '@lucide/vue'

defineProps({
  label: String,
  value: String,
  note: String,
  trend: Number,
  inverseTrend: Boolean,
  trendHint: { type: String, default: 'Dibanding periode sebelumnya' },
  comparison: Object,
  tone: { type: String, default: 'sage' },
})

const comparisonOpen = ref(false)
const comparisonRoot = ref(null)

const currency = (value) => new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
}).format(Number(value || 0))

function handleOutsideClick(event) {
  if (!comparisonRoot.value?.contains(event.target)) comparisonOpen.value = false
}

onMounted(() => document.addEventListener('click', handleOutsideClick))
onBeforeUnmount(() => document.removeEventListener('click', handleOutsideClick))
</script>

<template>
  <article class="metric-card" :class="`tone-${tone}`">
    <div class="metric-top">
      <span class="metric-icon"><slot name="icon" /></span>
      <div v-if="comparison" ref="comparisonRoot" class="metric-comparison">
        <button
          type="button"
          class="trend"
          :class="{ negative: trend !== null && (inverseTrend ? trend > 0 : trend < 0) }"
          :title="`${trendHint} · Klik untuk melihat perhitungan`"
          :aria-expanded="comparisonOpen"
          @click.stop="comparisonOpen = !comparisonOpen"
        >
          <Minus v-if="trend === null || trend === undefined" :size="14" />
          <ArrowUpRight v-else-if="trend >= 0" :size="14" />
          <ArrowDownRight v-else :size="14" />
          <template v-if="trend === null || trend === undefined">Baru</template>
          <template v-else>{{ Math.abs(trend).toLocaleString('id-ID', { maximumFractionDigits: 1 }) }}%</template>
        </button>
        <Transition name="dropdown">
          <div v-if="comparisonOpen" class="comparison-popover">
            <div class="comparison-heading">
              <span><Info :size="15" /></span>
              <div><strong>{{ comparison.title }}</strong><small>{{ comparison.description }}</small></div>
            </div>
            <div class="comparison-values">
              <div>
                <span>{{ comparison.currentLabel }}</span>
                <strong>{{ currency(comparison.currentValue) }}</strong>
              </div>
              <div>
                <span>{{ comparison.previousLabel }}</span>
                <strong>{{ currency(comparison.previousValue) }}</strong>
              </div>
            </div>
            <div class="comparison-result">
              <span>Selisih nominal</span>
              <strong :class="{ negative: comparison.inverse ? comparison.delta > 0 : comparison.delta < 0 }">
                {{ comparison.delta > 0 ? '+' : '' }}{{ currency(comparison.delta) }}
              </strong>
            </div>
            <p>{{ comparison.formula }}</p>
          </div>
        </Transition>
      </div>
      <span
        v-else-if="trend !== undefined && trend !== null"
        class="trend"
        :class="{ negative: inverseTrend ? trend > 0 : trend < 0 }"
        :title="trendHint"
      >
        <ArrowUpRight v-if="trend >= 0" :size="14" />
        <ArrowDownRight v-else :size="14" />
        {{ Math.abs(trend).toLocaleString('id-ID', { maximumFractionDigits: 1 }) }}%
      </span>
    </div>
    <p>{{ label }}</p>
    <h3>{{ value }}</h3>
    <small>{{ note }}</small>
  </article>
</template>
