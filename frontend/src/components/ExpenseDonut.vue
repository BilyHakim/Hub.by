<script setup>
import { computed } from 'vue'

const props = defineProps({ items: { type: Array, default: () => [] } })
const safeItems = computed(() => props.items || [])
const total = computed(() => safeItems.value.reduce((sum, item) => sum + item.value, 0))
const gradient = computed(() => {
  let current = 0
  const stops = safeItems.value.map((item) => {
    const start = current
    current += total.value ? (item.value / total.value) * 100 : 0
    return `${item.color} ${start}% ${current}%`
  })
  return stops.length ? `conic-gradient(${stops.join(',')})` : 'conic-gradient(#eceae4 0 100%)'
})
const shortCurrency = (value) => `${(value / 1_000_000).toLocaleString('id-ID', { maximumFractionDigits: 1 })} jt`
</script>

<template>
  <div class="expense-content">
    <div class="donut" :style="{ background: gradient }">
      <div><small>Total</small><strong>{{ shortCurrency(total) }}</strong></div>
    </div>
    <div class="expense-legend">
      <div v-for="item in safeItems" :key="item.name">
        <span><i :style="{ background: item.color }" />{{ item.name }}</span>
        <strong>{{ shortCurrency(item.value) }}</strong>
      </div>
    </div>
  </div>
</template>
