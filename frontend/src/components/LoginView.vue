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
    errorMessage.value = error.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <section class="login-card">
      <div class="login-brand">
        <span class="login-brand-mark"><HeartHandshake :size="29" stroke-width="1.8" /></span>
        <div><strong>hubby</strong><small>finance</small></div>
      </div>
      <div class="login-heading">
        <p class="eyebrow">Ruang keuangan pribadi</p>
        <h1>Selamat datang kembali</h1>
        <p>Masuk untuk melanjutkan cerita keuanganmu.</p>
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
      <p class="login-security"><LockKeyhole :size="13" /> Sesi dilindungi dengan cookie aman.</p>
    </section>
    <p class="login-footnote">Hubby Finance · Ruang tenang untuk keuangan keluarga</p>
  </main>
</template>
