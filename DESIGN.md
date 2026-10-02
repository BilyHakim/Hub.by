# Landing page Hub.by

Arah disetujui pemilik dalam percakapan: mengikuti UI proyek dengan latar kertas terang, hijau sage, DM Sans dan Manrope. Susunan: pembuka, fitur ketiga modul, dan tombol masuk.

Design Read: landing page bagi pengguna yang mencatat keuangan, tontonan, dan bacaan; gaya hangat dan tenang. ENERGY 2 / RHYTHM 2 / MOTION 1.

- Warna kertas menjaga kesinambungan dengan aplikasi; sage menandai tindakan masuk, dengan teks hijau gelap untuk kontras.
- DM Sans untuk teks dan Manrope untuk judul mengikuti identitas aplikasi yang sudah ada.
- Pembuka dan daftar isi ruang menjelaskan kegunaan tanpa gambar atau data contoh.
- Finance diberi bidang lebih luas untuk memuat transaksi, anggaran, dan tujuan; Watch dan Books berdampingan sebagai pustaka pribadi.
- Jarak pembuka lebih besar untuk menonjolkan judul; fitur terkait lebih rapat agar mudah dipindai.
- Motif berulang berupa nama modul dan judul tentang kegiatan sehari-hari, disertai pemisah seperti halaman catatan.
- Tidak membuat logo, ilustrasi, statistik, testimoni, atau klaim keamanan baru; Hub.by ditulis sebagai nama produk.
- Tema terang mengikuti arah yang disetujui, tanpa toggle tema. Tidak ada animasi otomatis.
- Semua tautan menuju bagian yang tersedia atau rute modul dan login nyata.

Landing page publik berada di `/`, login di `/login`, portal di `/hub`. Modul tetap di `/finance`, `/watch`, dan `/books`.

## Hubby Finance

Arah pemilik: susunan dashboard mengikuti gambar referensi, dengan color space aplikasi yang sudah ada. Design Read: dashboard keuangan pribadi dan bersama, panel kertas di dalam bingkai sage, ENERGY 2 / RHYTHM 2 / MOTION 1.

- Bingkai sage gelap menghubungkan sidebar dan header; navigasi aktif memakai warna kertas agar menyatu dengan ruang konten.
- Panel pembuka menjadi fokus untuk mencatat transaksi; angka disusun vertikal di kanan seperti referensi, grafik dan rincian tetap memakai data Finance.
- Sage, sand, dan lilac memakai token existing untuk membedakan jenis metrik; DM Sans dan Manrope mempertahankan identitas dan keterbacaan aplikasi.
- Panel besar memiliki radius 22px, metrik 16px, dan kontrol memakai radius existing untuk membedakan fungsi setiap bidang. Panel tidak diberi bayangan; popover tetap memakai elevasi karena berada di atas konten.
- Ikon dompet, bank, tabungan, dan tren memakai komponen existing karena sesuai dengan metrik keuangan. Tidak menambah ilustrasi, logo, avatar, atau statistik buatan.
- Pada layar sempit, kolom kanan turun mengikuti pembuka dan navigasi memakai drawer existing. Fokus keyboard memakai sand; gerak hanya transisi kontrol dan mengikuti reduced motion.
