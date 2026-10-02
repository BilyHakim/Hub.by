<script setup>
import { ref } from 'vue'
import { Eye, EyeOff, HeartHandshake, LockKeyhole, Mail } from '@lucide/vue'
import { api } from '../services/api'

const emit = defineEmits(['authenticated'])
const email = ref('')
const password = ref('')
const showPassword = ref(false)
const submitting = ref(false)
const errorMessage = ref('')

async function submit() {
  errorMessage.value = ''
  submitting.value = true
  try {
    await api.login({ email: email.value, password: password.value })
    password.value = ''
    emit('authenticated')
  } catch (error) {
    errorMessage.value = error instanceof TypeError
      ? 'Tidak dapat terhubung ke server. Periksa koneksi lalu coba masuk kembali.'
      : error.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <RouterLink class="login-back" to="/">Kembali ke Hub.by</RouterLink>
    <section class="login-card">
      <div class="login-brand">
        <span class="login-brand-mark"><HeartHandshake :size="29" stroke-width="1.8" /></span>
        <div><strong>hubby</strong><small>personal hub</small></div>
      </div>
      <div class="login-heading">
        <p class="eyebrow">Ruang personalmu</p>
        <h1>Selamat datang kembali</h1>
        <p>Masuk untuk membuka semua modul pribadimu.</p>
      </div>
      <form class="login-form" @submit.prevent="submit">
        <label>
          Email
          <span class="login-input">
            <Mail :size="18" />
            <input v-model.trim="email" type="email" autocomplete="username" placeholder="nama@email.com" autofocus required />
          </span>
        </label>
        <label>
          Kata sandi
          <span class="login-input">
            <LockKeyhole :size="18" />
            <input v-model="password" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" placeholder="Masukkan kata sandi" minlength="10" required />
            <button type="button" :aria-label="showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'" @click="showPassword = !showPassword">
              <EyeOff v-if="showPassword" :size="17" />
              <Eye v-else :size="17" />
            </button>
          </span>
        </label>
        <p v-if="errorMessage" class="login-error" role="alert">{{ errorMessage }}</p>
        <button class="primary-button login-submit" :disabled="submitting">{{ submitting ? 'Memeriksa...' : 'Masuk ke Hubby' }}</button>
      </form>
    </section>
    <p class="login-footnote">Hubby · Semua yang penting, dalam satu tempat</p>
  </main>
</template>

<style scoped>
.login-page { background: #f6f4ef; overflow: visible; }
.login-page::before, .login-page::after { display: none; }
.login-card { background: #fffefa; backdrop-filter: none; }
.login-back { display: inline-flex; align-items: center; min-height: 44px; padding: 8px; font-size: .9rem; text-decoration: underline; text-underline-offset: 4px; }
.login-page :focus-visible { outline: 3px solid #304c42; outline-offset: 4px; }
.login-heading > p:last-child, .login-footnote { color: #526159; font-size: .875rem; }
.login-input { border-color: #758079; color: #526159; }
.login-input button { width: 44px; height: 44px; color: #526159; }
.login-input input { font-size: 1rem; }
.login-input input::placeholder { color: #526159; opacity: 1; }
.login-form label { font-size: .875rem; }
.login-error { color: #853b34; font-size: .875rem; }
.login-submit { min-height: 44px; }
</style>
