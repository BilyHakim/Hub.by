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
const catalogQuery = ref('')
const catalogResults = ref([])
const catalogSearching = ref(false)
const selectedCatalog = ref(null)
const episodeCatalog = ref([])
const episodeProgress = ref([])
const loadingEpisodes = ref(false)
const overview = ref({
  summary: { totalTitles: 0, watchingTitles: 0, completedTitles: 0, totalMinutes: 0, monthMinutes: 0 },
  titles: [], recentSessions: [], dailyActivity: [],
})
const titleForm = reactive({ mediaType: 'movie', title: '', synopsis: '', genre: '', releaseYear: '', runtimeMinutes: 120, totalEpisodes: '', catalogId: '', posterUrl: '', totalSeasons: 0 })
const sessionForm = reactive({ titleId: '', watchedAt: today(), durationMinutes: 45, seasonNumber: 1, episodeNumber: 1, episodeFrom: 1, episodeTo: 1, notes: '', isBackfill: false })

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
const watchedEpisodeNumbers = computed(() => {
  const season = episodeProgress.value.find((item) => item.seasonNumber === Number(sessionForm.seasonNumber))
  return season?.watchedEpisodes || []
})
const availableEndEpisodes = computed(() => episodeCatalog.value.filter((item) => item.episodeNumber >= Number(sessionForm.episodeFrom)))
const maxDailyMinutes = computed(() => Math.max(...(overview.value.dailyActivity || []).map((item) => Number(item.minutes)), 1))
const weekMinutes = computed(() => (overview.value.dailyActivity || []).reduce((total, item) => total + Number(item.minutes), 0))

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
function shortDay(value) {
  return new Intl.DateTimeFormat('id-ID', { weekday: 'short' }).format(new Date(`${value}T00:00:00`))
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
  Object.assign(titleForm, { mediaType: 'movie', title: '', synopsis: '', genre: '', releaseYear: '', runtimeMinutes: 120, totalEpisodes: '', catalogId: '', posterUrl: '', totalSeasons: 0 })
  catalogQuery.value = ''
  catalogResults.value = []
  selectedCatalog.value = null
  error.value = ''
  titleModalOpen.value = true
}
async function openSessionModal(item = overview.value.titles[0]) {
  if (!item) { openTitleModal(); return }
  Object.assign(sessionForm, {
    titleId: item.id, watchedAt: today(), durationMinutes: item.runtimeMinutes,
    seasonNumber: item.mediaType === 'series' ? (item.lastSeason || 1) : 0,
    episodeNumber: item.mediaType === 'series' ? (item.lastEpisode + 1 || 1) : 0,
    episodeFrom: item.mediaType === 'series' ? (item.lastEpisode + 1 || 1) : 0,
    episodeTo: item.mediaType === 'series' ? (item.lastEpisode + 1 || 1) : 0, notes: '', isBackfill: false,
  })
  error.value = ''
  sessionModalOpen.value = true
  if (item.mediaType === 'series') await loadSeriesData()
}
async function syncSessionDefaults() {
  const item = selectedTitle.value
  if (!item) return
  sessionForm.durationMinutes = item.runtimeMinutes
  sessionForm.seasonNumber = item.mediaType === 'series' ? (item.lastSeason || 1) : 0
  sessionForm.episodeNumber = item.mediaType === 'series' ? (item.lastEpisode + 1 || 1) : 0
  sessionForm.episodeFrom = sessionForm.episodeNumber
  sessionForm.episodeTo = sessionForm.episodeNumber
  episodeCatalog.value = []
  episodeProgress.value = []
  if (item.mediaType === 'series') await loadSeriesData()
}
async function searchCatalog() {
  if (catalogQuery.value.trim().length < 2) return
  catalogSearching.value = true
  error.value = ''
  selectedCatalog.value = null
  try {
    const result = await api.searchWatchCatalog(catalogQuery.value.trim())
    catalogResults.value = result.items
  } catch (requestError) { error.value = requestError.message; catalogResults.value = [] }
  finally { catalogSearching.value = false }
}
async function selectCatalogItem(item) {
  catalogSearching.value = true
  error.value = ''
  try {
    const detail = await api.watchCatalogTitle(item.catalogId, item.mediaType)
    selectedCatalog.value = detail
    Object.assign(titleForm, { ...detail, synopsis: detail.plot || '', totalEpisodes: 0 })
  } catch (requestError) { error.value = requestError.message }
  finally { catalogSearching.value = false }
}
async function loadSeriesData() {
  const item = selectedTitle.value
  if (!item || item.mediaType !== 'series') return
  loadingEpisodes.value = true
  error.value = ''
  try {
    const requests = [api.watchProgress(item.id)]
    if (item.catalogId) requests.push(api.watchCatalogSeason(item.catalogId, Number(sessionForm.seasonNumber)))
    const [progress, catalog] = await Promise.all(requests)
    episodeProgress.value = progress.seasons
    episodeCatalog.value = catalog?.episodes || []
    const nextEpisode = item.lastSeason === Number(sessionForm.seasonNumber) ? item.lastEpisode + 1 : 1
    sessionForm.episodeFrom = Math.min(Math.max(nextEpisode, 1), episodeCatalog.value.length || nextEpisode)
    sessionForm.episodeTo = sessionForm.episodeFrom
  } catch (requestError) { error.value = requestError.message; episodeCatalog.value = [] }
  finally { loadingEpisodes.value = false }
}
function handleSeasonChange() {
  sessionForm.episodeFrom = 1
  sessionForm.episodeTo = 1
  loadSeriesData()
}
function handleEpisodeFromChange() {
  if (Number(sessionForm.episodeTo) < Number(sessionForm.episodeFrom)) sessionForm.episodeTo = sessionForm.episodeFrom
}
function selectEntireSeason() {
  sessionForm.episodeFrom = 1
  sessionForm.episodeTo = episodeCatalog.value.at(-1)?.episodeNumber || 1
}
async function markUntilEpisode() {
  const item = selectedTitle.value
  const targetSeason = Number(sessionForm.seasonNumber)
  const targetEpisode = Number(sessionForm.episodeTo)
  if (!item?.catalogId || targetSeason < 1 || targetEpisode < 1) {
    error.value = 'Pilih season dan episode terakhir yang sudah kamu tonton.'
    return
  }
  if (!window.confirm(`Tandai ${item.title} sampai Season ${targetSeason} Episode ${targetEpisode} sebagai selesai ditonton?`)) return
  saving.value = true
  error.value = ''
  try {
    for (let season = 1; season <= targetSeason; season += 1) {
      const catalog = season === targetSeason && episodeCatalog.value.length
        ? { episodes: episodeCatalog.value }
        : await api.watchCatalogSeason(item.catalogId, season)
      const episodeTo = season === targetSeason ? targetEpisode : catalog.episodes.at(-1)?.episodeNumber
      if (!episodeTo) continue
      await api.createWatchSessionBatch({
        titleId: item.id,
        watchedAt: sessionForm.watchedAt,
        durationMinutes: Number(sessionForm.durationMinutes),
        seasonNumber: season,
        episodeFrom: 1,
        episodeTo,
        notes: sessionForm.notes,
        isBackfill: sessionForm.isBackfill,
      })
    }
    sessionModalOpen.value = false
    await loadWatch()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function markEntireSeries() {
  const item = selectedTitle.value
  if (!item?.catalogId || !item.totalSeasons) {
    error.value = 'Data jumlah season belum tersedia untuk series ini.'
    return
  }
  if (!window.confirm(`Tandai seluruh ${item.totalSeasons} season ${item.title} sebagai selesai ditonton?`)) return
  saving.value = true
  error.value = ''
  try {
    for (let season = 1; season <= item.totalSeasons; season += 1) {
      const catalog = await api.watchCatalogSeason(item.catalogId, season)
      const lastEpisode = catalog.episodes.at(-1)?.episodeNumber
      if (!lastEpisode) continue
      await api.createWatchSessionBatch({
        titleId: item.id,
        watchedAt: sessionForm.watchedAt,
        durationMinutes: Number(sessionForm.durationMinutes),
        seasonNumber: season,
        episodeFrom: 1,
        episodeTo: lastEpisode,
        notes: sessionForm.notes,
        isBackfill: sessionForm.isBackfill,
      })
    }
    await api.updateWatchTitleStatus(item.id, 'completed')
    sessionModalOpen.value = false
    await loadWatch()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function addTitle() {
  saving.value = true
  error.value = ''
  try {
    await api.createWatchTitle({
      title: titleForm.title,
      synopsis: titleForm.synopsis,
      mediaType: titleForm.mediaType,
      genre: titleForm.genre,
      releaseYear: Number(titleForm.releaseYear || 0),
      runtimeMinutes: Number(titleForm.runtimeMinutes),
      totalEpisodes: 0,
      catalogId: titleForm.catalogId,
      posterUrl: titleForm.posterUrl,
      totalSeasons: Number(titleForm.totalSeasons || 0),
    })
    titleModalOpen.value = false
    await loadWatch()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function addSession() {
  saving.value = true
  error.value = ''
  try {
    if (selectedTitle.value?.mediaType === 'series') {
      await api.createWatchSessionBatch({
        titleId: Number(sessionForm.titleId),
        watchedAt: sessionForm.watchedAt,
        durationMinutes: Number(sessionForm.durationMinutes),
        seasonNumber: Number(sessionForm.seasonNumber),
        episodeFrom: Number(sessionForm.episodeFrom),
        episodeTo: Number(sessionForm.episodeTo),
        notes: sessionForm.notes,
        isBackfill: sessionForm.isBackfill,
      })
    } else {
      await api.createWatchSession({
        titleId: Number(sessionForm.titleId),
        watchedAt: sessionForm.watchedAt,
        durationMinutes: Number(sessionForm.durationMinutes),
        seasonNumber: 0,
        episodeNumber: 0,
        notes: sessionForm.notes,
        isBackfill: sessionForm.isBackfill,
      })
    }
    sessionModalOpen.value = false
    await loadWatch()
  } catch (requestError) { error.value = requestError.message }
  finally { saving.value = false }
}
async function changeStatus(item, event) {
  const previous = item.status
  const nextStatus = event.target.value
  item.status = nextStatus
  try {
    if (nextStatus === 'completed' && item.mediaType === 'movie' && !item.sessionCount) {
      await api.createWatchSession({
        titleId: item.id,
        watchedAt: today(),
        durationMinutes: item.runtimeMinutes,
        seasonNumber: 0,
        episodeNumber: 0,
        notes: 'Film ditandai selesai',
        isBackfill: false,
      })
    } else {
      await api.updateWatchTitleStatus(item.id, nextStatus)
    }
    await loadWatch()
  }
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

    <section class="watch-activity-chart watch-panel">
      <div class="watch-section-heading">
        <div><p class="eyebrow">7 hari terakhir</p><h2>Aktivitas menonton</h2></div>
        <div class="chart-total"><small>Total minggu ini</small><strong>{{ formatDuration(weekMinutes) }}</strong></div>
      </div>
      <div class="watch-bars" role="img" aria-label="Grafik waktu menonton tujuh hari terakhir">
        <article v-for="item in overview.dailyActivity" :key="item.date" :title="`${formatDate(item.date)}: ${formatDuration(item.minutes)}`">
          <span class="bar-value">{{ item.minutes ? formatDuration(item.minutes) : '–' }}</span>
          <span class="bar-track"><i :style="{ height: `${Math.max(item.minutes ? 8 : 0, (item.minutes / maxDailyMinutes) * 100)}%` }" /></span>
          <strong>{{ shortDay(item.date) }}</strong>
        </article>
      </div>
    </section>

    <section v-if="continueWatching.length" class="watch-section">
      <div class="watch-section-heading"><div><p class="eyebrow">Lanjutkan</p><h2>Terakhir kamu tonton</h2></div></div>
      <div class="continue-grid">
        <RouterLink v-for="item in continueWatching" :key="item.id" class="continue-card" :to="`/watch/${item.id}`">
          <span class="continue-art"><img v-if="item.posterUrl" :src="item.posterUrl" :alt="`Poster ${item.title}`" /><Tv v-else-if="item.mediaType === 'series'" :size="27" /><Film v-else :size="27" /></span>
          <span class="continue-copy"><small>{{ item.mediaType === 'series' ? 'Series' : 'Film' }} · {{ item.genre || 'Tanpa genre' }}</small><strong>{{ item.title }}</strong><span>{{ episodeLabel(item) }} · {{ formatDate(item.lastWatchedAt) }}</span></span>
          <span class="continue-play"><Play :size="17" fill="currentColor" /></span>
        </RouterLink>
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
            <RouterLink class="title-art" :to="`/watch/${item.id}`"><img v-if="item.posterUrl" :src="item.posterUrl" alt="" /><Tv v-else-if="item.mediaType === 'series'" :size="21" /><Film v-else :size="21" /></RouterLink>
            <RouterLink class="title-main" :to="`/watch/${item.id}`"><strong>{{ item.title }}</strong><span>{{ item.mediaType === 'series' ? 'Series' : 'Film' }}<template v-if="item.releaseYear"> · {{ item.releaseYear }}</template><template v-if="item.genre"> · {{ item.genre }}</template></span></RouterLink>
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
          <article v-for="item in overview.recentSessions" :key="item.id"><span class="history-dot" /><div><strong>{{ item.title }}</strong><span><template v-if="item.mediaType === 'series'">S{{ item.seasonNumber }} E{{ item.episodeNumber }} · </template>{{ formatDuration(item.durationMinutes) }}</span><small>{{ item.isBackfill ? 'Riwayat lama' : formatDate(item.watchedAt) }}</small></div><button type="button" :aria-label="`Hapus riwayat ${item.title}`" @click="removeSession(item)"><Trash2 :size="14" /></button></article>
        </div>
      </aside>
    </div>

    <Teleport to="body">
      <div v-if="titleModalOpen" class="modal-backdrop" @click.self="titleModalOpen = false">
        <form class="modal watch-modal watch-catalog-modal" @submit.prevent="addTitle">
          <button class="modal-close" type="button" aria-label="Tutup" @click="titleModalOpen = false"><X :size="18" /></button>
          <p class="eyebrow">Katalog TMDB</p><h2>Cari film atau series</h2><p class="modal-description">Cari judul dari TMDB, lalu tambahkan hasil yang tepat ke pustakamu.</p>
          <label>Cari judul<span class="catalog-search-input"><Search :size="17" /><input v-model.trim="catalogQuery" autofocus placeholder="Contoh: Breaking Bad" @keydown.enter.prevent="searchCatalog" /><button type="button" :disabled="catalogSearching || catalogQuery.length < 2" @click="searchCatalog">{{ catalogSearching ? 'Mencari...' : 'Cari' }}</button></span></label>
          <div v-if="catalogResults.length && !selectedCatalog" class="catalog-results">
            <button v-for="item in catalogResults" :key="item.catalogId" type="button" @click="selectCatalogItem(item)">
              <span class="catalog-poster"><img v-if="item.posterUrl" :src="item.posterUrl" alt="" /><Film v-else :size="20" /></span>
              <span><strong>{{ item.title }}</strong><small>{{ item.mediaType === 'series' ? 'Series' : 'Film' }} · {{ item.year }}</small></span><Plus :size="17" />
            </button>
          </div>
          <article v-if="selectedCatalog" class="selected-catalog">
            <span class="selected-poster"><img v-if="selectedCatalog.posterUrl" :src="selectedCatalog.posterUrl" :alt="`Poster ${selectedCatalog.title}`" /><Film v-else :size="28" /></span>
            <div><small>{{ selectedCatalog.mediaType === 'series' ? 'Series' : 'Film' }} · {{ selectedCatalog.releaseYear }}</small><strong>{{ selectedCatalog.title }}</strong><p>{{ selectedCatalog.genre }}<template v-if="selectedCatalog.totalSeasons"> · {{ selectedCatalog.totalSeasons }} season</template> · {{ selectedCatalog.runtimeMinutes }} menit</p><button type="button" @click="selectedCatalog = null">Pilih judul lain</button></div>
          </article>
          <p v-if="error" class="form-error">{{ error }}</p><button class="primary-button full-button" :disabled="saving || !selectedCatalog">{{ saving ? 'Menambahkan...' : 'Tambahkan ke pustaka' }}</button>
          <p class="catalog-attribution"><a href="https://www.themoviedb.org" target="_blank" rel="noopener noreferrer"><img src="https://www.themoviedb.org/assets/2/v4/logos/v2/blue_long_2-9665a76b1ae401a510ec1e0ca40ddcb3b0cfe45f1d51b77a308fea0845885648.svg" alt="The Movie Database (TMDB)" /></a><span>This product uses the TMDB API but is not endorsed or certified by TMDB.</span></p>
        </form>
      </div>

      <div v-if="sessionModalOpen" class="modal-backdrop watch-session-backdrop" @click.self="sessionModalOpen = false">
        <form class="modal watch-modal watch-session-modal" @submit.prevent="addSession">
          <button class="modal-close" type="button" aria-label="Tutup" @click="sessionModalOpen = false"><X :size="18" /></button>
          <p class="eyebrow">Watch log</p><h2>Catat tontonan</h2>
          <label>Judul<select v-model="sessionForm.titleId" required @change="syncSessionDefaults"><option v-for="item in overview.titles" :key="item.id" :value="item.id">{{ item.title }}</option></select></label>
          <div class="form-grid"><label>Tanggal ditonton<input v-model="sessionForm.watchedAt" type="date" required /></label><label>{{ selectedTitle?.mediaType === 'series' ? 'Durasi per episode' : 'Durasi' }} (menit)<input v-model="sessionForm.durationMinutes" type="number" min="1" max="1440" required /></label></div>
          <template v-if="selectedTitle?.mediaType === 'series'">
            <label>Season<select v-if="selectedTitle.totalSeasons" v-model="sessionForm.seasonNumber" @change="handleSeasonChange"><option v-for="season in selectedTitle.totalSeasons" :key="season" :value="season">Season {{ season }}</option></select><input v-else v-model="sessionForm.seasonNumber" type="number" min="1" required @change="handleSeasonChange" /></label>
            <div class="episode-bulk-actions">
              <button type="button" :disabled="loadingEpisodes || !episodeCatalog.length" @click="selectEntireSeason"><Check :size="14" /> Centang seluruh season ini</button>
              <button type="button" :disabled="saving || loadingEpisodes" @click="markUntilEpisode"><Play :size="14" /> Centang sampai episode ini</button>
              <button type="button" :disabled="saving || !selectedTitle.totalSeasons" @click="markEntireSeries"><ListVideo :size="14" /> Centang seluruh series</button>
            </div>
            <div class="form-grid">
              <label>Episode awal<select v-if="episodeCatalog.length" v-model="sessionForm.episodeFrom" @change="handleEpisodeFromChange"><option v-for="episode in episodeCatalog" :key="episode.episodeNumber" :value="episode.episodeNumber">E{{ episode.episodeNumber }} · {{ episode.title }}</option></select><input v-else v-model="sessionForm.episodeFrom" type="number" min="1" required /></label>
              <label>Episode akhir<select v-if="episodeCatalog.length" v-model="sessionForm.episodeTo"><option v-for="episode in availableEndEpisodes" :key="episode.episodeNumber" :value="episode.episodeNumber">E{{ episode.episodeNumber }} · {{ episode.title }}</option></select><input v-else v-model="sessionForm.episodeTo" type="number" :min="sessionForm.episodeFrom" required /></label>
            </div>
            <p v-if="loadingEpisodes" class="episode-loading">Memuat episode dari TMDB...</p>
            <div v-else-if="episodeCatalog.length" class="episode-preview">
              <span v-for="episode in episodeCatalog" :key="episode.episodeNumber" :class="{ watched: watchedEpisodeNumbers.includes(episode.episodeNumber), selected: episode.episodeNumber >= sessionForm.episodeFrom && episode.episodeNumber <= sessionForm.episodeTo }" :title="episode.title">{{ episode.episodeNumber }}<Check v-if="watchedEpisodeNumbers.includes(episode.episodeNumber)" :size="10" /></span>
            </div>
            <p class="episode-range-summary">Episode {{ sessionForm.episodeFrom }}–{{ sessionForm.episodeTo }} akan ditandai selesai ({{ Number(sessionForm.episodeTo) - Number(sessionForm.episodeFrom) + 1 }} episode).</p>
          </template>
          <label class="checkbox watch-backfill"><input v-model="sessionForm.isBackfill" type="checkbox" /> <span><strong>Ini tontonan lama</strong><small>Total jam tetap dihitung, tetapi tidak masuk statistik bulan ini.</small></span></label>
          <label>Catatan (opsional)<input v-model.trim="sessionForm.notes" maxlength="500" placeholder="Pendapat singkat atau momen penting" /></label>
          <p v-if="error" class="form-error">{{ error }}</p><button class="primary-button full-button watch-session-submit" :disabled="saving"><ListVideo :size="17" /> {{ saving ? 'Menyimpan...' : 'Simpan sesi menonton' }}</button>
        </form>
      </div>
    </Teleport>
  </section>
</template>
