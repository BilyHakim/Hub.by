<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  BookOpen, BookPlus, CalendarDays, Check, ChevronRight, History,
  Library, Plus, Search, Trash2, X,
} from '@lucide/vue'
import { api } from '../services/api'

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const titleModalOpen = ref(false)
const sessionModalOpen = ref(false)
const search = ref('')
const filter = ref('all')
const catalogQuery = ref('')
const catalogResults = ref([])
const catalogSearching = ref(false)
const selectedCatalog = ref(null)
const overview = ref({
  summary: { totalTitles: 0, readingTitles: 0, completedTitles: 0, totalPages: 0, monthPages: 0 },
  titles: [], recentSessions: [], dailyActivity: [],
})
const bookForm = reactive({ catalogId: '', title: '', author: '', description: '', coverUrl: '', publishYear: 0, totalPages: 300 })
const sessionForm = reactive({ titleId: '', readAt: today(), currentPage: 1, notes: '' })

const selectedTitle = computed(() => overview.value.titles.find((item) => item.id === Number(sessionForm.titleId)))
const filteredTitles = computed(() => {
  const keyword = search.value.trim().toLocaleLowerCase('id-ID')
  return overview.value.titles.filter((item) => {
    const matchesFilter = filter.value === 'all' || item.status === filter.value
    const matchesSearch = !keyword || `${item.title} ${item.author}`.toLocaleLowerCase('id-ID').includes(keyword)
    return matchesFilter && matchesSearch
  })
})
const continueReading = computed(() => overview.value.titles.filter((item) => item.status === 'reading').slice(0, 4))
const maxDailyPages = computed(() => Math.max(...overview.value.dailyActivity.map((item) => Number(item.pages)), 1))
const weekPages = computed(() => overview.value.dailyActivity.reduce((total, item) => total + Number(item.pages), 0))

function today() { return new Date().toISOString().slice(0, 10) }
function formatNumber(value = 0) { return new Intl.NumberFormat('id-ID').format(Number(value)) }
function formatDate(value) {
  if (!value) return 'Belum dibaca'
  return new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(`${value}T00:00:00`))
}
function shortDay(value) { return new Intl.DateTimeFormat('id-ID', { weekday: 'short' }).format(new Date(`${value}T00:00:00`)) }
function progress(item) { return item.totalPages ? Math.min(100, Math.round((item.currentPage / item.totalPages) * 100)) : 0 }

async function loadBooks() {
  loading.value = true
  error.value = ''
  try { overview.value = await api.books() }
  catch (requestError) { error.value = requestError.message }
  finally { loading.value = false }
}
function openTitleModal() {
  Object.assign(bookForm, { catalogId: '', title: '', author: '', description: '', coverUrl: '', publishYear: 0, totalPages: 300 })
  catalogQuery.value = ''
  catalogResults.value = []
  selectedCatalog.value = null
  error.value = ''
  titleModalOpen.value = true
}
function openSessionModal(item = overview.value.titles.find((book) => book.status !== 'completed')) {
  if (!item) { openTitleModal(); return }
  Object.assign(sessionForm, { titleId: item.id, readAt: today(), currentPage: Math.min(item.currentPage + 1, item.totalPages), notes: '' })
  error.value = ''
  sessionModalOpen.value = true
}
function syncSessionDefaults() {
  const item = selectedTitle.value
  if (item) sessionForm.currentPage = Math.min(item.currentPage + 1, item.totalPages)
}
async function searchCatalog() {
  if (catalogQuery.value.trim().length < 2) return
  catalogSearching.value = true
  error.value = ''
  selectedCatalog.value = null
  try { catalogResults.value = (await api.searchBookCatalog(catalogQuery.value.trim())).items }
  catch (requestError) { error.value = requestError.message; catalogResults.value = [] }
  finally { catalogSearching.value = false }
}
async function selectCatalogItem(item) {
  catalogSearching.value = true
  const detail = await api.bookCatalogWork(item.catalogId).catch(() => ({}))
  selectedCatalog.value = item
  Object.assign(bookForm, {
    catalogId: item.catalogId,
    title: detail.title || item.title,
    author: item.author,
    description: detail.description || '',
    coverUrl: detail.coverUrl || item.coverUrl,
    publishYear: item.publishYear,
    totalPages: item.totalPages,
  })
  catalogSearching.value = false
}
async function addTitle() {
  saving.value = true
  error.value = ''
  try {
    await api.createBookTitle({
      catalogId: bookForm.catalogId,
      title: bookForm.title,
      author: bookForm.author,
      description: bookForm.description,
      coverUrl: bookForm.coverUrl,
      publishYear: Number(bookForm.publishYear || 0),
      totalPages: Number(bookForm.totalPages),
    })
    titleModalOpen.value = false
    await loadBooks()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function addSession() {
  saving.value = true
  error.value = ''
  try {
    await api.createReadingSession({ titleId: Number(sessionForm.titleId), readAt: sessionForm.readAt, currentPage: Number(sessionForm.currentPage), notes: sessionForm.notes })
    sessionModalOpen.value = false
    await loadBooks()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function changeStatus(item, event) {
  const previous = item.status
  item.status = event.target.value
  try { await api.updateBookTitleStatus(item.id, item.status); await loadBooks() }
  catch (requestError) { item.status = previous; error.value = requestError.message }
}
async function removeTitle(item) {
  if (!window.confirm(`Hapus ${item.title} beserta seluruh riwayat bacanya?`)) return
  try { await api.deleteBookTitle(item.id); await loadBooks() }
  catch (requestError) { error.value = requestError.message }
}
async function removeSession(item) {
  if (!window.confirm(`Hapus riwayat membaca ${item.title}?`)) return
  try { await api.deleteReadingSession(item.id); await loadBooks() }
  catch (requestError) { error.value = requestError.message }
}
function handleWorkspaceChange() { loadBooks() }
onMounted(() => { loadBooks(); window.addEventListener('hubby:workspace-changed', handleWorkspaceChange) })
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page watch-page books-page">
    <div class="page-heading watch-heading">
      <div><p class="eyebrow">Personal reading tracker</p><h1>Hubby Books</h1><p>Catat progres membaca dan lihat seberapa jauh perjalanan bukumu.</p></div>
      <div class="heading-actions">
        <button class="secondary-button" type="button" :disabled="!overview.titles.length" @click="openSessionModal()"><BookOpen :size="17" /> Catat progres</button>
        <button class="primary-button" type="button" @click="openTitleModal"><Plus :size="17" /> Tambah buku</button>
      </div>
    </div>

    <p v-if="error && !titleModalOpen && !sessionModalOpen" class="watch-error">{{ error }}</p>
    <div class="watch-metrics" :class="{ shimmer: loading }">
      <article><span class="watch-metric-icon tone-lilac"><Library :size="20" /></span><div><small>Total halaman dibaca</small><strong>{{ formatNumber(overview.summary.totalPages) }} halaman</strong></div></article>
      <article><span class="watch-metric-icon tone-sand"><CalendarDays :size="20" /></span><div><small>Bulan ini</small><strong>{{ formatNumber(overview.summary.monthPages) }} halaman</strong></div></article>
      <article><span class="watch-metric-icon tone-blue"><BookOpen :size="20" /></span><div><small>Sedang dibaca</small><strong>{{ overview.summary.readingTitles }} buku</strong></div></article>
      <article><span class="watch-metric-icon tone-sage"><Check :size="20" /></span><div><small>Selesai</small><strong>{{ overview.summary.completedTitles }} buku</strong></div></article>
    </div>

    <section class="watch-activity-chart watch-panel">
      <div class="watch-section-heading"><div><p class="eyebrow">7 hari terakhir</p><h2>Aktivitas membaca</h2></div><div class="chart-total"><small>Total minggu ini</small><strong>{{ formatNumber(weekPages) }} halaman</strong></div></div>
      <div class="watch-bars" role="img" aria-label="Grafik halaman dibaca tujuh hari terakhir">
        <article v-for="item in overview.dailyActivity" :key="item.date"><span class="bar-value">{{ item.pages || '–' }}</span><span class="bar-track"><i :style="{ height: `${(Number(item.pages) / maxDailyPages) * 100}%` }" /></span><strong>{{ shortDay(item.date) }}</strong></article>
      </div>
    </section>

    <section v-if="continueReading.length" class="watch-section">
      <div class="watch-section-heading"><div><p class="eyebrow">Lanjutkan</p><h2>Buku yang sedang dibaca</h2></div></div>
      <div class="continue-grid">
        <RouterLink v-for="item in continueReading" :key="item.id" class="continue-card" :to="`/books/${item.id}`">
          <span class="continue-art"><img v-if="item.coverUrl" :src="item.coverUrl" alt="" /><BookOpen v-else :size="22" /></span>
          <span class="continue-copy"><small>{{ item.author || 'Penulis tidak diketahui' }}</small><strong>{{ item.title }}</strong><span>Halaman {{ item.currentPage }}/{{ item.totalPages }} · {{ progress(item) }}%</span></span><span class="continue-play"><ChevronRight :size="17" /></span>
        </RouterLink>
      </div>
    </section>

    <div class="watch-content-grid">
      <section class="watch-panel library-panel">
        <div class="watch-section-heading library-heading"><div><p class="eyebrow">Koleksi</p><h2>Pustaka buku</h2></div><div class="watch-tools"><label class="watch-search"><Search :size="15" /><input v-model="search" placeholder="Cari judul atau penulis" /></label><select v-model="filter"><option value="all">Semua status</option><option value="planned">Ingin dibaca</option><option value="reading">Sedang dibaca</option><option value="completed">Selesai</option><option value="dropped">Dihentikan</option></select></div></div>
        <div v-if="!loading && !filteredTitles.length" class="watch-empty"><Library :size="28" /><strong>{{ overview.titles.length ? 'Buku tidak ditemukan' : 'Pustakamu masih kosong' }}</strong><p>{{ overview.titles.length ? 'Coba kata kunci atau filter lain.' : 'Tambahkan buku pertama dari katalog Open Library.' }}</p><button v-if="!overview.titles.length" class="secondary-button" @click="openTitleModal"><BookPlus :size="15" /> Tambah buku</button></div>
        <div v-else class="watch-library-list">
          <article v-for="item in filteredTitles" :key="item.id" class="watch-title-row book-title-row">
            <RouterLink class="title-art" :to="`/books/${item.id}`"><img v-if="item.coverUrl" :src="item.coverUrl" alt="" /><BookOpen v-else :size="20" /></RouterLink>
            <RouterLink class="title-main" :to="`/books/${item.id}`"><strong>{{ item.title }}</strong><span>{{ item.author || 'Penulis tidak diketahui' }}<template v-if="item.publishYear"> · {{ item.publishYear }}</template></span></RouterLink>
            <div class="title-progress"><strong>Halaman {{ item.currentPage }}/{{ item.totalPages }}</strong><span>{{ progress(item) }}% selesai</span></div>
            <select class="status-select" :value="item.status" @change="changeStatus(item, $event)"><option value="planned">Ingin dibaca</option><option value="reading">Dibaca</option><option value="completed">Selesai</option><option value="dropped">Dihentikan</option></select>
            <button class="row-icon-button play" type="button" aria-label="Catat progres" :disabled="item.currentPage >= item.totalPages" @click="openSessionModal(item)"><BookOpen :size="16" /></button><button class="row-icon-button danger" type="button" aria-label="Hapus buku" @click="removeTitle(item)"><Trash2 :size="15" /></button>
          </article>
        </div>
      </section>

      <aside class="watch-panel history-panel"><div class="watch-section-heading"><div><p class="eyebrow">Aktivitas</p><h2>Riwayat terbaru</h2></div><History :size="19" /></div><div v-if="!overview.recentSessions.length" class="watch-empty compact"><BookOpen :size="25" /><strong>Belum ada progres</strong><p>Sesi membaca akan muncul di sini.</p></div><div v-else class="history-list"><article v-for="item in overview.recentSessions" :key="item.id"><span class="history-dot" /><div><strong>{{ item.title }}</strong><span>Halaman {{ item.startPage + 1 }}–{{ item.endPage }} · {{ item.pagesRead }} halaman</span><small>{{ formatDate(item.readAt) }}</small></div><button @click="removeSession(item)"><Trash2 :size="14" /></button></article></div></aside>
    </div>

    <Teleport to="body">
      <div v-if="titleModalOpen" class="modal-backdrop" @click.self="titleModalOpen = false"><form class="modal watch-modal watch-catalog-modal" @submit.prevent="addTitle"><button class="modal-close" type="button" @click="titleModalOpen = false"><X :size="18" /></button><p class="eyebrow">Katalog Open Library</p><h2>Cari buku</h2><p class="modal-description">Cari berdasarkan judul, penulis, atau ISBN.</p><label>Cari buku<span class="catalog-search-input"><Search :size="17" /><input v-model.trim="catalogQuery" autofocus placeholder="Contoh: Atomic Habits" @keydown.enter.prevent="searchCatalog" /><button type="button" :disabled="catalogSearching || catalogQuery.length < 2" @click="searchCatalog">{{ catalogSearching ? 'Mencari...' : 'Cari' }}</button></span></label><div v-if="catalogResults.length && !selectedCatalog" class="catalog-results"><button v-for="item in catalogResults" :key="item.catalogId" type="button" @click="selectCatalogItem(item)"><span class="catalog-poster"><img v-if="item.coverUrl" :src="item.coverUrl" alt="" /><BookOpen v-else :size="20" /></span><span><strong>{{ item.title }}</strong><small>{{ item.author || 'Penulis tidak diketahui' }} · {{ item.publishYear || 'Tahun tidak tersedia' }}</small></span><Plus :size="17" /></button></div><article v-if="selectedCatalog" class="selected-catalog"><span class="selected-poster"><img v-if="bookForm.coverUrl" :src="bookForm.coverUrl" :alt="`Sampul ${bookForm.title}`" /><BookOpen v-else :size="28" /></span><div><small>{{ bookForm.author }}</small><strong>{{ bookForm.title }}</strong><p>{{ bookForm.publishYear || 'Tahun tidak tersedia' }}</p><button type="button" @click="selectedCatalog = null">Pilih buku lain</button></div></article><label v-if="selectedCatalog">Jumlah halaman<input v-model="bookForm.totalPages" type="number" min="1" max="100000" required /><small class="field-help">Sesuaikan dengan edisi buku yang kamu baca.</small></label><p v-if="error" class="form-error">{{ error }}</p><button class="primary-button full-button" :disabled="saving || !selectedCatalog">{{ saving ? 'Menambahkan...' : 'Tambahkan ke pustaka' }}</button><p class="book-attribution">Data buku dan sampul disediakan oleh <a href="https://openlibrary.org" target="_blank" rel="noopener noreferrer">Open Library</a>.</p></form></div>
      <div v-if="sessionModalOpen" class="modal-backdrop" @click.self="sessionModalOpen = false"><form class="modal watch-modal" @submit.prevent="addSession"><button class="modal-close" type="button" @click="sessionModalOpen = false"><X :size="18" /></button><p class="eyebrow">Reading log</p><h2>Catat progres membaca</h2><label>Buku<select v-model="sessionForm.titleId" required @change="syncSessionDefaults"><option v-for="item in overview.titles.filter((book) => book.currentPage < book.totalPages)" :key="item.id" :value="item.id">{{ item.title }}</option></select></label><div class="form-grid"><label>Tanggal membaca<input v-model="sessionForm.readAt" type="date" required /></label><label>Halaman saat ini<input v-model="sessionForm.currentPage" type="number" :min="selectedTitle?.currentPage + 1" :max="selectedTitle?.totalPages" required /></label></div><p v-if="selectedTitle" class="reading-progress-hint">Progres sebelumnya halaman {{ selectedTitle.currentPage }} dari {{ selectedTitle.totalPages }}.</p><label>Catatan (opsional)<input v-model.trim="sessionForm.notes" maxlength="500" placeholder="Insight atau kutipan favorit" /></label><p v-if="error" class="form-error">{{ error }}</p><button class="primary-button full-button" :disabled="saving">{{ saving ? 'Menyimpan...' : 'Simpan progres' }}</button></form></div>
    </Teleport>
  </section>
</template>
