# Hubby

Hubby (Hub Bily) adalah workspace aplikasi bily. Produk pertamanya, **Hubby Finance**, mengubah alur kerja pada workbook `Dashboard Keuangan AmBil _ 2025.xlsx` menjadi aplikasi web yang lebih nyaman dipakai. Produk keduanya, **Hubby Watch**, mencatat film dan series, episode terakhir, serta total waktu menonton.

## Struktur

```text
.
├── backend/       Go REST API + migration PostgreSQL
├── frontend/      Vue 3 single-page app
└── docker-compose.yml
```

Modul pada workbook dipetakan menjadi:

- dashboard dan financial check-up;
- budget, transaksi, serta arus kas bulanan;
- progress kekayaan dan dana darurat;
- tujuan keuangan;
- monitor dan rebalancing investasi;
- kalkulator KPR dan persiapan pensiun;
- piramida keuangan dan glosarium.

Hubby Watch menyediakan:

- pustaka film dan series dengan status watchlist, sedang ditonton, selesai, atau dihentikan;
- watch log per tanggal beserta durasi dan catatan;
- posisi season dan episode terakhir;
- ringkasan waktu menonton keseluruhan dan bulan berjalan;
- daftar lanjut menonton dan riwayat terbaru.

Setelah login, pengguna masuk ke portal Hubby untuk memilih modul. Hubby Finance tersedia di `/finance`, sedangkan Hubby Watch tersedia di `/watch`; keduanya memakai akun dan workspace yang sama, tetapi memiliki navigasi produk yang terpisah.

Versi awal ini sudah memiliki dashboard, transaksi, tujuan, data investasi, financial check-up, skema data untuk seluruh modul, serta halaman perencanaan. Kalkulator KPR, pensiun, rebalancing, dan formulir konfigurasi lanjutan ditandai sebagai tahap berikutnya.

## Menjalankan lokal

Prasyarat:

- Go 1.25 atau lebih baru;
- Node.js `^20.19.0` atau `>=22.12.0`;
- Docker Desktop untuk PostgreSQL.

### 1. Database

```powershell
docker compose up -d postgres
docker compose run --rm migrate up
```

Migration dikelola dengan [Goose](https://github.com/pressly/goose). Service `migrate` menjalankan migration tertunda dan mencatat versinya pada tabel `goose_db_version`.

Migration juga dapat dijalankan langsung dari folder `backend`:

```powershell
$env:DATABASE_URL="postgres://hubby:hubby@localhost:5432/hubby?sslmode=disable"
go run ./cmd/migrate up
go run ./cmd/migrate status
```

Perintah yang tersedia adalah `up`, `up-by-one`, `down`, `status`, dan `version`. Migration SQL disematkan ke binary sehingga file migration tidak perlu disalin terpisah pada deployment.

### 2. Backend

```powershell
cd backend
Copy-Item .env.example .env
go run ./cmd/api
```

Backend otomatis membaca konfigurasi lokal dari `backend/.env`. Environment variable dari sistem tetap memiliki prioritas sehingga konfigurasi deployment tidak akan ditimpa. API tersedia di `http://localhost:8080`; health check berada di `http://localhost:8080/health`.

### Telegram Bot

Backend dapat menjalankan Telegram bot dengan long polling, sehingga tidak membutuhkan
domain publik atau webhook. Tambahkan konfigurasi berikut ke `backend/.env`:

```dotenv
TELEGRAM_BOT_TOKEN=token_dari_botfather
TELEGRAM_PAIRING_CODE=kode_rahasia_untuk_menghubungkan_akun
TELEGRAM_LOCAL_USER_ID=1
TELEGRAM_TIMEZONE=Asia/Jakarta
```

Setelah API dijalankan, buka bot dan hubungkan akun:

```text
/start kode_rahasia_untuk_menghubungkan_akun
```

Perintah yang tersedia:

- `/ruang` memilih ruang keuangan secara dinamis;
- `/ruangaktif` menampilkan ruang yang digunakan bot;
- `/keluar 25000 makan siang` mencatat pengeluaran;
- `/masuk 5jt gaji bulanan` mencatat pemasukan;
- `/hariini` dan `/bulanini` menampilkan ringkasan;
- `/saldo` menampilkan rekening dan saldo pada ruang aktif;
- `/batal` membatalkan transaksi yang belum dikonfirmasi.

Ruang aktif bot disimpan per akun Telegram dan tidak mengubah ruang aktif di website.
Kategori dan rekening selalu diambil dari ruang yang dipilih. Transaksi baru disimpan
setelah pengguna menekan tombol konfirmasi.

### 3. Frontend

Di terminal lain:

```powershell
cd frontend
Copy-Item .env.example .env
npm install
npm run dev
```

Buka `http://localhost:5173`. Jika API belum berjalan, frontend otomatis memakai data pratinjau sehingga desain tetap dapat diperiksa.

## Endpoint utama

| Method      | Endpoint                          | Kegunaan                        |
| ----------- | --------------------------------- | ------------------------------- |
| `GET`       | `/api/v1/dashboard?month=YYYY-MM` | Ringkasan dan check-up          |
| `GET/POST`  | `/api/v1/transactions`            | Daftar dan pencatatan transaksi |
| `DELETE`    | `/api/v1/transactions/{id}`       | Menghapus transaksi             |
| `GET`       | `/api/v1/accounts`                | Rekening dan aset               |
| `GET/PATCH` | `/api/v1/goals`                   | Tujuan keuangan                 |
| `GET`       | `/api/v1/investments`             | Portofolio investasi            |
| `GET`       | `/api/v1/watch`                   | Ringkasan dan pustaka tontonan  |
| `POST`      | `/api/v1/watch/titles`            | Menambahkan film atau series    |
| `POST`      | `/api/v1/watch/sessions`          | Mencatat sesi menonton          |

Semua nominal disimpan sebagai bilangan bulat rupiah untuk menghindari masalah pembulatan.

## Pemeriksaan

Backend:

```powershell
cd backend
go test ./...
go vet ./...
```

Frontend:

```powershell
cd frontend
npm run build
```
