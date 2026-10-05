<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, CalendarDays, Check, CircleAlert, History, Package, Pencil, Plus, Search, Trash2 } from '@lucide/vue'
import { api } from '../services/api'
import MetricCard from '../components/MetricCard.vue'
import EmptyState from '../components/EmptyState.vue'
import MoneyInput from '../components/MoneyInput.vue'
import MaintenanceDialog from '../components/MaintenanceDialog.vue'
import { formatMaintenanceDate as date, intervalLabels, itemStatus, localToday, optionalNumber, statusLabels, statusRank, usageDuration, usageLabels } from '../utils/maintenance'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const formError = ref('')
const toast = ref('')
const modal = ref('')
const editingId = ref(null)
const selectedRule = ref(null)
const search = ref('')
const filter = ref('all')
const data = ref({ items: [], rules: [], histories: [], categories: [], today: localToday() })
const itemForm = reactive({})
const ruleForm = reactive({})
const completionForm = reactive({})
let requestSequence = 0
let toastTimer
const item = computed(() => data.value.items.find(entry => entry.id === Number(route.params.id)))
const isDetail = computed(() => route.name === 'maintenance-detail')
const isHistory = computed(() => route.name === 'maintenance-history')
const isDashboard = computed(() => route.name === 'maintenance')
const rules = computed(() => data.value.rules.filter(rule => rule.itemId === item.value?.id))
const histories = computed(() => data.value.histories.filter(entry => !isDetail.value || entry.itemId === item.value?.id))
const status = entry => itemStatus(data.value.rules.filter(rule => rule.itemId === entry.id))
const number = value => new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(value)
const currency = value => value === null || value === undefined ? 'Belum dicatat' : new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const filteredItems = computed(() => data.value.items.filter(entry => (filter.value === 'all' || status(entry) === filter.value) && `${entry.name} ${entry.category} ${entry.brand} ${entry.location}`.toLocaleLowerCase('id-ID').includes(search.value.toLocaleLowerCase('id-ID'))))
const filteredHistory = computed(() => histories.value.filter(entry => `${entry.itemName} ${entry.ruleName} ${entry.vendor} ${entry.notes}`.toLocaleLowerCase('id-ID').includes(search.value.toLocaleLowerCase('id-ID'))))
const attention = computed(() => data.value.rules.filter(rule => rule.active && ['overdue', 'due_today'].includes(rule.status)).sort((a, b) => statusRank[b.status] - statusRank[a.status]))
const upcoming = computed(() => data.value.rules.filter(rule => rule.active && ['due_soon', 'usage_unknown', 'good'].includes(rule.status)).sort((a, b) => statusRank[b.status] - statusRank[a.status] || (a.nextDueDate || '9999').localeCompare(b.nextDueDate || '9999')))
const metrics = computed(() => [
  { label: 'Total barang', value: data.value.items.length, icon: Package, tone: 'sage' },
  { label: 'Jadwal terlambat', value: data.value.rules.filter(rule => rule.status === 'overdue').length, icon: CircleAlert, tone: 'rose' },
  { label: 'Segera / hari ini', value: data.value.rules.filter(rule => ['due_soon', 'due_today'].includes(rule.status)).length, icon: CalendarDays, tone: 'sand' },
  { label: 'Barang kondisi baik', value: data.value.items.filter(entry => status(entry) === 'good').length, icon: Check, tone: 'sage' },
])
const textFields = [ ['brand', 'Merek'], ['model', 'Model'], ['serialNumber', 'Nomor seri'], ['location', 'Lokasi'] ]
const dateFields = [ ['purchaseDate', 'Tanggal pembelian'], ['startUsageDate', 'Mulai digunakan'], ['warrantyExpiry', 'Garansi berakhir'] ]
const suggestedCategories = computed(() => [...new Set([...data.value.categories, 'Home Appliance', 'Vehicle', 'Electronics', 'Personal Care', 'Household', 'Other'])])
const itemName = id => data.value.items.find(entry => entry.id === id)?.name || ''
const unit = id => usageLabels[data.value.items.find(entry => entry.id === id)?.usageUnit] || 'km'
const interval = rule => [rule.intervalValue ? `Setiap ${rule.intervalValue} ${intervalLabels[rule.intervalUnit]}` : '', rule.usageInterval ? `Setiap ${number(rule.usageInterval)} ${unit(rule.itemId)}` : ''].filter(Boolean).join(' atau ')
function due(rule) {
  const parts = []
  if (rule.nextDueDate) {
    const days = Math.round((new Date(`${rule.nextDueDate}T00:00:00Z`) - new Date(`${data.value.today}T00:00:00Z`)) / 86400000)
    parts.push(`${date(rule.nextDueDate)} (${days === 0 ? 'hari ini' : days < 0 ? `terlambat ${-days} hari` : `${days} hari lagi`})`)
  }
  if (rule.nextDueUsage !== null && rule.nextDueUsage !== undefined) {
    const current = data.value.items.find(entry => entry.id === rule.itemId)?.currentUsage
    parts.push(`${number(rule.nextDueUsage)} ${unit(rule.itemId)}${current === null || current === undefined ? ' (penggunaan belum dicatat)' : ` (${number(Math.abs(rule.nextDueUsage - current))} ${unit(rule.itemId)} ${rule.nextDueUsage >= current ? 'lagi' : 'terlewat'})`}`)
  }
  return parts.join(' atau ') || 'Aturan nonaktif'
}
async function load() {
  const sequence = ++requestSequence
  loading.value = true
  error.value = ''
  try { const result = await api.maintenance(); if (sequence === requestSequence) data.value = result }
  catch (requestError) { if (sequence === requestSequence) error.value = requestError.message }
  finally { if (sequence === requestSequence) loading.value = false }
}
function showToast(message) {
  toast.value = message
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value = '' }, 3500)
}
function openItem(entry = null) {
  editingId.value = entry?.id || null
  Object.assign(itemForm, { name: '', category: '', brand: '', model: '', purchaseDate: '', startUsageDate: '', purchasePrice: '', serialNumber: '', warrantyExpiry: '', location: '', imageUrl: '', notes: '', currentUsage: '', usageUnit: 'km' }, entry || {})
  itemForm.purchasePrice ??= ''
  itemForm.currentUsage ??= ''
  formError.value = ''
  modal.value = 'item'
}
function openRule(rule = null) {
  editingId.value = rule?.id || null
  Object.assign(ruleForm, { name: '', kind: 'maintenance', intervalValue: 3, intervalUnit: 'months', usageInterval: '', startDate: item.value?.startUsageDate || item.value?.purchaseDate || data.value.today, startUsage: item.value?.currentUsage || 0, reminderDays: 14, reminderUsage: 0, active: true }, rule || {})
  ruleForm.intervalValue ??= ''
  ruleForm.usageInterval ??= ''
  formError.value = ''
  modal.value = 'rule'
}
function openCompletion(rule) {
  selectedRule.value = rule
  Object.assign(completionForm, { completedAt: data.value.today, cost: '', vendor: '', usageValue: data.value.items.find(entry => entry.id === rule.itemId)?.currentUsage ?? '', notes: '' })
  formError.value = ''
  modal.value = 'completion'
}
async function submit() {
  if (saving.value) return
  saving.value = true
  formError.value = ''
  const sequence = requestSequence
  try {
    if (modal.value === 'item') {
      const payload = Object.fromEntries(['name','category','brand','model','purchaseDate','startUsageDate','serialNumber','warrantyExpiry','location','imageUrl','notes','usageUnit'].map(key => [key, itemForm[key]]))
      Object.assign(payload, { purchasePrice: optionalNumber(itemForm.purchasePrice), currentUsage: optionalNumber(itemForm.currentUsage) })
      if (editingId.value) await api.updateMaintenanceItem(editingId.value, payload)
      else await api.createMaintenanceItem(payload)
    } else if (modal.value === 'rule') {
      const value = optionalNumber(ruleForm.intervalValue)
      const payload = { name: ruleForm.name, kind: ruleForm.kind, intervalValue: value, intervalUnit: value === null ? '' : ruleForm.intervalUnit, usageInterval: optionalNumber(ruleForm.usageInterval), startDate: ruleForm.startDate, startUsage: Number(ruleForm.startUsage), reminderDays: Number(ruleForm.reminderDays), reminderUsage: Number(ruleForm.reminderUsage), active: ruleForm.active }
      if (!payload.intervalValue && !payload.usageInterval) throw new Error('Isi interval waktu atau interval penggunaan.')
      if (editingId.value) await api.updateMaintenanceRule(editingId.value, payload)
      else await api.createMaintenanceRule(item.value.id, payload)
    } else {
      await api.completeMaintenance(selectedRule.value.id, { ...completionForm, cost: optionalNumber(completionForm.cost), usageValue: optionalNumber(completionForm.usageValue) })
    }
    if (sequence !== requestSequence) return
    modal.value = ''
    showToast('Maintenance berhasil disimpan.')
    await load()
  } catch (requestError) { if (sequence === requestSequence) formError.value = requestError.message }
  finally { saving.value = false }
}
async function removeItem() {
  if (saving.value || !window.confirm(`Hapus ${item.value.name} beserta seluruh aturan dan riwayat maintenance?`)) return
  saving.value = true
  try { await api.deleteMaintenanceItem(item.value.id); await router.push('/maintenance/items'); await load(); showToast('Barang berhasil dihapus.') }
  catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
function workspaceChanged() { modal.value = ''; search.value = ''; filter.value = 'all'; data.value = { items: [], rules: [], histories: [], categories: [], today: localToday() }; load() }
function headerSearch(event) { search.value = event.detail; if (isDetail.value || isDashboard.value) router.push('/maintenance/items') }
watch(() => route.fullPath, () => { modal.value = ''; filter.value = 'all' })
onMounted(() => { load(); window.addEventListener('hubby:workspace-changed', workspaceChanged); window.addEventListener('hubby:maintenance-search', headerSearch) })
onBeforeUnmount(() => { requestSequence++; clearTimeout(toastTimer); window.removeEventListener('hubby:workspace-changed', workspaceChanged); window.removeEventListener('hubby:maintenance-search', headerSearch) })
</script>

<template>
  <section class="page watch-page maintenance-page">
    <RouterLink v-if="isDetail" class="back-link" to="/maintenance/items"><ArrowLeft :size="16" /> Kembali ke daftar barang</RouterLink>
    <div class="page-heading watch-heading">
      <div><h1>{{ isDetail ? item?.name || 'Detail barang' : isHistory ? 'Riwayat maintenance' : isDashboard ? 'Hubby Maintenance' : 'Daftar barang' }}</h1><p>{{ isDetail ? 'Informasi barang, jadwal perawatan, dan riwayat servis.' : 'Pantau umur barang dan jadwal perawatan atau penggantiannya.' }}</p></div>
      <div v-if="!isDetail" class="heading-actions"><button class="primary-button" type="button" @click="openItem()"><Plus :size="17" /> Tambah barang</button></div>
      <div v-else-if="item && !loading" class="heading-actions"><button class="secondary-button" type="button" :disabled="saving" @click="openItem(item)"><Pencil :size="16" /> Edit barang</button><button class="secondary-button" type="button" :disabled="saving" @click="removeItem"><Trash2 :size="16" /> Hapus barang</button></div>
    </div>
    <p v-if="error" class="watch-error" role="alert">{{ error }} <button class="secondary-button" type="button" @click="load">Coba lagi</button></p>
    <div v-if="loading" class="watch-detail-loading" role="status">Memuat maintenance...</div>
    <template v-else-if="!error">
      <template v-if="isDetail">
        <EmptyState v-if="!item" title="Barang tidak ditemukan" text="Barang mungkin sudah dihapus atau berada di workspace lain." />
        <template v-else>
          <section class="watch-panel maintenance-info">
            <img v-if="item.imageUrl" class="maintenance-image" :src="item.imageUrl" :alt="item.name" />
            <div><span class="maintenance-status" :class="`maintenance-${status(item)}`">{{ statusLabels[status(item)] }}</span><p>{{ item.category || 'Tanpa kategori' }}<template v-if="item.brand"> · {{ item.brand }}</template><template v-if="item.model"> · {{ item.model }}</template></p></div>
            <dl class="maintenance-facts">
              <div><dt>Tanggal pembelian</dt><dd>{{ date(item.purchaseDate) }}</dd></div><div><dt>Mulai digunakan</dt><dd>{{ date(item.startUsageDate) }}</dd></div><div><dt>Umur pemakaian</dt><dd>{{ usageDuration(item.startUsageDate || item.purchaseDate, data.today) }}</dd></div><div><dt>Harga pembelian</dt><dd>{{ currency(item.purchasePrice) }}</dd></div>
              <div><dt>Nomor seri</dt><dd>{{ item.serialNumber || 'Belum dicatat' }}</dd></div><div><dt>Garansi berakhir</dt><dd>{{ date(item.warrantyExpiry) }}</dd></div><div><dt>Lokasi</dt><dd>{{ item.location || 'Belum dicatat' }}</dd></div><div><dt>Penggunaan saat ini</dt><dd>{{ item.currentUsage === null ? 'Belum dicatat' : `${number(item.currentUsage)} ${usageLabels[item.usageUnit]}` }}</dd></div>
            </dl><p v-if="item.notes" class="maintenance-notes">{{ item.notes }}</p>
          </section>
          <section class="watch-panel maintenance-section">
            <div class="watch-section-heading"><h2>Aturan perawatan & penggantian</h2><button class="secondary-button" type="button" @click="openRule()"><Plus :size="16" /> Tambah aturan</button></div>
            <EmptyState v-if="!rules.length" title="Belum ada jadwal" text="Tambahkan aturan berdasarkan waktu atau penggunaan untuk memulai pengingat." />
            <article v-for="rule in rules" :key="rule.id" class="maintenance-row">
              <div class="maintenance-row-copy"><strong>{{ rule.name }}</strong><span>{{ rule.kind === 'replacement' ? 'Penggantian' : 'Perawatan' }} · {{ interval(rule) }}</span><span>Berikutnya: {{ due(rule) }}</span><span>Terakhir: {{ date(rule.lastMaintenance) }}</span></div>
              <span class="maintenance-status" :class="`maintenance-${rule.status}`">{{ statusLabels[rule.status] }}</span>
              <div class="maintenance-row-actions"><button class="secondary-button" type="button" @click="openRule(rule)"><Pencil :size="15" /> Edit aturan</button><button v-if="rule.active" class="primary-button" type="button" @click="openCompletion(rule)"><Check :size="15" /> Selesai</button></div>
            </article>
          </section>
        </template>
      </template>
      <template v-else-if="isDashboard">
        <div class="maintenance-metrics"><MetricCard v-for="metric in metrics" :key="metric.label" :label="metric.label" :value="String(metric.value)" :tone="metric.tone"><template #icon><component :is="metric.icon" :size="20" /></template></MetricCard></div>
        <div class="watch-content-grid maintenance-section">
          <section v-for="section in [{ title: 'Perlu perhatian', entries: attention }, { title: 'Jadwal mendatang', entries: upcoming }]" :key="section.title" class="watch-panel">
            <div class="watch-section-heading"><h2>{{ section.title }}</h2></div>
            <EmptyState v-if="!section.entries.length" :title="section.title === 'Perlu perhatian' ? 'Tidak ada jadwal terlambat atau hari ini' : 'Belum ada jadwal mendatang'" text="Jadwal dihitung dari aturan dan riwayat maintenance barang." />
            <article v-for="rule in section.entries" :key="rule.id" class="maintenance-row maintenance-schedule-row">
              <div class="maintenance-row-copy"><RouterLink :to="`/maintenance/items/${rule.itemId}`"><strong>{{ itemName(rule.itemId) }}</strong></RouterLink><span>{{ rule.name }}</span><span>{{ due(rule) }}</span><span class="maintenance-status" :class="`maintenance-${rule.status}`">{{ statusLabels[rule.status] }}</span></div>
              <button class="secondary-button" type="button" :aria-label="`Selesaikan ${rule.name} untuk ${itemName(rule.itemId)}`" @click="openCompletion(rule)"><Check :size="15" /> Selesai</button>
            </article>
          </section>
        </div>
      </template>
      <section v-if="!isDetail && !isHistory" class="watch-panel maintenance-section">
        <div class="watch-section-heading library-heading"><h2>Daftar barang</h2><div class="watch-tools"><label class="watch-search"><Search :size="15" /><input v-model="search" aria-label="Cari barang" placeholder="Cari barang, kategori, lokasi" /></label><select v-model="filter" aria-label="Filter status"><option value="all">Semua status</option><option v-for="key in ['good', 'due_soon', 'due_today', 'overdue', 'usage_unknown', 'unscheduled']" :key="key" :value="key">{{ statusLabels[key] }}</option></select></div></div>
        <EmptyState v-if="!filteredItems.length" :title="data.items.length ? 'Barang tidak ditemukan' : 'Belum ada barang'" :text="data.items.length ? 'Coba kata kunci atau filter lain.' : 'Tambahkan AC, ban, sikat gigi, atau barang lainnya untuk mencatat perawatannya.'" />
        <article v-for="entry in filteredItems" :key="entry.id" class="maintenance-row">
          <RouterLink class="maintenance-row-copy" :to="`/maintenance/items/${entry.id}`"><strong>{{ entry.name }}</strong><span>{{ entry.category || 'Tanpa kategori' }}<template v-if="entry.location"> · {{ entry.location }}</template></span><span>Umur pemakaian: {{ usageDuration(entry.startUsageDate || entry.purchaseDate, data.today) }}</span></RouterLink>
          <span class="maintenance-status" :class="`maintenance-${status(entry)}`">{{ statusLabels[status(entry)] }}</span><RouterLink class="secondary-button" :to="`/maintenance/items/${entry.id}`">Lihat detail</RouterLink>
        </article>
      </section>
      <section v-if="isHistory || (isDetail && item)" class="watch-panel maintenance-section">
        <div class="watch-section-heading"><h2>Riwayat maintenance</h2><label v-if="isHistory" class="watch-search"><Search :size="15" /><input v-model="search" aria-label="Cari riwayat" placeholder="Cari barang, servis, vendor" /></label><History v-else :size="19" /></div>
        <EmptyState v-if="!filteredHistory.length" title="Belum ada riwayat yang sesuai" text="Maintenance yang ditandai selesai akan tersimpan di sini." />
        <article v-for="entry in filteredHistory" :key="entry.id" class="maintenance-row">
          <div class="maintenance-row-copy"><RouterLink :to="`/maintenance/items/${entry.itemId}`"><strong>{{ entry.itemName }} · {{ entry.ruleName }}</strong></RouterLink><span>{{ date(entry.completedAt) }}<template v-if="entry.vendor"> · {{ entry.vendor }}</template></span><span v-if="entry.usageValue !== null">{{ number(entry.usageValue) }} {{ unit(entry.itemId) }}</span><p v-if="entry.notes" class="maintenance-notes">{{ entry.notes }}</p></div><strong>{{ currency(entry.cost) }}</strong>
        </article>
      </section>
    </template>

    <MaintenanceDialog v-if="modal" :title="modal === 'item' ? editingId ? 'Edit barang' : 'Tambah barang' : modal === 'rule' ? editingId ? 'Edit aturan' : 'Tambah aturan' : 'Catat maintenance selesai'" :saving="saving" @close="modal = ''" @submit="submit">
      <template v-if="modal === 'item'">
        <label>Nama barang<input v-model.trim="itemForm.name" maxlength="200" required /></label>
        <label>Kategori (opsional)<input v-model.trim="itemForm.category" list="maintenance-categories" maxlength="100" placeholder="Pilih atau tulis kategori baru" /><datalist id="maintenance-categories"><option v-for="category in suggestedCategories" :key="category" :value="category" /></datalist></label>
        <div class="form-grid"><label v-for="[key, label] in textFields" :key="key">{{ label }} (opsional)<input v-model.trim="itemForm[key]" maxlength="200" /></label><label v-for="[key, label] in dateFields" :key="key">{{ label }} (opsional)<input v-model="itemForm[key]" type="date" min="1900-01-01" max="2200-12-31" /></label><label>Harga pembelian (opsional)<MoneyInput v-model="itemForm.purchasePrice" /></label></div>
        <div class="form-grid"><label>Penggunaan saat ini (opsional)<input v-model="itemForm.currentUsage" type="number" min="0" max="1000000000000" step="0.01" /><small>Perbarui nilai ini untuk pengingat berbasis penggunaan.</small></label><label>Satuan penggunaan<select v-model="itemForm.usageUnit"><option v-for="(label, key) in usageLabels" :key="key" :value="key">{{ label }}</option></select></label></div>
        <label>URL gambar (opsional)<input v-model.trim="itemForm.imageUrl" type="url" maxlength="1000" /></label><label>Catatan (opsional)<textarea v-model.trim="itemForm.notes" maxlength="4000" rows="3" /></label>
      </template>
      <template v-else-if="modal === 'rule'">
        <label>Nama aturan<input v-model.trim="ruleForm.name" maxlength="200" required placeholder="Cuci AC / ganti oli / ganti sikat gigi" /></label><label>Jenis<select v-model="ruleForm.kind"><option value="maintenance">Perawatan</option><option value="replacement">Penggantian</option></select></label>
        <div class="form-grid"><label>Interval waktu (opsional)<input v-model="ruleForm.intervalValue" type="number" min="1" :max="ruleForm.intervalUnit === 'years' ? 1000 : 10000" step="1" placeholder="Kosongkan untuk penggunaan saja" /></label><label>Satuan waktu<select v-model="ruleForm.intervalUnit"><option v-for="(label, key) in intervalLabels" :key="key" :value="key">{{ label }}</option></select></label><label>Interval penggunaan ({{ unit(item.id) }}, opsional)<input v-model="ruleForm.usageInterval" type="number" min="0.01" max="1000000000000" step="0.01" /></label><label>Penggunaan acuan ({{ unit(item.id) }})<input v-model="ruleForm.startUsage" type="number" min="0" max="1000000000000" step="0.01" required /></label></div>
        <p class="modal-description">Isi salah satu atau kedua interval. Jika keduanya diisi, jadwal yang tercapai lebih dulu menentukan status.</p>
        <label>Tanggal acuan pertama<input v-model="ruleForm.startDate" type="date" min="1900-01-01" max="2200-12-31" required /><small>Setelah servis, jadwal berikutnya dihitung dari penyelesaian terakhir.</small></label>
        <div class="form-grid"><label>Ingatkan berapa hari sebelumnya<input v-model="ruleForm.reminderDays" type="number" min="0" max="365" required /></label><label>Ingatkan penggunaan sebelumnya ({{ unit(item.id) }})<input v-model="ruleForm.reminderUsage" type="number" min="0" max="1000000000000" step="0.01" required /></label></div>
        <label class="maintenance-checkbox"><input v-model="ruleForm.active" type="checkbox" /> Aturan aktif</label>
      </template>
      <template v-else>
        <p>{{ itemName(selectedRule.itemId) }} · {{ selectedRule.name }}</p><label>Tanggal maintenance<input v-model="completionForm.completedAt" type="date" :min="selectedRule.startDate" :max="data.today" required /></label><label>Biaya (opsional)<MoneyInput v-model="completionForm.cost" /></label><label>Vendor / tempat servis (opsional)<input v-model.trim="completionForm.vendor" maxlength="200" /></label><label>Penggunaan ({{ unit(selectedRule.itemId) }}{{ selectedRule.usageInterval ? ', wajib' : ', opsional' }})<input v-model="completionForm.usageValue" type="number" :min="selectedRule.usageInterval ? selectedRule.startUsage : 0" max="1000000000000" step="0.01" :required="!!selectedRule.usageInterval" /></label><label>Catatan (opsional)<textarea v-model.trim="completionForm.notes" maxlength="4000" rows="3" /></label>
      </template>
      <p v-if="formError" class="form-error" role="alert">{{ formError }}</p><button class="primary-button full-button" :disabled="saving">{{ saving ? 'Menyimpan...' : modal === 'completion' ? 'Simpan penyelesaian' : 'Simpan' }}</button>
    </MaintenanceDialog>
    <Teleport to="body"><Transition name="toast"><div v-if="toast" class="app-toast" role="status"><Check :size="16" />{{ toast }}</div></Transition></Teleport>
  </section>
</template>

<style scoped>
.maintenance-page { --muted: #59645e; }
.maintenance-page :deep(.empty-state p) { color: var(--muted); }
.maintenance-page .back-link { display: inline-flex; margin-bottom: 18px; }
.maintenance-metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; margin-bottom: 25px; }
.maintenance-section { margin-top: 18px; }
.maintenance-info { display: grid; gap: 16px; }
.maintenance-info p { margin: 8px 0 0; }
.maintenance-image { width: 120px; height: 120px; object-fit: cover; border-radius: 12px; }
.maintenance-facts { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 18px; margin: 0; }
.maintenance-facts dt { color: var(--muted); font-size: 11px; margin-bottom: 6px; }
.maintenance-facts dd { margin: 0; font-size: 13px; overflow-wrap: anywhere; }
.maintenance-row { display: flex; align-items: center; gap: 16px; padding: 16px 0; border-bottom: 1px solid var(--line); }
.maintenance-row:last-child { border-bottom: 0; }
.maintenance-row-copy { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 6px; font-size: 13px; overflow-wrap: anywhere; }
.maintenance-row-copy span { color: var(--muted); font-size: 12px; }
.maintenance-row-copy a:hover, a.maintenance-row-copy:hover { text-decoration: underline; }
.maintenance-row-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.maintenance-notes { white-space: pre-wrap; overflow-wrap: anywhere; margin: 0; font-size: 13px; }
.maintenance-status { width: fit-content; padding: 5px 8px; border-radius: 7px; background: var(--surface-2); color: var(--ink) !important; font-size: 11px !important; }
.maintenance-good { background: var(--sage-light); color: var(--sage-dark) !important; }
.maintenance-due_soon, .maintenance-due_today { background: var(--sand-light); color: #724a22 !important; }
.maintenance-overdue { background: var(--rose-light); color: #813d36 !important; }
.maintenance-schedule-row { align-items: start; }
.maintenance-page :is(button, a, input, select):focus-visible { outline: 2px solid var(--sage-dark); outline-offset: 3px; }
.maintenance-page input, .maintenance-page select { min-height: 40px; }
.maintenance-page .watch-search, .maintenance-page .watch-tools select { border-color: #758079; }
.maintenance-row :is(.primary-button, .secondary-button) { min-height: 40px; flex-shrink: 0; }
@media (max-width: 1100px) { .maintenance-metrics, .maintenance-facts { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 600px) {
  .maintenance-row { flex-wrap: wrap; gap: 12px; }
  .maintenance-row-copy { flex-basis: 100%; }
  .maintenance-row-actions { width: 100%; }
  .maintenance-row-actions button { flex: 1; }
  .maintenance-metrics { gap: 9px; }
  .maintenance-page .watch-section-heading { align-items: start; flex-wrap: wrap; }
  .maintenance-page .watch-search { width: 100%; }
}
</style>

<style>
.maintenance-dialog { width: min(600px, 100%); max-height: calc(100dvh - 40px); overflow-y: auto; --muted: #59645e; }
.maintenance-dialog h2 { padding-right: 45px; }
.maintenance-dialog p, .maintenance-dialog small { color: #59645e; }
.maintenance-dialog p.form-error { color: #813d36; }
.maintenance-dialog :is(button, input, select, textarea):focus-visible { outline: 2px solid var(--sage-dark); outline-offset: 2px; }
.maintenance-dialog :is(input:not([type=checkbox]), select, textarea) { border-color: #758079; }
.maintenance-dialog .maintenance-checkbox { flex-direction: row; align-items: center; }
.maintenance-dialog .maintenance-checkbox input { width: 18px; height: 18px; }
@media (max-width: 600px) { .maintenance-dialog .form-grid { grid-template-columns: 1fr; } .maintenance-dialog-backdrop { padding: 12px; } .maintenance-dialog { max-height: calc(100dvh - 24px); } }
</style>
