<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { CalendarRange, Check, Info, Save } from '@lucide/vue'
import { api } from '../services/api'

const loading = ref(true)
const saving = ref(false)
const saved = ref(false)
const periodMode = ref('fixed_day')
const periodStartDay = ref(1)
const setting = ref({ currentPeriodLabel: '', example: { start: '', end: '' } })
const exampleMonth = ref(new Date().toISOString().slice(0, 7))
const dateLabel = (value) => value ? new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(`${value.slice(0,10)}T00:00:00`)) : '—'
const exampleLabel = computed(() => {
  const month = setting.value.currentPeriodLabel || exampleMonth.value
  return new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric' }).format(new Date(`${month}-01T00:00:00`))
})

async function load() {
  loading.value = true
  try {
    setting.value = await api.financePeriodSetting()
    periodMode.value = setting.value.periodMode || 'fixed_day'
    periodStartDay.value = setting.value.periodStartDay
    exampleMonth.value = setting.value.currentPeriodLabel
  } finally { loading.value = false }
}
async function preview() {
  try { setting.value = await api.financePeriodSetting(exampleMonth.value) } catch { /* keep previous preview */ }
}
async function save() {
  saving.value = true
  saved.value = false
  try {
    setting.value = await api.updateFinancePeriodSetting(periodMode.value, Number(periodStartDay.value))
    exampleMonth.value = setting.value.currentPeriodLabel
    saved.value = true
    window.dispatchEvent(new CustomEvent('hubby:period-changed'))
    window.setTimeout(() => { saved.value = false }, 2400)
  } finally { saving.value = false }
}
function handleWorkspaceChange() { load() }
onMounted(() => {
  load()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page settings-page">
    <div class="page-heading compact">
      <div><p class="eyebrow">Workspace aktif</p><h1>Pengaturan</h1><p>Atur cara Hubby mengelompokkan transaksi untuk ruang keuangan ini.</p></div>
    </div>
    <article class="panel period-settings-card" :class="{ shimmer: loading }">
      <div class="setting-icon"><CalendarRange :size="24" /></div>
      <div class="setting-content">
        <div class="panel-heading"><div><h2>Awal periode keuangan</h2><p>Sesuaikan periode dengan tanggal gajian atau siklus pencatatanmu.</p></div></div>
        <form @submit.prevent="save">
          <fieldset class="period-mode-picker">
            <legend>Mode periode</legend>
            <label :class="{ active: periodMode === 'fixed_day' }">
              <input v-model="periodMode" type="radio" value="fixed_day">
              <span><strong>Tanggal tetap</strong><small>Pilih tanggal yang sama setiap bulan.</small></span>
            </label>
            <label :class="{ active: periodMode === 'end_of_month' }">
              <input v-model="periodMode" type="radio" value="end_of_month">
              <span><strong>Hari terakhir setiap bulan</strong><small>Otomatis mengikuti 28, 29, 30, atau 31.</small></span>
            </label>
          </fieldset>
          <label v-if="periodMode === 'fixed_day'">Periode dimulai setiap tanggal
            <select v-model.number="periodStartDay">
              <option v-for="day in 31" :key="day" :value="day">Tanggal {{ day }}</option>
            </select>
          </label>
          <div class="period-explanation">
            <Info :size="18" />
            <p v-if="periodMode === 'fixed_day'">Tanggal awal termasuk periode baru. Agar tidak ada transaksi ganda, hari sebelum tanggal awal berikutnya menjadi akhir periode. Untuk tanggal 29–31, Hubby otomatis memakai hari terakhir pada bulan yang lebih pendek.</p>
            <p v-else>Periode baru selalu dimulai pada hari terakhir bulan sebelumnya. Contohnya, periode Oktober 2026 berjalan dari 30 September sampai 30 Oktober.</p>
          </div>
          <button class="primary-button" :disabled="saving"><Save v-if="!saved" :size="16" /><Check v-else :size="16" />{{ saving ? 'Menyimpan...' : saved ? 'Tersimpan' : 'Simpan pengaturan' }}</button>
        </form>
      </div>
      <aside class="period-preview">
        <small>Periode aktif · {{ exampleLabel }}</small>
        <strong>{{ dateLabel(setting.example?.start) }}</strong>
        <span>sampai</span>
        <strong>{{ dateLabel(setting.example?.end) }}</strong>
        <p>Semua dashboard, transaksi, check-up, dan rekomendasi dana darurat menggunakan rentang ini.</p>
      </aside>
    </article>
  </section>
</template>
