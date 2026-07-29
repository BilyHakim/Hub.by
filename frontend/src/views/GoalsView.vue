<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  ArrowRight, CalendarDays, CheckCircle2, CircleAlert, Clock, GraduationCap,
  Home, Pencil, Plane, Plus, Save, Target, Trash2, TrendingUp, X,
} from '@lucide/vue'
import { api } from '../services/api'
import MoneyInput from '../components/MoneyInput.vue'

const goals = ref([])
const loading = ref(true)
const saving = ref(false)
const showForm = ref(false)
const editingID = ref(null)
const error = ref('')
const icons = { plane: Plane, home: Home, 'graduation-cap': GraduationCap, target: Target }
const statusLabels = { completed: 'Tercapai', on_track: 'Sesuai rencana', at_risk: 'Perlu penyesuaian', overdue: 'Melewati target' }
const blankForm = () => {
  const date = new Date()
  date.setFullYear(date.getFullYear() + 1)
  return { name: '', targetAmount: 0, currentAmount: 0, monthlyContribution: 0, targetDate: date.toISOString().slice(0, 10), icon: 'target', expectedReturn: 5 }
}
const form = reactive(blankForm())
const currency = (value = 0) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const dateLabel = (date) => new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric' }).format(new Date(`${date}T00:00:00`))
const totalTarget = computed(() => goals.value.reduce((sum, goal) => sum + goal.targetAmount, 0))
const totalCurrent = computed(() => goals.value.reduce((sum, goal) => sum + goal.currentAmount, 0))
const overallProgress = computed(() => totalTarget.value ? totalCurrent.value / totalTarget.value * 100 : 0)
let requestSequence = 0

async function loadGoals() {
  const requestID = ++requestSequence
  loading.value = true
  error.value = ''
  try {
    const result = await api.goals()
    if (requestID === requestSequence) goals.value = result || []
  } catch (loadError) {
    if (requestID === requestSequence) {
      goals.value = []
      error.value = loadError.message
    }
  } finally {
    if (requestID === requestSequence) loading.value = false
  }
}
function openCreate() {
  editingID.value = null
  Object.assign(form, blankForm())
  showForm.value = true
}
function openEdit(goal) {
  editingID.value = goal.id
  Object.assign(form, {
    name: goal.name, targetAmount: goal.targetAmount, currentAmount: goal.currentAmount,
    monthlyContribution: goal.monthlyContribution, targetDate: goal.targetDate,
    icon: goal.icon || 'target', expectedReturn: goal.expectedReturn,
  })
  showForm.value = true
}
async function save() {
  saving.value = true
  error.value = ''
  const payload = {
    ...form, targetAmount: Number(form.targetAmount), currentAmount: Number(form.currentAmount),
    monthlyContribution: Number(form.monthlyContribution), expectedReturn: Number(form.expectedReturn),
  }
  try {
    if (editingID.value) await api.replaceGoal(editingID.value, payload)
    else await api.createGoal(payload)
    showForm.value = false
    await loadGoals()
  } catch (saveError) {
    error.value = saveError.message
  } finally { saving.value = false }
}
async function remove(goal) {
  if (!window.confirm(`Hapus tujuan “${goal.name}”?`)) return
  try {
    await api.deleteGoal(goal.id)
    await loadGoals()
  } catch (deleteError) { error.value = deleteError.message }
}
function estimationLabel(months) {
  if (months < 0) return 'Belum dapat diproyeksikan'
  if (months === 0) return 'Sudah tercapai'
  const years = Math.floor(months / 12)
  const rest = months % 12
  return [years ? `${years} tahun` : '', rest ? `${rest} bulan` : ''].filter(Boolean).join(' ')
}
function handleWorkspaceChange() {
  goals.value = []
  showForm.value = false
  loadGoals()
}
onMounted(() => {
  loadGoals()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page">
    <div class="page-heading compact">
      <div><p class="eyebrow">Rencana masa depan</p><h1>Tujuan keuangan</h1><p>Ubah mimpi besar menjadi langkah kecil yang bisa dilihat.</p></div>
      <button class="primary-button" @click="openCreate"><Plus :size="18" /> Buat tujuan</button>
    </div>
    <p v-if="error" class="form-error goal-page-error">{{ error }}</p>
    <div class="goal-highlight" :class="{ shimmer: loading }">
      <span><Target :size="28" /></span>
      <div><small>Total targetmu</small><strong>{{ currency(totalTarget) }}</strong><p>Dari {{ goals.length }} tujuan aktif</p></div>
      <div class="goal-overall-progress">
        <strong>{{ overallProgress.toFixed(1) }}%</strong>
        <div class="progress-track"><span :style="{ width: `${Math.min(overallProgress, 100)}%` }" /></div>
        <small>{{ currency(totalCurrent) }} sudah terkumpul</small>
      </div>
    </div>
    <div v-if="goals.length" class="goals-grid">
      <article v-for="goal in goals" :key="goal.id" class="panel goal-card">
        <div class="goal-top">
          <span class="goal-icon"><component :is="icons[goal.icon] || Target" :size="22" /></span>
          <span class="goal-status" :class="`status-${goal.status}`">{{ statusLabels[goal.status] }}</span>
        </div>
        <h2>{{ goal.name }}</h2>
        <p><CalendarDays :size="15" /> Target {{ dateLabel(goal.targetDate) }}</p>
        <div class="goal-numbers"><strong>{{ currency(goal.currentAmount) }}</strong><span>dari {{ currency(goal.targetAmount) }}</span></div>
        <div class="progress-track"><span :style="{ width: `${Math.min(goal.progress, 100)}%` }" /></div>
        <div class="progress-caption"><span>{{ goal.progress.toFixed(0) }}% terkumpul</span><span>{{ currency(goal.remainingAmount) }} lagi</span></div>
        <div class="goal-plan-details">
          <div><TrendingUp :size="15" /><span><small>Tabungan bulanan</small><strong>{{ currency(goal.monthlyContribution) }}</strong></span></div>
          <div><Clock :size="15" /><span><small>Estimasi dengan investasi</small><strong>{{ estimationLabel(goal.estimatedMonthsWithInvestment) }}</strong></span></div>
          <div :class="{ warning: goal.status === 'at_risk' || goal.status === 'overdue' }">
            <component :is="goal.status === 'completed' || goal.status === 'on_track' ? CheckCircle2 : CircleAlert" :size="15" />
            <span><small>Setoran agar sesuai target</small><strong>{{ currency(goal.requiredMonthlyContribution) }}/bulan</strong></span>
          </div>
        </div>
        <div class="goal-actions">
          <button class="secondary-button" @click="openEdit(goal)">Edit rencana <Pencil :size="15" /></button>
          <button class="goal-delete" title="Hapus tujuan" @click="remove(goal)"><Trash2 :size="16" /></button>
        </div>
      </article>
    </div>
    <div v-else-if="!loading" class="panel goal-empty">
      <span><Target :size="26" /></span><h2>Belum ada tujuan keuangan</h2>
      <p>Buat tujuan pertama untuk mulai menghitung target dan tabungan bulanannya.</p>
      <button class="primary-button" @click="openCreate"><Plus :size="16" />Buat tujuan</button>
    </div>
    <div v-if="showForm" class="modal-backdrop" @click.self="showForm = false">
      <div class="modal goal-modal">
        <button class="modal-close" @click="showForm = false"><X :size="21" /></button>
        <p class="eyebrow">Rencana masa depan</p><h2>{{ editingID ? 'Edit tujuan' : 'Buat tujuan baru' }}</h2>
        <p class="modal-description">Isi target, kondisi saat ini, dan kemampuan menabung seperti pada workbook.</p>
        <form class="calculator-form" @submit.prevent="save">
          <label>Nama tujuan<input v-model.trim="form.name" required maxlength="100" placeholder="Contoh: DP rumah"></label>
          <div class="compact-form-grid">
            <label>Ikon<select v-model="form.icon"><option value="target">Target</option><option value="home">Rumah</option><option value="plane">Perjalanan</option><option value="graduation-cap">Pendidikan</option></select></label>
            <label>Tanggal target<input v-model="form.targetDate" type="date" required></label>
          </div>
          <label>Target dana<MoneyInput v-model="form.targetAmount" required /></label>
          <label>Dana yang sudah terkumpul<MoneyInput v-model="form.currentAmount" /></label>
          <label>Rencana tabungan per bulan<MoneyInput v-model="form.monthlyContribution" /></label>
          <label>Estimasi imbal hasil per tahun (%)<input v-model.number="form.expectedReturn" type="number" min="0" max="100" step="0.1"></label>
          <button class="primary-button full-button" :disabled="saving"><Save :size="16" />{{ saving ? 'Menyimpan...' : 'Simpan tujuan' }}</button>
        </form>
      </div>
    </div>
  </section>
</template>
