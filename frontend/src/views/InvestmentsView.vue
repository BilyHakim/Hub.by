<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ArrowLeft, ChartNoAxesCombined, Pencil, Plus, Save, Trash2, X } from '@lucide/vue'
import ExpenseDonut from '../components/ExpenseDonut.vue'
import { api } from '../services/api'
import MoneyInput from '../components/MoneyInput.vue'

const loading = ref(true)
const saving = ref(false)
const items = ref([])
const editingID = ref(null)
const showForm = ref(false)
const colors = ['#49685c', '#e8a65d', '#7894a0', '#9a8bb7', '#d77268', '#b4a464']
const blank = () => ({ assetType: 'Reksa Dana', name: '', platform: '', purchaseValue: 0, currentValue: 0, targetAllocation: 0 })
const form = reactive(blank())
const currency = (value = 0) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const totalPurchase = computed(() => items.value.reduce((sum, item) => sum + item.purchaseValue, 0))
const totalCurrent = computed(() => items.value.reduce((sum, item) => sum + item.currentValue, 0))
const totalGain = computed(() => totalCurrent.value - totalPurchase.value)
const totalReturn = computed(() => totalPurchase.value ? totalGain.value / totalPurchase.value * 100 : 0)
const distribution = computed(() => {
  const groups = new Map()
  items.value.forEach((item) => groups.set(item.assetType, (groups.get(item.assetType) || 0) + item.currentValue))
  return [...groups].map(([name, value], index) => ({ name, value, color: colors[index % colors.length] }))
})

async function load() {
  loading.value = true
  try { items.value = await api.investments() || [] } finally { loading.value = false }
}
function openCreate() { editingID.value = null; Object.assign(form, blank()); showForm.value = true }
function openEdit(item) {
  editingID.value = item.id
  Object.assign(form, { assetType: item.assetType, name: item.name, platform: item.platform, purchaseValue: item.purchaseValue, currentValue: item.currentValue, targetAllocation: item.targetAllocation })
  showForm.value = true
}
async function save() {
  saving.value = true
  try {
    const payload = { ...form, purchaseValue: Number(form.purchaseValue), currentValue: Number(form.currentValue), targetAllocation: Number(form.targetAllocation) }
    if (editingID.value) await api.updateInvestment(editingID.value, payload)
    else await api.createInvestment(payload)
    showForm.value = false
    await load()
  } finally { saving.value = false }
}
async function remove(item) {
  if (!window.confirm(`Hapus investasi ${item.name}?`)) return
  await api.deleteInvestment(item.id)
  await load()
}
function handleWorkspaceChange() { showForm.value = false; load() }
onMounted(() => {
  load()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page planning-detail-page">
    <RouterLink class="back-link" to="/finance/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div><p class="eyebrow">Portofolio workspace</p><h1>Monitor investasi</h1><p>Pantau nilai pembelian, nilai saat ini, dan performa seluruh aset.</p></div>
      <button class="primary-button" @click="openCreate"><Plus :size="17" />Tambah investasi</button>
    </div>
    <div class="investment-summary" :class="{ shimmer: loading }">
      <article class="panel result-card"><small>Nilai pembelian</small><strong>{{ currency(totalPurchase) }}</strong><span>modal tercatat</span></article>
      <article class="panel result-card"><small>Nilai sekarang</small><strong>{{ currency(totalCurrent) }}</strong><span>market value</span></article>
      <article class="panel result-card" :class="{ loss: totalGain < 0 }"><small>Imbal hasil</small><strong>{{ currency(totalGain) }}</strong><span>{{ totalReturn.toFixed(2) }}%</span></article>
    </div>
    <div class="investment-layout">
      <article class="panel module-table-card">
        <div class="panel-heading"><div><h2>Daftar aset</h2><p>Nilai diperbarui manual sesuai platform investasi.</p></div></div>
        <div v-if="items.length" class="investment-list">
          <div v-for="item in items" :key="item.id" class="investment-row">
            <span class="investment-symbol"><ChartNoAxesCombined :size="18" /></span>
            <div><strong>{{ item.name }}</strong><small>{{ item.assetType }} · {{ item.platform || 'Tanpa platform' }}</small></div>
            <div><small>Nilai beli</small><span>{{ currency(item.purchaseValue) }}</span></div>
            <div><small>Nilai sekarang</small><strong>{{ currency(item.currentValue) }}</strong></div>
            <div :class="item.gain >= 0 ? 'positive' : 'negative'"><small>Imbal hasil</small><strong>{{ currency(item.gain) }} · {{ item.returnPercentage.toFixed(2) }}%</strong></div>
            <button title="Edit" @click="openEdit(item)"><Pencil :size="15" /></button>
            <button class="danger" title="Hapus" @click="remove(item)"><Trash2 :size="15" /></button>
          </div>
        </div>
        <div v-else class="empty-module">Belum ada investasi di workspace ini.</div>
      </article>
      <article class="panel distribution-card">
        <div class="panel-heading"><div><h2>Distribusi portofolio</h2><p>Dikelompokkan berdasarkan jenis aset.</p></div></div>
        <ExpenseDonut :items="distribution" />
        <RouterLink class="secondary-button full-button" to="/finance/modules/rebalancing">Atur target alokasi</RouterLink>
      </article>
    </div>
    <div v-if="showForm" class="modal-backdrop" @click.self="showForm = false">
      <div class="modal investment-modal">
        <button class="modal-close" @click="showForm = false"><X :size="22" /></button>
        <p class="eyebrow">Portofolio</p><h2>{{ editingID ? 'Edit investasi' : 'Tambah investasi' }}</h2>
        <form class="calculator-form" @submit.prevent="save">
          <div class="compact-form-grid">
            <label>Jenis aset<input v-model.trim="form.assetType" required placeholder="Contoh: Saham Indonesia"></label>
            <label>Platform<input v-model.trim="form.platform" placeholder="Contoh: Bibit"></label>
          </div>
          <label>Nama produk/aset<input v-model.trim="form.name" required placeholder="Contoh: BBCA"></label>
          <label>Nilai pembelian<MoneyInput v-model="form.purchaseValue" /></label>
          <label>Nilai sekarang<MoneyInput v-model="form.currentValue" /></label>
          <label>Target alokasi (%)<input v-model.number="form.targetAllocation" type="number" min="0" max="100" step="0.01"></label>
          <button class="primary-button full-button" :disabled="saving"><Save :size="16" />{{ saving ? 'Menyimpan...' : 'Simpan investasi' }}</button>
        </form>
      </div>
    </div>
  </section>
</template>
