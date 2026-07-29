<script setup>
import { ref, computed, onMounted } from 'vue'
import { Plus, Search, ArrowDownLeft, ArrowUpRight, Trash2, X, CalendarDays } from '@lucide/vue'
import { api } from '../services/api'
import { demoTransactions } from '../data/demo'
import EmptyState from '../components/EmptyState.vue'
import MonthPicker from '../components/MonthPicker.vue'

const transactions = ref([])
const loading = ref(true)
const modalOpen = ref(false)
const query = ref('')
const month = ref(new Date().toISOString().slice(0, 7))
const saving = ref(false)
const form = ref({ type: 'expense', categoryId: 3, accountId: 1, amount: '', description: '', occurredAt: new Date().toISOString().slice(0, 10), isDebtPayment: false })
const categories = {
  expense: [{ id: 3, name: 'Makanan' }, { id: 4, name: 'Transportasi' }, { id: 5, name: 'Tempat Tinggal' }, { id: 6, name: 'Tagihan' }, { id: 7, name: 'Belanja' }, { id: 8, name: 'Hiburan' }, { id: 9, name: 'Cicilan' }],
  income: [{ id: 1, name: 'Gaji' }, { id: 2, name: 'Freelance' }],
}

const filtered = computed(() => transactions.value.filter((item) =>
  `${item.description} ${item.category.name} ${item.account.name}`.toLowerCase().includes(query.value.toLowerCase()),
))
const income = computed(() => transactions.value.filter((x) => x.type === 'income').reduce((sum, x) => sum + x.amount, 0))
const expense = computed(() => transactions.value.filter((x) => x.type === 'expense').reduce((sum, x) => sum + x.amount, 0))
const currency = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const dateLabel = (date) => new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(`${date}T00:00:00`))

async function load() {
  loading.value = true
  try { transactions.value = await api.transactions(month.value) } catch { transactions.value = demoTransactions }
  loading.value = false
}
function setType(type) {
  form.value.type = type
  form.value.categoryId = categories[type][0].id
}
async function save() {
  saving.value = true
  try {
    await api.createTransaction({ ...form.value, amount: Number(form.value.amount) })
    modalOpen.value = false
    await load()
  } catch {
    const category = categories[form.value.type].find((x) => x.id === Number(form.value.categoryId))
    transactions.value.unshift({ id: Date.now(), ...form.value, amount: Number(form.value.amount), category, account: { id: 1, name: 'BCA Utama' } })
    modalOpen.value = false
  } finally { saving.value = false }
}
async function remove(id) {
  transactions.value = transactions.value.filter((item) => item.id !== id)
  try { await api.deleteTransaction(id) } catch { /* demo mode */ }
}
onMounted(load)
</script>

<template>
  <section class="page">
    <div class="page-heading compact">
      <div><p class="eyebrow">Arus kas</p><h1>Transaksi</h1><p>Catat setiap rupiah agar keputusan terasa lebih ringan.</p></div>
      <button class="primary-button" @click="modalOpen = true"><Plus :size="18" /> Tambah transaksi</button>
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
          <label>Nominal <div class="money-input"><span>Rp</span><input v-model="form.amount" inputmode="numeric" type="number" min="1" placeholder="0" required /></div></label>
          <div class="form-grid">
            <label>Kategori<select v-model="form.categoryId"><option v-for="category in categories[form.type]" :key="category.id" :value="category.id">{{ category.name }}</option></select></label>
            <label>Tanggal<input v-model="form.occurredAt" type="date" required /></label>
          </div>
          <label>Keterangan<input v-model="form.description" placeholder="Contoh: Belanja mingguan" required /></label>
          <label v-if="form.type === 'expense'" class="checkbox"><input v-model="form.isDebtPayment" type="checkbox" /> Ini pembayaran cicilan/kewajiban</label>
          <button class="primary-button full-button" :disabled="saving">{{ saving ? 'Menyimpan...' : 'Simpan transaksi' }}</button>
        </form>
      </div>
    </Teleport>
  </section>
</template>
