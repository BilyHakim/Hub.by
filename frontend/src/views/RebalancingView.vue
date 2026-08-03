<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ArrowDown, ArrowLeft, ArrowUp, CheckCircle2, Save, Scale } from '@lucide/vue'
import { api } from '../services/api'

const loading = ref(true)
const saving = ref(false)
const data = ref({ totalValue: 0, totalTargetAllocation: 0, isBalancedTarget: false, items: [] })
const holdings = ref([])
const currency = (value = 0) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value)
const targetTotal = computed(() => holdings.value.reduce((sum, item) => sum + Number(item.targetAllocation || 0), 0))
const targetValid = computed(() => Math.abs(targetTotal.value - 100) < 0.01)
const recommendations = computed(() => {
  const total = holdings.value.reduce((sum, item) => sum + item.currentValue, 0)
  return holdings.value.map((item) => {
    const targetValue = Math.round(total * Number(item.targetAllocation || 0) / 100)
    const difference = targetValue - item.currentValue
    return { ...item, source: item, currentAllocation: total ? item.currentValue / total * 100 : 0, targetValue, difference, action: difference > 0 ? 'buy' : difference < 0 ? 'sell' : 'hold' }
  })
})

async function load() {
  loading.value = true
  try {
    const [portfolio, result] = await Promise.all([api.investments(), api.rebalancing()])
    holdings.value = portfolio || []
    data.value = result
  } finally { loading.value = false }
}
async function save() {
  if (!targetValid.value) return
  saving.value = true
  try {
    await Promise.all(holdings.value.map((item) => api.updateInvestment(item.id, {
      assetType: item.assetType, name: item.name, platform: item.platform,
      purchaseValue: item.purchaseValue, currentValue: item.currentValue,
      targetAllocation: Number(item.targetAllocation),
    })))
    await load()
  } finally { saving.value = false }
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
    <RouterLink class="back-link" to="/finance/modules"><ArrowLeft :size="16" /> Kembali ke perencanaan</RouterLink>
    <div class="page-heading compact">
      <div><p class="eyebrow">Komposisi ideal</p><h1>Rebalancing investasi</h1><p>Bandingkan alokasi aktual dengan target dan lihat nominal penyesuaiannya.</p></div>
    </div>
    <article class="planning-hero">
      <span><Scale :size="25" /></span>
      <div><small>Total nilai portofolio</small><strong>{{ currency(data.totalValue) }}</strong></div>
      <div class="allocation-total" :class="{ invalid: !targetValid }">
        <small>Total target</small><strong>{{ targetTotal.toFixed(2) }}%</strong><span>{{ targetValid ? 'Siap dihitung' : 'Harus tepat 100%' }}</span>
      </div>
    </article>
    <article class="panel module-table-card" :class="{ shimmer: loading }">
      <div class="panel-heading"><div><h2>Target per aset</h2><p>Ubah kolom target seperti area hijau pada workbook.</p></div></div>
      <div v-if="holdings.length" class="rebalance-table">
        <div class="rebalance-head"><span>Aset</span><span>Nilai sekarang</span><span>Alokasi sekarang</span><span>Target</span><span>Nilai target</span><span>Penyesuaian</span></div>
        <div v-for="item in recommendations" :key="item.id" class="rebalance-row">
          <div><strong>{{ item.name }}</strong><small>{{ item.assetType }}</small></div>
          <strong>{{ currency(item.currentValue) }}</strong>
          <span>{{ item.currentAllocation.toFixed(2) }}%</span>
          <label><input v-model.number="item.source.targetAllocation" type="number" min="0" max="100" step="0.01"><span>%</span></label>
          <strong>{{ currency(item.targetValue) }}</strong>
          <div class="rebalance-action" :class="item.action">
            <ArrowUp v-if="item.action === 'buy'" :size="15" />
            <ArrowDown v-else-if="item.action === 'sell'" :size="15" />
            <CheckCircle2 v-else :size="15" />
            <span>{{ item.action === 'buy' ? 'Tambah' : item.action === 'sell' ? 'Kurangi' : 'Sesuai' }}</span>
            <strong>{{ currency(Math.abs(item.difference)) }}</strong>
          </div>
        </div>
      </div>
      <div v-else class="empty-module">Tambahkan aset melalui Monitor Investasi terlebih dahulu.</div>
      <div class="rebalance-footer">
        <p>Nominal adalah rekomendasi matematis, bukan instruksi membeli atau menjual aset.</p>
        <button class="primary-button" :disabled="saving || !targetValid || !holdings.length" @click="save"><Save :size="16" />{{ saving ? 'Menyimpan...' : 'Simpan target' }}</button>
      </div>
    </article>
  </section>
</template>
