<script setup>
import { computed, ref } from 'vue'
import { ArrowLeft, BookOpenText, Search, X } from '@lucide/vue'
import { glossaryTerms } from '../data/glossary'

const search = ref('')
const category = ref('Semua')
const categories = ['Semua', ...new Set(glossaryTerms.map((item) => item.category))]
const filteredTerms = computed(() => {
  const needle = search.value.trim().toLocaleLowerCase('id-ID')
  return glossaryTerms.filter((item) => {
    const matchesCategory = category.value === 'Semua' || item.category === category.value
    const matchesSearch = !needle || `${item.term} ${item.definition}`.toLocaleLowerCase('id-ID').includes(needle)
    return matchesCategory && matchesSearch
  })
})
const groupedTerms = computed(() => {
  const groups = new Map()
  filteredTerms.value.forEach((item) => {
    if (!groups.has(item.category)) groups.set(item.category, [])
    groups.get(item.category).push(item)
  })
  return [...groups].map(([name, items]) => ({ name, items }))
})
</script>

<template>
  <section class="page planning-detail-page">
    <RouterLink class="back-link" to="/finance/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div><p class="eyebrow">Kamus sederhana</p><h1>Glosarium finansial</h1><p>Istilah keuangan dari workbook, dijelaskan dengan bahasa yang mudah dipahami.</p></div>
    </div>
    <article class="glossary-hero">
      <span><BookOpenText :size="26" /></span>
      <div><strong>{{ glossaryTerms.length }} istilah finansial</strong><p>Cari istilah tentang arus kas, investasi, kredit, dan pensiun.</p></div>
      <label class="glossary-search"><Search :size="17" /><input v-model="search" placeholder="Cari istilah atau penjelasan..."><button v-if="search" @click="search = ''"><X :size="15" /></button></label>
    </article>
    <div class="glossary-filters">
      <button v-for="item in categories" :key="item" :class="{ active: category === item }" @click="category = item">{{ item }}</button>
    </div>
    <div v-if="groupedTerms.length" class="glossary-groups">
      <section v-for="group in groupedTerms" :key="group.name">
        <div class="glossary-group-heading"><span>{{ group.name }}</span><small>{{ group.items.length }} istilah</small></div>
        <div class="glossary-grid">
          <article v-for="item in group.items" :key="item.term" class="panel glossary-card">
            <span>{{ item.term.slice(0, 1) }}</span>
            <div><h2>{{ item.term }}</h2><p>{{ item.definition }}</p></div>
          </article>
        </div>
      </section>
    </div>
    <div v-else class="panel empty-module">Istilah “{{ search }}” belum ditemukan.</div>
  </section>
</template>
