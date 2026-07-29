<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ArrowLeft, Check, Layers3, Sparkles } from '@lucide/vue'
import { api } from '../services/api'

const loading = ref(true)
const data = ref({ levels: [], overallProgress: 0, completedItems: 0, totalItems: 0 })
const updating = ref(new Set())

async function load() {
  loading.value = true
  try { data.value = await api.pyramid() } finally { loading.value = false }
}
async function toggle(item) {
  const previous = item.isCompleted
  item.isCompleted = !previous
  updating.value.add(item.id)
  try {
    await api.updatePyramidItem(item.id, item.isCompleted)
    await load()
  } catch {
    item.isCompleted = previous
  } finally {
    updating.value.delete(item.id)
  }
}
function handleWorkspaceChange() { load() }
onMounted(() => {
  load()
  window.addEventListener('hubby:workspace-changed', handleWorkspaceChange)
})
onBeforeUnmount(() => window.removeEventListener('hubby:workspace-changed', handleWorkspaceChange))
</script>

<template>
  <section class="page planning-detail-page">
    <RouterLink class="back-link" to="/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div><p class="eyebrow">Prioritas 1—7</p><h1>Piramida keuangan</h1><p>Bangun fondasi secara berurutan dan tandai pencapaian sesuai kondisimu.</p></div>
    </div>
    <div class="planning-hero pyramid-hero">
      <span><Layers3 :size="27" /></span>
      <div><small>Progress keseluruhan</small><strong>{{ data.completedItems }} dari {{ data.totalItems }} pencapaian</strong></div>
      <div class="hero-progress"><strong>{{ data.overallProgress.toFixed(0) }}%</strong><div class="progress-track"><span :style="{ width: `${data.overallProgress}%` }" /></div></div>
    </div>
    <div class="pyramid-levels" :class="{ shimmer: loading }">
      <article v-for="level in data.levels" :key="level.priority" class="panel pyramid-level">
        <div class="level-number">{{ level.priority }}</div>
        <div class="level-content">
          <div class="level-heading">
            <div><h2>{{ level.title }}</h2><p>{{ level.description }}</p></div>
            <span>{{ level.progress.toFixed(0) }}%</span>
          </div>
          <div class="progress-track"><span :style="{ width: `${level.progress}%` }" /></div>
          <div class="pyramid-checklist">
            <button v-for="item in level.items" :key="item.id" type="button" :class="{ completed: item.isCompleted }" :disabled="updating.has(item.id)" @click="toggle(item)">
              <span><Check v-if="item.isCompleted" :size="15" /></span>{{ item.title }}
            </button>
          </div>
        </div>
      </article>
    </div>
    <p class="planning-footnote"><Sparkles :size="15" /> Checklist mengikuti struktur prioritas pada workbook dan bersifat manual agar bisa disesuaikan dengan kondisi keluarga.</p>
  </section>
</template>

