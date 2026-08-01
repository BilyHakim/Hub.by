<script setup>
import { ref, computed, onBeforeUnmount, onMounted, reactive } from 'vue'
import { Plus, Search, ArrowDownLeft, ArrowUpRight, ArrowLeftRight, WalletCards, Trash2, X } from '@lucide/vue'
import { api } from '../services/api'
import { demoTransactions } from '../data/demo'
import EmptyState from '../components/EmptyState.vue'
import MonthPicker from '../components/MonthPicker.vue'
import ResourceManagerModal from '../components/ResourceManagerModal.vue'
import MoneyInput from '../components/MoneyInput.vue'

const transactions = ref([])
const loading = ref(true)
const modalOpen = ref(false)
const transferModalOpen = ref(false)
const query = ref('')
const month = ref(new Date().toISOString().slice(0, 7))
const saving = ref(false)
const savingTransfer = ref(false)
const transferError = ref('')
const managerType = ref(null)
const form = ref({ type: 'expense', categoryId: null, accountId: null, amount: '', description: '', occurredAt: new Date().toISOString().slice(0, 10), isDebtPayment: false })
const transferForm = ref({ sourceAccountId: null, destinationAccountId: null, amount: '', description: '', occurredAt: new Date().toISOString().slice(0, 10) })
const categories = reactive({
  expense: [{ id: 3, name: 'Makanan', expenseClass: 'essential' }, { id: 4, name: 'Transportasi', expenseClass: 'essential' }, { id: 5, name: 'Tempat Tinggal', expenseClass: 'essential' }, { id: 6, name: 'Tagihan', expenseClass: 'obligation' }, { id: 7, name: 'Belanja', expenseClass: 'discretionary' }, { id: 8, name: 'Hiburan', expenseClass: 'discretionary' }, { id: 9, name: 'Cicilan', expenseClass: 'obligation' }],
  income: [{ id: 1, name: 'Gaji' }, { id: 2, name: 'Freelance' }],
})
const accounts = ref([{ id: 1, name: 'BCA Utama', kind: 'bank', balance: 0 }])
const accountKindLabels = {
  bank: 'Rekening bank', cash: 'Tunai', ewallet: 'E-wallet', investment: 'Investasi',
  property: 'Properti', liability: 'Kewajiban/utang',
}
let requestSequence = 0

const filtered = computed(() => transactions.value.filter((item) =>
  `${item.description} ${item.category?.name || ''} ${item.account?.name || ''} ${item.destinationAccount?.name || ''}`.toLowerCase().includes(query.value.toLowerCase()),
))
const income = computed(() => transactions.value.filter((x) => x.type === 'income').reduce((sum, x) => sum + x.amount, 0))
const expense = computed(() => transactions.value.filter((x) => x.type === 'expense').reduce((sum, x) => sum + x.amount, 0))
const selectedSourceAccount = computed(() => accounts.value.find((account) => account.id === Number(transferForm.value.sourceAccountId)))
const selectedDestinationAccount = computed(() => accounts.value.find((account) => account.id === Number(transferForm.value.destinationAccountId)))
const canSaveTransfer = computed(() => {
  const amount = Number(transferForm.value.amount)
  return !savingTransfer.value && transferForm.value.sourceAccountId && transferForm.value.destinationAccountId &&
    transferForm.value.sourceAccountId !== transferForm.value.destinationAccountId && amount > 0 &&
    amount <= (selectedSourceAccount.value?.balance ?? -1)
})
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
function openTransferModal() {
  transferError.value = ''
  transferForm.value = {
    sourceAccountId: accounts.value[0]?.id || null,
    destinationAccountId: accounts.value.find((account) => account.id !== accounts.value[0]?.id)?.id || null,
    amount: '',
    description: '',
    occurredAt: new Date().toISOString().slice(0, 10),
  }
  transferModalOpen.value = true
}
function swapTransferAccounts() {
  const source = transferForm.value.sourceAccountId
  transferForm.value.sourceAccountId = transferForm.value.destinationAccountId
  transferForm.value.destinationAccountId = source
  transferError.value = ''
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
    await Promise.all([load(), loadMetadata()])
  } catch {
    const category = categories[form.value.type].find((x) => x.id === Number(form.value.categoryId))
    const account = accounts.value.find((x) => x.id === Number(form.value.accountId))
    transactions.value.unshift({ id: Date.now(), ...form.value, amount: Number(form.value.amount), category, account })
    modalOpen.value = false
  } finally { saving.value = false }
}
async function saveTransfer() {
  if (!canSaveTransfer.value) return
  savingTransfer.value = true
  transferError.value = ''
  try {
    await api.createTransfer({ ...transferForm.value, amount: Number(transferForm.value.amount) })
    transferModalOpen.value = false
    await Promise.all([load(), loadMetadata()])
  } catch (error) {
    transferError.value = error.message
  } finally { savingTransfer.value = false }
}
async function remove(id) {
  try {
    await api.deleteTransaction(id)
    await Promise.all([load(), loadMetadata()])
  } catch {
    transactions.value = transactions.value.filter((item) => item.id !== id)
  }
}
async function removeItem(item) {
  if (item.type !== 'transfer') {
    await remove(item.id)
    return
  }
  try {
    await api.deleteTransfer(item.id)
    await Promise.all([load(), loadMetadata()])
  } catch {
    transactions.value = transactions.value.filter((entry) => !(entry.type === 'transfer' && entry.id === item.id))
  }
}
async function handleTransactionsUpdated() {
  await Promise.all([load(), loadMetadata()])
}
onMounted(() => {
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
  window.addEventListener('hubby:transactions-updated', handleTransactionsUpdated)
  initializePeriod()
})
onBeforeUnmount(() => {
  window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange)
  window.removeEventListener('hubby:transactions-updated', handleTransactionsUpdated)
})

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
      <div class="heading-actions">
        <button class="secondary-button" :disabled="accounts.length < 2" @click="openTransferModal"><ArrowLeftRight :size="18" /> Transfer</button>
        <button class="primary-button" @click="openModal"><Plus :size="18" /> Tambah transaksi</button>
      </div>
    </div>

    <div class="summary-strip">
      <div><span class="summary-icon income"><ArrowDownLeft :size="19" /></span><p>Pemasukan bulan ini<strong>{{ currency(income) }}</strong></p></div>
      <div><span class="summary-icon expense"><ArrowUpRight :size="19" /></span><p>Pengeluaran bulan ini<strong>{{ currency(expense) }}</strong></p></div>
      <div><span class="summary-icon balance">=</span><p>Selisih<strong>{{ currency(income - expense) }}</strong></p></div>
    </div>

    <article class="panel account-balance-panel">
      <div class="account-balance-heading">
        <div><p class="eyebrow">Posisi dana</p><h2>Saldo rekening</h2></div>
        <button type="button" class="text-link account-manage-button" @click="managerType = 'account'">Kelola rekening</button>
      </div>
      <div class="account-balance-grid">
        <div v-for="account in accounts" :key="account.id" class="account-balance-card">
          <span><WalletCards :size="18" /></span>
          <div>
            <small>{{ accountKindLabels[account.kind] || account.kind }}<template v-if="account.isEmergencyFund"> · Dana darurat</template></small>
            <strong>{{ account.name }}</strong>
          </div>
          <b :class="{ negative: account.balance < 0 }">{{ currency(account.balance) }}</b>
        </div>
      </div>
    </article>

    <article class="panel table-panel">
      <div class="table-toolbar">
        <div class="search-box table-search"><Search :size="17" /><input v-model="query" placeholder="Cari transaksi..." /></div>
        <MonthPicker v-model="month" compact @change="load" />
      </div>
      <div v-if="filtered.length" class="transaction-list">
        <div v-for="item in filtered" :key="`${item.type}-${item.id}`" class="transaction-row">
          <span class="transaction-icon" :class="item.type">
            <ArrowDownLeft v-if="item.type === 'income'" :size="19" />
            <ArrowLeftRight v-else-if="item.type === 'transfer'" :size="19" />
            <ArrowUpRight v-else :size="19" />
          </span>
          <div class="transaction-name">
            <strong>{{ item.description }}</strong>
            <small v-if="item.type === 'transfer'">{{ item.account.name }} → {{ item.destinationAccount.name }}</small>
            <small v-else>{{ item.category.name }} · {{ item.account.name }}</small>
          </div>
          <span class="transaction-date">{{ dateLabel(item.occurredAt) }}</span>
          <strong class="transaction-amount" :class="item.type">{{ item.type === 'income' ? '+' : item.type === 'expense' ? '−' : '' }} {{ currency(item.amount) }}</strong>
          <button class="icon-button delete-button" :aria-label="item.type === 'transfer' ? 'Hapus transfer' : 'Hapus transaksi'" @click="removeItem(item)"><Trash2 :size="17" /></button>
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
      <div v-if="transferModalOpen" class="modal-backdrop" @click.self="transferModalOpen = false">
        <form class="modal transfer-modal" @submit.prevent="saveTransfer">
          <div class="modal-heading"><div><p class="eyebrow">Pemindahan dana</p><h2>Transfer antar rekening</h2></div><button type="button" class="icon-button" @click="transferModalOpen = false"><X :size="20" /></button></div>
          <p class="modal-description">Saldo rekening asal akan berkurang dan saldo rekening tujuan akan bertambah dengan nominal yang sama.</p>
          <div class="transfer-account-grid">
            <div class="form-field">
              <div class="field-label-row"><span>Dari rekening</span><small v-if="selectedSourceAccount">Saldo {{ currency(selectedSourceAccount.balance) }}</small></div>
              <select v-model="transferForm.sourceAccountId" required @change="transferError = ''">
                <option v-for="account in accounts" :key="account.id" :value="account.id" :disabled="account.id === transferForm.destinationAccountId">{{ account.name }}</option>
              </select>
            </div>
            <button type="button" class="transfer-swap-button" aria-label="Tukar rekening" @click="swapTransferAccounts"><ArrowLeftRight :size="17" /></button>
            <div class="form-field">
              <div class="field-label-row"><span>Ke rekening</span><small v-if="selectedDestinationAccount">Saldo {{ currency(selectedDestinationAccount.balance) }}</small></div>
              <select v-model="transferForm.destinationAccountId" required @change="transferError = ''">
                <option v-for="account in accounts" :key="account.id" :value="account.id" :disabled="account.id === transferForm.sourceAccountId">{{ account.name }}</option>
              </select>
            </div>
          </div>
          <label>Nominal <MoneyInput v-model="transferForm.amount" required /></label>
          <label>Tanggal<input v-model="transferForm.occurredAt" type="date" required /></label>
          <label>Keterangan<input v-model.trim="transferForm.description" placeholder="Contoh: Pindah dana tabungan" /></label>
          <p v-if="Number(transferForm.amount) > (selectedSourceAccount?.balance ?? 0)" class="form-error">Saldo rekening asal tidak mencukupi.</p>
          <p v-else-if="transferError" class="form-error">{{ transferError }}</p>
          <button class="primary-button full-button" :disabled="!canSaveTransfer">{{ savingTransfer ? 'Memindahkan...' : 'Transfer dana' }}</button>
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
