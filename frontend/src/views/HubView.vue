<script setup>
import { ArrowUpRight, BarChart3, BookOpen, BookMarked, Clapperboard, Clock3, Film, PiggyBank, Sparkles, WalletCards } from '@lucide/vue'

const modules = [
  {
    name: 'Hubby Finance',
    description: 'Kelola arus kas, tujuan, investasi, dan rencana keuangan dalam satu ruang yang tenang.',
    to: '/finance',
    icon: WalletCards,
    tone: 'finance',
    features: [
      { icon: BarChart3, label: 'Ringkasan keuangan' },
      { icon: PiggyBank, label: 'Budget & tujuan' },
    ],
  },
  {
    name: 'Hubby Watch',
    description: 'Catat film dan series, lanjutkan episode terakhir, dan pahami waktu menontonmu.',
    to: '/watch',
    icon: Clapperboard,
    tone: 'watch',
    features: [
      { icon: Film, label: 'Pustaka tontonan' },
      { icon: Clock3, label: 'Waktu menonton' },
    ],
  },
  {
    name: 'Hubby Books',
    description: 'Bangun kebiasaan membaca, simpan pustaka, dan lanjutkan dari halaman terakhir.',
    to: '/books',
    icon: BookOpen,
    tone: 'books',
    features: [
      { icon: BookMarked, label: 'Progres halaman' },
      { icon: BarChart3, label: 'Aktivitas membaca' },
    ],
  },
]
</script>

<template>
  <main class="hub-home">
    <section class="hub-hero">
      <span class="hub-hero-mark"><Sparkles :size="24" /></span>
      <p class="eyebrow">Your personal hub</p>
      <h1>Satu tempat untuk hal-hal yang penting.</h1>
      <p>Pilih ruang yang ingin kamu buka hari ini. Setiap modul berdiri sendiri, dengan satu akun Hubby yang sama.</p>
    </section>

    <section class="hub-module-grid" aria-label="Modul Hubby">
      <RouterLink v-for="item in modules" :key="item.name" :to="item.to" class="hub-module-card" :class="`hub-module-${item.tone}`">
        <span class="hub-module-icon"><component :is="item.icon" :size="30" stroke-width="1.7" /></span>
        <span class="hub-module-state">Tersedia</span>
        <h2>{{ item.name }}</h2>
        <p>{{ item.description }}</p>
        <span class="hub-feature-list">
          <span v-for="feature in item.features" :key="feature.label"><component :is="feature.icon" :size="15" /> {{ feature.label }}</span>
        </span>
        <span class="hub-open-module">Buka modul <ArrowUpRight :size="17" /></span>
      </RouterLink>
    </section>
  </main>
</template>
