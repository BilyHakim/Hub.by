<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  WalletCards, TrendingUp, Landmark, PiggyBank,
  ArrowRight, CircleCheck, CircleAlert, Sparkles, Plus,
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

const currency = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const compactCurrency = (value) => `Rp${new Intl.NumberFormat('id-ID', { notation: 'compact', maximumFractionDigits: 1 }).format(value)}`

async function loadDashboard() {
  loading.value = true
  try {
    data.value = await api.dashboard(month.value)
    usingDemo.value = false
  } catch {
    data.value = demoDashboard
    usingDemo.value = true
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
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
        <MonthPicker v-model="month" @change="loadDashboard" />
        <RouterLink class="primary-button" to="/transactions"><Plus :size="18" /> Catat transaksi</RouterLink>
      </div>
    </div>

    <div v-if="usingDemo" class="demo-notice">
      <Sparkles :size="17" />
      <span>Mode pratinjau aktif. Jalankan backend untuk memakai data PostgreSQL.</span>
    </div>

    <div class="metrics-grid" :class="{ shimmer: loading }">
      <MetricCard label="Pemasukan" :value="compactCurrency(data.income)" note="Bulan berjalan" :trend="8.4" tone="sage">
        <template #icon><WalletCards :size="20" /></template>
      </MetricCard>
      <MetricCard label="Pengeluaran" :value="compactCurrency(data.expense)" note="59% dari pemasukan" :trend="-3.1" tone="sand">
        <template #icon><Landmark :size="20" /></template>
      </MetricCard>
      <MetricCard label="Uang tersisa" :value="compactCurrency(data.savings)" :note="`${data.savingsRate.toFixed(1)}% berhasil disimpan`" :trend="14.2" tone="moss">
        <template #icon><PiggyBank :size="20" /></template>
      </MetricCard>
      <MetricCard label="Nilai investasi" :value="compactCurrency(data.investmentValue)" :note="`${data.investmentReturn.toFixed(1)}% total imbal hasil`" :trend="data.investmentReturn" tone="lilac">
        <template #icon><TrendingUp :size="20" /></template>
      </MetricCard>
    </div>

    <div class="dashboard-grid">
      <article class="panel cashflow-panel">
        <div class="panel-heading">
          <div><h2>Arus kas</h2><p>Perbandingan 6 bulan terakhir</p></div>
          <RouterLink to="/transactions">Lihat detail <ArrowRight :size="16" /></RouterLink>
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
          <span class="score-badge">2/3 sehat</span>
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
        <RouterLink class="text-link" to="/modules">Lihat pemeriksaan lengkap <ArrowRight :size="16" /></RouterLink>
      </article>
    </div>
  </section>
</template>
