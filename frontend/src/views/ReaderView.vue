<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router'
import DOMPurify from 'dompurify'
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import { api } from '../services/api'
import { bookPage, pdfEngine, readingTime } from '../services/reader'

const route = useRoute()
const router = useRouter()
const book = ref(null)
const loading = ref(true)
const error = ref('')
const status = ref('')
const page = ref(1)
const count = ref(1)
const location = ref('')
const canonicalPage = ref(1)
const timer = ref(false)
const elapsed = ref(0)
const baseSeconds = ref(0)
const marks = ref([])
const quote = ref('')
const note = ref('')
const selectionLocation = ref('')
const selectionRects = ref([])
const savingMark = ref(false)
const rendering = ref(false)
const readerHost = ref(null)
const toolbar = ref(null)
const fullscreenButton = ref(null)
const fullscreen = ref(false)
const navigationHidden = ref(false)
const hideNavigationButton = ref(null)
const showNavigationButton = ref(null)
const contentHost = ref(null)
const pdfScale = ref('width')
const epubScale = ref('100')
const pdfHost = ref(null)
const canvas = ref(null)
const textHost = ref(null)
const epubHost = ref(null)
const toc = ref([])
const chapter = ref('')
const sessionID = crypto.randomUUID()
let pdf, pdfModule, epub, rendition, interval, observer, renderTask, textLayer
let disposed = false
let tickAt = Date.now()
let ticks = 0
let saveQueue = Promise.resolve()
let previousBodyOverflow = ''
let epubResizeFrame
const currentMarks = computed(() => marks.value.filter(m => m.titleId === book.value?.id))
const pageRects = computed(() => currentMarks.value.filter(m => m.location === location.value).flatMap(m => m.rects || []))
const canAnnotate = computed(() => !loading.value && !rendering.value && !savingMark.value && !!location.value)

function resizeEPUB() {
  if (!rendition || loading.value || disposed) return
  cancelAnimationFrame(epubResizeFrame)
  epubResizeFrame = requestAnimationFrame(() => {
    if (disposed || rendering.value) return
    const height = fullscreen.value ? Math.max(180, contentHost.value.clientHeight - 24) : 620
    const width = epubHost.value.clientWidth
    const container = epubHost.value.querySelector('.epub-container')
    if (container?.clientWidth === width && container.clientHeight === height) return
    rendering.value = true
    relocateEPUB(() => rendition.resize(width, height, location.value || undefined))
      .catch(e => { error.value = e.message })
      .finally(() => { rendering.value = false; if (!disposed) resizeEPUB() })
  })
}
async function changeScale(event) {
  clearSelection()
  if (book.value.format === 'pdf') {
    pdfScale.value = event.target.value
    await renderPDF()
  } else {
    epubScale.value = event.target.value
    const position = location.value
    rendering.value = true
    try {
      await relocateEPUB(() => {
        rendition.themes.fontSize(`${18 * Number(epubScale.value) / 100}px`)
        return rendition.display(position)
      })
    } catch (e) { error.value = e.message }
    finally { rendering.value = false; resizeEPUB() }
  }
}
async function relocateEPUB(action) {
  let relocated
  const positionReady = new Promise(resolve => { relocated = resolve; rendition.once('relocated', relocated) })
  try { await action(); await positionReady }
  finally { rendition.off('relocated', relocated) }
}
async function toggleNavigation() {
  navigationHidden.value = !navigationHidden.value
  await nextTick()
  resizeEPUB()
  const button = navigationHidden.value ? showNavigationButton.value : hideNavigationButton.value
  button?.focus({ preventScroll: true })
}
async function setFullscreen(active) {
  if (fullscreen.value === active) return
  if (active) previousBodyOverflow = document.body.style.overflow
  fullscreen.value = active
  navigationHidden.value = false
  document.body.style.overflow = active ? 'hidden' : previousBodyOverflow
  await nextTick()
  if (disposed) return
  readerHost.value.scrollTop = 0
  resizeEPUB()
  fullscreenButton.value?.focus({ preventScroll: true })
}
async function toggleFullscreen() {
  if (fullscreen.value) {
    if (document.fullscreenElement === readerHost.value) await document.exitFullscreen().catch(() => {})
    await setFullscreen(false)
  } else {
    const request = readerHost.value.requestFullscreen?.()
    await setFullscreen(true)
    if (request) await request.catch(() => {})
  }
}
function fullscreenChanged() {
  if (document.fullscreenElement !== readerHost.value && fullscreen.value) setFullscreen(false)
}
function fullscreenKeydown(event) {
  if (event.key === 'Escape' && fullscreen.value) toggleFullscreen()
}

function accountTime() {
  const now = Date.now()
  if (timer.value && !loading.value && !document.hidden) elapsed.value += Math.min(5, (now - tickAt) / 1000)
  tickAt = now
}
function save() {
  if (!book.value || !location.value || loading.value) return Promise.resolve(true)
  accountTime()
  const id = book.value.id
  const payload = { location: location.value, currentPage: canonicalPage.value, timerEnabled: timer.value, sessionId: sessionID, seconds: Math.min(86400, Math.floor(elapsed.value)) }
  status.value = 'Menyimpan...'
  saveQueue = saveQueue.catch(() => {}).then(() => api.saveReaderProgress(id, payload)).then(() => {
    status.value = 'Posisi baca tersimpan'
    return true
  }).catch(e => { error.value = e.message; status.value = 'Belum tersimpan. Coba simpan lagi.'; return false })
  return saveQueue
}
function toggleTimer() {
  accountTime()
  timer.value = !timer.value
  save()
}
function clearSelection() { quote.value = ''; selectionLocation.value = ''; selectionRects.value = [] }
function capturePDFSelection() {
  const selection = window.getSelection()
  if (!selection?.rangeCount || !textHost.value?.contains(selection.anchorNode)) return
  const text = selection.toString().trim()
  if (!text) return
  const bounds = pdfHost.value.getBoundingClientRect()
  quote.value = text.slice(0, 10000)
  selectionLocation.value = location.value
  selectionRects.value = Array.from(selection.getRangeAt(0).getClientRects()).map(r => ({
    x: Math.max(0, (r.left - bounds.left) / bounds.width), y: Math.max(0, (r.top - bounds.top) / bounds.height),
    width: Math.min(1, r.width / bounds.width), height: Math.min(1, r.height / bounds.height),
  })).filter(r => r.width > 0 && r.height > 0 && r.x + r.width <= 1.001 && r.y + r.height <= 1.001).slice(0, 100)
}
async function renderPDF() {
  if (!pdf || disposed) return
  rendering.value = true
  clearSelection()
  try {
    renderTask?.cancel(); textLayer?.cancel()
    try { await renderTask?.promise } catch { /* RenderingCancelledException is expected when resizing. */ }
    const source = await pdf.getPage(page.value)
    if (disposed) return
    const original = source.getViewport({ scale: 1 })
    const available = Math.max(240, pdfHost.value.parentElement.clientWidth - 24)
    const availableHeight = Math.max(180, (fullscreen.value ? contentHost.value.clientHeight : window.innerHeight - (toolbar.value?.offsetHeight || 0) - 120) - 24)
    const scale = pdfScale.value === 'width' ? available / original.width : pdfScale.value === 'page' ? Math.min(available / original.width, availableHeight / original.height) : Number(pdfScale.value) / 100
    const viewport = source.getViewport({ scale })
    const ratio = window.devicePixelRatio || 1
    canvas.value.width = Math.floor(viewport.width * ratio); canvas.value.height = Math.floor(viewport.height * ratio)
    pdfHost.value.style.width = `${viewport.width}px`; pdfHost.value.style.height = `${viewport.height}px`
    pdfHost.value.style.setProperty('--scale-factor', viewport.scale)
    pdfHost.value.style.setProperty('--total-scale-factor', viewport.scale)
    renderTask = source.render({ canvasContext: canvas.value.getContext('2d'), viewport, transform: [ratio, 0, 0, ratio, 0, 0] })
    await renderTask.promise
    textHost.value.replaceChildren()
    textLayer = new pdfModule.TextLayer({ textContentSource: await source.getTextContent(), container: textHost.value, viewport })
    await textLayer.render()
  } catch (e) { if (e.name !== 'RenderingCancelledException' && !disposed) error.value = e.message }
  finally { rendering.value = false }
}
async function navigate(target) {
  if (loading.value || rendering.value) return
  error.value = ''
  try {
    if (book.value.format === 'pdf') {
      page.value = Math.max(1, Math.min(count.value, Number(target) || 1))
      location.value = String(page.value)
      canonicalPage.value = bookPage(page.value, count.value, book.value.totalPages)
      await renderPDF()
      await save()
    } else {
      clearSelection()
      rendering.value = true
      try {
        await relocateEPUB(() => target === 'next' ? rendition.next() : target === 'prev' ? rendition.prev() : rendition.display(target))
      } finally { rendering.value = false; resizeEPUB() }
    }
  } catch (e) { error.value = e.message }
}
async function addMark(kind) {
  if (!canAnnotate.value) return
  savingMark.value = true; error.value = ''
  try {
    const mark = { kind, location: (kind !== 'bookmark' && quote.value ? selectionLocation.value : location.value), page: canonicalPage.value, quote: kind === 'bookmark' ? '' : quote.value, note: kind === 'bookmark' ? '' : note.value, rects: kind === 'bookmark' ? [] : selectionRects.value }
    const created = await api.createReaderAnnotation(book.value.id, mark)
    marks.value.unshift({ ...mark, id: created.id, titleId: book.value.id })
    if (book.value.format === 'epub' && mark.quote) rendition.annotations.highlight(mark.location, {}, undefined, 'reader-highlight', { fill: '#e4bc61', 'fill-opacity': '0.4' })
    if (kind !== 'bookmark') { note.value = ''; clearSelection() }
    status.value = 'Anotasi tersimpan'
  } catch (e) { error.value = e.message }
  finally { savingMark.value = false }
}
async function jumpMark(mark) {
  await navigate(book.value.format === 'pdf' ? Number(mark.location) : mark.location)
}
function flattenTOC(items, depth = 0) { return items.flatMap(item => [{ href: item.href, label: `${'· '.repeat(depth)}${item.label}` }, ...flattenTOC(item.subitems || [], depth + 1)]) }
async function load() {
  try {
    const [library, annotations] = await Promise.all([api.reader(), api.readerAnnotations()])
    if (disposed) return
    book.value = library.books.find(b => b.id === Number(route.params.id))
    if (!book.value) throw new Error('Buku digital tidak ditemukan di workspace ini.')
    marks.value = annotations
    timer.value = book.value.timerEnabled
    baseSeconds.value = Number(book.value.seconds)
    const marked = currentMarks.value.find(m => m.id === Number(route.query.mark))
    const target = marked?.location || book.value.location
    const data = await api.readerFile(book.value.id)
    if (disposed) return
    await nextTick()
    if (book.value.format === 'pdf') {
      pdfModule = await pdfEngine()
      pdf = await pdfModule.getDocument({ data, isEvalSupported: false }).promise
      if (disposed) { await pdf.loadingTask.destroy(); return }
      count.value = pdf.numPages
      page.value = Math.max(1, Math.min(count.value, Number(target) || 1))
      location.value = String(page.value)
      canonicalPage.value = bookPage(page.value, count.value, book.value.totalPages)
      await nextTick(); await renderPDF()
      observer = new ResizeObserver(() => { if (!rendering.value) renderPDF() })
      observer.observe(pdfHost.value.parentElement)
      observer.observe(contentHost.value)
    } else {
      const { default: ePub } = await import('epubjs')
      epub = ePub()
      await epub.open(data, 'binary')
      if (disposed) { epub.destroy(); return }
      await epub.ready
      epub.spine.hooks.content.register(doc => {
        const clean = DOMPurify.sanitize(doc.documentElement.outerHTML, { WHOLE_DOCUMENT: true, FORBID_TAGS: ['script', 'iframe', 'object', 'embed', 'form', 'base'] })
        const parsed = new DOMParser().parseFromString(clean, 'text/html')
        doc.documentElement.replaceChildren(...Array.from(parsed.documentElement.childNodes).map(n => doc.importNode(n, true)))
      })
      await epub.locations.generate(1600)
      if (disposed) { epub.destroy(); return }
      toc.value = flattenTOC((await epub.loaded.navigation).toc)
      rendition = epub.renderTo(epubHost.value, { width: '100%', height: 620, spread: 'none', allowScriptedContent: false, allowPopups: false })
      rendition.themes.default({ body: { color: '#24332d', background: '#fffefa', 'line-height': '1.8', 'font-size': '18px', padding: '12px !important' } })
      rendition.on('selected', (cfi, contents) => {
        quote.value = contents.window.getSelection().toString().slice(0, 10000)
        selectionLocation.value = cfi; selectionRects.value = []
      })
      rendition.on('keydown', fullscreenKeydown)
      rendition.on('relocated', loc => {
        location.value = loc.start.cfi
        const index = epub.locations.locationFromCfi(loc.start.cfi)
        canonicalPage.value = loc.atEnd ? book.value.totalPages : bookPage(Math.max(1, index + 1), Math.max(1, epub.locations.length()), book.value.totalPages)
        if (!loading.value) save()
      })
      await relocateEPUB(() => rendition.display(target || undefined))
      observer = new ResizeObserver(resizeEPUB)
      observer.observe(toolbar.value)
      observer.observe(contentHost.value)
      for (const mark of currentMarks.value.filter(m => m.quote)) rendition.annotations.highlight(mark.location, {}, undefined, 'reader-highlight', { fill: '#e4bc61', 'fill-opacity': '0.4' })
    }
  } catch (e) { error.value = e.message || 'Buku tidak dapat dibuka. Pastikan file tidak terkunci atau rusak.' }
  finally { loading.value = false; tickAt = Date.now() }
}
async function visibilityChange() { accountTime(); await save() }
function workspaceChange() { disposed = true; router.replace('/reader') }
onBeforeRouteLeave(async () => disposed || await save())
onBeforeRouteUpdate(async to => {
  if (to.params.id !== route.params.id) return await save()
  const mark = currentMarks.value.find(m => m.id === Number(to.query.mark))
  if (mark) await jumpMark(mark)
})
onMounted(() => {
  load()
  interval = setInterval(() => { accountTime(); ticks++; if (ticks % 10 === 0 && !document.hidden) save() }, 1000)
  document.addEventListener('visibilitychange', visibilityChange)
  document.addEventListener('fullscreenchange', fullscreenChanged)
  window.addEventListener('keydown', fullscreenKeydown)
  window.addEventListener('resize', resizeEPUB)
  window.addEventListener('pagehide', save)
  window.addEventListener('hubby:workspace-changed', workspaceChange)
})
onBeforeUnmount(() => {
  disposed = true; clearInterval(interval); observer?.disconnect(); renderTask?.cancel(); textLayer?.cancel(); pdf?.loadingTask.destroy(); epub?.destroy()
  cancelAnimationFrame(epubResizeFrame)
  if (fullscreen.value) document.body.style.overflow = previousBodyOverflow
  if (document.fullscreenElement === readerHost.value) document.exitFullscreen().catch(() => {})
  document.removeEventListener('visibilitychange', visibilityChange)
  document.removeEventListener('fullscreenchange', fullscreenChanged)
  window.removeEventListener('keydown', fullscreenKeydown)
  window.removeEventListener('resize', resizeEPUB)
  window.removeEventListener('pagehide', save)
  window.removeEventListener('hubby:workspace-changed', workspaceChange)
})
</script>

<template>
  <section ref="readerHost" class="page reader-page" :class="{ 'reader-is-fullscreen': fullscreen }" :aria-label="fullscreen ? `Layar penuh: ${book?.title || 'Reader'}` : undefined">
    <RouterLink v-show="!fullscreen" class="back-link" to="/reader">Kembali ke pustaka digital</RouterLink>
    <div v-show="!fullscreen" class="page-heading"><div><p class="eyebrow">Hubby Reader</p><h1>{{ book?.title || 'Membuka buku' }}</h1><p>{{ book?.author }}</p></div><RouterLink class="secondary-button" to="/reader/annotations">Semua mark & anotasi</RouterLink></div>
    <p v-if="loading" role="status">Memuat dan menyiapkan buku...</p>
    <p v-if="error" role="alert" class="watch-error">{{ error }}</p>
    <div v-if="book" v-show="!fullscreen || !navigationHidden" id="reader-toolbar" ref="toolbar" class="reader-toolbar watch-panel">
      <button ref="fullscreenButton" class="secondary-button" :aria-pressed="fullscreen" :disabled="loading || rendering" @click="toggleFullscreen">{{ fullscreen ? 'Keluar layar penuh' : 'Layar penuh' }}</button>
      <button v-if="fullscreen" ref="hideNavigationButton" class="secondary-button" aria-controls="reader-toolbar" :aria-expanded="!navigationHidden" @click="toggleNavigation">Sembunyikan navigasi</button>
      <div class="reader-page-nav"><button class="secondary-button" :disabled="loading || rendering || (book.format === 'pdf' && page <= 1)" @click="navigate(book.format === 'pdf' ? page - 1 : 'prev')">Sebelumnya</button><form v-if="book.format === 'pdf'" @submit.prevent="navigate(page)"><label>Halaman PDF<input :value="page" type="number" min="1" :max="count" :disabled="loading || rendering" @change="navigate($event.target.value)" /></label><span>/ {{ count }}</span></form><span v-else>Halaman estimasi {{ canonicalPage }} / {{ book.totalPages }}</span><button class="secondary-button" :disabled="loading || rendering || (book.format === 'pdf' && page >= count)" @click="navigate(book.format === 'pdf' ? page + 1 : 'next')">Berikutnya</button></div>
      <label v-if="book.format === 'epub'">Daftar isi<select v-model="chapter" :disabled="loading || rendering" @change="navigate(chapter)"><option value="" disabled>Pilih bab</option><option v-for="item in toc" :key="item.href" :value="item.href">{{ item.label }}</option></select></label>
      <label class="reader-scale">{{ book.format === 'pdf' ? 'Skala' : 'Ukuran teks' }}<select :aria-label="book.format === 'pdf' ? 'Skala' : 'Ukuran teks'" :value="book.format === 'pdf' ? pdfScale : epubScale" :disabled="loading || rendering" @change="changeScale"><template v-if="book.format === 'pdf'"><option value="width">Sesuai lebar</option><option value="page">Satu halaman penuh</option></template><option v-for="value in [50, 75, 100, 125, 150, 200]" :key="value" :value="String(value)">{{ value }}%</option></select></label>
      <button class="secondary-button" role="switch" :aria-checked="timer" :disabled="loading" @click="toggleTimer">Timer {{ timer ? 'ON' : 'OFF' }}</button><span>Waktu: {{ readingTime(baseSeconds + elapsed) }}</span><button class="secondary-button" :disabled="loading" @click="save">Simpan posisi</button><span role="status">{{ status }}</span>
    </div>
    <div v-if="fullscreen && navigationHidden" class="reader-nav-show" role="group" aria-label="Navigasi halaman cepat">
      <button class="reader-icon-button" type="button" aria-label="Halaman sebelumnya" title="Halaman sebelumnya" :disabled="loading || rendering || (book.format === 'pdf' && page <= 1)" @click="navigate(book.format === 'pdf' ? page - 1 : 'prev')"><ChevronLeft :size="22" aria-hidden="true" /></button>
      <button class="reader-icon-button" type="button" aria-label="Halaman berikutnya" title="Halaman berikutnya" :disabled="loading || rendering || (book.format === 'pdf' && page >= count)" @click="navigate(book.format === 'pdf' ? page + 1 : 'next')"><ChevronRight :size="22" aria-hidden="true" /></button>
      <button ref="showNavigationButton" class="secondary-button" aria-controls="reader-toolbar" :aria-expanded="!navigationHidden" @click="toggleNavigation">Tampilkan navigasi</button>
    </div>
    <div ref="contentHost" class="reader-content-grid">
      <div class="reader-paper" :aria-busy="loading || rendering">
        <div v-show="book?.format === 'pdf'" ref="pdfHost" class="reader-pdf-page" @mouseup="capturePDFSelection" @touchend="capturePDFSelection" @keyup="capturePDFSelection"><canvas ref="canvas" aria-label="Halaman PDF" /><div ref="textHost" class="textLayer" tabindex="0" aria-label="Teks halaman PDF" /><div class="reader-pdf-highlights" aria-hidden="true"><i v-for="(rect, i) in pageRects" :key="i" :style="{ left: `${rect.x * 100}%`, top: `${rect.y * 100}%`, width: `${rect.width * 100}%`, height: `${rect.height * 100}%` }" /></div></div>
        <div v-show="book?.format === 'epub'" ref="epubHost" class="reader-epub" />
      </div>
      <aside class="watch-panel reader-annotation-panel"><h2>Tandai & catat</h2><p>Pilih teks untuk highlight atau kutipan. Bookmark menyimpan posisi saat ini.</p><button class="secondary-button" :disabled="!canAnnotate || savingMark" @click="addMark('bookmark')">Bookmark posisi ini</button><blockquote v-if="quote">{{ quote }}</blockquote><button class="secondary-button" :disabled="!canAnnotate || !quote || savingMark" @click="addMark('highlight')">Simpan highlight</button><label>Catatan<textarea v-model.trim="note" rows="4" maxlength="4000" :disabled="!canAnnotate" placeholder="Tulis catatan pada bagian ini" /></label><button class="primary-button" :disabled="!canAnnotate || !note || savingMark" @click="addMark('note')">Simpan catatan</button><small v-if="book?.format === 'pdf'">PDF hasil scan tanpa teks tetap mendukung bookmark dan catatan halaman.</small><h2>Mark buku ini</h2><p v-if="!currentMarks.length">Belum ada bagian yang ditandai.</p><button v-for="mark in currentMarks" :key="mark.id" class="reader-mark-link" :disabled="!canAnnotate" @click="jumpMark(mark)">{{ mark.kind === 'bookmark' ? 'Bookmark' : mark.quote || mark.note }} <small>{{ book?.format === 'pdf' ? `Halaman PDF ${mark.location}` : `Halaman estimasi ${mark.page}` }}</small></button></aside>
    </div>
  </section>
</template>
