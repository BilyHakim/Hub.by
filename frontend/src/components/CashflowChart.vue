<script setup>
import { computed } from 'vue'

const props = defineProps({ points: { type: Array, default: () => [] } })
const width = 640
const height = 225
const pad = 18

const max = computed(() => Math.max(...props.points.flatMap((p) => [p.income, p.expense]), 1) * 1.12)
const pointString = (key) => props.points.map((point, index) => {
  const x = pad + index * ((width - pad * 2) / Math.max(props.points.length - 1, 1))
  const y = height - pad - (point[key] / max.value) * (height - pad * 2)
  return `${x},${y}`
}).join(' ')
</script>

<template>
  <div class="cashflow-chart">
    <div class="chart-legend">
      <span><i class="legend-dot income-dot" />Pemasukan</span>
      <span><i class="legend-dot expense-dot" />Pengeluaran</span>
    </div>
    <svg :viewBox="`0 0 ${width} ${height}`" role="img" aria-label="Grafik pemasukan dan pengeluaran enam bulan">
      <defs>
        <linearGradient id="incomeArea" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stop-color="#49685c" stop-opacity=".2" />
          <stop offset="100%" stop-color="#49685c" stop-opacity="0" />
        </linearGradient>
      </defs>
      <line v-for="n in 4" :key="n" :x1="pad" :x2="width-pad" :y1="n*43" :y2="n*43" class="grid-line" />
      <polygon v-if="points.length" :points="`${pad},${height-pad} ${pointString('income')} ${width-pad},${height-pad}`" fill="url(#incomeArea)" />
      <polyline :points="pointString('income')" class="line line-income" />
      <polyline :points="pointString('expense')" class="line line-expense" />
      <g v-for="(point, index) in points" :key="point.month">
        <circle :cx="pad + index * ((width-pad*2) / Math.max(points.length-1,1))" :cy="height-pad-(point.income/max)*(height-pad*2)" r="4" class="point point-income" />
        <circle :cx="pad + index * ((width-pad*2) / Math.max(points.length-1,1))" :cy="height-pad-(point.expense/max)*(height-pad*2)" r="4" class="point point-expense" />
        <text :x="pad + index * ((width-pad*2) / Math.max(points.length-1,1))" :y="height-1" text-anchor="middle">{{ point.month }}</text>
      </g>
    </svg>
  </div>
</template>

