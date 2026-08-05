<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft, BookOpen, CalendarDays, Clock3, History, Library, Trash2 } from '@lucide/vue'
import { api } from '../services/api'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const detail = ref({ title: {}, sessions: [] })
const title = computed(() => detail.value.title || {})
const progress = computed(() => title.value.totalPages ? Math.min(100, Math.round((title.value.currentPage / title.value.totalPages) * 100)) : 0)
const firstReadAt = computed(() => {
  if (title.value.firstReadAt) return title.value.firstReadAt
  return detail.value.sessions
    .map((session) => session.readAt)
    .filter(Boolean)
    .sort()[0] || ''
})
const readingDays = computed(() => {
  if (!firstReadAt.value) return 0
  const start = new Date(`${firstReadAt.value}T00:00:00`)
  const end = title.value.status === 'completed' && title.value.lastReadAt
    ? new Date(`${title.value.lastReadAt}T00:00:00`)
    : new Date()
  end.setHours(0, 0, 0, 0)
  return Math.max(1, Math.floor((end - start) / 86400000) + 1)
})

function formatDate(value) {
  if (!value) return 'Belum pernah dibaca'
  return new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(`${value}T00:00:00`))
}
function statusLabel(status) { return { planned: 'Ingin dibaca', reading: 'Sedang dibaca', completed: 'Selesai', dropped: 'Dihentikan' }[status] || status }
async function loadDetail() {
  loading.value = true
  error.value = ''
  try { detail.value = await api.bookTitle(route.params.id) }
  catch (requestError) { error.value = requestError.message }
  finally { loading.value = false }
}
async function removeSession(session) {
  if (!window.confirm('Hapus riwayat membaca ini?')) return
  try { await api.deleteReadingSession(session.id); await loadDetail() }
  catch (requestError) { error.value = requestError.message }
}
function handleWorkspaceChange() { loadDetail() }
onMounted(() => { loadDetail(); window.addEventListener('hubby:workspace-changed', handleWorkspaceChange) })
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page watch-detail-page book-detail-page">
    <RouterLink class="back-link" to="/books"><ArrowLeft :size="16" /> Kembali ke Hubby Books</RouterLink>
    <p v-if="error" class="watch-error">{{ error }}</p>
    <div v-if="loading" class="watch-detail-loading">Memuat detail buku...</div>
    <template v-else>
      <section class="watch-detail-hero watch-panel">
        <span class="detail-poster"><img v-if="title.coverUrl" :src="title.coverUrl" :alt="`Sampul ${title.title}`" /><BookOpen v-else :size="42" /></span>
        <div class="detail-copy">
          <div class="detail-badges"><span>Buku</span><span :class="`status-${title.status}`">{{ statusLabel(title.status) }}</span></div>
          <h1>{{ title.title }}</h1>
          <p class="detail-meta">{{ title.author || 'Penulis tidak diketahui' }}<template v-if="title.publishYear"> · {{ title.publishYear }}</template></p>
          <p class="detail-plot">{{ title.description || 'Buku ini ditambahkan dari katalog Open Library. Catat progres membaca untuk melihat perjalananmu.' }}</p>
          <div class="detail-facts book-detail-facts"><span><Library :size="16" /><strong>{{ title.currentPage }}/{{ title.totalPages }}</strong><small>Halaman saat ini</small></span><span><CalendarDays :size="16" /><strong>{{ firstReadAt ? formatDate(firstReadAt) : '–' }}</strong><small>Mulai dibaca</small></span><span><Clock3 :size="16" /><strong>{{ readingDays ? `${readingDays} hari` : '–' }}</strong><small>Durasi membaca</small></span><span><CalendarDays :size="16" /><strong>{{ title.lastReadAt ? formatDate(title.lastReadAt) : '–' }}</strong><small>Terakhir dibaca</small></span></div>
        </div>
      </section>

      <div class="watch-detail-grid">
        <section class="watch-panel detail-progress-panel"><div class="watch-section-heading"><div><p class="eyebrow">Reading progress</p><h2>Perjalanan membaca</h2></div><strong>{{ progress }}%</strong></div><div class="book-progress-track"><i :style="{ width: `${progress}%` }" /></div><div class="book-progress-labels"><span>Halaman 0</span><strong>{{ title.currentPage }} dari {{ title.totalPages }}</strong><span>Selesai</span></div><p v-if="title.catalogId" class="book-source-link"><a :href="`https://openlibrary.org/works/${title.catalogId}`" target="_blank" rel="noopener noreferrer">Lihat buku di Open Library</a></p></section>
        <aside class="watch-panel detail-history-panel"><div class="watch-section-heading"><div><p class="eyebrow">Reading log</p><h2>Riwayat membaca</h2></div><History :size="19" /></div><div v-if="!detail.sessions.length" class="watch-empty compact"><BookOpen :size="25" /><strong>Belum ada progres</strong></div><div v-else class="detail-history-list"><article v-for="session in detail.sessions" :key="session.id"><span class="history-dot" /><div><strong>Halaman {{ session.startPage + 1 }}–{{ session.endPage }}</strong><span>{{ session.pagesRead }} halaman dibaca</span><small>{{ formatDate(session.readAt) }}</small></div><button @click="removeSession(session)"><Trash2 :size="14" /></button></article></div></aside>
      </div>
    </template>
  </section>
</template>
