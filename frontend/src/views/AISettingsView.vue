<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ArrowLeft, Check, KeyRound, Pencil, Plus, Save, Trash2 } from '@lucide/vue'
import { api } from '../services/api'

const loading = ref(true)
const busy = ref(false)
const error = ref('')
const status = ref('')
const config = ref(null)
const formOpen = ref(false)
const envText = ref('')
const models = ref([])
const modelsLoading = ref(false)
const modelsStatus = ref('')
const modelSearch = ref('')
let modelController
let modelEpoch = 0
const deleteTarget = ref(null)
const clearHistory = ref(false)
const preferences = reactive({ defaultCredentialId: null, financeEnabled: false, includeBalances: false, includeGoals: false, includeObligations: false, retentionDays: 30, dailyLimit: 20 })
const credential = reactive({ id: 0, name: '', provider: 'commandcode', baseUrl: 'https://api.commandcode.ai/provider/v1', format: 'chat_completions', apiKey: '', model: '', maxTokens: 4096, zdr: true, enabled: true })
let controller
let epoch = 0
const activeCredentials = computed(() => (config.value?.credentials || []).filter(item => item.enabled))
const filteredModels = computed(() => models.value.filter(model => model.toLowerCase().includes(modelSearch.value.trim().toLowerCase())))
function resetModels() {
  modelController?.abort()
  modelEpoch++
  models.value = []
  modelSearch.value = ''
  modelsStatus.value = ''
  modelsLoading.value = false
}
watch(() => [credential.provider, credential.baseUrl, credential.format, credential.apiKey, credential.zdr], resetModels, { flush: 'sync' })
watch(() => [credential.provider, credential.baseUrl, credential.format], () => { credential.model = '' }, { flush: 'sync' })
async function fetchModels() {
  resetModels()
  const current = modelEpoch
  modelController = new AbortController()
  modelsLoading.value = true
  error.value = ''
  try {
    const result = await api.aiModels({ ...credential }, modelController.signal)
    if (current !== modelEpoch) return
    models.value = result.models
    modelsStatus.value = result.models.length
      ? `${result.models.length} model tersedia. Pilih model di bawah. Akses dan kuota diperiksa saat chat dikirim.`
      : 'Tidak ada model untuk format API ini. Periksa format API atau masukkan ID model secara manual.'
  } catch (e) {
    if (current === modelEpoch && e.name !== 'AbortError') error.value = e.message
  } finally {
    if (current === modelEpoch) modelsLoading.value = false
  }
}
const presets = {
  commandcode: ['https://api.commandcode.ai/provider/v1', 'chat_completions'],
  openai: ['https://api.openai.com/v1', 'chat_completions'],
  anthropic: ['https://api.anthropic.com/v1', 'messages'],
  ollama: ['http://localhost:11434', 'ollama'],
  custom: ['', 'chat_completions'],
}
function changeProvider() {
  const [baseUrl, format] = presets[credential.provider]
  Object.assign(credential, { baseUrl, format, apiKey: '' })
  models.value = []
}
function editCredential(item) {
  resetModels()
  Object.assign(credential, item || { id: 0, name: '', provider: 'commandcode', baseUrl: presets.commandcode[0], format: 'chat_completions', model: '', maxTokens: 4096, zdr: true, enabled: true })
  credential.apiKey = ''
  envText.value = ''
  models.value = []
  formOpen.value = true
  error.value = ''
  status.value = ''
}
function closeForm() {
  resetModels()
  formOpen.value = false
  credential.apiKey = ''
  envText.value = ''
}
function importEnv() {
  const values = {}
  const allowed = ['AI_PROVIDER', 'AI_API_BASE_URL', 'AI_API_KEY', 'AI_CHAT_MODEL', 'AI_API_FORMAT', 'AI_MAX_TOKENS', 'COMMANDCODE_ZDR']
  try {
    for (const line of envText.value.split(/\r?\n/)) {
      if (!line.trim() || line.trim().startsWith('#')) continue
      const match = line.match(/^\s*(?:export\s+)?([A-Z_]+)\s*=\s*(.*?)\s*$/)
      if (!match || !allowed.includes(match[1])) throw new Error('Gunakan hanya variabel AI yang tercantum pada panduan impor.')
      let value = match[2]
      if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) value = value.slice(1, -1)
      values[match[1]] = value
    }
    let provider = values.AI_PROVIDER
    if (values.AI_API_BASE_URL?.startsWith('https://api.commandcode.ai/')) provider = 'commandcode'
    if (provider === 'api') provider = 'custom'
    if (provider && !presets[provider]) throw new Error('Provider impor tidak didukung.')
    if (provider) { credential.provider = provider; changeProvider() }
    if (values.AI_API_BASE_URL) credential.baseUrl = values.AI_API_BASE_URL
    if (values.AI_API_KEY) credential.apiKey = values.AI_API_KEY
    if (values.AI_CHAT_MODEL) credential.model = values.AI_CHAT_MODEL
    if (values.AI_API_FORMAT) credential.format = values.AI_API_FORMAT
    if (values.AI_MAX_TOKENS) credential.maxTokens = Number(values.AI_MAX_TOKENS)
    if (values.COMMANDCODE_ZDR) {
      const zdr = values.COMMANDCODE_ZDR.toLowerCase()
      if (!['true', 'false', '1', '0'].includes(zdr)) throw new Error('COMMANDCODE_ZDR harus true, false, 1, atau 0.')
      credential.zdr = zdr === 'true' || zdr === '1'
    }
    envText.value = ''
    status.value = 'Konfigurasi diisi ke formulir. Periksa lalu simpan credential.'
    error.value = ''
  } catch (e) { error.value = e.message }
}
async function load() {
  controller?.abort()
  controller = new AbortController()
  const current = ++epoch
  closeForm()
  config.value = null
  deleteTarget.value = null
  clearHistory.value = false
  loading.value = true
  busy.value = false
  error.value = ''
  status.value = ''
  try {
    const result = await api.aiSettings(controller.signal)
    if (current !== epoch) return
    config.value = result
    Object.assign(preferences, result.settings)
  } catch (e) { if (current === epoch && e.name !== 'AbortError') error.value = e.message }
  finally { if (current === epoch) loading.value = false }
}
async function run(action, message, refreshPreferences = false) {
  const current = epoch
  busy.value = true
  error.value = ''
  status.value = ''
  try {
    await action()
    if (current !== epoch) return
    const result = await api.aiSettings(controller.signal)
    if (current !== epoch) return
    config.value = result
    if (refreshPreferences) Object.assign(preferences, result.settings)
    if (preferences.defaultCredentialId && !result.credentials.some(item => item.id === preferences.defaultCredentialId && item.enabled)) preferences.defaultCredentialId = null
    status.value = message
  } catch (e) { if (current === epoch && e.name !== 'AbortError') error.value = e.message }
  finally { if (current === epoch) busy.value = false }
}
async function saveCredential() {
  const current = epoch
  const input = { ...credential }
  await run(async () => {
    const result = await api.saveAICredential(input)
    if (current !== epoch) return
    credential.id = result.id
    credential.apiKey = ''
    closeForm()
  }, 'Credential tersimpan dan API key dienkripsi.')
}
async function testCredential(item) {
  const current = epoch
  await run(async () => {
    const result = await api.testAICredential(item.id, controller.signal)
    if (current !== epoch) return
    if (formOpen.value && credential.id === item.id) models.value = result.models
  }, 'Koneksi berhasil. Izin generasi dan kuota model diperiksa saat mengirim chat.')
}
async function removeCredential() {
  const id = deleteTarget.value?.id
  if (!id) return
  const current = epoch
  await run(async () => {
    await api.deleteAICredential(id)
    if (current !== epoch) return
    if (credential.id === id) closeForm()
    deleteTarget.value = null
  }, 'Credential dihapus.')
}
async function removeHistory() {
  const current = epoch
  await run(async () => { await api.clearAIConversations(); if (current === epoch) clearHistory.value = false }, 'Riwayat chat pada workspace ini dihapus. Metadata penggunaan tetap mengikuti masa retensi.')
}
onMounted(() => { load(); window.addEventListener('hubby:workspace-changed', load) })
onBeforeUnmount(() => { epoch++; controller?.abort(); resetModels(); window.removeEventListener('hubby:workspace-changed', load) })
</script>

<template>
  <section class="page ai-page">
    <div class="page-heading compact"><div><p class="eyebrow">Hubby AI</p><h1>Pengaturan AI</h1><p>Credential dan izin data pribadi untuk workspace aktif.</p></div><RouterLink to="/ai" class="secondary-button"><ArrowLeft :size="18" />Percakapan</RouterLink></div>
    <p v-if="loading" role="status">Memuat pengaturan...</p>
    <p v-if="error" role="alert" class="ai-notice ai-error">{{ error }}</p>
    <p v-if="status" role="status" class="ai-notice"><Check :size="18" />{{ status }}</p>
    <div v-if="config && !config.encryptionReady" class="ai-notice"><div><strong>Pasang kunci enkripsi server</strong><p>Administrator perlu mengisi AI_ENCRYPTION_KEY dengan kunci acak 32 byte dalam format Base64. Simpan kunci ini bersama backup server agar credential dapat dipulihkan.</p></div></div>
    <template v-if="config">
      <section class="ai-settings-section">
        <div class="ai-section-heading"><div><h2><KeyRound :size="21" />Credential AI</h2><p>Simpan beberapa koneksi dan pilih credential yang ingin digunakan saat chat.</p></div><button class="secondary-button" :disabled="busy || !config.encryptionReady" @click="editCredential(null)"><Plus :size="17" />Tambah credential</button></div>
        <p v-if="!config.credentials.length" class="ai-muted">Belum ada credential. Tambahkan API key melalui formulir atau impor konfigurasi env.</p>
        <div v-for="item in config.credentials" :key="item.id" class="ai-credential-row"><div><strong>{{ item.name }}</strong><span>{{ item.provider }} · {{ item.enabled ? 'Aktif' : 'Nonaktif' }} · {{ item.model }}</span><small>{{ item.baseUrl }} · API key tersimpan terenkripsi</small></div><div class="ai-row-actions"><button class="secondary-button" :disabled="busy || !item.enabled || !config.encryptionReady" @click="testCredential(item)">Tes koneksi</button><button class="ai-icon-action" :disabled="busy || !config.encryptionReady" :aria-label="`Edit ${item.name}`" @click="editCredential(item)"><Pencil :size="18" /></button><button class="ai-icon-action" :disabled="busy" :aria-label="`Hapus ${item.name}`" @click="deleteTarget = item"><Trash2 :size="18" /></button></div></div>
        <div v-if="deleteTarget" class="ai-confirm" role="group" aria-label="Konfirmasi hapus credential"><p>Hapus credential {{ deleteTarget.name }}?</p><button class="secondary-button" :disabled="busy" @click="deleteTarget = null">Batal</button><button class="secondary-button" :disabled="busy" @click="removeCredential">Hapus credential</button></div>
        <form v-if="formOpen" class="ai-credential-form" @submit.prevent="saveCredential">
          <h3>{{ credential.id ? 'Edit credential' : 'Credential baru' }}</h3>
          <fieldset :disabled="busy" class="ai-form-grid">
            <label>Nama credential<input v-model="credential.name" required minlength="2" maxlength="100" placeholder="CommandCode pribadi" /></label>
            <label>Provider<select v-model="credential.provider" @change="changeProvider"><option value="commandcode">CommandCode</option><option value="openai">OpenAI</option><option value="anthropic">Anthropic</option><option value="ollama">Ollama</option><option value="custom">API kompatibel lainnya</option></select></label>
            <label>Base URL<input v-model="credential.baseUrl" type="url" required :readonly="['commandcode', 'openai', 'anthropic'].includes(credential.provider)" /></label>
            <label>Format API<select v-model="credential.format" :disabled="['openai', 'anthropic', 'ollama'].includes(credential.provider)"><option value="chat_completions">Chat Completions</option><option value="messages">Anthropic Messages</option><option v-if="credential.provider === 'ollama'" value="ollama">Ollama</option></select></label>
            <label>API key<input v-model="credential.apiKey" type="password" autocomplete="new-password" :required="!credential.id && credential.provider !== 'ollama'" :placeholder="credential.id ? 'Kosongkan untuk mempertahankan key' : 'Masukkan API key'" maxlength="8192" /></label>
            <div class="ai-model-picker">
              <button type="button" class="secondary-button" :disabled="modelsLoading || (!credential.apiKey.trim() && !credential.id && credential.provider !== 'ollama')" @click="fetchModels">{{ modelsLoading ? 'Mengambil model...' : 'Ambil daftar model' }}</button>
              <p id="ai-model-status" class="ai-muted" role="status">{{ modelsLoading ? 'Menghubungi provider untuk mengambil daftar model...' : modelsStatus || 'Isi API key, lalu ambil daftar model. Credential belum perlu disimpan.' }}</p>
              <label v-if="models.length">Cari model<input v-model="modelSearch" type="search" placeholder="Cari nama atau ID model" /></label>
              <label>Model
                <select v-if="models.length" v-model="credential.model" required aria-describedby="ai-model-status">
                  <option disabled value="">Pilih model</option>
                  <option v-if="credential.model && !filteredModels.includes(credential.model)" :value="credential.model">{{ credential.model }} (pilihan saat ini)</option>
                  <option v-for="model in filteredModels" :key="model" :value="model">{{ model }}</option>
                </select>
                <input v-else v-model="credential.model" required maxlength="200" placeholder="Ambil daftar model atau isi ID secara manual" aria-describedby="ai-model-status" />
              </label>
              <p v-if="models.length && !filteredModels.length" class="ai-muted" role="status">Tidak ada model yang cocok dengan pencarian.</p>
            </div>
            <label>Batas token jawaban<input v-model.number="credential.maxTokens" type="number" required min="256" max="8192" step="1" /></label>
            <label class="ai-checkbox"><input v-model="credential.enabled" type="checkbox" />Credential aktif</label>
            <label v-if="credential.provider === 'commandcode'" class="ai-checkbox ai-form-wide"><input v-model="credential.zdr" type="checkbox" />Wajibkan zero data retention (ZDR). Model yang tidak mendukung akan menolak permintaan.</label>
          </fieldset>
          <p class="ai-muted" v-if="credential.provider === 'commandcode'">Model Claude menggunakan format Anthropic Messages. Paket CommandCode harus mendukung Provider API.</p>
          <p class="ai-muted" v-if="['custom', 'ollama'].includes(credential.provider)">Endpoint lokal atau jaringan privat perlu diizinkan administrator melalui AI_ALLOWED_HOSTS dengan host dan port yang tepat.</p>
          <details class="ai-env-import"><summary>Impor konfigurasi env</summary><p>Variabel yang didukung AI_PROVIDER, AI_API_BASE_URL, AI_API_KEY, AI_CHAT_MODEL, AI_API_FORMAT, AI_MAX_TOKENS, COMMANDCODE_ZDR.</p><label>Konfigurasi<textarea v-model="envText" rows="6" autocomplete="off" spellcheck="false" placeholder="AI_API_BASE_URL=https://api.commandcode.ai/provider/v1&#10;AI_API_KEY=...&#10;AI_CHAT_MODEL=..." /></label><button type="button" class="secondary-button" :disabled="busy || !envText.trim()" @click="importEnv">Isi formulir dari env</button></details>
          <div class="ai-row-actions"><button class="primary-button" :disabled="busy"><Save :size="17" />{{ busy ? 'Menyimpan...' : 'Simpan credential' }}</button><button type="button" class="secondary-button" :disabled="busy" @click="closeForm">Batal</button><button v-if="credential.id" type="button" class="secondary-button" :disabled="busy" @click="testCredential(credential)">Muat daftar model tersimpan</button></div>
        </form>
      </section>
      <form class="ai-settings-section" @submit.prevent="run(() => api.saveAISettings({ ...preferences }), 'Pengaturan AI tersimpan.', true)">
        <h2>Sumber data dan penggunaan</h2>
        <fieldset :disabled="busy" class="ai-preferences">
          <label>Credential default<select v-model="preferences.defaultCredentialId"><option :value="null">Pilih saat memulai chat</option><option v-for="item in activeCredentials" :key="item.id" :value="item.id">{{ item.name }} · {{ item.model }}</option></select></label>
          <label class="ai-checkbox"><input v-model="preferences.financeEnabled" type="checkbox" /><span><strong>Izinkan analisis Hubby Finance</strong><small>Kirim ringkasan arus kas empat periode, nama kategori, budget, dan simulasi ke provider saat Anda mengirim chat.</small></span></label>
          <div class="ai-scope-options"><label class="ai-checkbox"><input v-model="preferences.includeBalances" type="checkbox" :disabled="!preferences.financeEnabled" />Sertakan saldo agregat dan dana darurat</label><label class="ai-checkbox"><input v-model="preferences.includeGoals" type="checkbox" :disabled="!preferences.financeEnabled" />Sertakan tujuan keuangan</label><label class="ai-checkbox"><input v-model="preferences.includeObligations" type="checkbox" :disabled="!preferences.financeEnabled" />Sertakan sisa utang dan piutang</label></div>
          <div class="ai-form-grid"><label>Batas permintaan per 24 jam<input v-model.number="preferences.dailyLimit" type="number" min="1" max="200" required /></label><label>Retensi riwayat (hari)<input v-model.number="preferences.retentionDays" type="number" min="1" max="365" required /></label></div>
        </fieldset>
        <p class="ai-muted">Nama kategori dan tujuan dapat mengandung informasi pribadi. Riwayat tersimpan di database Hubby. Percakapan yang melewati retensi disembunyikan dan dibersihkan saat modul digunakan.</p>
        <button class="primary-button" :disabled="busy"><Save :size="17" />Simpan pengaturan</button>
      </form>
      <section class="ai-settings-section"><h2>Penggunaan dan riwayat</h2><p>{{ config.usage.requests }} permintaan dalam 24 jam terakhir · {{ config.usage.inputTokens.toLocaleString('id-ID') }} token masuk · {{ config.usage.outputTokens.toLocaleString('id-ID') }} token keluar.</p><p class="ai-muted">Percobaan gagal ikut dihitung pada batas permintaan. Token tercatat dari respons yang berhasil disimpan. Tagihan aktual mengikuti provider.</p><button class="secondary-button" :disabled="busy" @click="clearHistory = true"><Trash2 :size="17" />Hapus seluruh riwayat saya di workspace ini</button><div v-if="clearHistory" class="ai-confirm" role="group" aria-label="Konfirmasi hapus seluruh riwayat"><p>Hapus seluruh percakapan pribadi pada workspace aktif?</p><button class="secondary-button" :disabled="busy" @click="clearHistory = false">Batal</button><button class="secondary-button" :disabled="busy" @click="removeHistory">Hapus riwayat</button></div></section>
    </template>
  </section>
</template>
