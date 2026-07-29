<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Calculator, House, Info, Save } from '@lucide/vue'
import { api } from '../services/api'
import MoneyInput from '../components/MoneyInput.vue'

const loading = ref(true)
const saving = ref(false)
const data = ref({ input: {}, summary: {}, schedule: [] })
const form = reactive({
  propertyPrice: 500000000, downPaymentPercent: 20, tenorYears: 15,
  fixedRate: 5, fixedYears: 5, floatingRate: 10,
  monthlyIncome: 20000000, otherInstallments: 0, otherCosts: 0,
  startDate: new Date().toISOString().slice(0, 10),
})
const currency = (value = 0) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const percent = (value = 0) => `${Number(value).toLocaleString('id-ID', { maximumFractionDigits: 1 })}%`
const visibleSchedule = computed(() => data.value.schedule?.slice(0, 12) || [])
const statusLabels = { healthy: 'Sehat', safe: 'Batas aman', burdened: 'Membebani', unhealthy: 'Tidak sehat' }

async function load() {
  loading.value = true
  try {
    data.value = await api.mortgage()
    Object.assign(form, data.value.input)
  } finally { loading.value = false }
}
async function save() {
  saving.value = true
  try {
    data.value = await api.updateMortgage({
      ...form,
      propertyPrice: Number(form.propertyPrice), downPaymentPercent: Number(form.downPaymentPercent),
      tenorYears: Number(form.tenorYears), fixedRate: Number(form.fixedRate),
      fixedYears: Number(form.fixedYears), floatingRate: Number(form.floatingRate),
      monthlyIncome: Number(form.monthlyIncome), otherInstallments: Number(form.otherInstallments),
      otherCosts: Number(form.otherCosts),
    })
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
  <section class="page planning-detail-page">
    <RouterLink class="back-link" to="/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div><p class="eyebrow">Kalkulator properti</p><h1>Simulasi KPR</h1><p>Hitung fase bunga fixed dan floating serta dampaknya pada cashflow.</p></div>
    </div>
    <div class="mortgage-grid" :class="{ shimmer: loading }">
      <article class="panel module-form-card">
        <div class="panel-heading"><div><h2>Asumsi KPR</h2><p>Diadaptasi dari kolom input workbook.</p></div><Calculator :size="21" /></div>
        <form class="calculator-form" @submit.prevent="save">
          <label>Harga properti<MoneyInput v-model="form.propertyPrice" required /></label>
          <div class="compact-form-grid">
            <label>DP (%)<input v-model.number="form.downPaymentPercent" type="number" min="0" max="99" step="0.1"></label>
            <label>Tenor (tahun)<input v-model.number="form.tenorYears" type="number" min="1" max="30"></label>
            <label>Bunga fixed (%)<input v-model.number="form.fixedRate" type="number" min="0" max="100" step="0.01"></label>
            <label>Periode fixed (tahun)<input v-model.number="form.fixedYears" type="number" min="0" :max="form.tenorYears"></label>
            <label>Bunga floating (%)<input v-model.number="form.floatingRate" type="number" min="0" max="100" step="0.01"></label>
            <label>Mulai KPR<input v-model="form.startDate" type="date"></label>
          </div>
          <label>Penghasilan bulanan<MoneyInput v-model="form.monthlyIncome" /></label>
          <label>Cicilan lain per bulan<MoneyInput v-model="form.otherInstallments" /></label>
          <label>Estimasi biaya lainnya<MoneyInput v-model="form.otherCosts" /></label>
          <button class="primary-button full-button" :disabled="saving"><Save :size="16" />{{ saving ? 'Menghitung...' : 'Hitung & simpan' }}</button>
        </form>
      </article>
      <div class="mortgage-results">
        <article class="planning-hero mortgage-health">
          <span><House :size="25" /></span>
          <div><small>Kesimpulan kemampuan cicilan</small><strong>{{ statusLabels[data.summary?.status] || '—' }}</strong><p>{{ data.summary?.conclusion }}</p></div>
          <div class="mortgage-ratio"><small>Rasio maksimum</small><strong>{{ percent(data.summary?.maxRatio) }}</strong></div>
        </article>
        <div class="result-grid">
          <article class="panel result-card"><small>Pokok pinjaman</small><strong>{{ currency(data.summary?.principal) }}</strong><span>DP {{ currency(data.summary?.downPayment) }}</span></article>
          <article class="panel result-card"><small>Cicilan fixed</small><strong>{{ currency(data.summary?.fixedPayment) }}</strong><span>per bulan</span></article>
          <article class="panel result-card"><small>Cicilan floating</small><strong>{{ currency(data.summary?.floatingPayment) }}</strong><span>estimasi per bulan</span></article>
          <article class="panel result-card"><small>Total bunga</small><strong>{{ currency(data.summary?.totalInterest) }}</strong><span>selama tenor</span></article>
        </div>
        <article class="panel mortgage-summary">
          <div><span>Uang muka + biaya awal</span><strong>{{ currency(data.summary?.upfrontCost) }}</strong></div>
          <div><span>Rasio cicilan minimum</span><strong>{{ percent(data.summary?.minRatio) }}</strong></div>
          <div><span>Rasio cicilan maksimum</span><strong>{{ percent(data.summary?.maxRatio) }}</strong></div>
        </article>
        <p class="planning-footnote"><Info :size="15" /> Simulasi menggunakan metode anuitas. Bunga floating merupakan asumsi dan dapat berubah mengikuti kebijakan bank.</p>
      </div>
    </div>
    <article class="panel module-table-card">
      <div class="panel-heading"><div><h2>12 cicilan pertama</h2><p>Rincian pokok, bunga, dan sisa pinjaman.</p></div></div>
      <div class="module-table">
        <div class="module-table-head"><span>Bulan</span><span>Fase</span><span>Cicilan</span><span>Pokok</span><span>Bunga</span><span>Sisa pinjaman</span></div>
        <div v-for="row in visibleSchedule" :key="row.month" class="module-table-row">
          <span>{{ row.month }}</span><span class="pill">{{ row.rateType === 'fixed' ? 'Fixed' : 'Floating' }}</span>
          <strong>{{ currency(row.payment) }}</strong><span>{{ currency(row.principalPayment) }}</span>
          <span>{{ currency(row.interestPayment) }}</span><strong>{{ currency(row.remainingBalance) }}</strong>
        </div>
      </div>
    </article>
  </section>
</template>
