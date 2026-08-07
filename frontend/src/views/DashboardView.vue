<script setup>
import { computed, ref, onBeforeUnmount, onMounted } from 'vue'
import {
  WalletCards, TrendingUp, Landmark, PiggyBank,
  ArrowRight, CircleCheck, CircleAlert, Sparkles, Plus, ClipboardList,
} from '@lucide/vue'
import MetricCard from '../components/MetricCard.vue'
import CashflowChart from '../components/CashflowChart.vue'
import ExpenseDonut from '../components/ExpenseDonut.vue'
import MonthPicker from '../components/MonthPicker.vue'
import { api } from '../services/api'
import { demoDashboard } from '../data/demo'

const loading = ref(true)
const usingDemo = ref(false)
const data = ref(demoDashboard)
const month = ref(new Date().toISOString().slice(0, 7))
let requestSequence = 0

const currency = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const compactCurrency = (value) => `Rp${new Intl.NumberFormat('id-ID', { notation: 'compact', maximumFractionDigits: 1 }).format(value)}`
const expenseShare = computed(() => data.value.income > 0 ? data.value.expense / data.value.income * 100 : 0)
const healthyCheckCount = computed(() => data.value.financialCheckup.filter((item) => item.status === 'healthy').length)
const shortDate = (value) => value
  ? new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short' }).format(new Date(`${value}T00:00:00`))
  : ''
const periodLabel = (start, end, fallback) =>
  start && end ? `${shortDate(start)} – ${shortDate(end)}` : fallback
const trendFormula = (current, previous, trend) => previous === 0
  ? 'Persentase belum dapat dihitung karena nilai periode sebelumnya adalah nol.'
  : `(${currency(current)} − ${currency(previous)}) ÷ ${currency(Math.abs(previous))} × 100 = ${Math.abs(trend || 0).toLocaleString('id-ID', { maximumFractionDigits: 1 })}%`
const metricComparisons = computed(() => {
  const currentPeriod = periodLabel(data.value.periodStart, data.value.periodEnd, 'Periode ini')
  const previousPeriod = periodLabel(data.value.previousPeriodStart, data.value.previousPeriodEnd, 'Periode sebelumnya')
  const createPeriodComparison = (title, current, previous, trend, inverse = false) => ({
    title,
    description: 'Dibandingkan dengan satu periode sebelumnya',
    currentLabel: currentPeriod,
    previousLabel: previousPeriod,
    currentValue: current,
    previousValue: previous,
    delta: current - previous,
    inverse,
    formula: trendFormula(current, previous, trend),
  })
  return {
    income: createPeriodComparison('Perubahan pemasukan', data.value.income, data.value.previousIncome, data.value.incomeTrend),
    expense: createPeriodComparison('Perubahan pengeluaran', data.value.expense, data.value.previousExpense, data.value.expenseTrend, true),
    savings: createPeriodComparison('Perubahan uang tersisa', data.value.savings, data.value.previousSavings, data.value.savingsTrend),
    investment: {
      title: 'Imbal hasil investasi',
      description: 'Nilai saat ini dibandingkan dengan modal pembelian',
      currentLabel: 'Nilai investasi saat ini',
      previousLabel: 'Total modal pembelian',
      currentValue: data.value.investmentValue,
      previousValue: data.value.investmentCost,
      delta: data.value.investmentValue - data.value.investmentCost,
      formula: data.value.investmentCost === 0
        ? 'Imbal hasil belum dapat dihitung karena modal pembelian masih nol.'
        : `(${currency(data.value.investmentValue)} − ${currency(data.value.investmentCost)}) ÷ ${currency(data.value.investmentCost)} × 100 = ${Math.abs(data.value.investmentReturn).toLocaleString('id-ID', { maximumFractionDigits: 1 })}%`,
    },
  }
})

async function loadDashboard() {
  const requestID = ++requestSequence
  loading.value = true
  try {
    const result = await api.dashboard(month.value)
    if (requestID !== requestSequence) return
    data.value = {
      ...result,
      cashflow: result.cashflow || [],
      expenseBreakdown: result.expenseBreakdown || [],
      financialCheckup: result.financialCheckup || [],
    }
    usingDemo.value = false
  } catch {
    if (requestID !== requestSequence) return
    data.value = demoDashboard
    usingDemo.value = true
  } finally {
    if (requestID === requestSequence) loading.value = false
  }
}
async function initializePeriod() {
  try {
    const setting = await api.financePeriodSetting()
    month.value = setting.currentPeriodLabel
  } catch { /* default to calendar month */ }
  await loadDashboard()
}

function handleWorkspaceChange() {
  data.value = {
    ...demoDashboard,
    income: 0,
    expense: 0,
    savings: 0,
    previousIncome: 0,
    previousExpense: 0,
    previousSavings: 0,
    incomeTrend: null,
    expenseTrend: null,
    savingsTrend: null,
    savingsRate: 0,
    netWorth: 0,
    emergencyFund: 0,
    emergencyTarget: 0,
    emergencyProgress: 0,
    investmentValue: 0,
    investmentCost: 0,
    investmentReturn: 0,
    cashflow: [],
    expenseBreakdown: [],
    financialCheckup: [],
  }
  initializePeriod()
}

onMounted(() => {
  initializePeriod()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page dashboard-page">
    <div class="page-heading">
      <div>
        <p class="eyebrow">Hubby finance</p>
        <h1>Selamat datang kembali, Bily <span>👋</span></h1>
        <p>Ini cerita keuanganmu bulan ini. Pelan-pelan, yang penting konsisten.</p>
      </div>
      <div class="heading-actions">
        <div class="period-picker-group">
          <MonthPicker v-model="month" @change="loadDashboard" />
          <small v-if="data.periodStart">{{ data.periodStart }} — {{ data.periodEnd }}</small>
        </div>
        <RouterLink class="secondary-button dashboard-plan-shortcut" to="/finance/modules/budget"><ClipboardList :size="18" /> Rencana keuangan</RouterLink>
        <RouterLink class="primary-button" to="/finance/transactions"><Plus :size="18" /> Catat transaksi</RouterLink>
      </div>
    </div>

    <div v-if="usingDemo" class="demo-notice">
      <Sparkles :size="17" />
      <span>Mode pratinjau aktif. Jalankan backend untuk memakai data PostgreSQL.</span>
    </div>

    <div class="metrics-grid" :class="{ shimmer: loading }">
      <MetricCard label="Pemasukan" :value="compactCurrency(data.income)" note="Bulan berjalan" :trend="data.incomeTrend" :comparison="metricComparisons.income" tone="sage">
        <template #icon><WalletCards :size="20" /></template>
      </MetricCard>
      <MetricCard
        label="Pengeluaran"
        :value="compactCurrency(data.expense)"
        :note="`${expenseShare.toFixed(1)}% dari pemasukan`"
        :trend="data.expenseTrend"
        :comparison="metricComparisons.expense"
        inverse-trend
        tone="sand"
      >
        <template #icon><Landmark :size="20" /></template>
      </MetricCard>
      <MetricCard label="Uang tersisa" :value="compactCurrency(data.savings)" :note="`${data.savingsRate.toFixed(1)}% berhasil disimpan`" :trend="data.savingsTrend" :comparison="metricComparisons.savings" tone="moss">
        <template #icon><PiggyBank :size="20" /></template>
      </MetricCard>
      <MetricCard label="Nilai investasi" :value="compactCurrency(data.investmentValue)" :note="`${data.investmentReturn.toFixed(1)}% total imbal hasil`" :trend="data.investmentReturn" :comparison="metricComparisons.investment" trend-hint="Total imbal hasil investasi" tone="lilac">
        <template #icon><TrendingUp :size="20" /></template>
      </MetricCard>
    </div>

    <div class="dashboard-grid">
      <article class="panel cashflow-panel">
        <div class="panel-heading">
          <div><h2>Arus kas</h2><p>Perbandingan 6 bulan terakhir</p></div>
          <RouterLink to="/finance/transactions">Lihat detail <ArrowRight :size="16" /></RouterLink>
        </div>
        <CashflowChart :points="data.cashflow" />
      </article>

      <article class="panel expense-panel">
        <div class="panel-heading">
          <div><h2>Ke mana uangmu pergi?</h2><p>Pengeluaran bulan ini</p></div>
        </div>
        <ExpenseDonut :items="data.expenseBreakdown" />
      </article>

      <article class="panel emergency-card">
        <div class="panel-heading">
          <div><h2>Dana darurat</h2><p>Target 6 bulan pengeluaran</p></div>
          <span class="pill">Prioritas #3</span>
        </div>
        <div class="emergency-total">
          <div>
            <small>Terkumpul</small>
            <strong>{{ currency(data.emergencyFund) }}</strong>
          </div>
          <div class="align-right">
            <small>Target</small>
            <strong>{{ currency(data.emergencyTarget) }}</strong>
          </div>
        </div>
        <div class="progress-track"><span :style="{ width: `${Math.min(data.emergencyProgress, 100)}%` }" /></div>
        <div class="progress-caption">
          <span>{{ data.emergencyProgress.toFixed(0) }}% tercapai</span>
          <span>Kurang {{ currency(Math.max(data.emergencyTarget - data.emergencyFund, 0)) }}</span>
        </div>
        <p class="gentle-note"><Sparkles :size="16" /> Dengan ritme saat ini, targetmu bisa tercapai sekitar 8 bulan lagi.</p>
      </article>

      <article class="panel checkup-card">
        <div class="panel-heading">
          <div><h2>Financial check-up</h2><p>Tiga indikator kesehatan utama</p></div>
          <span class="score-badge">{{ healthyCheckCount }}/{{ data.financialCheckup.length }} sehat</span>
        </div>
        <div class="checkup-list">
          <div v-for="item in data.financialCheckup" :key="item.label" class="checkup-row">
            <span class="status-icon" :class="item.status">
              <CircleCheck v-if="item.status === 'healthy'" :size="19" />
              <CircleAlert v-else :size="19" />
            </span>
            <div><strong>{{ item.label }}</strong><small>{{ item.recommendation }}</small></div>
            <strong class="check-value">{{ item.value.toFixed(1) }}%</strong>
          </div>
        </div>
        <RouterLink class="text-link" to="/finance/modules">Lihat pemeriksaan lengkap <ArrowRight :size="16" /></RouterLink>
      </article>
    </div>
  </section>
</template>
