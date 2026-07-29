<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { CalendarDays, ChevronDown, ChevronLeft, ChevronRight } from '@lucide/vue'

const props = defineProps({
  modelValue: { type: String, required: true },
  compact: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'change'])

const open = ref(false)
const viewYear = ref(Number(props.modelValue.slice(0, 4)) || new Date().getFullYear())
const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des']

const selectedYear = computed(() => Number(props.modelValue.slice(0, 4)))
const selectedMonth = computed(() => Number(props.modelValue.slice(5, 7)) - 1)
const label = computed(() => new Intl.DateTimeFormat('id-ID', {
  month: props.compact ? 'short' : 'long',
  year: 'numeric',
}).format(new Date(`${props.modelValue}-02T00:00:00`)))

watch(() => props.modelValue, (value) => {
  if (!open.value) viewYear.value = Number(value.slice(0, 4))
})

function toggle() {
  open.value = !open.value
  if (open.value) viewYear.value = selectedYear.value
}
function selectMonth(index) {
  const value = `${viewYear.value}-${String(index + 1).padStart(2, '0')}`
  emit('update:modelValue', value)
  emit('change', value)
  open.value = false
}
function selectCurrentMonth() {
  const now = new Date()
  viewYear.value = now.getFullYear()
  selectMonth(now.getMonth())
}
function close() {
  open.value = false
}
function onKeydown(event) {
  if (event.key === 'Escape') close()
}

onMounted(() => {
  document.addEventListener('click', close)
  document.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', close)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div class="month-select" :class="{ compact }" @click.stop>
    <button class="month-picker" type="button" :aria-expanded="open" @click="toggle">
      <CalendarDays v-if="compact" :size="17" />
      <span>{{ label }}</span>
      <ChevronDown :size="17" :class="{ rotated: open }" />
    </button>
    <Transition name="dropdown">
      <div v-if="open" class="month-popover">
        <div class="month-year-nav">
          <button type="button" aria-label="Tahun sebelumnya" @click="viewYear--"><ChevronLeft :size="18" /></button>
          <strong>{{ viewYear }}</strong>
          <button type="button" aria-label="Tahun berikutnya" @click="viewYear++"><ChevronRight :size="18" /></button>
        </div>
        <div class="month-grid">
          <button
            v-for="(name, index) in monthNames"
            :key="name"
            type="button"
            :class="{ selected: viewYear === selectedYear && index === selectedMonth }"
            @click="selectMonth(index)"
          >
            {{ name }}
          </button>
        </div>
        <button class="current-month-button" type="button" @click="selectCurrentMonth">Bulan ini</button>
      </div>
    </Transition>
  </div>
</template>

