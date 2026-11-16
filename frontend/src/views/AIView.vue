<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { Plus, Send, Settings, Sparkles, Trash2 } from '@lucide/vue'
import { api } from '../services/api'
import AIContextCard from '../components/AIContextCard.vue'
import AIAnswer from '../components/AIAnswer.vue'
import { aiRangeError, defaultAIStartMonth } from '../utils/aiPeriodRange'

const loading = ref(true)
const sending = ref(false)
const opening = ref(false)
const error = ref('')
const config = ref(null)
const conversations = ref([])
const conversation = ref(null)
const question = ref('')
const pendingQuestion = ref('')
const credentialId = ref('')
const month = ref('')
const startMonth = ref('')
const reduction = ref(10)
const preview = ref(null)
const previewBusy = ref(false)
const messageEnd = ref(null)
const deleting = ref(false)
const deleteId = ref(null)
let controller
let epoch = 0
let conversationEpoch = 0
let previewEpoch = 0
let retryRequest = null
const credentials = computed(() => (config.value?.credentials || []).filter(item => item.enabled))
const selectedCredential = computed(() => credentials.value.find(item => item.id === Number(credentialId.value)))
const rangeError = computed(() => aiRangeError(startMonth.value, month.value))
const ready = computed(() => !loading.value && !sending.value && !opening.value && config.value?.encryptionReady && !!selectedCredential.value && !rangeError.value)
const prompts = ['Evaluasi kondisi keuangan saya pada periode ini.', 'Pos pengeluaran mana yang bisa saya sesuaikan?', 'Jelaskan dampak simulasi ini terhadap cashflow saya.']

function resetConversation() {
  conversationEpoch++
  conversation.value = null
  error.value = ''
  retryRequest = null
}
function changePeriod() {
  resetConversation()
  invalidatePreview()
}
function invalidatePreview() {
  previewEpoch++
  preview.value = null
  previewBusy.value = false
}
async function load() {
  controller?.abort()
  controller = new AbortController()
  const current = ++epoch
  resetConversation()
  config.value = null
  conversations.value = []
  credentialId.value = ''
  question.value = ''
  pendingQuestion.value = ''
  invalidatePreview()
  deleteId.value = null
  sending.value = false
  opening.value = false
  loading.value = true
  try {
    const [settings, history, period] = await Promise.all([api.aiSettings(controller.signal), api.aiConversations(controller.signal), api.financePeriodSetting()])
    if (current !== epoch) return
    config.value = settings
    conversations.value = history
    month.value = period.currentPeriodLabel
    startMonth.value = defaultAIStartMonth(month.value)
    const defaultCredential = credentials.value.find(item => item.id === settings.settings.defaultCredentialId)
    credentialId.value = defaultCredential?.id || credentials.value[0]?.id || ''
  } catch (e) {
    if (current === epoch && e.name !== 'AbortError') error.value = e.message
  } finally { if (current === epoch) loading.value = false }
}
async function openConversation(id) {
  const current = epoch
  const selection = ++conversationEpoch
  opening.value = true
  error.value = ''
  try {
    const item = await api.aiConversation(id, controller.signal)
    if (current !== epoch || selection !== conversationEpoch) return
    conversation.value = item
    month.value = item.month
    startMonth.value = item.startMonth || defaultAIStartMonth(item.month)
    invalidatePreview()
    retryRequest = null
    await nextTick()
    messageEnd.value?.scrollIntoView({ block: 'nearest' })
  } catch (e) { if (current === epoch && e.name !== 'AbortError') error.value = e.message }
  finally { if (current === epoch && selection === conversationEpoch) opening.value = false }
}
async function showContext() {
  const current = epoch
  const request = ++previewEpoch
  previewBusy.value = true
  error.value = ''
  try {
    const data = await api.aiContext(startMonth.value, month.value, reduction.value, controller.signal)
    if (current === epoch && request === previewEpoch) preview.value = data
  } catch (e) { if (current === epoch && e.name !== 'AbortError') error.value = e.message }
  finally { if (current === epoch && request === previewEpoch) previewBusy.value = false }
}
async function send() {
  if (!ready.value || question.value.trim().length < 2) return
  const current = epoch
  const payload = {
    credentialId: Number(credentialId.value), conversationId: conversation.value?.id || 0,
    revision: conversation.value?.revision || 0, startMonth: startMonth.value, month: month.value,
    question: question.value.trim(), reductionPercent: Number(reduction.value),
  }
  const serialized = JSON.stringify(payload)
  const requestId = retryRequest?.serialized === serialized ? retryRequest.requestId : crypto.randomUUID()
  sending.value = true
  pendingQuestion.value = payload.question
  error.value = ''
  try {
    const result = await api.sendAIMessage({ ...payload, requestId }, controller.signal)
    if (current !== epoch) return
    conversation.value = result
    question.value = ''
    preview.value = null
    retryRequest = null
    const history = await api.aiConversations(controller.signal)
    if (current !== epoch) return
    conversations.value = history
    await nextTick()
    if (current === epoch) messageEnd.value?.scrollIntoView({ block: 'nearest' })
  } catch (e) {
    if (current !== epoch || e.name === 'AbortError') return
    error.value = e.message
    retryRequest = !e.status ? { serialized, requestId } : null
    const history = await api.aiConversations(controller.signal).catch(() => null)
    if (current === epoch && history) conversations.value = history
  } finally { if (current === epoch) { sending.value = false; pendingQuestion.value = '' } }
}
async function removeConversation() {
  if (!deleteId.value) return
  const current = epoch
  const id = deleteId.value
  deleting.value = true
  try {
    await api.deleteAIConversation(id)
    if (current !== epoch) return
    conversations.value = conversations.value.filter(item => item.id !== id)
    if (conversation.value?.id === id) resetConversation()
    deleteId.value = null
  } catch (e) { if (current === epoch) error.value = e.message }
  finally { if (current === epoch) deleting.value = false }
}
onMounted(() => { load(); window.addEventListener('hubby:workspace-changed', load) })
onBeforeUnmount(() => { epoch++; controller?.abort(); window.removeEventListener('hubby:workspace-changed', load) })
</script>

<template>
  <section class="page ai-page">
    <div class="page-heading compact"><div><p class="eyebrow">Hubby AI</p><h1>Pahami keuanganmu.</h1><p>Evaluasi catatan Finance dan pertimbangkan langkah berikutnya.</p></div><RouterLink to="/ai/settings" class="secondary-button"><Settings :size="18" />Pengaturan AI</RouterLink></div>
    <p v-if="loading" role="status">Memuat ruang AI...</p>
    <p v-if="error" class="ai-notice ai-error" role="alert">{{ error }}</p>
    <div v-if="config && !config.encryptionReady" class="ai-notice">Credential memerlukan kunci enkripsi pada server. <RouterLink to="/ai/settings">Lihat pengaturan</RouterLink></div>
    <div v-else-if="config && !credentials.length" class="ai-notice">Tambahkan credential AI untuk memulai percakapan. <RouterLink to="/ai/settings">Tambah credential</RouterLink></div>
    <div v-if="config" class="ai-workspace">
      <aside class="ai-history" aria-label="Riwayat percakapan">
        <button class="secondary-button" :disabled="sending || opening" @click="resetConversation"><Plus :size="18" />Percakapan baru</button>
        <h2>Riwayat pribadi</h2>
        <p v-if="!conversations.length" class="ai-muted">Percakapan tersimpan di workspace ini setelah AI menjawab.</p>
        <div v-for="item in conversations" :key="item.id" class="ai-history-row" :class="{ active: conversation?.id === item.id }">
          <button :disabled="sending || opening" :aria-current="conversation?.id === item.id ? 'true' : undefined" @click="openConversation(item.id)"><strong>{{ item.title }}</strong><small>{{ item.startMonth && item.startMonth !== item.month ? `${item.startMonth} sampai ${item.month}` : item.month }}</small></button>
          <button class="ai-icon-action" :disabled="sending" :aria-label="`Hapus percakapan ${item.title}`" @click="deleteId = item.id"><Trash2 :size="16" /></button>
        </div>
        <div v-if="deleteId" class="ai-confirm" role="group" aria-label="Konfirmasi hapus percakapan"><p>Hapus percakapan ini?</p><button class="secondary-button" :disabled="deleting" @click="deleteId = null">Batal</button><button class="secondary-button" :disabled="deleting" @click="removeConversation">Hapus</button></div>
      </aside>
      <div class="ai-chat-panel">
        <div class="ai-chat-options">
          <label>Credential dan model<select v-model="credentialId" :disabled="sending || opening"><option value="" disabled>Pilih credential</option><option v-for="item in credentials" :key="item.id" :value="item.id">{{ item.name }} · {{ item.model }}</option></select></label>
          <fieldset class="ai-period-range" :disabled="sending || opening || !!conversation" aria-describedby="ai-range-help">
            <legend class="sr-only">Periode Finance</legend>
            <div class="ai-period-inputs">
              <label>Periode awal<input v-model="startMonth" type="month" min="1900-01" :max="month || '2200-12'" @change="changePeriod" /></label>
              <label>Periode akhir<input v-model="month" type="month" :min="startMonth || '1900-01'" max="2200-12" @change="changePeriod" /></label>
            </div>
          </fieldset>
          <label>Simulasi pengurangan pos fleksibel<select v-model.number="reduction" :disabled="sending || opening" @change="invalidatePreview"><option v-for="value in [0, 5, 10, 15, 20, 30, 50]" :key="value" :value="value">{{ value }}%</option></select></label>
          <p id="ai-range-help" class="ai-range-help">Maksimal 12 bulan. Budget dan simulasi memakai periode akhir.<span v-if="conversation"> Mulai percakapan baru untuk mengganti rentang.</span></p>
        </div>
        <p v-if="!loading && rangeError" class="ai-notice ai-error" role="alert">{{ rangeError }}</p>
        <div class="ai-source-bar"><span>{{ config.settings.financeEnabled ? 'Sumber · Hubby Finance' : 'Sumber Finance belum diaktifkan' }}</span><button class="ai-text-button" :disabled="sending || opening || previewBusy || !!rangeError" @click="showContext">{{ previewBusy ? 'Memuat...' : 'Periksa data yang akan dikirim' }}</button></div>
        <AIContextCard v-if="preview" :context="preview" />
        <div class="ai-messages" aria-label="Percakapan">
          <div v-if="!conversation && !sending" class="ai-empty"><Sparkles :size="32" aria-hidden="true" /><h2>Apa yang ingin kamu evaluasi?</h2><p>Pilih periode, lalu tanyakan pola pengeluaran atau dampak penyesuaian budget.</p><div class="ai-prompts"><button v-for="prompt in prompts" :key="prompt" @click="question = prompt">{{ prompt }}</button></div></div>
          <article v-for="item in conversation?.messages || []" :key="item.id" class="ai-message" :class="`ai-message-${item.role}`"><header>{{ item.role === 'user' ? 'Anda' : 'Hubby AI' }}<small v-if="item.model">{{ item.model }}</small></header><AIAnswer v-if="item.role === 'assistant'" :content="item.content" /><div v-else class="ai-answer">{{ item.content }}</div><AIContextCard v-if="item.context" :context="item.context" /></article>
          <article v-if="sending" class="ai-message ai-message-user"><header>Anda</header><div class="ai-answer">{{ pendingQuestion }}</div></article>
          <p v-if="sending" class="ai-processing" role="status">Mengambil data dan menyusun analisis...</p>
          <div ref="messageEnd" />
        </div>
        <form class="ai-composer" @submit.prevent="send">
          <label for="ai-question">Pertanyaan Anda</label>
          <textarea id="ai-question" v-model="question" rows="3" maxlength="6000" :disabled="sending" placeholder="Contoh: pos mana yang paling memungkinkan saya kurangi?" required />
          <div class="ai-composer-footer"><p>Dengan mengirim, pertanyaan, hingga 6 balasan terakhir, dan konteks yang diaktifkan diteruskan ke {{ selectedCredential?.name || 'provider pilihan' }}. Tinjau saran sebelum bertindak.</p><button class="primary-button" :disabled="!ready || question.trim().length < 2"><Send :size="17" />{{ sending ? 'Memproses...' : 'Kirim' }}</button></div>
        </form>
      </div>
    </div>
  </section>
</template>
