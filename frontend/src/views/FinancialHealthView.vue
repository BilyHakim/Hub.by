<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Activity, ArrowDownRight, ArrowUpRight, CalendarDays, CircleAlert,
  CircleCheck, Landmark, PiggyBank, ReceiptText, Sparkles, WalletCards,
} from '@lucide/vue'
import LifetimeExpenseDonut from '../components/LifetimeExpenseDonut.vue'
import { api } from '../services/api'
import { demoFinancialHealth } from '../data/demo'

const loading = ref(true)
const usingDemo = ref(false)
const data = ref(demoFinancialHealth)
const selectedCategoryId = ref(null)

const currency = (value) => new Intl.NumberFormat('id-ID', {
  style: 'currency', currency: 'IDR', maximumFractionDigits: 0,
}).format(value || 0)
const compactCurrency = (value) => `Rp${new Intl.NumberFormat('id-ID', {
  notation: 'compact', maximumFractionDigits: 1,
}).format(value || 0)}`
const date = (value) => value ? new Intl.DateTimeFormat('id-ID', {
  day: 'numeric', month: 'short', year: 'numeric',
}).format(new Date(`${value}T00:00:00`)) : 'Belum ada transaksi'
const kindLabel = { cash: 'Tunai', bank: 'Bank', ewallet: 'E-wallet', investment: 'Investasi', property: 'Properti', liability: 'Kewajiban' }

const selectedCategory = computed(() => data.value.expenseCategories.find((item) => item.id === selectedCategoryId.value) || data.value.expenseCategories[0])
const topCategory = computed(() => data.value.expenseCategories[0])
const positiveAssets = computed(() => data.value.accounts.filter((item) => item.kind !== 'liability' && item.balance > 0))
const healthChecks = computed(() => [
  { label: 'Arus kas sepanjang masa positif', ok: data.value.lifetimeSavings >= 0, value: compactCurrency(data.value.lifetimeSavings) },
  { label: 'Rasio simpan minimal 20%', ok: data.value.savingsRate >= 20, value: `${data.value.savingsRate.toFixed(1)}%` },
  { label: 'Dana darurat mencapai target', ok: data.value.emergencyProgress >= 100, value: `${data.value.emergencyProgress.toFixed(0)}%` },
  { label: 'Kekayaan bersih positif', ok: data.value.netWorth >= 0, value: compactCurrency(data.value.netWorth) },
])
const healthyCount = computed(() => healthChecks.value.filter((item) => item.ok).length)
const healthScore = computed(() => Math.round(healthyCount.value / healthChecks.value.length * 100))

async function load() {
  loading.value = true
  try {
    data.value = await api.financialHealth()
    usingDemo.value = false
  } catch {
    data.value = demoFinancialHealth
    usingDemo.value = true
  } finally {
    selectedCategoryId.value = data.value.expenseCategories[0]?.id ?? null
    loading.value = false
  }
}
function handleWorkspaceChange() { load() }
onMounted(() => {
  load()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
  window.addEventListener('hubby:transactions-updated', load)
})
onBeforeUnmount(() => {
  window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange)
  window.removeEventListener('hubby:transactions-updated', load)
})
</script>

<template>
  <section class="page financial-health-page" :class="{ shimmer: loading }">
    <div class="page-heading health-heading">
      <div>
        <p class="eyebrow">Gambaran sepanjang masa</p>
        <h1>Kesehatan keuangan</h1>
        <p>Seluruh angka dihitung sejak transaksi pertama, tanpa batasan periode.</p>
      </div>
      <div class="lifetime-range"><CalendarDays :size="17" /><span><small>Rentang data</small><strong>{{ date(data.firstTransaction) }} — {{ date(data.lastTransaction) }}</strong></span></div>
    </div>

    <div v-if="usingDemo" class="demo-notice"><Sparkles :size="17" /><span>Mode pratinjau aktif. Jalankan backend untuk memakai data PostgreSQL.</span></div>

    <div class="health-metrics">
      <article class="health-metric balance"><span><WalletCards :size="20" /></span><div><small>Total saldo saat ini</small><strong>{{ compactCurrency(data.totalBalance) }}</strong><p>Kas, bank, dan e-wallet</p></div></article>
      <article class="health-metric"><span><ArrowDownRight :size="20" /></span><div><small>Total pemasukan</small><strong>{{ compactCurrency(data.lifetimeIncome) }}</strong><p>{{ data.transactionCount }} transaksi tercatat</p></div></article>
      <article class="health-metric expense"><span><ArrowUpRight :size="20" /></span><div><small>Total pengeluaran</small><strong>{{ compactCurrency(data.lifetimeExpense) }}</strong><p>Rata-rata {{ compactCurrency(data.averageMonthlyExpense) }}/bulan</p></div></article>
      <article class="health-metric"><span><PiggyBank :size="20" /></span><div><small>Uang tersisa</small><strong>{{ compactCurrency(data.lifetimeSavings) }}</strong><p>Rasio simpan {{ data.savingsRate.toFixed(1) }}%</p></div></article>
    </div>

    <div class="health-overview-grid">
      <article class="panel health-score-card">
        <div class="panel-heading"><div><h2>Skor kesehatan</h2><p>Empat fondasi utama kondisi keuanganmu</p></div><span class="health-score-pill">{{ healthyCount }}/4 sehat</span></div>
        <div class="health-score-main"><div class="health-score-ring" :style="{ '--score': `${healthScore * 3.6}deg` }"><span><strong>{{ healthScore }}</strong><small>/100</small></span></div><div><strong>{{ healthScore >= 75 ? 'Fondasi keuanganmu kuat' : healthScore >= 50 ? 'Sudah di jalur yang baik' : 'Ada fondasi yang perlu diperkuat' }}</strong><p>Gunakan indikator di bawah sebagai prioritas perbaikan berikutnya.</p></div></div>
        <div class="health-check-list">
          <div v-for="item in healthChecks" :key="item.label"><span :class="item.ok ? 'healthy' : 'attention'"><CircleCheck v-if="item.ok" :size="17" /><CircleAlert v-else :size="17" /></span><p>{{ item.label }}</p><strong>{{ item.value }}</strong></div>
        </div>
      </article>

      <article class="panel wealth-card">
        <div class="panel-heading"><div><h2>Posisi kekayaan</h2><p>Aset dikurangi kewajiban saat ini</p></div><Landmark :size="20" /></div>
        <strong class="wealth-total">{{ currency(data.netWorth) }}</strong>
        <div class="wealth-comparison"><div><small>Total aset</small><strong>{{ compactCurrency(data.totalAssets) }}</strong></div><span>−</span><div><small>Total kewajiban</small><strong>{{ compactCurrency(data.totalLiabilities) }}</strong></div></div>
        <div class="account-list">
          <div v-for="account in positiveAssets" :key="account.id"><span><i :class="{ emergency: account.isEmergencyFund }" /><span><strong>{{ account.name }}</strong><small>{{ kindLabel[account.kind] || account.kind }}<template v-if="account.isEmergencyFund"> · Dana darurat</template></small></span></span><strong>{{ compactCurrency(account.balance) }}</strong></div>
        </div>
      </article>
    </div>

    <div class="health-expense-grid">
      <article class="panel category-explorer">
        <div class="panel-heading"><div><h2>Ke mana uangmu paling banyak pergi?</h2><p>Klik bagian donut atau nama kategori untuk melihat detail</p></div><span v-if="topCategory" class="top-category-badge">#1 {{ topCategory.name }}</span></div>
        <LifetimeExpenseDonut v-if="data.expenseCategories.length" v-model:selected-id="selectedCategoryId" :items="data.expenseCategories" />
        <div v-else class="health-empty"><ReceiptText :size="25" /><strong>Belum ada pengeluaran</strong><p>Kategori akan muncul otomatis setelah transaksi dicatat.</p></div>
      </article>

      <article class="panel category-insight">
        <template v-if="selectedCategory">
          <div class="category-insight-title"><span :style="{ background: selectedCategory.color }" /><div><small>Detail kategori</small><h2>{{ selectedCategory.name }}</h2></div></div>
          <strong class="category-total">{{ currency(selectedCategory.value) }}</strong>
          <p class="category-share">{{ selectedCategory.share.toFixed(1) }}% dari seluruh pengeluaranmu</p>
          <div class="category-stat-grid"><div><small>Jumlah transaksi</small><strong>{{ selectedCategory.transactionCount }}</strong></div><div><small>Rata-rata transaksi</small><strong>{{ compactCurrency(selectedCategory.averageTransaction) }}</strong></div><div><small>Pengeluaran terbesar</small><strong>{{ compactCurrency(selectedCategory.largestAmount) }}</strong></div><div><small>Terakhir digunakan</small><strong>{{ date(selectedCategory.lastSpentAt) }}</strong></div></div>
          <div class="largest-expense-note"><Sparkles :size="17" /><p>Transaksi terbesar di kategori ini adalah <strong>{{ selectedCategory.largestDescription }}</strong> senilai {{ currency(selectedCategory.largestAmount) }}.</p></div>
        </template>
        <template v-else><Activity :size="28" /><p>Pilih kategori untuk melihat insight.</p></template>
      </article>
    </div>
  </section>
</template>
