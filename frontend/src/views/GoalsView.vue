<script setup>
import { ref, onBeforeUnmount, onMounted } from 'vue'
import { Target, Plane, Home, GraduationCap, Plus, CalendarDays, ArrowRight } from '@lucide/vue'
import { api } from '../services/api'
import { demoGoals } from '../data/demo'

const goals = ref([])
const icons = { plane: Plane, home: Home, 'graduation-cap': GraduationCap }
const currency = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const dateLabel = (date) => new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric' }).format(new Date(`${date}T00:00:00`))
let requestSequence = 0
async function loadGoals() {
  const requestID = ++requestSequence
  try {
    const result = await api.goals()
    if (requestID === requestSequence) goals.value = result || []
  } catch {
    if (requestID === requestSequence) goals.value = demoGoals
  }
}
function handleWorkspaceChange() {
  goals.value = []
  loadGoals()
}
onMounted(() => {
  loadGoals()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page">
    <div class="page-heading compact">
      <div><p class="eyebrow">Rencana masa depan</p><h1>Tujuan keuangan</h1><p>Ubah mimpi besar menjadi langkah kecil yang bisa dilihat.</p></div>
      <button class="primary-button"><Plus :size="18" /> Buat tujuan</button>
    </div>
    <div class="goal-highlight">
      <span><Target :size="28" /></span>
      <div><small>Total targetmu</small><strong>{{ currency(goals.reduce((sum, goal) => sum + goal.targetAmount, 0)) }}</strong><p>Dari {{ goals.length }} tujuan aktif</p></div>
      <div class="highlight-progress"><strong>{{ currency(goals.reduce((sum, goal) => sum + goal.currentAmount, 0)) }}</strong><small>sudah terkumpul</small></div>
    </div>
    <div class="goals-grid">
      <article v-for="goal in goals" :key="goal.id" class="panel goal-card">
        <div class="goal-top">
          <span class="goal-icon"><component :is="icons[goal.icon] || Target" :size="22" /></span>
          <span class="goal-status">Aktif</span>
        </div>
        <h2>{{ goal.name }}</h2>
        <p><CalendarDays :size="15" /> Target {{ dateLabel(goal.targetDate) }}</p>
        <div class="goal-numbers"><strong>{{ currency(goal.currentAmount) }}</strong><span>dari {{ currency(goal.targetAmount) }}</span></div>
        <div class="progress-track"><span :style="{ width: `${Math.min(goal.currentAmount / goal.targetAmount * 100, 100)}%` }" /></div>
        <div class="progress-caption"><span>{{ (goal.currentAmount / goal.targetAmount * 100).toFixed(0) }}% terkumpul</span><span>{{ currency(goal.targetAmount - goal.currentAmount) }} lagi</span></div>
        <button class="secondary-button">Lihat rencana <ArrowRight :size="16" /></button>
      </article>
    </div>
  </section>
</template>
