<script setup>
import { computed } from 'vue'
const props = defineProps({ context: { type: Object, required: true } })
const period = computed(() => props.context.periods?.[0])
const firstPeriod = computed(() => props.context.periods?.at(-1))
const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value || 0)
</script>

<template>
  <details class="ai-context-card">
    <summary>Data pendukung <span v-if="period">{{ firstPeriod.start }} sampai {{ period.end }}</span></summary>
    <div v-if="period" class="ai-context-body">
      <div class="ai-table-scroll" tabindex="0" role="region" aria-label="Perbandingan arus kas per periode">
        <table>
          <caption>Perbandingan arus kas per periode Finance</caption>
          <thead><tr><th scope="col">Periode</th><th scope="col">Pemasukan</th><th scope="col">Pengeluaran</th><th scope="col">Surplus</th><th scope="col">Status</th></tr></thead>
          <tbody><tr v-for="item in context.periods" :key="item.month"><th scope="row">{{ item.month }}<small>{{ item.start }} sampai {{ item.end }}</small></th><td>{{ money(item.income) }}</td><td>{{ money(item.expense) }}</td><td>{{ money(item.surplus) }}</td><td>{{ item.complete ? 'Telah berakhir' : 'Belum selesai' }}</td></tr></tbody>
        </table>
      </div>
      <h3>Rincian periode akhir: {{ period.month }}</h3>
      <p>{{ period.transactionCount }} transaksi tercatat · {{ period.complete ? 'Periode telah berakhir' : 'Periode belum selesai' }}</p>
      <dl class="ai-context-numbers">
        <div><dt>Pemasukan</dt><dd>{{ money(period.income) }}</dd></div>
        <div><dt>Pengeluaran</dt><dd>{{ money(period.expense) }}</dd></div>
        <div><dt>Surplus tercatat</dt><dd>{{ money(period.surplus) }}</dd></div>
      </dl>
      <h3>Simulasi pengeluaran fleksibel</h3>
      <p>Pengurangan {{ context.reductionPercent }}% dari pos discretionary yang tercatat memberi ruang {{ money(context.scenarioSaving) }}. Surplus skenario {{ money(context.scenarioSurplus) }}.</p>
      <div class="ai-table-scroll" v-if="context.categories?.length">
        <table><caption>Budget dan pengeluaran periode akhir {{ period.month }}</caption><thead><tr><th scope="col">Pos</th><th scope="col">Budget</th><th scope="col">Aktual</th><th scope="col">Ruang simulasi</th></tr></thead>
          <tbody><tr v-for="(item, index) in context.categories" :key="index"><th scope="row">{{ item.name }}<small>{{ item.class }}</small></th><td>{{ money(item.planned) }}</td><td>{{ money(item.actual) }}</td><td>{{ money(item.scenarioSaving) }}</td></tr></tbody>
        </table>
      </div>
      <p v-if="context.transactions">{{ context.transactions.length }} detail transaksi pemasukan dan pengeluaran dalam rentang pilihan disertakan, termasuk deskripsi/catatan dan nama rekening. Lihat seluruh konteks di bawah untuk memeriksa setiap catatan. Akses data Finance hanya-baca.</p>
      <h3>Sumber</h3>
      <ul><li v-for="source in context.sources" :key="source">{{ source }}</li></ul>
      <p v-for="note in context.notes" :key="note" class="ai-context-note">{{ note }}</p>
      <small>Diambil {{ new Date(context.generatedAt).toLocaleString('id-ID') }} · Snapshot tersimpan bersama jawaban.</small>
      <details><summary>Lihat seluruh konteks yang dikirim</summary><pre>{{ JSON.stringify(context, null, 2) }}</pre></details>
    </div>
    <p v-else class="ai-context-body">Akses Finance dinonaktifkan. Percakapan ini memakai pertanyaan dan riwayat chat.</p>
  </details>
</template>
