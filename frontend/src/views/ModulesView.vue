<script setup>
import {
  Pyramid, HeartPulse, ShieldCheck, House, ChartNoAxesCombined,
  Scale, Umbrella, BookOpenText, ArrowUpRight, BadgeCheck,
} from '@lucide/vue'

const modules = [
  { title: 'Piramida keuangan', text: 'Lihat urutan prioritas dan progres fondasi keuanganmu.', icon: Pyramid, state: 'Tersedia', tone: 'sage' },
  { title: 'Financial check-up', text: 'Periksa rasio tabungan, kewajiban, dan likuiditas.', icon: HeartPulse, state: 'Tersedia', tone: 'rose' },
  { title: 'Dana darurat', text: 'Hitung target dana aman berdasarkan kebutuhan bulanan.', icon: ShieldCheck, state: 'Tersedia', tone: 'sand' },
  { title: 'Simulasi KPR', text: 'Uji uang muka, tenor, bunga, dan kemampuan cicilan.', icon: House, state: 'Segera', tone: 'blue' },
  { title: 'Monitor investasi', text: 'Pantau nilai, imbal hasil, dan distribusi portofolio.', icon: ChartNoAxesCombined, state: 'Tersedia', tone: 'lilac' },
  { title: 'Rebalancing', text: 'Bandingkan alokasi saat ini dengan komposisi ideal.', icon: Scale, state: 'Segera', tone: 'moss' },
  { title: 'Persiapan pensiun', text: 'Proyeksikan kebutuhan pensiun dengan pendekatan 4%.', icon: Umbrella, state: 'Segera', tone: 'rose' },
  { title: 'Glosarium finansial', text: 'Pahami istilah keuangan dalam bahasa yang sederhana.', icon: BookOpenText, state: 'Tersedia', tone: 'blue' },
]
</script>

<template>
  <section class="page">
    <div class="page-heading compact">
      <div><p class="eyebrow">Toolkit pribadi</p><h1>Perencanaan keuangan</h1><p>Semua kalkulator dari workbook, dirapikan menjadi alat yang mudah dipakai.</p></div>
    </div>
    <div class="insight-banner">
      <span><BadgeCheck :size="24" /></span>
      <div><strong>4 dari 7 fondasi sudah terpenuhi</strong><p>Fokus berikutnya: lengkapi dana darurat hingga 6 bulan pengeluaran.</p></div>
      <button class="secondary-button">Lihat piramida <ArrowUpRight :size="16" /></button>
    </div>
    <div class="modules-grid">
      <article v-for="item in modules" :key="item.title" class="module-card">
        <span class="module-icon" :class="`tone-${item.tone}`"><component :is="item.icon" :size="23" /></span>
        <span class="module-state" :class="{ soon: item.state === 'Segera' }">{{ item.state }}</span>
        <h2>{{ item.title }}</h2>
        <p>{{ item.text }}</p>
        <button :disabled="item.state === 'Segera'">Buka modul <ArrowUpRight :size="15" /></button>
      </article>
    </div>
  </section>
</template>
