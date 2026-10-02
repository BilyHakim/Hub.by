<script setup>
defineProps({ authenticated: Boolean })
</script>

<template>
  <div class="landing">
    <a class="landing-skip" href="#konten">Lewati ke konten</a>
    <header class="landing-header landing-width">
      <RouterLink class="landing-wordmark" to="/" aria-label="Hub.by, beranda">Hub.by</RouterLink>
      <nav aria-label="Navigasi landing page">
        <a href="#fitur">Lihat fitur</a>
        <RouterLink class="landing-entry" :to="authenticated ? '/hub' : '/login'">{{ authenticated ? 'Buka ruang saya' : 'Masuk' }}</RouterLink>
      </nav>
    </header>

    <main id="konten" class="landing-width" tabindex="-1">
      <section class="landing-intro" aria-labelledby="landing-title">
        <div>
          <h1 id="landing-title">Catatan sehari-hari,<br />punya tempat sendiri.</h1>
          <p class="landing-lead">Kelola keuangan, simpan tontonan, dan lanjutkan bacaan. Hub.by menyatukan catatanmu dalam satu akun dan ruang bersama.</p>
          <RouterLink class="landing-cta" :to="authenticated ? '/hub' : '/login'">{{ authenticated ? 'Buka ruang saya' : 'Masuk ke Hub.by' }}</RouterLink>
          <p class="landing-account-note">Gunakan akun yang sudah tersedia untuk masuk.</p>
        </div>
        <aside class="landing-directory" aria-label="Isi ruang Hub.by">
          <p>Satu ruang, untuk catatanmu</p>
          <dl>
            <div><dt>Finance</dt><dd>Pemasukan, pengeluaran, dan tujuan</dd></div>
            <div><dt>Watch</dt><dd>Film, series, dan progres episode</dd></div>
            <div><dt>Books</dt><dd>Pustaka dan halaman terakhir</dd></div>
          </dl>
          <span>Catat hari ini. Lanjutkan saat kamu kembali.</span>
        </aside>
      </section>

      <section id="fitur" class="landing-features" aria-labelledby="feature-title">
        <div class="landing-section-heading">
          <h2 id="feature-title">Apa yang ingin kamu catat?</h2>
          <p>Setiap modul punya tempatnya. Pilih sesuai kegiatanmu.</p>
        </div>
        <article class="landing-finance">
          <div class="landing-feature-copy">
            <p class="landing-module-name">Hubby Finance</p>
            <h3>Kenali ke mana<br />uangmu pergi.</h3>
            <p>Catat pemasukan dan pengeluaran, tinjau arus kas bulanan, dan pantau tujuan keuangan dalam ruang yang kamu pilih.</p>
            <RouterLink class="landing-module-link" to="/finance">Buka Finance</RouterLink>
          </div>
          <ul class="landing-finance-list">
            <li><strong>Transaksi dan rekening</strong><span>Simpan transaksi beserta kategori dan rekeningnya.</span></li>
            <li><strong>Anggaran bulanan</strong><span>Susun rencana pengeluaran dan bandingkan dengan realisasinya.</span></li>
            <li><strong>Tujuan keuangan</strong><span>Catat target dana dan perbarui progres yang terkumpul.</span></li>
          </ul>
        </article>
        <div class="landing-personal">
          <article>
            <p class="landing-module-name">Hubby Watch</p>
            <h3>Ingat episode<br />terakhirmu.</h3>
            <p>Simpan film dan series, atur status tontonan, lalu catat sesi menonton beserta durasi dan catatan. Progres episode membantumu melanjutkan series.</p>
            <p class="landing-source-note">Pencarian katalog menggunakan TMDB dan memerlukan konfigurasi token oleh pengelola.</p>
            <RouterLink class="landing-module-link" to="/watch">Buka Watch</RouterLink>
          </article>
          <article>
            <p class="landing-module-name">Hubby Books</p>
            <h3>Kembali ke<br />halaman terakhir.</h3>
            <p>Cari buku melalui Open Library, simpan ke pustaka, dan catat halaman terakhir yang dibaca. Lihat progres serta riwayat membaca setiap buku.</p>
            <p class="landing-source-note">Pencarian buku menggunakan Open Library tanpa API key.</p>
            <RouterLink class="landing-module-link" to="/books">Buka Books</RouterLink>
          </article>
        </div>
      </section>
      <section class="landing-closing" aria-labelledby="closing-title">
        <div><h2 id="closing-title">Catatanmu, dalam ruang yang kamu pilih.</h2><p>Setelah masuk, pilih modul dan ruang bersama untuk mengakses catatanmu.</p></div>
        <RouterLink class="landing-cta" :to="authenticated ? '/hub' : '/login'">{{ authenticated ? 'Pilih modul saya' : 'Masuk dan pilih modul' }}</RouterLink>
      </section>
    </main>
    <footer class="landing-footer landing-width"><span>Hub.by</span><p>Keuangan, tontonan, dan bacaan dalam satu ruang.</p><a href="#landing-title">Kembali ke atas</a></footer>
  </div>
</template>

<style scoped>
.landing { --landing-muted: #526159; color: #24332d; background: #f6f4ef; line-height: 1.65; overflow-wrap: anywhere; }
.landing-width { width: min(1120px, calc(100% - 40px)); margin-inline: auto; }
.landing a, .landing button { min-height: 44px; max-width: 100%; display: inline-flex; align-items: center; justify-content: center; }
.landing :focus-visible { outline: 3px solid #304c42; outline-offset: 5px; }
.landing-skip { position: absolute; top: 8px; left: 20px; padding: 8px 16px; background: #fffefa; z-index: 5; transform: translateY(-160%); }
.landing-skip:focus { transform: translateY(0); }
.landing-header { padding-block: 20px; display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 16px; border-bottom: 1px solid #cccfc6; }
.landing-wordmark { font-family: 'Manrope', sans-serif; font-size: 1.65rem; font-weight: 700; letter-spacing: -.06em; }
.landing-header nav { display: flex; flex-wrap: wrap; gap: 16px; align-items: center; font-size: .9rem; }
.landing-entry { border: 1px solid #526159; padding: 7px 16px; border-radius: 6px; }
.landing-intro { display: grid; gap: 40px; padding-block: 56px; }
.landing h1 { font-size: clamp(2.35rem, 5vw, 4rem); line-height: 1.12; margin-bottom: 24px; max-width: 750px; }
.landing-lead { max-width: 540px; font-size: 1.125rem; color: var(--landing-muted); margin-bottom: 28px; }
.landing-cta { color: #fffefa; background: #49685c; padding: 12px 22px; border-radius: 6px; font-weight: 600; text-align: center; }
.landing-cta:hover, .landing-cta:active { background: #304c42; }
.landing-account-note { margin: 12px 0 0; color: var(--landing-muted); font-size: .875rem; }
.landing-directory { background: #ebe8df; padding: 28px; border-radius: 18px 18px 18px 4px; align-self: center; }
.landing-directory > p { font-weight: 600; margin-bottom: 22px; }
.landing-directory dl { margin: 0; }
.landing-directory dl > div { padding: 15px 0; border-top: 1px solid #c6c8be; }
.landing-directory dt { font-family: 'Manrope', sans-serif; font-size: 1.25rem; font-weight: 700; }
.landing-directory dd { margin: 2px 0 0; color: var(--landing-muted); font-size: .9rem; }
.landing-directory > span { display: block; margin-top: 20px; font-size: .8rem; color: var(--landing-muted); }
.landing-features { scroll-margin-top: 24px; }
.landing-section-heading { padding-block: 16px 28px; }
.landing h2 { font-size: clamp(1.6rem, 3vw, 2.15rem); line-height: 1.25; margin-bottom: 12px; }
.landing-section-heading p { color: var(--landing-muted); margin: 0; }
.landing-finance { background: #dfe8e2; padding: 28px; border-radius: 8px; display: grid; gap: 24px; }
.landing-module-name { font-size: .9rem; font-weight: 600; margin-bottom: 16px; }
.landing h3 { font-size: clamp(1.8rem, 3.5vw, 2.65rem); line-height: 1.16; margin-bottom: 20px; }
.landing-feature-copy > p:not(.landing-module-name), .landing-personal article > p:not(.landing-module-name) { color: var(--landing-muted); max-width: 470px; }
.landing-module-link { font-weight: 600; text-decoration: underline; text-underline-offset: 5px; padding-block: 8px; }
.landing a:not(.landing-cta):hover { text-decoration: underline; text-underline-offset: 5px; }
.landing-finance-list { list-style: none; margin: 0; padding: 0; align-self: center; }
.landing-finance-list li { padding: 20px 0; border-bottom: 1px solid #aebdb2; }
.landing-finance-list li:first-child { border-top: 1px solid #aebdb2; }
.landing-finance-list strong, .landing-finance-list span { display: block; }
.landing-finance-list span { color: var(--landing-muted); margin-top: 4px; }
.landing-personal { display: grid; }
.landing-personal article { padding: 32px 0; border-bottom: 1px solid #cccfc6; }
.landing-source-note { font-size: .875rem; }
.landing-closing { padding-block: 40px; display: grid; align-items: center; gap: 20px; }
.landing-closing h2 { max-width: 650px; }
.landing-closing p { margin: 0; color: var(--landing-muted); }
.landing-closing .landing-cta { justify-self: start; }
.landing-footer { display: flex; flex-wrap: wrap; gap: 12px 24px; align-items: center; padding-block: 24px; border-top: 1px solid #cccfc6; font-size: .875rem; }
.landing-footer > span { font-weight: 700; font-size: 1.1rem; }
.landing-footer p { margin: 0; color: var(--landing-muted); }
@media (min-width: 640px) {
  .landing-width { width: min(1120px, calc(100% - 64px)); }
  .landing-intro { padding-block: 72px; }
  .landing-directory dl { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 24px; }
  .landing-personal { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 40px; }
  .landing-personal article { padding-block: 48px; }
  .landing-finance { padding: 40px; }
  .landing-footer a { margin-left: auto; }
}
@media (min-width: 960px) {
  .landing-intro { grid-template-columns: minmax(0, 1.65fr) minmax(0, 1fr); gap: 64px; padding-block: 96px; }
  .landing-directory dl { display: block; }
  .landing-finance { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 64px; }
  .landing-closing { grid-template-columns: minmax(0, 1fr) auto; gap: 40px; padding-block: 64px; }
}
@media (prefers-reduced-motion: reduce) { .landing * { scroll-behavior: auto; } }
</style>
