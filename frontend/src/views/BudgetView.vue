<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  ArrowLeft, BadgeCheck, CalendarDays, CircleAlert, Equal, Eye, ReceiptText, Save,
  Sparkles, Target, TrendingDown, WalletCards, X,
} from '@lucide/vue'
import MonthPicker from '../components/MonthPicker.vue'
import MoneyInput from '../components/MoneyInput.vue'
import { api } from '../services/api'

const month = ref(new Date().toISOString().slice(0, 7))
const loadedMonth = ref(month.value)
const loading = ref(true)
const saving = ref(false)
const dirty = ref(false)
const error = ref('')
const saved = ref(false)
const transactions = ref([])
const selectedCategoryId = ref(null)
const transactionError = ref('')
const detailOpen = ref(false)
const data = ref({
  month: month.value, periodStart: '', periodEnd: '',
  planned: 0, actual: 0, remaining: 0, items: [],
})
const plans = ref({})

const currency = (value) => new Intl.NumberFormat('id-ID', {
  style: 'currency', currency: 'IDR', maximumFractionDigits: 0,
}).format(Number(value || 0))
const totalPlanned = computed(() => Object.values(plans.value).reduce((sum, value) => sum + Number(value || 0), 0))
const totalActual = computed(() => data.value.items.reduce((sum, item) => sum + Number(item.actual || 0), 0))
const totalRemaining = computed(() => totalPlanned.value - totalActual.value)
const usage = computed(() => totalPlanned.value > 0 ? totalActual.value / totalPlanned.value * 100 : 0)
const selectedCategory = computed(() => data.value.items.find((item) => item.categoryId === selectedCategoryId.value) || null)
const selectedTransactions = computed(() => transactions.value.filter((item) =>
  item.type === 'expense' && Number(item.category?.id) === Number(selectedCategoryId.value),
))
const dateLabel = (date) => new Intl.DateTimeFormat('id-ID', {
  day: 'numeric', month: 'short', year: 'numeric',
}).format(new Date(`${date}T00:00:00`))

function actualStatus(item) {
  const planned = Number(plans.value[item.categoryId] || 0)
  if (planned === 0) return item.actual > 0 ? 'unplanned' : 'empty'
  return item.actual > planned ? 'over' : item.actual === planned ? 'exact' : 'safe'
}
function rowRemaining(item) {
  return Number(plans.value[item.categoryId] || 0) - Number(item.actual || 0)
}
function rowUsage(item) {
  const planned = Number(plans.value[item.categoryId] || 0)
  if (planned <= 0) return item.actual > 0 ? 100 : 0
  return Math.min(item.actual / planned * 100, 100)
}
function updatePlan(categoryId, value) {
  plans.value = { ...plans.value, [categoryId]: Number(value || 0) }
  dirty.value = true
  saved.value = false
}
function openDetail(categoryId) {
  selectedCategoryId.value = categoryId
  detailOpen.value = true
}
function closeDetail() {
  detailOpen.value = false
}
function handleKeydown(event) {
  if (event.key === 'Escape' && detailOpen.value) closeDetail()
}

async function load(targetMonth = month.value) {
  loading.value = true
  error.value = ''
  saved.value = false
  transactionError.value = ''
  try {
    const result = await api.budget(targetMonth)
    data.value = result
    plans.value = Object.fromEntries(result.items.map((item) => [item.categoryId, item.planned]))
    month.value = result.month
    loadedMonth.value = result.month
    if (!result.items.some((item) => item.categoryId === selectedCategoryId.value)) {
      selectedCategoryId.value = result.items[0]?.categoryId || null
    }
    dirty.value = false
    try {
      transactions.value = await api.transactions(result.month) || []
    } catch {
      transactions.value = []
      transactionError.value = 'Rincian transaksi belum dapat dimuat.'
    }
  } catch (loadError) {
    error.value = loadError.message
  } finally {
    loading.value = false
  }
}
async function initializePeriod() {
  try {
    const setting = await api.financePeriodSetting()
    month.value = setting.currentPeriodLabel
  } catch { /* use calendar month */ }
  await load(month.value)
}
async function changeMonth(value) {
  if (dirty.value && !window.confirm('Rencana yang belum disimpan akan hilang. Tetap pindah periode?')) {
    month.value = loadedMonth.value
    return
  }
  await load(value)
}
async function save() {
  saving.value = true
  error.value = ''
  saved.value = false
  try {
    const items = data.value.items.map((item) => ({
      categoryId: item.categoryId,
      plannedAmount: Number(plans.value[item.categoryId] || 0),
    }))
    const result = await api.updateBudget(month.value, items)
    data.value = result
    plans.value = Object.fromEntries(result.items.map((item) => [item.categoryId, item.planned]))
    dirty.value = false
    saved.value = true
  } catch (saveError) {
    error.value = saveError.message
  } finally {
    saving.value = false
  }
}
function handleWorkspaceChange() { initializePeriod() }
function handleTransactionsUpdated() {
  if (!dirty.value) load(month.value)
}
function beforeUnload(event) {
  if (!dirty.value) return
  event.preventDefault()
}
onMounted(() => {
  initializePeriod()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
  window.addEventListener('hubby:transactions-updated', handleTransactionsUpdated)
  window.addEventListener('beforeunload', beforeUnload)
  window.addEventListener('keydown', handleKeydown)
})
onBeforeUnmount(() => {
  window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange)
  window.removeEventListener('hubby:transactions-updated', handleTransactionsUpdated)
  window.removeEventListener('beforeunload', beforeUnload)
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <section class="page planning-detail-page budget-page">
    <RouterLink class="back-link" to="/finance/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div>
        <p class="eyebrow">Anggaran bulanan</p>
        <h1>Rencana vs pengeluaran sebenarnya</h1>
        <p>Isi hanya kolom rencana. Realisasi dihitung otomatis dari transaksi pada kategori dan periode yang sama.</p>
      </div>
      <div class="period-picker-group">
        <MonthPicker v-model="month" @change="changeMonth" />
        <small v-if="data.periodStart">{{ data.periodStart }} — {{ data.periodEnd }}</small>
      </div>
    </div>

    <div v-if="error" class="inline-alert error budget-alert"><CircleAlert :size="17" />{{ error }}</div>
    <div v-if="saved" class="inline-alert success budget-alert"><BadgeCheck :size="17" />Rencana pengeluaran berhasil disimpan.</div>

    <div class="budget-summary-grid" :class="{ shimmer: loading }">
      <article class="panel">
        <span class="budget-summary-icon planned"><Target :size="20" /></span>
        <div><small>Total rencana</small><strong>{{ currency(totalPlanned) }}</strong></div>
      </article>
      <article class="panel">
        <span class="budget-summary-icon actual"><ReceiptText :size="20" /></span>
        <div><small>Pengeluaran sebenarnya</small><strong>{{ currency(totalActual) }}</strong></div>
      </article>
      <article class="panel" :class="{ 'budget-over': totalRemaining < 0 }">
        <span class="budget-summary-icon remaining"><WalletCards :size="20" /></span>
        <div><small>{{ totalRemaining < 0 ? 'Melebihi rencana' : 'Sisa anggaran' }}</small><strong>{{ currency(Math.abs(totalRemaining)) }}</strong></div>
      </article>
      <article class="panel">
        <span class="budget-summary-icon usage"><TrendingDown :size="20" /></span>
        <div><small>Anggaran terpakai</small><strong>{{ usage.toFixed(1) }}%</strong></div>
      </article>
    </div>

    <article class="panel budget-table-panel" :class="{ shimmer: loading }">
      <div class="panel-heading budget-table-heading">
        <div><h2>Rincian per kategori</h2><p>Nilai sebenarnya akan berubah otomatis saat transaksi ditambah atau dihapus.</p></div>
        <button class="primary-button" type="button" :disabled="saving || loading || !dirty" @click="save">
          <Save :size="16" />{{ saving ? 'Menyimpan...' : dirty ? 'Simpan rencana' : 'Tersimpan' }}
        </button>
      </div>

      <div class="budget-table-scroll">
        <div class="budget-table-head">
          <span>Kategori</span><span>Rencana</span><span>Sebenarnya</span><span>Sisa / kekurangan</span><span>Detail</span>
        </div>
        <div v-if="!loading && data.items.length === 0" class="budget-empty">
          <ReceiptText :size="28" />
          <strong>Belum ada kategori pengeluaran</strong>
          <p>Tambahkan kategori melalui halaman Arus kas, lalu kembali ke halaman ini.</p>
        </div>
        <div
          v-for="item in data.items"
          :key="item.categoryId"
          class="budget-row"
          :class="{ selected: detailOpen && selectedCategoryId === item.categoryId }"
        >
          <div class="budget-category">
            <span :style="{ background: item.color }" />
            <div><strong>{{ item.categoryName }}</strong><small>Pengeluaran kategori ini</small></div>
          </div>
          <MoneyInput
            :model-value="plans[item.categoryId] || 0"
            :aria-label="`Rencana ${item.categoryName}`"
            @update:model-value="updatePlan(item.categoryId, $event)"
          />
          <div class="budget-actual">
            <strong>{{ currency(item.actual) }}</strong>
            <div class="budget-progress"><span :class="actualStatus(item)" :style="{ width: `${rowUsage(item)}%` }" /></div>
          </div>
          <div class="budget-difference" :class="actualStatus(item)">
            <component :is="actualStatus(item) === 'exact' ? Equal : actualStatus(item) === 'over' || actualStatus(item) === 'unplanned' ? CircleAlert : BadgeCheck" :size="16" />
            <div>
              <strong>{{ rowRemaining(item) < 0 ? '-' : '' }}{{ currency(Math.abs(rowRemaining(item))) }}</strong>
              <small v-if="actualStatus(item) === 'over'">Melebihi rencana</small>
              <small v-else-if="actualStatus(item) === 'unplanned'">Belum direncanakan</small>
              <small v-else-if="actualStatus(item) === 'exact'">Tepat sesuai rencana</small>
              <small v-else>Sisa anggaran</small>
            </div>
          </div>
          <button class="budget-detail-button" type="button" @click="openDetail(item.categoryId)"><Eye :size="15" /> Lihat detail</button>
        </div>
        <div v-if="data.items.length" class="budget-total-row">
          <strong>Total pengeluaran</strong>
          <strong>{{ currency(totalPlanned) }}</strong>
          <strong>{{ currency(totalActual) }}</strong>
          <strong :class="{ negative: totalRemaining < 0 }">{{ totalRemaining < 0 ? '-' : '' }}{{ currency(Math.abs(totalRemaining)) }}</strong>
          <span aria-hidden="true" />
        </div>
      </div>
    </article>

    <p class="planning-footnote"><Sparkles :size="15" /> Realisasi memakai tanggal periode keuangan aktif, jadi tetap mengikuti pengaturan tanggal gajian Anda.</p>

    <Teleport to="body">
      <div v-if="detailOpen && selectedCategory" class="modal-backdrop" @click.self="closeDetail">
        <article class="modal budget-detail-modal" role="dialog" aria-modal="true" :aria-label="`Detail transaksi ${selectedCategory.categoryName}`">
          <button class="modal-close" type="button" aria-label="Tutup detail" @click="closeDetail"><X :size="18" /></button>
          <div class="budget-detail-heading">
            <p class="eyebrow">Rincian kategori</p>
            <h2>Transaksi {{ selectedCategory.categoryName }}</h2>
            <p>Pengeluaran pada periode {{ data.periodStart }} — {{ data.periodEnd }}.</p>
            <span class="budget-transaction-count">{{ selectedTransactions.length }} transaksi · {{ currency(selectedCategory.actual) }}</span>
          </div>
          <div v-if="transactionError" class="budget-transaction-empty"><CircleAlert :size="22" /><span>{{ transactionError }}</span></div>
          <div v-else-if="selectedTransactions.length" class="budget-transaction-list">
            <div v-for="transaction in selectedTransactions" :key="transaction.id" class="budget-transaction-row">
              <span class="budget-transaction-icon" :style="{ color: selectedCategory.color, background: `${selectedCategory.color}1a` }"><ReceiptText :size="17" /></span>
              <div class="budget-transaction-name">
                <strong>{{ transaction.description || selectedCategory.categoryName }}</strong>
                <small>{{ transaction.account?.name || 'Rekening tidak diketahui' }}</small>
              </div>
              <span class="budget-transaction-date"><CalendarDays :size="14" />{{ dateLabel(transaction.occurredAt) }}</span>
              <strong class="budget-transaction-amount">{{ currency(transaction.amount) }}</strong>
            </div>
          </div>
          <div v-else class="budget-transaction-empty">
            <ReceiptText :size="24" />
            <div><strong>Belum ada transaksi {{ selectedCategory.categoryName }}</strong><span>Realisasi kategori ini masih kosong pada periode terpilih.</span></div>
          </div>
        </article>
      </div>
    </Teleport>
  </section>
</template>
