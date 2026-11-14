# Hubby Reader

Reader mengikuti style Hub.by: kertas terang, DM Sans/Manrope, bingkai navigasi yang sama. Biru slate membedakannya dari Books yang memakai sand. Kontrol dan teks memakai slate lebih gelap untuk memenuhi kontras. ENERGY 1 / RHYTHM 1 / MOTION 1. Isi buku menjadi fokus; anotasi berada di samping pada layar besar dan di bawah pada layar sempit. Ikon buku, upload, dan hapus memakai library aplikasi untuk menunjukkan fungsi yang sama. Tidak ada ilustrasi atau konten contoh yang disisipkan ke database pengguna.

## Data dan perilaku

`reader_books` melampirkan satu PDF/EPUB ke `book_titles` melalui foreign key dengan workspace yang sama. Pustaka Reader hanya membaca lampiran tersebut. Upload baru membuat kedua entri dalam satu transaksi; melampirkan ke buku lama memakai ID yang sama. Foreign key menghapus file, timer, dan anotasi jika buku atau workspace dihapus. Menghapus lampiran saja mempertahankan buku dan reading log.

Posisi Reader dapat bergerak mundur; reading log Books hanya bertambah jika halaman melewati progres tertinggi. PDF menyimpan nomor halaman file. EPUB menyimpan CFI dengan estimasi halaman berdasarkan lokasi teks, dan menandai selesai ketika mencapai akhir buku. Jumlah halaman buku lama tetap dipertahankan dan progres file dipetakan proporsional.

Timer menghitung waktu ketika reader terlihat dan timer on. Penyimpanan berkala berjalan setiap sepuluh detik, juga saat berpindah halaman, menyimpan manual, meninggalkan rute, atau menyembunyikan tab. ID sesi dan akumulasi detik membuat retry idempoten. Menutup browser secara paksa atau kehilangan koneksi sebelum penyimpanan berhasil dapat kehilangan perubahan terakhir. Tidak ada dukungan membaca offline.

PDF dirender memakai [PDF.js](https://mozilla.github.io/pdf.js/examples/), dengan text layer untuk seleksi dan persegi highlight yang dinormalisasi terhadap halaman. EPUB memakai [epub.js](https://github.com/futurepress/epub.js), DOMPurify, serta iframe tanpa izin menjalankan script atau popup. File disimpan di PostgreSQL, maksimal 50 MB. EPUB diperiksa sebagai ZIP dengan mimetype dan container; ukuran hasil ekstraksi dibatasi 200 MB. Buku dengan DRM atau password tidak didukung; PDF scan mendukung catatan halaman dan bookmark, tetapi highlight memerlukan teks.

## Validasi

### Layar penuh

Tombol **Layar penuh** pada toolbar memperluas area buku, menyembunyikan navigasi aplikasi dan panel anotasi, serta mempertahankan navigasi halaman/bab, timer, dan penyimpanan progres. Keluar melalui tombol **Keluar layar penuh** atau Escape. Isi anotasi dan posisi baca tetap dipertahankan; panel kembali ketika keluar. EPUB menyesuaikan ukuran sambil membawa posisi CFI terakhir. PDF dirender ulang sesuai lebar layar.

Toolbar fullscreen berada di bawah area baca dan mendapat ruang tersendiri sehingga tidak menutupi halaman. Tombol **Sembunyikan navigasi** memberi seluruh area tersebut ke buku; tombol **Tampilkan navigasi** tetap tersedia di bilah bawah dan menerima fokus keyboard. Keluar fullscreen mengembalikan toolbar biasa. Pada layar sempit, toolbar dapat digulir dan tingginya dibatasi agar isi buku tetap terlihat.

Navigasi halaman memakai baris tersendiri: tombol **Sebelumnya** dengan panah kiri, posisi halaman di tengah, dan **Berikutnya** dengan panah kanan serta warna utama. Target tombol minimal 52px dengan jarak dari kontrol skala, timer, dan fullscreen. Pada layar hingga 480px, tombol memakai ikon dengan label aksesibel dan tooltip agar baris tetap muat.

Setiap navigasi ke halaman sebelumnya/berikutnya mengembalikan posisi vertikal area baca ke paling atas setelah halaman selesai dirender, termasuk saat navigasi disembunyikan dalam fullscreen. Perilaku ini berlaku untuk PDF dan EPUB, juga saat melompat melalui nomor halaman, daftar isi, atau mark. Mengubah skala atau ukuran layar saja tidak mereset scroll.

Saat toolbar disembunyikan, ikon chevron kiri/kanan tetap tersedia bersama tombol **Tampilkan navigasi**. Ikon memakai library aplikasi, dengan label dan tooltip **Halaman sebelumnya/berikutnya**, serta target sentuh 44px. Tombol mengikuti status loading/rendering dan batas halaman PDF, memakai fungsi navigasi serta penyimpanan progres yang sama. Bilah ringkas mendapat ruang tersendiri di bawah buku agar tidak menutupi teks. Tombol sebelumnya berada di kiri dan berikutnya di kanan, dipisahkan tombol **Tampilkan navigasi** di tengah.

Validasi navigasi cepat sebelum perubahan tata letak: build produksi berhasil; ikon PDF diuji dari halaman pertama ke terakhir dan kembali tanpa membuka toolbar; tombol batas halaman nonaktif; ikon EPUB diuji next/previous pada layar 320px, ukuran tombol minimal 44px, dan seluruh grup muat dalam viewport. Mode native, fallback, dan fullscreen ditolak semuanya lulus tanpa page error. Screenshot desktop dan mobile diperiksa. Validasi ini memakai data contoh lokal, belum deployment VPS.

PDF menyediakan skala **Sesuai lebar**, **Satu halaman penuh**, dan **50/75/100/125/150/200%**. Ukuran persentase memakai ukuran halaman PDF; teks, canvas, dan highlight mengikuti skala yang sama. Halaman yang lebih lebar dari panel dapat digeser horizontal. EPUB menyediakan ukuran teks **50–200%** dari ukuran dasar 18px dan membawa CFI saat teks mengalir ulang. Perubahan skala dan navigasi menunggu posisi EPUB selesai diperbarui.

Implementasi memakai [Fullscreen API](https://developer.mozilla.org/en-US/docs/Web/API/Element/requestFullscreen) dengan fallback memenuhi viewport ketika API tidak tersedia atau permintaan ditolak. Pada fallback, bingkai browser tetap terlihat. Tidak memerlukan perubahan backend atau migration; cukup rebuild dan recreate frontend.

Validasi fullscreen memakai build produksi dan data API contoh lokal di Chrome headless. PDF/EPUB diuji pada desktop 1440px dan mobile 390px: masuk, keluar lewat tombol/Escape, pergantian halaman/bab, timer, penyimpanan posisi, pengembalian panel, dan tidak ada overflow horizontal. Mode native, API tidak tersedia, dan permintaan ditolak semuanya lulus tanpa page error. Screenshot desktop/mobile diperiksa dan EPUB tetap menampilkan teks setelah perubahan ukuran. Pengujian ini tidak mengakses database VPS.

Validasi skala dan toolbar bawah: rasio lebar PDF 100% ke 150% diperiksa, zoom 200% menghasilkan area geser horizontal, mode satu halaman muat di area baca, toolbar berada di bawah konten, hide/unhide memperbesar dan mengembalikan area baca, serta ukuran EPUB 150% menghasilkan 27px. Bab EPUB dipastikan tetap sama setelah perubahan ukuran dan resize mobile. Semua mode native/fallback/rejection lulus tanpa page error. Build dan `git diff --check` lulus.

Hard Gate PASS: kontrol fullscreen dan keluar bekerja, Escape diuji, panel tersembunyi tidak dapat ditab, dan build produksi berhasil.

Purpose Gate PASS: area layar diberikan ke isi buku; toolbar tetap tersedia untuk berpindah halaman, mengatur timer, dan keluar. Warna dan tipografi memakai token Reader yang sudah terukur.

Liveliness PASS: ENERGY 1 / RHYTHM 1 / MOTION 1 tetap; isi buku menjadi fokus tunggal tanpa animasi tambahan.

Craftsmanship PASS: native/fallback/rejection diuji dengan PDF dan EPUB, termasuk resize mobile dan pengembalian tampilan. Belum diuji pada Safari/perangkat fisik atau dideploy ke VPS.

### Pustaka dan anotasi

- Unit Go memeriksa format file, timer/progres, dan batas koordinat anotasi.
- `TestReaderDatabaseLifecycle` memakai `READER_TEST_DATABASE_URL` untuk memeriksa upload/link, retry timer, progres mundur, isolasi workspace, penghapusan, dan cascade. Jalankan hanya terhadap database QA yang telah dimigrasikan.
- Uji browser lokal memakai PDF dua halaman dan EPUB dua bab: upload, lanjut baca, timer, highlight kedua format, bookmark, catatan, lompat ke anotasi, daftar isi, navigasi, serta Books. Script dalam EPUB contoh tidak berjalan. Tidak ditemukan page error pada alur tersebut.
- Screenshot diperiksa pada desktop 1440px dan mobile 390px; CSS text layer dibatasi ke area PDF agar tidak memengaruhi sidebar aplikasi.
- Pemeriksaan tambahan pada 320/390/768/1440px memastikan PDF memenuhi lebar area baca tanpa overflow. Kontrol timer diuji dengan keyboard dan fokus terlihat. Upload melalui detail Books mempertahankan ID buku tanpa duplikat; pencarian, penghapusan anotasi/file, kondisi kosong, serta retry setelah error API berhasil diuji.
- Kontras terhitung: tombol putih pada slate `#456B79` 5.78:1; teks utama slate pada kertas 7.91:1; teks pendukung pada panel 5.55:1; teks kutipan 9.23:1; batas kontrol terhadap panel 3.91:1.

Audit npm masih melaporkan lima temuan high pada dependensi proyek di luar reader: Vue/server-renderer, Vite, nanoid, dan source-map-js. Parser XML transitive epub.js telah diperbarui melalui override `@xmldom/xmldom` ke versi yang sudah menutup temuan audit. Pengujian browser memakai Chrome; belum memverifikasi Safari, perangkat fisik, atau deployment produksi.

## Antislop delivery gate

Hard Gate PASS: alur PDF/EPUB berjalan dalam browser, kontrol inti memiliki handler/rute, tersedia loading/error/empty state, dan teks memakai pasangan warna terukur. Tidak ada klaim pengguna, testimoni, atau statistik buatan.

Purpose Gate PASS: warna menunjukkan identitas Reader; font, panel, navigasi, dan bentuk kontrol mengikuti Hub.by sesuai arahan pemilik. Tidak menambahkan dekorasi atau animasi otomatis.

Liveliness PASS: ENERGY 1 / RHYTHM 1 / MOTION 1; isi buku menjadi fokus dengan kontrol baca terkelompok dan panel anotasi terpisah. Motif kertas dan bingkai modul mempertahankan identitas aplikasi.

Craftsmanship PASS: build dan tes integrasi dijalankan, alur utama diklik melalui browser headless, layout mobile diperiksa, dan isolasi workspace serta batas file diuji. Batas DRM, scan tanpa teks, dan penyimpanan saat koneksi terputus dijelaskan di atas.
