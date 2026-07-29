<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Pyramid, HeartPulse, ShieldCheck, House, ChartNoAxesCombined,
  Scale, Umbrella, BookOpenText, ArrowUpRight, BadgeCheck,
} from '@lucide/vue'
import { api } from '../services/api'

const modules = [
  { title: 'Piramida keuangan', text: 'Lihat urutan prioritas dan progres fondasi keuanganmu.', icon: Pyramid, state: 'Tersedia', tone: 'sage', route: '/modules/pyramid' },
  { title: 'Financial check-up', text: 'Periksa rasio tabungan, kewajiban, dan likuiditas.', icon: HeartPulse, state: 'Tersedia', tone: 'rose', route: '/modules/checkup' },
  { title: 'Dana darurat', text: 'Hitung target dana aman berdasarkan kebutuhan bulanan.', icon: ShieldCheck, state: 'Tersedia', tone: 'sand', route: '/modules/emergency-fund' },
  { title: 'Simulasi KPR', text: 'Uji uang muka, tenor, bunga, dan kemampuan cicilan.', icon: House, state: 'Tersedia', tone: 'blue', route: '/modules/mortgage' },
  { title: 'Monitor investasi', text: 'Pantau nilai, imbal hasil, dan distribusi portofolio.', icon: ChartNoAxesCombined, state: 'Tersedia', tone: 'lilac', route: '/modules/investments' },
  { title: 'Rebalancing', text: 'Bandingkan alokasi saat ini dengan komposisi ideal.', icon: Scale, state: 'Tersedia', tone: 'moss', route: '/modules/rebalancing' },
  { title: 'Persiapan pensiun', text: 'Proyeksikan kebutuhan pensiun dengan pendekatan 4%.', icon: Umbrella, state: 'Tersedia', tone: 'rose', route: '/modules/retirement' },
  { title: 'Glosarium finansial', text: 'Pahami istilah keuangan dalam bahasa yang sederhana.', icon: BookOpenText, state: 'Tersedia', tone: 'blue', route: '/modules/glossary' },
]

const loading = ref(true)
const pyramid = ref({ levels: [] })
const completedLevels = computed(() => pyramid.value.levels.filter((level) => level.progress >= 100).length)
const totalLevels = computed(() => pyramid.value.levels.length || 7)
const nextLevel = computed(() => pyramid.value.levels.find((level) => level.progress < 100))
async function loadOverview() {
  loading.value = true
  try { pyramid.value = await api.pyramid() }
  catch { pyramid.value = { levels: [] } }
  finally { loading.value = false }
}
function handleWorkspaceChange() { loadOverview() }
onMounted(() => {
  loadOverview()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page">
    <div class="page-heading compact">
      <div><p class="eyebrow">Toolkit pribadi</p><h1>Perencanaan keuangan</h1><p>Semua kalkulator dari workbook, dirapikan menjadi alat yang mudah dipakai.</p></div>
    </div>
    <div class="insight-banner" :class="{ shimmer: loading }">
      <span><BadgeCheck :size="24" /></span>
      <div>
        <strong>{{ completedLevels }} dari {{ totalLevels }} fondasi sudah terpenuhi</strong>
        <p v-if="nextLevel">Fokus berikutnya: {{ nextLevel.title }} — {{ nextLevel.progress.toFixed(0) }}% selesai.</p>
        <p v-else>Seluruh fondasi sudah terpenuhi. Pertahankan dan evaluasi secara berkala.</p>
      </div>
      <RouterLink class="secondary-button" to="/modules/pyramid">Lihat piramida <ArrowUpRight :size="16" /></RouterLink>
    </div>
    <div class="modules-grid">
      <article v-for="item in modules" :key="item.title" class="module-card">
        <span class="module-icon" :class="`tone-${item.tone}`"><component :is="item.icon" :size="23" /></span>
        <span class="module-state" :class="{ soon: item.state === 'Segera' }">{{ item.state }}</span>
        <h2>{{ item.title }}</h2>
        <p>{{ item.text }}</p>
        <RouterLink v-if="item.route" :to="item.route">Buka modul <ArrowUpRight :size="15" /></RouterLink>
        <button v-else disabled>Buka modul <ArrowUpRight :size="15" /></button>
      </article>
    </div>
  </section>
</template>
