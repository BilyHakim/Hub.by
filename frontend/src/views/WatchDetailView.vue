<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft, CalendarDays, Check, Clock3, Film, History, Star, Trash2, Tv } from '@lucide/vue'
import { api } from '../services/api'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const detail = ref({ title: {}, sessions: [], seasons: [] })
const catalogDetail = ref(null)
const seasonCatalog = ref([])
const selectedSeason = ref(1)
const loadingSeason = ref(false)

const title = computed(() => detail.value.title || {})
const watchedEpisodes = computed(() => detail.value.seasons.find((item) => item.seasonNumber === selectedSeason.value)?.watchedEpisodes || [])
const seasonProgress = computed(() => seasonCatalog.value.length ? Math.round((watchedEpisodes.value.length / seasonCatalog.value.length) * 100) : 0)

function formatDuration(minutes = 0) {
  const hours = Math.floor(Number(minutes) / 60)
  const rest = Number(minutes) % 60
  return hours ? `${hours} jam${rest ? ` ${rest} menit` : ''}` : `${rest} menit`
}
function formatDate(value) {
  if (!value) return 'Belum pernah ditonton'
  return new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(`${value}T00:00:00`))
}
function statusLabel(status) { return { planned: 'Watchlist', watching: 'Sedang ditonton', completed: 'Selesai', dropped: 'Dihentikan' }[status] || status }
async function loadSeason() {
  if (title.value.mediaType !== 'series' || !title.value.imdbId) return
  loadingSeason.value = true
  try { seasonCatalog.value = (await api.watchCatalogSeason(title.value.imdbId, selectedSeason.value)).episodes }
  catch (requestError) { error.value = requestError.message; seasonCatalog.value = [] }
  finally { loadingSeason.value = false }
}
async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    detail.value = await api.watchTitle(route.params.id)
    selectedSeason.value = detail.value.title.lastSeason || 1
    if (detail.value.title.imdbId) {
      catalogDetail.value = await api.watchCatalogTitle(detail.value.title.imdbId).catch(() => null)
    }
    await loadSeason()
  } catch (requestError) { error.value = requestError.message }
  finally { loading.value = false }
}
async function removeSession(session) {
  if (!window.confirm('Hapus riwayat tontonan ini?')) return
  try { await api.deleteWatchSession(session.id); await loadDetail() }
  catch (requestError) { error.value = requestError.message }
}
function handleWorkspaceChange() { loadDetail() }
onMounted(() => { loadDetail(); window.addEventListener('hubby:workspace-changed', handleWorkspaceChange) })
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page watch-detail-page">
    <RouterLink class="back-link" to="/watch"><ArrowLeft :size="16" /> Kembali ke Hubby Watch</RouterLink>
    <p v-if="error" class="watch-error">{{ error }}</p>
    <div v-if="loading" class="watch-detail-loading">Memuat detail tontonan...</div>
    <template v-else>
      <section class="watch-detail-hero watch-panel">
        <span class="detail-poster"><img v-if="title.posterUrl" :src="title.posterUrl" :alt="`Poster ${title.title}`" /><Tv v-else-if="title.mediaType === 'series'" :size="42" /><Film v-else :size="42" /></span>
        <div class="detail-copy">
          <div class="detail-badges"><span>{{ title.mediaType === 'series' ? 'Series' : 'Film' }}</span><span :class="`status-${title.status}`">{{ statusLabel(title.status) }}</span></div>
          <h1>{{ title.title }}</h1>
          <p class="detail-meta">{{ title.releaseYear || 'Tahun tidak diketahui' }} · {{ title.genre || 'Tanpa genre' }} · {{ title.runtimeMinutes }} menit</p>
          <p class="detail-plot">{{ catalogDetail?.plot || 'Belum ada sinopsis untuk judul ini.' }}</p>
          <div class="detail-facts"><span><Clock3 :size="16" /><strong>{{ formatDuration(title.watchedMinutes) }}</strong><small>Total ditonton</small></span><span><History :size="16" /><strong>{{ title.sessionCount }}</strong><small>{{ title.mediaType === 'series' ? 'Episode tercatat' : 'Sesi tercatat' }}</small></span><span><CalendarDays :size="16" /><strong>{{ title.lastWatchedAt ? formatDate(title.lastWatchedAt) : '–' }}</strong><small>Terakhir ditonton</small></span></div>
        </div>
      </section>

      <div class="watch-detail-grid">
        <section v-if="title.mediaType === 'series'" class="watch-panel detail-progress-panel">
          <div class="watch-section-heading"><div><p class="eyebrow">Episode tracker</p><h2>Progres series</h2></div><strong>{{ seasonProgress }}%</strong></div>
          <div class="season-tabs"><button v-for="season in (title.totalSeasons || 1)" :key="season" :class="{ active: selectedSeason === season }" @click="selectedSeason = season; loadSeason()">S{{ season }}</button></div>
          <p v-if="loadingSeason" class="detail-muted">Memuat episode...</p>
          <div v-else class="detail-episode-list">
            <article v-for="episode in seasonCatalog" :key="episode.episodeNumber" :class="{ watched: watchedEpisodes.includes(episode.episodeNumber) }">
              <span>{{ episode.episodeNumber }}</span><div><strong>{{ episode.title }}</strong><small>{{ episode.released || 'Tanggal tidak tersedia' }}<template v-if="episode.imdbRating"> · <Star :size="10" fill="currentColor" /> {{ episode.imdbRating }}</template></small></div><Check v-if="watchedEpisodes.includes(episode.episodeNumber)" :size="17" />
            </article>
          </div>
        </section>

        <aside class="watch-panel detail-history-panel">
          <div class="watch-section-heading"><div><p class="eyebrow">Watch log</p><h2>Riwayat tontonan</h2></div><History :size="19" /></div>
          <div v-if="!detail.sessions.length" class="watch-empty compact"><History :size="25" /><strong>Belum ada riwayat</strong></div>
          <div v-else class="detail-history-list"><article v-for="session in detail.sessions" :key="session.id"><span class="history-dot" /><div><strong>{{ title.mediaType === 'series' ? `Season ${session.seasonNumber} · Episode ${session.episodeNumber}` : title.title }}</strong><span>{{ formatDuration(session.durationMinutes) }}</span><small>{{ session.isBackfill ? 'Riwayat lama' : formatDate(session.watchedAt) }}</small></div><button @click="removeSession(session)"><Trash2 :size="14" /></button></article></div>
        </aside>
      </div>
    </template>
  </section>
</template>
