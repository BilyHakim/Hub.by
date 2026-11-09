# Hubby Reader

Reader mengikuti style Hub.by: kertas terang, DM Sans/Manrope, bingkai navigasi yang sama. Biru slate membedakannya dari Books yang memakai sand. Kontrol dan teks memakai slate lebih gelap untuk memenuhi kontras. ENERGY 1 / RHYTHM 1 / MOTION 1. Isi buku menjadi fokus; anotasi berada di samping pada layar besar dan di bawah pada layar sempit. Ikon buku, upload, dan hapus memakai library aplikasi untuk menunjukkan fungsi yang sama. Tidak ada ilustrasi atau konten contoh yang disisipkan ke database pengguna.

## Data dan perilaku

`reader_books` melampirkan satu PDF/EPUB ke `book_titles` melalui foreign key dengan workspace yang sama. Pustaka Reader hanya membaca lampiran tersebut. Upload baru membuat kedua entri dalam satu transaksi; melampirkan ke buku lama memakai ID yang sama. Foreign key menghapus file, timer, dan anotasi jika buku atau workspace dihapus. Menghapus lampiran saja mempertahankan buku dan reading log.

Posisi Reader dapat bergerak mundur; reading log Books hanya bertambah jika halaman melewati progres tertinggi. PDF menyimpan nomor halaman file. EPUB menyimpan CFI dengan estimasi halaman berdasarkan lokasi teks, dan menandai selesai ketika mencapai akhir buku. Jumlah halaman buku lama tetap dipertahankan dan progres file dipetakan proporsional.

Timer menghitung waktu ketika reader terlihat dan timer on. Penyimpanan berkala berjalan setiap sepuluh detik, juga saat berpindah halaman, menyimpan manual, meninggalkan rute, atau menyembunyikan tab. ID sesi dan akumulasi detik membuat retry idempoten. Menutup browser secara paksa atau kehilangan koneksi sebelum penyimpanan berhasil dapat kehilangan perubahan terakhir. Tidak ada dukungan membaca offline.

PDF dirender memakai [PDF.js](https://mozilla.github.io/pdf.js/examples/), dengan text layer untuk seleksi dan persegi highlight yang dinormalisasi terhadap halaman. EPUB memakai [epub.js](https://github.com/futurepress/epub.js), DOMPurify, serta iframe tanpa izin menjalankan script atau popup. File disimpan di PostgreSQL, maksimal 50 MB. EPUB diperiksa sebagai ZIP dengan mimetype dan container; ukuran hasil ekstraksi dibatasi 200 MB. Buku dengan DRM atau password tidak didukung; PDF scan mendukung catatan halaman dan bookmark, tetapi highlight memerlukan teks.

## Validasi

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
