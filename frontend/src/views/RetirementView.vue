<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Info, Save, Sparkles, Umbrella } from '@lucide/vue'
import { api } from '../services/api'

const loading = ref(true)
const saving = ref(false)
const data = ref({ input: {}, summary: {}, projection: [] })
const form = reactive({
  currentAge: 25, retirementAge: 55, monthlyExpense: 5000000,
  inflationRate: 3, expectedReturn: 6, currentFund: 100000000,
  monthlyContribution: 4000000, withdrawalRate: 4,
})
const currency = (value = 0) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const shortCurrency = (value = 0) => `${(Number(value) / 1_000_000_000).toLocaleString('id-ID', { maximumFractionDigits: 2 })} M`
const progressWidth = computed(() => Math.min(Number(data.value.summary?.progress || 0), 100))
const projectionMilestones = computed(() => {
  const rows = data.value.projection || []
  return rows.filter((row, index) => index === 0 || row.year % 5 === 0 || index === rows.length - 1)
})

async function load() {
  loading.value = true
  try {
    data.value = await api.retirement()
    Object.assign(form, data.value.input)
  } finally { loading.value = false }
}
async function save() {
  saving.value = true
  try {
    data.value = await api.updateRetirement({
      currentAge: Number(form.currentAge), retirementAge: Number(form.retirementAge),
      monthlyExpense: Number(form.monthlyExpense), inflationRate: Number(form.inflationRate),
      expectedReturn: Number(form.expectedReturn), currentFund: Number(form.currentFund),
      monthlyContribution: Number(form.monthlyContribution), withdrawalRate: Number(form.withdrawalRate),
    })
  } finally { saving.value = false }
}
function useRecommendation() { form.monthlyContribution = data.value.summary?.requiredMonthlyContribution || 0 }
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
      <div><p class="eyebrow">Pendekatan 4% rule</p><h1>Persiapan pensiun</h1><p>Proyeksikan biaya hidup dan pertumbuhan dana hingga usia pensiun.</p></div>
    </div>
    <div class="retirement-grid" :class="{ shimmer: loading }">
      <article class="panel module-form-card">
        <div class="panel-heading"><div><h2>Asumsi perencanaan</h2><p>Sesuaikan dengan kondisi dan targetmu.</p></div><Umbrella :size="21" /></div>
        <form class="calculator-form" @submit.prevent="save">
          <div class="compact-form-grid">
            <label>Usia saat ini<input v-model.number="form.currentAge" type="number" min="15" max="99"></label>
            <label>Usia pensiun<input v-model.number="form.retirementAge" type="number" :min="form.currentAge + 1" max="100"></label>
          </div>
          <label>Pengeluaran bulanan saat ini<div class="money-input"><span>Rp</span><input v-model.number="form.monthlyExpense" type="number" min="0"></div></label>
          <div class="compact-form-grid">
            <label>Inflasi per tahun (%)<input v-model.number="form.inflationRate" type="number" min="0" max="100" step="0.1"></label>
            <label>Return investasi (%)<input v-model.number="form.expectedReturn" type="number" min="0" max="100" step="0.1"></label>
          </div>
          <label>Saldo investasi saat ini<div class="money-input"><span>Rp</span><input v-model.number="form.currentFund" type="number" min="0"></div></label>
          <label>Tabungan investasi per bulan<div class="money-input"><span>Rp</span><input v-model.number="form.monthlyContribution" type="number" min="0"></div></label>
          <label>Withdrawal rate (%)<input v-model.number="form.withdrawalRate" type="number" min="0.1" max="100" step="0.1"></label>
          <button class="primary-button full-button" :disabled="saving"><Save :size="16" />{{ saving ? 'Menghitung...' : 'Hitung & simpan' }}</button>
        </form>
      </article>
      <div class="retirement-results">
        <article class="planning-hero retirement-status" :class="{ shortfall: data.summary?.status === 'shortfall' }">
          <span><Sparkles :size="25" /></span>
          <div><small>Status proyeksi</small><strong>{{ data.summary?.status === 'on_track' ? 'Dana berada di jalur yang tepat' : 'Masih ada kekurangan dana' }}</strong><p>{{ data.summary?.message }}</p></div>
        </article>
        <div class="result-grid">
          <article class="panel result-card"><small>Pengeluaran saat pensiun</small><strong>{{ currency(data.summary?.monthlyExpenseAtRetirement) }}</strong><span>per bulan setelah inflasi</span></article>
          <article class="panel result-card"><small>Target dana pensiun</small><strong>{{ currency(data.summary?.targetFund) }}</strong><span>berdasarkan {{ form.withdrawalRate }}% rule</span></article>
          <article class="panel result-card"><small>Proyeksi dana terkumpul</small><strong>{{ currency(data.summary?.projectedFund) }}</strong><span>pada usia {{ form.retirementAge }}</span></article>
          <article class="panel result-card" :class="{ loss: data.summary?.gap < 0 }"><small>Selisih proyeksi</small><strong>{{ currency(data.summary?.gap) }}</strong><span>{{ data.summary?.gap >= 0 ? 'surplus' : 'kekurangan' }}</span></article>
        </div>
        <article class="panel retirement-progress-card">
          <div class="panel-heading"><div><h2>Progres terhadap target</h2><p>{{ data.summary?.yearsToRetirement }} tahun atau {{ data.summary?.monthsToRetirement }} bulan menuju pensiun.</p></div><strong>{{ Number(data.summary?.progress || 0).toFixed(1) }}%</strong></div>
          <div class="progress-track large"><span :style="{ width: `${progressWidth}%` }" /></div>
          <div class="retirement-recommendation">
            <div><small>Setoran bulanan saat ini</small><strong>{{ currency(form.monthlyContribution) }}</strong></div>
            <span>dibandingkan</span>
            <div><small>Setoran minimum hasil simulasi</small><strong>{{ currency(data.summary?.requiredMonthlyContribution) }}</strong></div>
            <button class="secondary-button" type="button" @click="useRecommendation">Gunakan rekomendasi</button>
          </div>
        </article>
        <p class="planning-footnote"><Info :size="15" /> 4% rule adalah pendekatan kasar, bukan jaminan dana akan bertahan. Evaluasi inflasi, return, dan kebutuhan secara berkala.</p>
      </div>
    </div>
    <article class="panel module-table-card">
      <div class="panel-heading"><div><h2>Proyeksi pertumbuhan</h2><p>Ringkasan tahun pertama, setiap lima tahun, dan tahun terakhir.</p></div></div>
      <div class="retirement-timeline">
        <div v-for="row in projectionMilestones" :key="row.year" class="retirement-point">
          <span>Usia {{ row.age }}</span><strong>{{ shortCurrency(row.endingFund) }}</strong><small>Tahun ke-{{ row.year }} · return {{ currency(row.returnAmount) }}</small>
        </div>
      </div>
    </article>
  </section>
</template>
