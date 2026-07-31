<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  ArrowLeft, BadgeCheck, CalendarDays, CreditCard, HandCoins, Pencil,
  Plus, ReceiptText, RotateCcw, Save, Trash2, WalletCards, X,
} from '@lucide/vue'
import MoneyInput from '../components/MoneyInput.vue'
import { api } from '../services/api'

const loading = ref(true)
const saving = ref(false)
const items = ref([])
const filter = ref('debt')
const showForm = ref(false)
const paymentItem = ref(null)
const editingID = ref(null)
const today = () => new Date().toISOString().slice(0, 10)
const blank = () => ({ type: filter.value, name: '', platform: '', originalAmount: 0, installmentCount: 3, startDate: today(), notes: '' })
const form = reactive(blank())
const payment = reactive({ amount: 0, paidAt: today(), notes: '' })
const currency = (value = 0) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const filtered = computed(() => items.value.filter((item) => item.type === filter.value))
const debts = computed(() => items.value.filter((item) => item.type === 'debt'))
const receivables = computed(() => items.value.filter((item) => item.type === 'receivable'))
const total = (list, field) => list.reduce((sum, item) => sum + Number(item[field] || 0), 0)
const totalDebt = computed(() => total(debts.value, 'remainingAmount'))
const totalReceivable = computed(() => total(receivables.value, 'remainingAmount'))
const completedCount = computed(() => items.value.filter((item) => item.remainingAmount === 0).length)

async function load() {
  loading.value = true
  try { items.value = await api.obligations() || [] } finally { loading.value = false }
}
function openCreate() {
  editingID.value = null
  Object.assign(form, blank())
  showForm.value = true
}
function openEdit(item) {
  editingID.value = item.id
  Object.assign(form, {
    type: item.type, name: item.name, platform: item.platform,
    originalAmount: item.originalAmount, installmentCount: item.installmentCount,
    startDate: item.startDate, notes: item.notes,
  })
  showForm.value = true
}
async function save() {
  saving.value = true
  try {
    const payload = { ...form, originalAmount: Number(form.originalAmount), installmentCount: Number(form.installmentCount) }
    if (editingID.value) await api.updateObligation(editingID.value, payload)
    else await api.createObligation(payload)
    showForm.value = false
    await load()
  } finally { saving.value = false }
}
function openPayment(item) {
  paymentItem.value = item
  payment.amount = Math.min(item.expectedInstallment, item.remainingAmount)
  payment.paidAt = today()
  payment.notes = ''
}
async function savePayment() {
  saving.value = true
  try {
    await api.createObligationPayment(paymentItem.value.id, { ...payment, amount: Number(payment.amount) })
    paymentItem.value = null
    await load()
  } finally { saving.value = false }
}
async function undoLast(item) {
  if (!item.lastPaymentId || !window.confirm(`Batalkan pembayaran terakhir untuk ${item.name}?`)) return
  await api.deleteObligationPayment(item.lastPaymentId)
  await load()
}
async function remove(item) {
  if (!window.confirm(`Hapus ${item.type === 'debt' ? 'utang' : 'piutang'} ${item.name} beserta histori pembayarannya?`)) return
  await api.deleteObligation(item.id)
  await load()
}
function switchFilter(value) {
  filter.value = value
}
function handleWorkspaceChange() {
  showForm.value = false
  paymentItem.value = null
  load()
}
onMounted(() => {
  load()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page planning-detail-page obligations-page">
    <RouterLink class="back-link" to="/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div><p class="eyebrow">Komitmen keuangan</p><h1>Utang & piutang</h1><p>Pantau paylater, cicilan, pinjaman, dan uang yang akan diterima dari berbagai pihak.</p></div>
      <button class="primary-button" @click="openCreate"><Plus :size="17" />Tambah catatan</button>
    </div>

    <div class="obligation-summary" :class="{ shimmer: loading }">
      <article class="panel"><span class="debt"><CreditCard :size="20" /></span><div><small>Sisa seluruh utang</small><strong>{{ currency(totalDebt) }}</strong><em>{{ debts.length }} kontrak</em></div></article>
      <article class="panel"><span class="receivable"><HandCoins :size="20" /></span><div><small>Piutang belum diterima</small><strong>{{ currency(totalReceivable) }}</strong><em>{{ receivables.length }} catatan</em></div></article>
      <article class="panel"><span class="done"><BadgeCheck :size="20" /></span><div><small>Sudah lunas</small><strong>{{ completedCount }}</strong><em>dari {{ items.length }} catatan</em></div></article>
    </div>

    <div class="obligation-tabs">
      <button :class="{ active: filter === 'debt' }" @click="switchFilter('debt')">Utang <span>{{ debts.length }}</span></button>
      <button :class="{ active: filter === 'receivable' }" @click="switchFilter('receivable')">Piutang <span>{{ receivables.length }}</span></button>
    </div>

    <div class="obligation-grid" :class="{ shimmer: loading }">
      <article v-for="item in filtered" :key="item.id" class="panel obligation-card" :class="{ completed: item.remainingAmount === 0 }">
        <div class="obligation-heading">
          <span :class="item.type"><CreditCard v-if="item.type === 'debt'" :size="19" /><HandCoins v-else :size="19" /></span>
          <div><h2>{{ item.name }}</h2><p>{{ item.platform || (item.type === 'debt' ? 'Tanpa platform' : 'Tanpa nama pihak') }}</p></div>
          <div class="obligation-actions">
            <button title="Edit" @click="openEdit(item)"><Pencil :size="14" /></button>
            <button class="danger" title="Hapus" @click="remove(item)"><Trash2 :size="14" /></button>
          </div>
        </div>
        <div class="obligation-amounts">
          <div><small>Sudah {{ item.type === 'debt' ? 'dibayar' : 'diterima' }}</small><strong>{{ currency(item.paidAmount) }}</strong></div>
          <div><small>Sisa</small><strong>{{ currency(item.remainingAmount) }}</strong></div>
        </div>
        <div class="progress-track"><span :style="{ width: `${Math.min(item.progress, 100)}%` }" /></div>
        <div class="obligation-progress-label"><span>{{ item.progress.toFixed(1) }}%</span><span>{{ item.paymentCount }}/{{ item.installmentCount }} pembayaran</span></div>
        <div class="obligation-meta">
          <span><ReceiptText :size="14" />{{ currency(item.expectedInstallment) }} / cicilan</span>
          <span><CalendarDays :size="14" />Mulai {{ item.startDate }}</span>
        </div>
        <div class="obligation-card-footer">
          <button v-if="item.remainingAmount > 0" class="primary-button" @click="openPayment(item)">
            <WalletCards :size="15" />Catat {{ item.type === 'debt' ? 'pembayaran' : 'penerimaan' }}
          </button>
          <span v-else class="obligation-paid"><BadgeCheck :size="16" />Lunas</span>
          <button v-if="item.lastPaymentId" class="undo-payment" title="Batalkan pembayaran terakhir" @click="undoLast(item)"><RotateCcw :size="14" />Urungkan terakhir</button>
        </div>
      </article>
      <div v-if="!loading && !filtered.length" class="panel empty-module obligation-empty">
        Belum ada {{ filter === 'debt' ? 'utang' : 'piutang' }} yang dicatat.
      </div>
    </div>

    <div v-if="showForm" class="modal-backdrop" @click.self="showForm = false">
      <div class="modal obligation-modal">
        <button class="modal-close" @click="showForm = false"><X :size="22" /></button>
        <p class="eyebrow">Utang & piutang</p><h2>{{ editingID ? 'Edit catatan' : 'Tambah catatan' }}</h2>
        <form class="calculator-form" @submit.prevent="save">
          <div class="type-switch">
            <button type="button" :class="{ active: form.type === 'debt' }" @click="form.type = 'debt'">Utang</button>
            <button type="button" :class="{ active: form.type === 'receivable' }" @click="form.type = 'receivable'">Piutang</button>
          </div>
          <div class="compact-form-grid">
            <label>Nama<input v-model.trim="form.name" required placeholder="Contoh: PayLater laptop"></label>
            <label>Platform / pihak<input v-model.trim="form.platform" placeholder="Contoh: ShopeePayLater"></label>
          </div>
          <label>Nilai awal<MoneyInput v-model="form.originalAmount" /></label>
          <div class="compact-form-grid">
            <label>Jumlah cicilan<input v-model.number="form.installmentCount" type="number" min="1" max="360" required></label>
            <label>Tanggal mulai<input v-model="form.startDate" type="date" required></label>
          </div>
          <div class="tenor-shortcuts"><span>Tenor cepat:</span><button v-for="tenor in [1,3,6,9,12]" :key="tenor" type="button" @click="form.installmentCount = tenor">{{ tenor }}×</button></div>
          <label>Catatan<input v-model.trim="form.notes" placeholder="Opsional"></label>
          <button class="primary-button full-button" :disabled="saving"><Save :size="16" />{{ saving ? 'Menyimpan...' : 'Simpan catatan' }}</button>
        </form>
      </div>
    </div>

    <div v-if="paymentItem" class="modal-backdrop" @click.self="paymentItem = null">
      <div class="modal payment-modal">
        <button class="modal-close" @click="paymentItem = null"><X :size="22" /></button>
        <p class="eyebrow">{{ paymentItem.type === 'debt' ? 'Pembayaran cicilan' : 'Penerimaan piutang' }}</p>
        <h2>{{ paymentItem.name }}</h2>
        <p class="payment-caption">Sisa saat ini {{ currency(paymentItem.remainingAmount) }}</p>
        <form class="calculator-form" @submit.prevent="savePayment">
          <label>Nominal<MoneyInput v-model="payment.amount" /></label>
          <label>Tanggal<input v-model="payment.paidAt" type="date" required></label>
          <label>Catatan<input v-model.trim="payment.notes" placeholder="Contoh: Cicilan ke-2"></label>
          <button class="primary-button full-button" :disabled="saving || payment.amount <= 0"><Save :size="16" />{{ saving ? 'Menyimpan...' : 'Simpan pembayaran' }}</button>
        </form>
      </div>
    </div>
  </section>
</template>
