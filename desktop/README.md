# Hubby Desktop

Aplikasi desktop ini adalah shell Tauri 2 untuk frontend Vue yang sudah ada. Desktop tidak
memiliki database atau migration terpisah: semua data tetap diakses melalui Go API dan schema
PostgreSQL pada `../backend/migrations`.

## Prasyarat Windows

- Node.js sesuai versi pada README utama;
- Rust stable melalui rustup;
- Microsoft C++ Build Tools dan WebView2;
- PostgreSQL, migration, dan backend Hubby dalam keadaan berjalan.

## Development

Dari root repository, hidupkan Docker Desktop lalu jalankan PostgreSQL:

```powershell
docker compose up -d postgres
```

Pastikan `backend/.env` memakai URL database untuk PostgreSQL Compose:

```dotenv
DATABASE_URL=postgres://hubby:hubby@localhost:5432/hubby?sslmode=disable
```

Jika `backend/.env` sudah ada, ubah baris tersebut di file yang ada; jangan menyalin ulang
`.env.example` di atas file lokal. Jalankan migration dan API dari terminal pertama:

```powershell
cd backend
go run ./cmd/migrate up
go run ./cmd/api
```

Terminal API harus tetap terbuka. Pastikan health check mengembalikan status `ok`:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Pada database yang baru dibuat, atur `AUTH_EMAIL` dan `AUTH_INITIAL_PASSWORD` di `backend/.env`
sebelum menjalankan API untuk pertama kali. Kata sandi awal minimal 10 karakter. Konfigurasi
tersebut hanya menginisialisasi akun kosong; jika database sudah pernah dipakai, gunakan email
dan kata sandi akun yang tersimpan di database.

Setelah API sehat, dari terminal kedua:

```powershell
cd desktop
npm install
npm run dev
```

Tauri akan menjalankan Vite secara otomatis. Pada mode development, request `/api` diteruskan
oleh proxy Vite ke `http://localhost:8080`. Installer yang sudah terpasang juga memerlukan
PostgreSQL dan API ini tetap berjalan di komputer yang sama.

## Build

```powershell
cd desktop
npm run build
```

Build production memakai `frontend/.env.desktop`. Untuk memakai API VPS, isi `VITE_API_URL`
dengan URL HTTPS API, misalnya `https://api.example.com/api/v1`, sebelum menjalankan build.
Backend dan PostgreSQL tetap berjalan di server; installer desktop hanya menyertakan UI.

Origin desktop production adalah `http://localhost:1420`. Pada konfigurasi backend VPS,
`FRONTEND_ORIGIN` tetap menunjuk ke origin website HTTPS, lalu tambahkan origin desktop ke
`CORS_ALLOWED_ORIGINS`, misalnya `http://localhost:1420`. Backend mengizinkan cookie lintas situs
dengan atribut `SameSite=None; Secure` ketika `FRONTEND_ORIGIN` memakai HTTPS. Deploy ulang backend
setelah mengubah konfigurasi atau kode.

Jika backend hanya dapat dicapai melalui reverse proxy tepercaya, set `TRUST_PROXY=true` dan
pastikan proxy menimpa `X-Real-IP` serta `X-Forwarded-For` dengan alamat klien. Biarkan nilainya
`false` jika port backend dapat diakses langsung.

Installer atau executable hasil build tersedia di `src-tauri/target/release/bundle`.
