# Ikon Hub.by

Sumber utama: `app-icon.svg`. Simbol hati dan jabat tangan melanjutkan identitas aplikasi: ruang pribadi dan bersama untuk merawat catatan sehari-hari. Bidang sage `#49685c` dan warna kertas `#fffefa` mengikuti UI. Hati dibuat solid supaya siluet tetap terbaca pada favicon; potongan jabat tangan menghubungkannya dengan simbol di dalam aplikasi. Sudut kiri bawah yang lebih kecil mengikuti bentuk brand mark existing.

Design Read: hangat, tenang, mudah dikenali pada tab browser maupun launcher. ENERGY 2 / RHYTHM 1 / MOTION 1. Satu simbol menjadi fokus, dengan ruang kosong di sekelilingnya. Tidak memakai teks, gradien, bayangan, atau animasi.

Regenerasi dari root repository:

```sh
npm --prefix desktop run icons
```

Perintah memakai Tauri CLI yang sudah terpasang, memeriksa dimensi PNG web dan kesamaan sumber SVG, lalu memperbarui:

- `frontend/public/favicon.svg`, `favicon.ico`, dan `icons/`: PNG 16, 32, 180, 192, 512, dan 1024 px;
- `desktop/src-tauri/icons/`: PNG, Windows ICO/AppX, macOS ICNS, dan aset iOS;
- `mobile-capacitor/android/app/src/main/res/mipmap-*/`: ikon launcher lima kepadatan, bulat, foreground adaptif, dan monokrom.

`app-icon-foreground.svg` menempatkan simbol di tengah bidang 108 unit, di dalam lingkaran aman 66 unit agar tidak terpotong ketika launcher mengubah bentuk ikon. `app-icon-monochrome.svg` memakai transparansi pada potongan tangan supaya mengikuti warna tema Android. Saat mengubah simbol, perbarui ketiga SVG sebelum regenerasi. Konfigurasi ekspor ada di `app-icon.json`.

Build script Rust memantau folder `icons` supaya perubahan ICO juga membangun ulang resource executable pada build incremental.

Web menghubungkan aset melalui `frontend/index.html` dan `frontend/public/site.webmanifest`. Manifest menyediakan metadata layar utama; aplikasi tetap membutuhkan koneksi internet. Aset iOS tersedia untuk penggunaan berikutnya, tetapi proyek native iOS belum ada.

Build ulang web, APK, dan desktop untuk membawa ikon baru ke paket aplikasi. Aplikasi yang sudah terpasang dan website produksi belum berubah sampai paket/deployment baru dipasang. Browser atau launcher dapat menyimpan ikon lama dalam cache.

## Validasi 9 November 2026

- Hard Gate PASS untuk cakupan ikon: pembuatan diminta pemilik; semua URL favicon, Apple touch icon, dan manifest memiliki aset di hasil build; dimensi PNG 16 sampai 1024 px diperiksa otomatis. Kontras kertas/sage dihitung 6,09:1. Tidak menambah kontrol, teks produk, atau tata letak.
- Purpose Gate PASS: motif mengikuti simbol hati/jabat tangan existing; warna mengikuti token aplikasi. Siluet solid menjaga keterbacaan ukuran kecil; foreground adaptif menyisakan ruang untuk pemotongan launcher. Tidak menambah efek dekoratif.
- Liveliness PASS: ENERGY 2 / RHYTHM 1 / MOTION 1; hati sebagai fokus, potongan tangan sebagai motif khas, sage sebagai satu aksen, dan ruang kosong sebagai batas simbol. PNG 16/32 px, ikon bulat, foreground monokrom, dan ikon utama diperiksa visual.
- Craftsmanship PASS: regenerasi dan pemeriksaan aset lulus; ICO memuat 16/24/32/48/64/256 px dan header ICNS valid. Build web, build APK debug, enam tes konfigurasi mobile, serta build installer Windows MSI/NSIS berhasil. `aapt2` mengonfirmasi ikon adaptif dan lapisan monokrom di APK; ikon diekstrak dari executable Windows hasil build untuk diperiksa.

Belum diuji dengan memasang installer/APK pada perangkat, mengubah tema launcher Android, atau menjalankan macOS/iOS. Tidak melakukan deployment website produksi.
