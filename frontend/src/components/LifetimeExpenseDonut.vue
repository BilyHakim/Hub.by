<script setup>
import { computed } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
  selectedId: { type: Number, default: null },
})
const emit = defineEmits(['update:selectedId'])

const total = computed(() => props.items.reduce((sum, item) => sum + Number(item.value || 0), 0))
const segments = computed(() => {
  let offset = 0
  return props.items.map((item) => {
    const share = total.value ? (Number(item.value || 0) / total.value) * 100 : 0
    const segment = { ...item, share, offset }
    offset += share
    return segment
  })
})
const selected = computed(() => props.items.find((item) => item.id === props.selectedId) || props.items[0])
const compactCurrency = (value) => `Rp${new Intl.NumberFormat('id-ID', { notation: 'compact', maximumFractionDigits: 1 }).format(value || 0)}`
</script>

<template>
  <div class="lifetime-donut-layout">
    <div class="interactive-donut" role="group" aria-label="Komposisi pengeluaran per kategori">
      <svg viewBox="0 0 42 42" aria-hidden="true">
        <circle class="donut-track" cx="21" cy="21" r="15.9155" pathLength="100" />
        <circle
          v-for="segment in segments"
          :key="segment.id"
          class="donut-segment"
          :class="{ selected: segment.id === selected?.id, muted: selected && segment.id !== selected.id }"
          cx="21" cy="21" r="15.9155" pathLength="100"
          :stroke="segment.color"
          :stroke-dasharray="`${segment.share} ${100 - segment.share}`"
          :stroke-dashoffset="-segment.offset"
          tabindex="0"
          role="button"
          :aria-label="`${segment.name}, ${segment.share.toFixed(1)} persen`"
          @click="emit('update:selectedId', segment.id)"
          @keydown.enter.prevent="emit('update:selectedId', segment.id)"
          @keydown.space.prevent="emit('update:selectedId', segment.id)"
        ><title>{{ segment.name }} · {{ compactCurrency(segment.value) }}</title></circle>
      </svg>
      <div class="interactive-donut-center">
        <small>{{ selected?.name || 'Total' }}</small>
        <strong>{{ selected ? `${selected.share.toFixed(1)}%` : compactCurrency(total) }}</strong>
      </div>
    </div>

    <div class="lifetime-donut-legend">
      <button
        v-for="item in items" :key="item.id" type="button"
        :class="{ active: item.id === selected?.id }"
        @click="emit('update:selectedId', item.id)"
      >
        <span><i :style="{ background: item.color }" />{{ item.name }}</span>
        <strong>{{ item.share.toFixed(1) }}%</strong>
      </button>
    </div>
  </div>
</template>
