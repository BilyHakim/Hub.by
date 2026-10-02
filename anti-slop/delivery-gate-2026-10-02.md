# Delivery Gate Hub.by

Tanggal: 2 Oktober 2026. Mode: during, session override. Status: PASS untuk landing page dan integrasi login yang diubah.

Lingkup: landing publik `/`, akses login `/login`, portal `/hub`, metadata, dan navigasi menuju modul. Modul Finance, Watch, dan Books yang sudah ada tidak diaudit ulang secara menyeluruh.

Arah pemilik dan alasan keputusan tercatat di [DESIGN.md](../DESIGN.md). Dials: ENERGY 2 / RHYTHM 2 / MOTION 1.

Bukti: [log Chrome](evidence/verification.txt), [desktop 1440px](evidence/landing-1440.png), [ponsel 375px](evidence/landing-375.png), dan [skrip pemeriksaan](verify-landing.mjs). Build produksi `npm.cmd run build` berhasil. `git diff --check` tidak menemukan kesalahan whitespace.

## 1. Hard Gate

- R-02 PASS: pemindaian LandingView.vue, index.html, dan DESIGN.md tidak menemukan em dash; copy UI menggunakan bahasa Indonesia.
- R-03 PASS: Chrome memeriksa 320, 375, 640, 768, 960, 1280, dan 1440px tanpa overflow; target tautan minimal 44px; teks 200% lulus pada 320, 768, dan 1440px.
- R-17 PASS: landing tidak menampilkan angka pengguna, uptime, pertumbuhan, atau statistik lain.
- R-18 PASS: tidak ada testimoni, identitas pelanggan, atau avatar rekaan di landing.
- R-23 PASS: pemilik menyetujui arah UI dan susunan pembuka, fitur modul, serta tombol masuk; nama Hub.by memakai teks, tanpa logo atau aset visual baru.
- R-24 PASS: Hub.by menuju `/`; Lihat fitur menuju `#fitur` yang ada; Masuk menuju `/login`; semuanya diklik di Chrome.
- R-25 PASS: pasangan teks landing memiliki rasio 5.22:1 sampai 12.04:1; CTA normal 6.09:1 dan hover 9.30:1, hasil formula WCAG tercatat di log.
- R-26 PASS: setiap tautan landing diklik pada 375 dan 1440px; setiap tautan memiliki rute atau anchor nyata; toggle sandi dan submit login bekerja.
- R-27 PASS: landing merupakan konten statis tanpa data pengguna; tetap tampil saat API mati. Form login kosong memakai validasi browser, proses submit menonaktifkan tombol dan menampilkan Memeriksa, serta kegagalan ditampilkan sebagai alert. Pemeriksaan sesi memiliki teks dengan role status.
- R-28 PASS: tidak menambahkan FAQ karena tidak tersedia sumber pertanyaan pengguna.
- R-32 PASS: urutan Tab dan Shift+Tab diperiksa; setiap tautan memiliki outline solid; Enter pada skip link memindahkan fokus ke main. Tidak ada menu atau dialog baru yang memerlukan Escape.
- R-33 PASS: perubahan fitur ditulis langsung pada komponen Vue, router, dan HTML melalui patch; skrip verifikasi hanya membaca build dan menyimpan bukti.
- R-34 PASS: satu tema terang sesuai arah pemilik; tidak menambahkan toggle tema atau tema kedua.
- R-35 PASS: build berhasil; semua tautan, toggle sandi, validasi dan submit login diklik; tidak ada error JavaScript aplikasi. Respons autentikasi diuji menggunakan API uji.
- R-36 PASS: tidak ada klaim keamanan, kepatuhan, pelanggan, atau kecepatan baru; klaim cookie aman di tampilan login dihapus.
- R-37 PASS: pemilik menyetujui gaya kertas terang, sage, DM Sans/Manrope dan struktur sebelum edit; Design Read diumumkan sebelum implementasi dan dicatat dalam DESIGN.md.
- R-38 PASS: fitur diambil dari komponen dan endpoint yang ada; anggaran memiliki GET/PUT budgets, tujuan memiliki endpoint goals, Watch memiliki sessions/progress/catalog, Books memiliki sessions/catalog. Kebutuhan token TMDB dijelaskan.

## 2. Purpose-Gate

- R-01 PASS: warna kertas dan sage mengikuti UI yang disetujui; bidang sage muda mengelompokkan fitur Finance; tidak ada gradient atau glow di landing.
- R-04 PASS: landing tidak memakai ikon dekoratif atau memilih pustaka ikon baru; ikon login yang dipertahankan menunjukkan email, sandi, dan visibilitas sandi.
- R-06 PASS: DM Sans untuk teks dan Manrope untuk judul meneruskan identitas UI, alasan tercatat di DESIGN.md; tidak ada heading monospace atau label dengan tracking berlebihan.
- R-07 PASS: tidak ada grid atau pola titik dekoratif; garis memisahkan isi direktori dan fitur yang berbeda.
- R-08 PASS: tidak menambahkan panah dekoratif pada tombol.
- R-09 PASS: tidak ada badge kapsul atau label duplikat di atas H1.
- R-10 PASS: landing memakai permukaan solid; backdrop blur pada login dinonaktifkan.
- R-12 PASS: landing tidak memakai shadow; shadow form login yang sudah ada membedakan bidang input dari halaman, bukan diterapkan ke semua komponen.
- R-13 PASS: tidak ada glow pada landing.
- R-14 PASS: Finance ditampilkan sebagai bidang luas dengan daftar transaksi, anggaran, tujuan; Watch/Books sebagai dua artikel pustaka, bukan tiga kartu fitur identik.
- R-19 PASS: landing hanya mengubah warna atau underline saat hover/active, sesuai MOTION 1; tidak ada reveal, parallax, atau animasi otomatis.
- R-22 PASS: direktori berisi deskripsi fungsi nyata, bukan ilustrasi generik atau screenshot produk rekaan.

## 3. Liveliness

- Dials PASS: ENERGY 2 / RHYTHM 2 / MOTION 1 tercatat dalam DESIGN.md dan Design Read.
- Konsistensi PASS: judul besar menjadi pembuka; komposisi berpindah dari pembuka/direktori ke Finance, artikel pustaka, dan penutup; motion sebatas feedback interaksi.
- Focal point PASS: judul utama memimpin pembuka, pertanyaan kegiatan memimpin fitur, dan judul ajakan memimpin penutup; screenshot desktop dan ponsel diperiksa.
- Whitespace PASS: jarak pembuka 56px di ponsel dan 96px di desktop; jarak fitur lebih rapat untuk menghubungkan isi yang terkait.
- Accent PASS: sage gelap menandai CTA masuk; elemen lain memakai teks dan permukaan netral.
- Identity motif PASS: bahasa catatan sehari-hari, judul kegiatan, nama Finance/Watch/Books, serta pemisah seperti halaman catatan berulang sepanjang halaman.
- Design Read PASS: diumumkan sebelum generasi dan dicatat dalam DESIGN.md, berdasarkan pilihan pemilik.

## 4. Craftsmanship & Quality Locks

- C-1 PASS: warna, tipografi, komposisi, spacing, dan motif memiliki alasan tertulis dalam DESIGN.md.
- C-2 PASS: click-through semua tautan landing dan kontrol login tersimpan dalam log; tidak ada kontrol baru tanpa fungsi.
- C-3 PASS: pembuka menjelaskan fungsi; direktori membantu memahami modul; bagian fitur menjelaskan kegiatan; penutup mengarahkan ke login yang tersedia.
- C-4 PASS: tujuh lebar layar, pembesaran teks, keyboard, sesi masuk/keluar, dan API tidak tersedia diperiksa; tidak ada clipping konten landing.
- C-5 PASS: tidak menampilkan statistik, pelanggan, kutipan, atau klaim produk yang dibuat-buat.
- R-05 PASS: komposisi berbeda mengikuti isi modul; tidak ada pricing, logo pelanggan, bento, atau footer empat kolom tanpa isi.
- R-11 PASS: CTA memakai radius 6px, bidang Finance 8px, direktori 18px dengan satu sudut 4px; tidak semua elemen berbentuk pill.
- R-15 PASS: CTA menyebut Masuk ke Hub.by, Buka Finance, Buka Watch, Buka Books, dan Masuk dan pilih modul sesuai aksinya.
- R-16 PASS: copy menjelaskan transaksi, episode, halaman, dan ruang; tidak ada AI Powered, revolutionary, atau klaim pemasaran sejenis.
- R-20 PASS: struktur mengikuti gabungan keuangan, tontonan, dan bacaan milik Hub.by; judul serta direktori menggunakan kegiatan produk yang spesifik.
- R-21 PASS: tema terang berasal dari persetujuan pemilik dan kesinambungan UI, bukan default tema gelap.
- R-29 PASS: landing menggunakan kertas/netral, ink hijau, sage sebagai accent, dan sage muda untuk permukaan Finance.
- R-30 PASS: tidak menggunakan referensi atau meniru layout produk lain; gaya diambil dari codebase Hub.by.
- R-31 PASS: alasan keputusan utama tertulis satu baris per keputusan di DESIGN.md; tidak membuat ilustrasi atau ikon dekoratif untuk mengisi ruang.

## Click-through

Daftar berikut dilakukan di Chrome pada 375px dan 1440px, sebelum login:

| Elemen | Hasil |
| --- | --- |
| Hub.by | `/`, landing publik |
| Lihat fitur | `/#fitur`, bagian fitur |
| Masuk | `/login`, form login |
| Masuk ke Hub.by | `/login`, form login |
| Buka Finance | `/finance`, form login saat belum memiliki sesi |
| Buka Watch | `/watch`, form login saat belum memiliki sesi |
| Buka Books | `/books`, form login saat belum memiliki sesi |
| Masuk dan pilih modul | `/login`, form login |
| Kembali ke atas | `/#landing-title`, judul pembuka |

Pemeriksaan tambahan: skip link dengan Enter memindahkan fokus ke `main`; form kosong memicu validasi; toggle sandi mengubah text/password; login ditolak menampilkan error; login berhasil dengan API uji menuju `/hub`; ketiga CTA dalam sesi aktif menuju `/hub`; logout kembali ke landing; tautan kembali dari login menuju `/`. Saat API tidak tersedia, landing tetap tampil dan login menjelaskan cara mencoba kembali dalam bahasa Indonesia.

## Batas verifikasi dan deployment

Pengujian dilakukan terhadap build lokal dengan Chrome headless dan respons API uji. Data akun pengujian tidak ditampilkan pada landing atau dikirim ke produksi. Font eksternal diblokir selama uji sehingga layout juga diperiksa dengan font fallback. Pengujian tidak mengklaim aksesibilitas tersertifikasi, login produksi, atau deployment domain sudah selesai.

Nginx proyek sudah melayani `/` dan fallback `index.html` untuk rute Vue. Setelah frontend ini dideploy ke server yang melayani `bilyhakim.site`, domain utama akan menampilkan landing publik. Panduan deployment dan verifikasi diperbarui dalam DEPLOYMENT.md.

Untuk mengulang verifikasi, build frontend, lalu jalankan `node anti-slop/verify-landing.mjs` dari root dengan `CHROME_TOOLS_MODULE` menunjuk module `third_party/index.js` dari instalasi chrome-devtools-mcp yang mengekspor Puppeteer. Skrip memakai Chrome yang terpasang pada path Windows standar, port lokal 4178, serta stub API, tanpa akun produksi.
