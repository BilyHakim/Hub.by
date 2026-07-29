# Hubby

Hubby (Hub Bily) adalah workspace aplikasi bily. Produk pertamanya, **Hubby Finance**, mengubah alur kerja pada workbook `Dashboard Keuangan AmBil _ 2025.xlsx` menjadi aplikasi web yang lebih nyaman dipakai.

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
