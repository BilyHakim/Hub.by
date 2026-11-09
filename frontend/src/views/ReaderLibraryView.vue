<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { BookOpen, Upload, Trash2 } from '@lucide/vue'
import { api } from '../services/api'
import { pdfEngine, readingTime } from '../services/reader'

const route = useRoute()
const marksPage = computed(() => route.name === 'reader-annotations')
const books = ref([])
const titles = ref([])
const marks = ref([])
const search = ref('')
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const title = ref('')
const author = ref('')
const titleId = ref(route.query.titleId || '')
const file = ref(null)
const fileInput = ref(null)
const filteredBooks = computed(() => books.value.filter(b => `${b.title} ${b.author}`.toLowerCase().includes(search.value.toLowerCase())))
const filteredMarks = computed(() => marks.value.filter(m => `${m.title} ${m.quote} ${m.note}`.toLowerCase().includes(search.value.toLowerCase())))
const availableTitles = computed(() => titles.value.filter(t => !books.value.some(b => b.id === t.id)))
const kindLabel = kind => ({ bookmark: 'Bookmark', highlight: 'Highlight', note: 'Catatan' }[kind])

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [library, overview, annotations] = await Promise.all([api.reader(), api.books(), api.readerAnnotations()])
    books.value = library.books; titles.value = overview.titles; marks.value = annotations
  } catch (e) { error.value = e.message }
  finally { loading.value = false }
}
function selectFile(event) {
  file.value = event.target.files[0] || null
  if (file.value && !title.value) title.value = file.value.name.replace(/\.(pdf|epub)$/i, '')
}
async function upload() {
  if (!file.value) return
  busy.value = true; error.value = ''; notice.value = ''
  let document
  try {
    if (file.value.size > 50 * 1024 * 1024) throw new Error('Upload maksimal 50 MB.')
    if (!/\.(pdf|epub)$/i.test(file.value.name)) throw new Error('Pilih file PDF atau EPUB.')
    const data = await file.value.arrayBuffer()
    let pages
    if (/\.pdf$/i.test(file.value.name)) {
      const pdf = await pdfEngine()
      document = await pdf.getDocument({ data, isEvalSupported: false }).promise
      pages = document.numPages
    } else {
      const { default: ePub } = await import('epubjs')
      document = ePub()
      await document.open(data, 'binary')
      await document.ready
      await document.locations.generate(1600)
      pages = Math.max(1, document.locations.length())
    }
    const form = new FormData()
    form.append('file', file.value)
    form.append('titleId', titleId.value || '0')
    form.append('title', title.value)
    form.append('author', author.value)
    form.append('totalPages', String(pages))
    await api.uploadReaderBook(form)
    file.value = null; title.value = ''; author.value = ''; titleId.value = ''; fileInput.value.value = ''
    notice.value = 'Buku digital tersimpan dan terhubung ke Hubby Books.'
    await load()
  } catch (e) { error.value = e.message || 'File tidak dapat dibuka. Pastikan PDF/EPUB valid dan tidak terkunci.' }
  finally {
    if (document?.loadingTask) await document.loadingTask.destroy()
    else document?.destroy()
    busy.value = false
  }
}
async function removeBook(book) {
  if (!window.confirm(`Hapus file digital, waktu baca, dan anotasi ${book.title}? Data buku serta riwayat halaman tetap ada di Hubby Books.`)) return
  try { await api.deleteReaderBook(book.id); await load() } catch (e) { error.value = e.message }
}
async function removeMark(mark) {
  if (!window.confirm('Hapus anotasi ini?')) return
  try { await api.deleteReaderAnnotation(mark.id); marks.value = marks.value.filter(m => m.id !== mark.id) } catch (e) { error.value = e.message }
}
function shellSearch(event) { search.value = event.detail }
onMounted(() => { load(); window.addEventListener('hubby:workspace-changed', load); window.addEventListener('hubby:reader-search', shellSearch) })
onBeforeUnmount(() => { window.removeEventListener('hubby:workspace-changed', load); window.removeEventListener('hubby:reader-search', shellSearch) })
</script>

<template>
  <section class="page reader-page">
    <div class="page-heading"><div><p class="eyebrow">Hubby Reader</p><h1>{{ marksPage ? 'Mark & anotasi' : 'Pustaka digital' }}</h1><p>{{ marksPage ? 'Kembali ke bagian yang kamu tandai dan catat.' : 'PDF dan EPUB milikmu, terhubung ke Hubby Books.' }}</p></div><RouterLink class="secondary-button" to="/books">Hubby Books</RouterLink></div>
    <nav class="reader-tabs" aria-label="Reader"><RouterLink to="/reader" :aria-current="!marksPage ? 'page' : undefined">Pustaka digital</RouterLink><RouterLink to="/reader/annotations" :aria-current="marksPage ? 'page' : undefined">Mark & anotasi</RouterLink></nav>
    <p v-if="error" role="alert" class="watch-error">{{ error }} <button type="button" @click="load">Muat ulang</button></p>
    <p v-if="notice" role="status">{{ notice }}</p>
    <form v-if="!marksPage" class="watch-panel reader-upload" @submit.prevent="upload">
      <div><h2>Tambahkan buku digital</h2><p>Lampirkan ke buku di Books atau buat entri baru. PDF/EPUB, maksimal 50 MB. EPUB memakai estimasi halaman yang tetap.</p></div>
      <label>File buku<input ref="fileInput" type="file" accept=".pdf,.epub,application/pdf,application/epub+zip" required :disabled="busy || loading" @change="selectFile" /></label>
      <label>Hubungkan ke<select v-model="titleId" :disabled="busy || loading"><option value="">Buku baru di Hubby Books</option><option v-for="book in availableTitles" :key="book.id" :value="book.id">{{ book.title }}</option></select></label>
      <template v-if="!titleId"><label>Judul<input v-model.trim="title" required maxlength="200" :disabled="busy" /></label><label>Penulis (opsional)<input v-model.trim="author" maxlength="200" :disabled="busy" /></label></template>
      <button class="primary-button" :disabled="busy || loading || !file"><Upload :size="17" />{{ busy ? 'Memeriksa dan mengunggah...' : 'Upload buku' }}</button>
    </form>
    <label class="reader-search">{{ marksPage ? 'Cari anotasi' : 'Cari buku' }}<input v-model="search" type="search" :placeholder="marksPage ? 'Judul, kutipan, atau catatan' : 'Judul atau penulis'" /></label>
    <p v-if="loading" role="status">Memuat pustaka digital...</p>
    <template v-else-if="!marksPage">
      <div v-if="!filteredBooks.length" class="watch-panel reader-empty"><BookOpen :size="30" /><h2>{{ books.length ? 'Buku tidak ditemukan' : 'Belum ada buku digital' }}</h2><p>{{ books.length ? 'Coba kata pencarian lain.' : 'Upload PDF atau EPUB melalui form di atas untuk mulai membaca.' }}</p></div>
      <div v-else class="reader-library-grid"><article v-for="book in filteredBooks" :key="book.id" class="watch-panel reader-book"><span class="reader-format">{{ book.format.toUpperCase() }}</span><h2>{{ book.title }}</h2><p>{{ book.author || 'Penulis belum diisi' }}</p><p>{{ book.currentPage ? `Posisi terakhir: ${book.format === 'pdf' ? 'halaman PDF ' + book.location : 'halaman estimasi ' + book.currentPage}` : 'Belum dibaca' }}</p><small>Waktu tercatat: {{ readingTime(book.seconds) }}</small><small v-if="book.lastReadAt">Terakhir dibaca: {{ book.lastReadAt }}</small><div class="reader-book-actions"><RouterLink class="primary-button" :to="`/reader/books/${book.id}`">{{ book.currentPage ? 'Lanjutkan membaca' : 'Baca buku' }}</RouterLink><RouterLink class="secondary-button" :to="`/books/${book.id}`">Detail Books</RouterLink><button class="reader-icon-button" :aria-label="`Hapus file digital ${book.title}`" @click="removeBook(book)"><Trash2 :size="18" /></button></div></article></div>
    </template>
    <template v-else>
      <div v-if="!filteredMarks.length" class="watch-panel reader-empty"><h2>{{ marks.length ? 'Anotasi tidak ditemukan' : 'Belum ada mark atau anotasi' }}</h2><p>Tandai halaman atau pilih teks saat membaca buku.</p></div>
      <div v-else class="reader-marks"><article v-for="mark in filteredMarks" :key="mark.id" class="watch-panel"><small>{{ kindLabel(mark.kind) }} · {{ mark.title }}</small><blockquote v-if="mark.quote">{{ mark.quote }}</blockquote><p v-if="mark.note">{{ mark.note }}</p><div class="reader-book-actions"><RouterLink class="secondary-button" :to="{ path: `/reader/books/${mark.titleId}`, query: { mark: mark.id } }">Buka bagian ini</RouterLink><button class="reader-icon-button" aria-label="Hapus anotasi" @click="removeMark(mark)"><Trash2 :size="18" /></button></div></article></div>
    </template>
  </section>
</template>
