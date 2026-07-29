<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Calculator, CircleDollarSign, Save, ShieldCheck, Sparkles } from '@lucide/vue'
import MonthPicker from '../components/MonthPicker.vue'
import { api } from '../services/api'

const month = ref(new Date().toISOString().slice(0, 7))
const loading = ref(true)
const saving = ref(false)
const data = ref({ monthlyExpense: 0, observedExpense: 0, targetMonths: 6, targetAmount: 0, currentAmount: 0, remainingAmount: 0, progress: 0 })
const form = reactive({ monthlyExpense: 0, targetMonths: 6 })
const currency = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const projectedTarget = computed(() => Number(form.monthlyExpense || 0) * Number(form.targetMonths || 0))

async function load() {
  loading.value = true
  try {
    data.value = await api.emergencyFund(month.value)
    form.monthlyExpense = data.value.monthlyExpense
    form.targetMonths = data.value.targetMonths
  } finally { loading.value = false }
}
async function save() {
  saving.value = true
  try {
    await api.updateEmergencyFund({ monthlyExpense: Number(form.monthlyExpense), targetMonths: Number(form.targetMonths) })
    await load()
  } finally { saving.value = false }
}
function useObservedExpense() { form.monthlyExpense = data.value.observedExpense }
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
      <div><p class="eyebrow">Bantalan keuangan</p><h1>Target dana darurat</h1><p>Target = pengeluaran bulanan × jumlah bulan perlindungan.</p></div>
      <MonthPicker v-model="month" @change="load" />
    </div>
    <div class="emergency-module-grid" :class="{ shimmer: loading }">
      <article class="panel emergency-config">
        <div class="panel-heading"><div><h2>Atur target</h2><p>Kolom input pada workbook dipindahkan ke formulir ini.</p></div><Calculator :size="21" /></div>
        <form @submit.prevent="save">
          <label>Pengeluaran bulanan
            <div class="money-input"><span>Rp</span><input v-model.number="form.monthlyExpense" type="number" min="0" /></div>
          </label>
          <button type="button" class="observed-button" @click="useObservedExpense">Gunakan pengeluaran bulan ini: {{ currency(data.observedExpense) }}</button>
          <label>Jumlah bulan perlindungan
            <select v-model.number="form.targetMonths"><option v-for="number in 24" :key="number" :value="number">{{ number }} bulan</option></select>
          </label>
          <div class="calculation-preview"><span>Target dana darurat</span><strong>{{ currency(projectedTarget) }}</strong><small>{{ currency(form.monthlyExpense) }} × {{ form.targetMonths }} bulan</small></div>
          <button class="primary-button full-button" :disabled="saving"><Save :size="16" />{{ saving ? 'Menyimpan...' : 'Simpan target' }}</button>
        </form>
      </article>
      <article class="panel emergency-progress-card">
        <div class="emergency-shield"><ShieldCheck :size="31" /></div>
        <small>Kondisi saat ini</small><h2>{{ currency(data.currentAmount) }}</h2>
        <p>dari target {{ currency(data.targetAmount) }}</p>
        <div class="progress-track large"><span :style="{ width: `${Math.min(data.progress,100)}%` }" /></div>
        <div class="progress-caption"><span>{{ data.progress.toFixed(1) }}% tercapai</span><span>{{ currency(data.remainingAmount) }} lagi</span></div>
        <div class="emergency-account-note"><CircleDollarSign :size="18" /><span>Nilai saat ini dijumlahkan dari rekening yang ditandai sebagai <strong>dana darurat</strong>.</span></div>
      </article>
    </div>
    <div class="recommendation-grid">
      <div><strong>3 bulan</strong><span>Single dengan pekerjaan dan pendapatan stabil</span></div>
      <div class="recommended"><strong>6 bulan</strong><span>Menikah atau memiliki tanggungan keluarga</span></div>
      <div><strong>12 bulan</strong><span>Freelancer, pengusaha, atau pendapatan tidak tetap</span></div>
    </div>
    <p class="planning-footnote"><Sparkles :size="15" /> Sesuaikan dengan kondisi dan kebutuhan masing-masing, seperti panduan pada workbook.</p>
  </section>
</template>
