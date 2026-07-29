<script setup>
import { ArrowUpRight, ArrowDownRight } from '@lucide/vue'

defineProps({
  label: String,
  value: String,
  note: String,
  trend: Number,
  tone: { type: String, default: 'sage' },
})
</script>

<template>
  <article class="metric-card" :class="`tone-${tone}`">
    <div class="metric-top">
      <span class="metric-icon"><slot name="icon" /></span>
      <span v-if="trend !== undefined" class="trend" :class="{ negative: trend < 0 }">
        <ArrowUpRight v-if="trend >= 0" :size="14" />
        <ArrowDownRight v-else :size="14" />
        {{ Math.abs(trend).toLocaleString('id-ID') }}%
      </span>
    </div>
    <p>{{ label }}</p>
    <h3>{{ value }}</h3>
    <small>{{ note }}</small>
  </article>
</template>
