<script setup>
import { computed, reactive, ref } from 'vue'
import { Pencil, Plus, Tags, Trash2, WalletCards, X } from '@lucide/vue'
import { api } from '../services/api'

const props = defineProps({
  type: { type: String, required: true },
  items: { type: Array, default: () => [] },
})
const emit = defineEmits(['close', 'changed'])

const editingID = ref(null)
const saving = ref(false)
const errorMessage = ref('')
const form = reactive({
  name: '',
  categoryType: 'expense',
  expenseClass: 'essential',
  kind: 'bank',
  balance: 0,
  isEmergencyFund: false,
})
const isCategory = computed(() => props.type === 'category')
const title = computed(() => isCategory.value ? 'Kelola kategori' : 'Kelola rekening')
const accountKinds = [
  { value: 'bank', label: 'Rekening bank' },
  { value: 'cash', label: 'Tunai' },
  { value: 'ewallet', label: 'E-wallet' },
  { value: 'investment', label: 'Investasi' },
  { value: 'property', label: 'Properti' },
  { value: 'liability', label: 'Kewajiban/utang' },
]
const expenseClasses = [
  { value: 'essential', label: 'Kebutuhan sehari-hari' },
  { value: 'obligation', label: 'Kewajiban dan tagihan' },
  { value: 'discretionary', label: 'Keinginan' },
  { value: 'future', label: 'Masa depan/investasi' },
]
const expenseClassLabels = Object.fromEntries(expenseClasses.map((item) => [item.value, item.label]))
const currency = (value) => new Intl.NumberFormat('id-ID', {
  style: 'currency', currency: 'IDR', maximumFractionDigits: 0,
}).format(value)

function resetForm() {
  editingID.value = null
  errorMessage.value = ''
  Object.assign(form, {
    name: '',
    categoryType: 'expense',
    expenseClass: 'essential',
    kind: 'bank',
    balance: 0,
    isEmergencyFund: false,
  })
}
function edit(item) {
  editingID.value = item.id
  errorMessage.value = ''
  Object.assign(form, {
    name: item.name,
    categoryType: item.type || 'expense',
    expenseClass: item.expenseClass || 'essential',
    kind: item.kind || 'bank',
    balance: item.balance || 0,
    isEmergencyFund: item.isEmergencyFund || false,
  })
}
async function save() {
  saving.value = true
  errorMessage.value = ''
  try {
    if (isCategory.value) {
      const payload = {
        name: form.name,
        type: form.categoryType,
        expenseClass: form.categoryType === 'expense' ? form.expenseClass : null,
      }
      if (editingID.value) await api.updateCategory(editingID.value, payload)
      else await api.createCategory(payload)
    } else {
      const payload = {
        name: form.name,
        kind: form.kind,
        balance: Number(form.balance),
        isEmergencyFund: form.isEmergencyFund,
      }
      if (editingID.value) await api.updateAccount(editingID.value, payload)
      else await api.createAccount(payload)
    }
    emit('changed')
    resetForm()
  } catch (error) {
    errorMessage.value = error.message
  } finally {
    saving.value = false
  }
}
async function remove(item) {
  const confirmed = window.confirm(`Hapus ${isCategory.value ? 'kategori' : 'rekening'} “${item.name}”?`)
  if (!confirmed) return
  errorMessage.value = ''
  try {
    if (isCategory.value) await api.deleteCategory(item.id)
    else await api.deleteAccount(item.id)
    if (editingID.value === item.id) resetForm()
    emit('changed')
  } catch (error) {
    errorMessage.value = error.message
  }
}
</script>

<template>
  <div class="modal-backdrop resource-manager-backdrop" @click.self="emit('close')">
    <div class="modal resource-manager-modal">
      <div class="modal-heading">
        <div><p class="eyebrow">Workspace aktif</p><h2>{{ title }}</h2></div>
        <button type="button" class="icon-button" aria-label="Tutup" @click="emit('close')"><X :size="20" /></button>
      </div>
      <p class="modal-description">Perubahan hanya berlaku untuk ruang keuangan yang sedang aktif.</p>

      <div class="resource-layout">
        <div class="resource-list">
          <div v-if="!items.length" class="resource-empty">Belum ada data.</div>
          <div v-for="item in items" :key="item.id" class="resource-row" :class="{ active: editingID === item.id }">
            <span class="resource-row-icon">
              <Tags v-if="isCategory" :size="17" />
              <WalletCards v-else :size="17" />
            </span>
            <div>
              <strong>{{ item.name }}</strong>
              <small v-if="isCategory">{{ item.type === 'income' ? 'Pemasukan' : expenseClassLabels[item.expenseClass] || 'Pengeluaran' }}</small>
              <small v-else>{{ currency(item.balance) }} · {{ item.kind }}</small>
            </div>
            <button type="button" aria-label="Edit" @click="edit(item)"><Pencil :size="15" /></button>
            <button type="button" class="danger" aria-label="Hapus" @click="remove(item)"><Trash2 :size="15" /></button>
          </div>
        </div>

        <form class="resource-form" @submit.prevent="save">
          <div class="resource-form-title">
            <strong>{{ editingID ? 'Edit data' : `Tambah ${isCategory ? 'kategori' : 'rekening'}` }}</strong>
            <button v-if="editingID" type="button" @click="resetForm">Batal edit</button>
          </div>
          <label>Nama<input v-model="form.name" minlength="2" maxlength="80" required :placeholder="isCategory ? 'Contoh: Kesehatan' : 'Contoh: Bank Mandiri'" /></label>
          <label v-if="isCategory">Jenis
            <select v-model="form.categoryType">
              <option value="expense">Pengeluaran</option>
              <option value="income">Pemasukan</option>
            </select>
          </label>
          <label v-if="isCategory && form.categoryType === 'expense'">Kelompok pengeluaran
            <select v-model="form.expenseClass">
              <option v-for="expenseClass in expenseClasses" :key="expenseClass.value" :value="expenseClass.value">{{ expenseClass.label }}</option>
            </select>
            <small class="field-help">Kebutuhan dan kewajiban masuk rekomendasi dana darurat.</small>
          </label>
          <template v-else>
            <label>Jenis rekening
              <select v-model="form.kind"><option v-for="kind in accountKinds" :key="kind.value" :value="kind.value">{{ kind.label }}</option></select>
            </label>
            <label>Saldo saat ini<input v-model.number="form.balance" type="number" step="1" /></label>
            <label class="checkbox"><input v-model="form.isEmergencyFund" type="checkbox" /> Tandai sebagai dana darurat</label>
          </template>
          <p v-if="errorMessage" class="form-error">{{ errorMessage }}</p>
          <button class="primary-button full-button" :disabled="saving">
            <Plus v-if="!editingID" :size="16" />
            {{ saving ? 'Menyimpan...' : editingID ? 'Simpan perubahan' : 'Tambahkan' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>
