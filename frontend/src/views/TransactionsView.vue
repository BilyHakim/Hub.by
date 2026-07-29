<script setup>
import { ref, computed, onBeforeUnmount, onMounted, reactive } from 'vue'
import { Plus, Search, ArrowDownLeft, ArrowUpRight, Trash2, X } from '@lucide/vue'
import { api } from '../services/api'
import { demoTransactions } from '../data/demo'
import EmptyState from '../components/EmptyState.vue'
import MonthPicker from '../components/MonthPicker.vue'
import ResourceManagerModal from '../components/ResourceManagerModal.vue'
import MoneyInput from '../components/MoneyInput.vue'

const transactions = ref([])
const loading = ref(true)
const modalOpen = ref(false)
const query = ref('')
const month = ref(new Date().toISOString().slice(0, 7))
const saving = ref(false)
const managerType = ref(null)
const form = ref({ type: 'expense', categoryId: null, accountId: null, amount: '', description: '', occurredAt: new Date().toISOString().slice(0, 10), isDebtPayment: false })
const categories = reactive({
  expense: [{ id: 3, name: 'Makanan', expenseClass: 'essential' }, { id: 4, name: 'Transportasi', expenseClass: 'essential' }, { id: 5, name: 'Tempat Tinggal', expenseClass: 'essential' }, { id: 6, name: 'Tagihan', expenseClass: 'obligation' }, { id: 7, name: 'Belanja', expenseClass: 'discretionary' }, { id: 8, name: 'Hiburan', expenseClass: 'discretionary' }, { id: 9, name: 'Cicilan', expenseClass: 'obligation' }],
  income: [{ id: 1, name: 'Gaji' }, { id: 2, name: 'Freelance' }],
})
const accounts = ref([{ id: 1, name: 'BCA Utama' }])
let requestSequence = 0

const filtered = computed(() => transactions.value.filter((item) =>
  `${item.description} ${item.category.name} ${item.account.name}`.toLowerCase().includes(query.value.toLowerCase()),
))
const income = computed(() => transactions.value.filter((x) => x.type === 'income').reduce((sum, x) => sum + x.amount, 0))
const expense = computed(() => transactions.value.filter((x) => x.type === 'expense').reduce((sum, x) => sum + x.amount, 0))
const currency = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const dateLabel = (date) => new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(`${date}T00:00:00`))

async function load() {
  const requestID = ++requestSequence
  loading.value = true
  try {
    const result = await api.transactions(month.value)
    if (requestID === requestSequence) transactions.value = result || []
  } catch {
    if (requestID === requestSequence) transactions.value = demoTransactions
  }
  if (requestID === requestSequence) loading.value = false
}
async function loadMetadata() {
  try {
    const [categoryItems, accountItems] = await Promise.all([api.categories(), api.accounts()])
    categories.expense = categoryItems.filter((item) => item.type === 'expense')
    categories.income = categoryItems.filter((item) => item.type === 'income')
    accounts.value = accountItems
  } catch { /* demo mode */ }
  form.value.categoryId = categories[form.value.type][0]?.id || null
  form.value.accountId = accounts.value[0]?.id || null
}
async function initializePeriod() {
  try {
    const setting = await api.financePeriodSetting()
    month.value = setting.currentPeriodLabel
  } catch { /* default to calendar month */ }
  await loadMetadata()
  await load()
}
function openModal() {
  form.value.categoryId = categories[form.value.type][0]?.id || null
  form.value.accountId = accounts.value[0]?.id || null
  modalOpen.value = true
}
function setType(type) {
  form.value.type = type
  form.value.categoryId = categories[type][0]?.id || null
}
async function save() {
  saving.value = true
  try {
    await api.createTransaction({ ...form.value, amount: Number(form.value.amount) })
    modalOpen.value = false
    await load()
  } catch {
    const category = categories[form.value.type].find((x) => x.id === Number(form.value.categoryId))
    const account = accounts.value.find((x) => x.id === Number(form.value.accountId))
    transactions.value.unshift({ id: Date.now(), ...form.value, amount: Number(form.value.amount), category, account })
    modalOpen.value = false
  } finally { saving.value = false }
}
async function remove(id) {
  transactions.value = transactions.value.filter((item) => item.id !== id)
  try { await api.deleteTransaction(id) } catch { /* demo mode */ }
}
onMounted(() => {
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
  initializePeriod()
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))

async function handleWorkspaceChange() {
  managerType.value = null
  transactions.value = []
  await initializePeriod()
}
</script>

<template>
  <section class="page">
    <div class="page-heading compact">
      <div><p class="eyebrow">Arus kas</p><h1>Transaksi</h1><p>Catat setiap rupiah agar keputusan terasa lebih ringan.</p></div>
      <button class="primary-button" @click="openModal"><Plus :size="18" /> Tambah transaksi</button>
    </div>

    <div class="summary-strip">
      <div><span class="summary-icon income"><ArrowDownLeft :size="19" /></span><p>Pemasukan bulan ini<strong>{{ currency(income) }}</strong></p></div>
      <div><span class="summary-icon expense"><ArrowUpRight :size="19" /></span><p>Pengeluaran bulan ini<strong>{{ currency(expense) }}</strong></p></div>
      <div><span class="summary-icon balance">=</span><p>Selisih<strong>{{ currency(income - expense) }}</strong></p></div>
    </div>

    <article class="panel table-panel">
      <div class="table-toolbar">
        <div class="search-box table-search"><Search :size="17" /><input v-model="query" placeholder="Cari transaksi..." /></div>
        <MonthPicker v-model="month" compact @change="load" />
      </div>
      <div v-if="filtered.length" class="transaction-list">
        <div v-for="item in filtered" :key="item.id" class="transaction-row">
          <span class="transaction-icon" :class="item.type">
            <ArrowDownLeft v-if="item.type === 'income'" :size="19" />
            <ArrowUpRight v-else :size="19" />
          </span>
          <div class="transaction-name"><strong>{{ item.description }}</strong><small>{{ item.category.name }} · {{ item.account.name }}</small></div>
          <span class="transaction-date">{{ dateLabel(item.occurredAt) }}</span>
          <strong class="transaction-amount" :class="item.type">{{ item.type === 'income' ? '+' : '−' }} {{ currency(item.amount) }}</strong>
          <button class="icon-button delete-button" aria-label="Hapus transaksi" @click="remove(item.id)"><Trash2 :size="17" /></button>
        </div>
      </div>
      <EmptyState v-else title="Belum ada transaksi" text="Mulai dengan mencatat pemasukan atau pengeluaran pertamamu." />
    </article>

    <Teleport to="body">
      <div v-if="modalOpen" class="modal-backdrop" @click.self="modalOpen = false">
        <form class="modal" @submit.prevent="save">
          <div class="modal-heading"><div><p class="eyebrow">Catatan baru</p><h2>Tambah transaksi</h2></div><button type="button" class="icon-button" @click="modalOpen = false"><X :size="20" /></button></div>
          <div class="type-switch">
            <button type="button" :class="{ active: form.type === 'expense' }" @click="setType('expense')">Pengeluaran</button>
            <button type="button" :class="{ active: form.type === 'income' }" @click="setType('income')">Pemasukan</button>
          </div>
          <label>Nominal <MoneyInput v-model="form.amount" required /></label>
          <div class="form-grid">
            <div class="form-field">
              <div class="field-label-row"><span>Kategori</span><button type="button" @click="managerType = 'category'">Kelola</button></div>
              <select v-model="form.categoryId"><option v-for="category in categories[form.type]" :key="category.id" :value="category.id">{{ category.name }}</option></select>
            </div>
            <div class="form-field">
              <div class="field-label-row"><span>Rekening</span><button type="button" @click="managerType = 'account'">Kelola</button></div>
              <select v-model="form.accountId"><option v-for="account in accounts" :key="account.id" :value="account.id">{{ account.name }}</option></select>
            </div>
          </div>
          <label>Tanggal<input v-model="form.occurredAt" type="date" required /></label>
          <label>Keterangan<input v-model="form.description" placeholder="Contoh: Belanja mingguan" required /></label>
          <label v-if="form.type === 'expense'" class="checkbox"><input v-model="form.isDebtPayment" type="checkbox" /> Ini pembayaran cicilan/kewajiban</label>
          <button class="primary-button full-button" :disabled="saving">{{ saving ? 'Menyimpan...' : 'Simpan transaksi' }}</button>
        </form>
      </div>
      <ResourceManagerModal
        v-if="managerType"
        :type="managerType"
        :items="managerType === 'category' ? [...categories.expense, ...categories.income] : accounts"
        @close="managerType = null"
        @changed="loadMetadata"
      />
    </Teleport>
  </section>
</template>
