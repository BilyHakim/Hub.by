<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ArrowLeft, CircleAlert, CircleCheck, HeartPulse } from '@lucide/vue'
import MonthPicker from '../components/MonthPicker.vue'
import { api } from '../services/api'

const month = ref(new Date().toISOString().slice(0, 7))
const loading = ref(true)
const data = ref({ items: [], healthyCount: 0, totalCount: 0 })
async function load() {
  loading.value = true
  try { data.value = await api.financialCheckup(month.value) } finally { loading.value = false }
}
async function initializePeriod() {
  try {
    const setting = await api.financePeriodSetting()
    month.value = setting.currentPeriodLabel
  } catch { /* default to calendar month */ }
  await load()
}
function handleWorkspaceChange() { initializePeriod() }
onMounted(() => {
  initializePeriod()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page planning-detail-page">
    <RouterLink class="back-link" to="/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div><p class="eyebrow">Pemeriksaan bulanan</p><h1>Financial check-up</h1><p>Rasio dihitung otomatis dari transaksi dan rekening pada workspace aktif.</p></div>
      <div class="period-picker-group">
        <MonthPicker v-model="month" @change="load" />
        <small v-if="data.periodStart">{{ data.periodStart }} — {{ data.periodEnd }}</small>
      </div>
    </div>
    <div class="planning-hero checkup-hero">
      <span><HeartPulse :size="27" /></span>
      <div><small>Skor kesehatan</small><strong>{{ data.healthyCount }} dari {{ data.totalCount }} indikator sehat</strong></div>
      <div class="checkup-score">{{ data.totalCount ? Math.round(data.healthyCount / data.totalCount * 100) : 0 }}<small>/100</small></div>
    </div>
    <div class="checkup-table panel" :class="{ shimmer: loading }">
      <div class="checkup-table-head"><span>Rasio</span><span>Formula</span><span>Rekomendasi</span><span>Hasil</span></div>
      <div v-for="item in data.items" :key="item.key" class="checkup-table-row">
        <span class="status-icon" :class="item.status"><CircleCheck v-if="item.status === 'healthy'" :size="18" /><CircleAlert v-else :size="18" /></span>
        <div><strong>{{ item.label }}</strong><small class="mobile-formula">{{ item.formula }}</small></div>
        <p>{{ item.formula }}</p>
        <span class="recommendation-pill">{{ item.recommendation }}</span>
        <strong class="ratio-result">{{ item.value.toFixed(1) }}{{ item.unit }}</strong>
      </div>
    </div>
  </section>
</template>
