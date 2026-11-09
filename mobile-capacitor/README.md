# Hubby Mobile — Capacitor

Proyek Android untuk HP dan tablet, memakai komponen Vue, router, dan styling dari `../frontend`. Folder ini memuat konfigurasi/build khusus mobile dan proyek native di `android/`. Tidak ada salinan komponen UI atau backend lokal.

API selalu `https://bilyhakim.site/api/v1`, ditetapkan dalam `scripts/config.mjs`. Akun, workspace, dan data memakai backend serta database produksi yang sama dengan web. Semua perubahan setelah login adalah perubahan data produksi. Aplikasi hanya menyimpan aset UI lokal dan sesi login perangkat; credential database tidak dimasukkan ke APK. Membaca data memerlukan koneksi internet; belum ada fitur offline/sinkronisasi lokal.

## Menyiapkan proyek

Gunakan Node.js 22.12+ atau 24+, Android Studio, Android SDK Platform 36/Build Tools 35+, dan JDK 21. Gradle proyek ini belum mendukung JDK 25. SDK Android dapat dipilih melalui `ANDROID_HOME` atau `android/local.properties`; file tersebut tidak masuk Git.

Jalankan dari root repository:

```sh
npm --prefix frontend ci
npm --prefix mobile-capacitor ci
cd mobile-capacitor
npm run check
npm run sync
npm run android
```

Di Windows PowerShell, gunakan `npm.cmd` jika execution policy memblokir `npm.ps1`. Pilih JDK 21 pada Android Studio Settings > Build Tools > Gradle > Gradle JDK, atau isi `JAVA_HOME` pada terminal untuk build CLI. Script `apk:debug` juga mengenali JDK 21 di `.tools/jdk-21...` jika tersedia; folder `.tools` tidak masuk Git dan tidak dibundel ke aplikasi.

## Menjalankan dan membuat APK

```sh
# Emulator atau perangkat Android dengan USB debugging
npm run run:android

# APK debug untuk instalasi pengujian
npm run apk:debug
```

Hasil APK: `android/app/build/outputs/apk/debug/app-debug.apk`. APK debug belum ditandatangani dengan kunci rilis Play Store. Untuk rilis, gunakan menu Generate Signed App Bundle / APK di Android Studio dan simpan keystore di luar Git.

Setelah mengubah UI pada `frontend`, jalankan `npm run sync` lalu build ulang aplikasi. `npm run build` menulis aset hanya ke `mobile-capacitor/www`, sehingga hasil build web di `frontend/dist` tidak tertimpa. Perubahan frontend pada VPS tidak otomatis memperbarui UI yang sudah dibundel di APK.

## Preview browser dengan API produksi

```sh
npm run dev
# atau npm run build kemudian npm run preview
```

Buka `http://localhost:1420`. Port ini mengikuti origin desktop yang sudah ada pada contoh konfigurasi backend. Preview browser memerlukan `CORS_ALLOWED_ORIGINS` produksi yang memuat `http://localhost:1420`. Tambahkan origin tersebut tanpa menghapus origin lain jika konfigurasi VPS berbeda; tidak perlu migration. Preview dan APK sama-sama ditargetkan ke API produksi, bukan `localhost:8080`.

Android memakai [CapacitorHttp](https://capacitorjs.com/docs/apis/http) untuk `fetch`/XHR native, termasuk cookie sesi backend. Tidak perlu menambahkan wildcard CORS, membuka PostgreSQL ke internet, atau memindahkan token login ke localStorage. Cookie tetap HttpOnly dari server. Build tidak berisi password akun maupun database.

## Perilaku mobile

- Membuka `/hub` saat startup; form login yang sama muncul bila belum ada sesi.
- Tombol Back Android keluar dari fullscreen Reader terlebih dahulu, kemudian mengikuti riwayat navigasi. Pada halaman awal, aplikasi diminimalkan.
- Tata letak HP/tablet, Reader, skala, anotasi, dan navigasi fullscreen memakai implementasi web yang sama. Area layar mengikuti safe-area perangkat dan zoom browser tetap diizinkan.
- Ikon aplikasi memakai aset Hubby dari proyek desktop. Tema native mengikuti latar kertas/sage yang sudah ada. Orientasi tidak dikunci.
- Worker PDF dibundel sebagai `.js` agar disajikan dengan MIME JavaScript oleh server aset native.
- Upload memakai pemilih file Android/WebView; tidak meminta akses seluruh penyimpanan. File besar tetap melewati bridge native dan perlu pengujian memori pada perangkat fisik.

Target yang disiapkan saat ini Android. iPhone/iPad memerlukan proyek iOS serta build/pengujian melalui macOS dan Xcode; tidak termasuk APK Android ini. [Persyaratan Capacitor](https://capacitorjs.com/docs/getting-started/environment-setup).

## Pemeriksaan

`npm run check` memeriksa endpoint produksi, lokasi build terpisah, konfigurasi HTTP native, viewport yang tetap dapat di-zoom, dan ekstensi worker. Pemeriksaan browser dengan fixture harus mencegat seluruh request API; jangan membuat data tes di produksi. Endpoint `/health` dapat diperiksa tanpa login.

Sebelum menggunakan data penting, verifikasi di perangkat: login, tutup/buka ulang aplikasi, logout, upload PDF/EPUB, seleksi/highlight, fullscreen, rotasi HP/tablet, dan tombol Back. Login ke produksi hanya dilakukan oleh pemilik dengan akun sendiri.

## Hasil validasi implementasi

- Build web mobile, sinkronisasi Capacitor, dan `assembleDebug` berhasil. APK debug tersedia pada lokasi di atas, dengan application ID `by.hubby.mobile` dan versi `0.1.0`.
- Enam tes Node lulus setelah build, mencakup endpoint produksi di bundle, tidak adanya connection string PostgreSQL di aset, konfigurasi native HTTP, lokasi output terpisah, dan perilaku tombol Back.
- Uji Chrome headless pada 390×844 dan 1024×768 lulus: login dengan fixture, portal modul, Reader PDF/EPUB, skala, fullscreen, serta navigasi cepat. Semua request API mengarah ke host produksi tetapi dicegat dan dipenuhi fixture lokal. Tidak ada akun atau data percobaan yang dibuat di produksi, dan tidak ditemukan page error.
- Pemeriksaan produksi hanya read-only: `/health` merespons 200 dan preflight origin preview `http://localhost:1420` merespons 204.
- Belum diuji pada perangkat fisik/emulator, belum login dengan akun produksi asli, dan belum menjalankan instrumented test Android. APK ini untuk pengujian, belum rilis Play Store. iOS belum diimplementasikan.

Hard Gate PASS untuk perubahan ini: build APK dan alur UI browser di atas berjalan, viewport tetap dapat di-zoom, tidak ada overflow pada ukuran HP/tablet yang diuji, dan tidak menambahkan kontrol UI baru yang belum terhubung.

Purpose Gate PASS: UI, warna, font, dan ikon memakai implementasi/aset Hubby yang sudah ada. CSS khusus mobile hanya memberi safe-area; tema native mengikuti warna kertas/sage dan tidak mengunci orientasi.

Liveliness PASS: mempertahankan arah aplikasi ENERGY 1 / RHYTHM 1 / MOTION 1 pada Reader; tidak menambahkan animasi atau dekorasi baru ke UI mobile.

Craftsmanship PASS untuk cakupan yang diverifikasi: enam tes Node, build produksi, sync Android, APK debug, dan uji browser HP/tablet lulus. Batas pengujian perangkat, akun asli, dan iOS dijelaskan di atas.
