<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  CalendarDays, Check, Clapperboard, Clock3, Film, History, ListVideo,
  Play, Plus, Search, Trash2, Tv, X,
} from '@lucide/vue'
import { api } from '../services/api'

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const titleModalOpen = ref(false)
const sessionModalOpen = ref(false)
const search = ref('')
const filter = ref('all')
const overview = ref({
  summary: { totalTitles: 0, watchingTitles: 0, completedTitles: 0, totalMinutes: 0, monthMinutes: 0 },
  titles: [], recentSessions: [],
})
const titleForm = reactive({ mediaType: 'movie', title: '', genre: '', releaseYear: '', runtimeMinutes: 120, totalEpisodes: '' })
const sessionForm = reactive({ titleId: '', watchedAt: today(), durationMinutes: 45, seasonNumber: 1, episodeNumber: 1, notes: '' })

const selectedTitle = computed(() => overview.value.titles.find((item) => item.id === Number(sessionForm.titleId)))
const filteredTitles = computed(() => {
  const keyword = search.value.trim().toLocaleLowerCase('id-ID')
  return overview.value.titles.filter((item) => {
    const matchesFilter = filter.value === 'all' || item.status === filter.value
    const matchesSearch = !keyword || `${item.title} ${item.genre}`.toLocaleLowerCase('id-ID').includes(keyword)
    return matchesFilter && matchesSearch
  })
})
const continueWatching = computed(() => overview.value.titles.filter((item) => item.status === 'watching').slice(0, 4))

function today() { return new Date().toISOString().slice(0, 10) }
function formatDuration(minutes = 0) {
  const hours = Math.floor(Number(minutes) / 60)
  const remainder = Number(minutes) % 60
  if (!hours) return `${remainder} menit`
  return remainder ? `${hours} jam ${remainder} menit` : `${hours} jam`
}
function formatDate(value) {
  if (!value) return 'Belum ditonton'
  return new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(`${value}T00:00:00`))
}
function episodeLabel(item) {
  if (item.mediaType === 'movie') return item.sessionCount ? 'Selesai ditonton' : 'Belum ditonton'
  return item.lastEpisode ? `S${item.lastSeason} E${item.lastEpisode}` : 'Belum ada episode'
}
async function loadWatch() {
  loading.value = true
  error.value = ''
  try { overview.value = await api.watch() }
  catch (requestError) { error.value = requestError.message }
  finally { loading.value = false }
}
function openTitleModal() {
  Object.assign(titleForm, { mediaType: 'movie', title: '', genre: '', releaseYear: '', runtimeMinutes: 120, totalEpisodes: '' })
  error.value = ''
  titleModalOpen.value = true
}
function openSessionModal(item = overview.value.titles[0]) {
  if (!item) { openTitleModal(); return }
  Object.assign(sessionForm, {
    titleId: item.id, watchedAt: today(), durationMinutes: item.runtimeMinutes,
    seasonNumber: item.mediaType === 'series' ? (item.lastSeason || 1) : 0,
    episodeNumber: item.mediaType === 'series' ? (item.lastEpisode + 1 || 1) : 0, notes: '',
  })
  error.value = ''
  sessionModalOpen.value = true
}
function syncSessionDefaults() {
  const item = selectedTitle.value
  if (!item) return
  sessionForm.durationMinutes = item.runtimeMinutes
  sessionForm.seasonNumber = item.mediaType === 'series' ? (item.lastSeason || 1) : 0
  sessionForm.episodeNumber = item.mediaType === 'series' ? (item.lastEpisode + 1 || 1) : 0
}
async function addTitle() {
  saving.value = true
  error.value = ''
  try {
    await api.createWatchTitle({ ...titleForm, releaseYear: Number(titleForm.releaseYear || 0), runtimeMinutes: Number(titleForm.runtimeMinutes), totalEpisodes: titleForm.mediaType === 'series' ? Number(titleForm.totalEpisodes || 0) : 0 })
    titleModalOpen.value = false
    await loadWatch()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function addSession() {
  saving.value = true
  error.value = ''
  try {
    await api.createWatchSession({ ...sessionForm, titleId: Number(sessionForm.titleId), durationMinutes: Number(sessionForm.durationMinutes), seasonNumber: Number(sessionForm.seasonNumber || 0), episodeNumber: Number(sessionForm.episodeNumber || 0) })
    sessionModalOpen.value = false
    await loadWatch()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function changeStatus(item, event) {
  const previous = item.status
  item.status = event.target.value
  try { await api.updateWatchTitleStatus(item.id, item.status); await loadWatch() }
  catch (requestError) { item.status = previous; error.value = requestError.message }
}
async function removeTitle(item) {
  if (!window.confirm(`Hapus ${item.title} beserta seluruh riwayat tontonnya?`)) return
  try { await api.deleteWatchTitle(item.id); await loadWatch() }
  catch (requestError) { error.value = requestError.message }
}
async function removeSession(item) {
  if (!window.confirm(`Hapus riwayat menonton ${item.title}?`)) return
  try { await api.deleteWatchSession(item.id); await loadWatch() }
  catch (requestError) { error.value = requestError.message }
}
function handleWorkspaceChange() { loadWatch() }
onMounted(() => { loadWatch(); window.addEventListener('hubby:workspace-changed', handleWorkspaceChange) })
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page watch-page">
    <div class="page-heading watch-heading">
      <div><p class="eyebrow">Personal entertainment tracker</p><h1>Hubby Watch</h1><p>Ingat tontonan terakhir dan lihat berapa banyak waktu yang kamu habiskan.</p></div>
      <div class="heading-actions">
        <button class="secondary-button" type="button" :disabled="!overview.titles.length" @click="openSessionModal()"><Play :size="17" /> Catat tontonan</button>
        <button class="primary-button" type="button" @click="openTitleModal"><Plus :size="17" /> Tambah judul</button>
      </div>
    </div>

    <p v-if="error && !titleModalOpen && !sessionModalOpen" class="watch-error">{{ error }}</p>

    <div class="watch-metrics" :class="{ shimmer: loading }">
      <article><span class="watch-metric-icon tone-lilac"><Clock3 :size="20" /></span><div><small>Total waktu menonton</small><strong>{{ formatDuration(overview.summary.totalMinutes) }}</strong></div></article>
      <article><span class="watch-metric-icon tone-sand"><CalendarDays :size="20" /></span><div><small>Bulan ini</small><strong>{{ formatDuration(overview.summary.monthMinutes) }}</strong></div></article>
      <article><span class="watch-metric-icon tone-blue"><Play :size="20" /></span><div><small>Sedang ditonton</small><strong>{{ overview.summary.watchingTitles }} judul</strong></div></article>
      <article><span class="watch-metric-icon tone-sage"><Check :size="20" /></span><div><small>Selesai</small><strong>{{ overview.summary.completedTitles }} judul</strong></div></article>
    </div>

    <section v-if="continueWatching.length" class="watch-section">
      <div class="watch-section-heading"><div><p class="eyebrow">Lanjutkan</p><h2>Terakhir kamu tonton</h2></div></div>
      <div class="continue-grid">
        <button v-for="item in continueWatching" :key="item.id" class="continue-card" type="button" @click="openSessionModal(item)">
          <span class="continue-art"><Tv v-if="item.mediaType === 'series'" :size="27" /><Film v-else :size="27" /></span>
          <span class="continue-copy"><small>{{ item.mediaType === 'series' ? 'Series' : 'Film' }} · {{ item.genre || 'Tanpa genre' }}</small><strong>{{ item.title }}</strong><span>{{ episodeLabel(item) }} · {{ formatDate(item.lastWatchedAt) }}</span></span>
          <span class="continue-play"><Play :size="17" fill="currentColor" /></span>
        </button>
      </div>
    </section>

    <div class="watch-content-grid">
      <section class="watch-panel library-panel">
        <div class="watch-section-heading library-heading">
          <div><p class="eyebrow">Koleksi</p><h2>Pustaka tontonan</h2></div>
          <div class="watch-tools"><label class="watch-search"><Search :size="15" /><input v-model="search" placeholder="Cari judul atau genre" /></label><select v-model="filter" aria-label="Filter status"><option value="all">Semua status</option><option value="planned">Watchlist</option><option value="watching">Sedang ditonton</option><option value="completed">Selesai</option><option value="dropped">Dihentikan</option></select></div>
        </div>
        <div v-if="!loading && !filteredTitles.length" class="watch-empty"><Clapperboard :size="28" /><strong>{{ overview.titles.length ? 'Judul tidak ditemukan' : 'Pustakamu masih kosong' }}</strong><p>{{ overview.titles.length ? 'Coba kata kunci atau filter lain.' : 'Tambahkan film atau series pertama untuk mulai tracking.' }}</p><button v-if="!overview.titles.length" class="secondary-button" @click="openTitleModal"><Plus :size="15" /> Tambah judul</button></div>
        <div v-else class="watch-library-list">
          <article v-for="item in filteredTitles" :key="item.id" class="watch-title-row">
            <span class="title-art"><Tv v-if="item.mediaType === 'series'" :size="21" /><Film v-else :size="21" /></span>
            <div class="title-main"><strong>{{ item.title }}</strong><span>{{ item.mediaType === 'series' ? 'Series' : 'Film' }}<template v-if="item.releaseYear"> · {{ item.releaseYear }}</template><template v-if="item.genre"> · {{ item.genre }}</template></span></div>
            <div class="title-progress"><strong>{{ episodeLabel(item) }}</strong><span>{{ formatDuration(item.watchedMinutes) }} tercatat</span></div>
            <select class="status-select" :value="item.status" :aria-label="`Status ${item.title}`" @change="changeStatus(item, $event)"><option value="planned">Watchlist</option><option value="watching">Ditonton</option><option value="completed">Selesai</option><option value="dropped">Dihentikan</option></select>
            <button class="row-icon-button play" type="button" aria-label="Catat tontonan" @click="openSessionModal(item)"><Play :size="16" /></button>
            <button class="row-icon-button danger" type="button" aria-label="Hapus judul" @click="removeTitle(item)"><Trash2 :size="15" /></button>
          </article>
        </div>
      </section>

      <aside class="watch-panel history-panel">
        <div class="watch-section-heading"><div><p class="eyebrow">Aktivitas</p><h2>Riwayat terbaru</h2></div><History :size="19" /></div>
        <div v-if="!overview.recentSessions.length" class="watch-empty compact"><History :size="25" /><strong>Belum ada riwayat</strong><p>Sesi yang kamu catat akan muncul di sini.</p></div>
        <div v-else class="history-list">
          <article v-for="item in overview.recentSessions" :key="item.id"><span class="history-dot" /><div><strong>{{ item.title }}</strong><span><template v-if="item.mediaType === 'series'">S{{ item.seasonNumber }} E{{ item.episodeNumber }} · </template>{{ formatDuration(item.durationMinutes) }}</span><small>{{ formatDate(item.watchedAt) }}</small></div><button type="button" :aria-label="`Hapus riwayat ${item.title}`" @click="removeSession(item)"><Trash2 :size="14" /></button></article>
        </div>
      </aside>
    </div>

    <Teleport to="body">
      <div v-if="titleModalOpen" class="modal-backdrop" @click.self="titleModalOpen = false">
        <form class="modal watch-modal" @submit.prevent="addTitle">
          <button class="modal-close" type="button" aria-label="Tutup" @click="titleModalOpen = false"><X :size="18" /></button>
          <p class="eyebrow">Pustaka Hubby Watch</p><h2>Tambah judul</h2><p class="modal-description">Masukkan detail dasar film atau series. Katalog dapat diperkaya nanti.</p>
          <div class="type-switch"><button type="button" :class="{ active: titleForm.mediaType === 'movie' }" @click="titleForm.mediaType = 'movie'">Film</button><button type="button" :class="{ active: titleForm.mediaType === 'series' }" @click="titleForm.mediaType = 'series'">Series</button></div>
          <label>Judul<input v-model.trim="titleForm.title" maxlength="160" autofocus placeholder="Contoh: Severance" required /></label>
          <div class="form-grid"><label>Genre<input v-model.trim="titleForm.genre" maxlength="80" placeholder="Drama, Sci-fi" /></label><label>Tahun rilis<input v-model="titleForm.releaseYear" type="number" min="1888" max="2200" placeholder="2022" /></label></div>
          <div class="form-grid"><label>{{ titleForm.mediaType === 'series' ? 'Durasi per episode' : 'Durasi film' }} (menit)<input v-model="titleForm.runtimeMinutes" type="number" min="1" max="1440" required /></label><label v-if="titleForm.mediaType === 'series'">Total episode (opsional)<input v-model="titleForm.totalEpisodes" type="number" min="1" /></label></div>
          <p v-if="error" class="form-error">{{ error }}</p><button class="primary-button full-button" :disabled="saving">{{ saving ? 'Menambahkan...' : 'Tambahkan ke pustaka' }}</button>
        </form>
      </div>

      <div v-if="sessionModalOpen" class="modal-backdrop" @click.self="sessionModalOpen = false">
        <form class="modal watch-modal" @submit.prevent="addSession">
          <button class="modal-close" type="button" aria-label="Tutup" @click="sessionModalOpen = false"><X :size="18" /></button>
          <p class="eyebrow">Watch log</p><h2>Catat tontonan</h2>
          <label>Judul<select v-model="sessionForm.titleId" required @change="syncSessionDefaults"><option v-for="item in overview.titles" :key="item.id" :value="item.id">{{ item.title }}</option></select></label>
          <div class="form-grid"><label>Tanggal ditonton<input v-model="sessionForm.watchedAt" type="date" required /></label><label>Durasi (menit)<input v-model="sessionForm.durationMinutes" type="number" min="1" max="1440" required /></label></div>
          <div v-if="selectedTitle?.mediaType === 'series'" class="form-grid"><label>Season<input v-model="sessionForm.seasonNumber" type="number" min="1" required /></label><label>Episode<input v-model="sessionForm.episodeNumber" type="number" min="1" required /></label></div>
          <label>Catatan (opsional)<input v-model.trim="sessionForm.notes" maxlength="500" placeholder="Pendapat singkat atau momen penting" /></label>
          <p v-if="error" class="form-error">{{ error }}</p><button class="primary-button full-button" :disabled="saving"><ListVideo :size="17" /> {{ saving ? 'Menyimpan...' : 'Simpan sesi menonton' }}</button>
        </form>
      </div>
    </Teleport>
  </section>
</template>
