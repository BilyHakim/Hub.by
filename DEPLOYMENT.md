# Deployment VPS

Panduan ini menjalankan PostgreSQL, migration, Go API, frontend, dan Caddy melalui
`compose.prod.yml`. Caddy melayani HTTP/HTTPS pada port 80 dan 443. Frontend juga
tersedia di loopback VPS pada port 8080; PostgreSQL dan backend tidak membuka port
langsung ke internet.

## Persiapan pertama

Di VPS, buat konfigurasi lokal yang tidak masuk Git:

```bash
cd /opt/hubby
if [ ! -f .env.production ]; then
  cp .env.production.example .env.production
fi
chmod 600 .env.production
nano .env.production
```

Jangan menyalin template di atas `.env.production` yang sudah ada. File produksi dapat berisi
credential aktif dan harus dipertahankan saat deploy berikutnya.

Nilai produksi utama:

```dotenv
FRONTEND_ORIGIN=https://bilyhakim.site
CORS_ALLOWED_ORIGINS=http://localhost:1420
TRUST_PROXY=true
```

Isi `POSTGRES_PASSWORD`, `DATABASE_URL`, `AUTH_EMAIL`, dan token opsional dengan
nilai sebenarnya. Jika password database mengandung karakter khusus seperti `@`,
encode bagian password pada `DATABASE_URL` (`@` menjadi `%40`).

`AUTH_INITIAL_PASSWORD` hanya menginisialisasi database baru. Setelah akun sudah
memiliki password hash, mengubah variabel tersebut tidak mengganti password akun.

Jika VPS sudah mempunyai `compose.prod.yml` lokal yang tidak tercatat Git, pindahkan
file tersebut sebelum pull pertama yang membawa file Compose dari repository:

```bash
mv compose.prod.yml compose.prod.yml.backup
git pull --ff-only origin main
```

Bandingkan konfigurasi lama bila ada nilai atau volume yang perlu dipertahankan.
`compose.prod.yml` baru menggunakan volume bernama logis `hubby_postgres`. Karena data lama
tidak boleh berpindah ke volume kosong, periksa mount database lama sebelum pertama kali
mengganti Compose:

```bash
docker compose -f compose.prod.yml.backup config
docker volume ls | grep hubby
```

Jika deployment lama memakai PostgreSQL eksternal, pertahankan `DATABASE_URL` lama dan sesuaikan
dependency `postgres` pada Compose sebelum menjalankan migration.

## Reverse proxy HTTPS

Caddy berada pada network `internal` yang sama dengan frontend dan meneruskan
request ke `frontend:80`. Konfigurasi repository ada di `deploy/Caddyfile`, dengan
domain `bilyhakim.site` dan `www.bilyhakim.site`. File `/opt/hubby/Caddyfile` lama
tetap dapat disimpan sebagai backup; service aktif memakai file di `deploy/`.

Sertifikat dan konfigurasi memakai volume eksternal yang sudah ada di VPS:
`hubby_caddy_data` dan `hubby_caddy_config`. Compose tidak membuat volume pengganti
yang kosong. Nama dapat disesuaikan melalui `CADDY_DATA_VOLUME` dan
`CADDY_CONFIG_VOLUME` pada `.env.production`; jika tidak diisi, nama di atas dipakai.
Pastikan kedua volume tersedia sebelum menjalankan service Caddy:

```bash
docker volume inspect hubby_caddy_data hubby_caddy_config --format '{{.Name}}'
```

Untuk instalasi baru saja, buat kedua volume dengan `docker volume create` sebelum
menjalankan Caddy. Untuk VPS yang sudah aktif, gunakan volume sertifikat lama.

Arahkan DNS kedua domain ke VPS dan pastikan port 80/TCP, 443/TCP, dan 443/UDP
tersedia untuk Caddy. Frontend hanya diterbitkan ke `127.0.0.1`, dan
`TRUST_PROXY=true` dipakai karena request publik melewati Caddy.

Jika sebelumnya Caddy dipulihkan lewat `compose.prod.yml.backup` dan
`docker network connect`, integrasikan service aktif setelah pull:

```bash
cd /opt/hubby
docker compose --env-file .env.production -f compose.prod.yml config --quiet
docker compose --env-file .env.production -f compose.prod.yml up -d --no-deps caddy
curl -I https://bilyhakim.site
```

Compose akan memakai service Caddy dalam project yang sama, dengan volume lama
dan network yang sekarang tercatat permanen. Recreate Caddy dapat memutus HTTPS
sebentar. Tidak perlu menjalankan `down` atau menghapus volume.

## Deploy pembaruan

Sebelum migration, buat backup database:

```bash
cd /opt/hubby
docker compose --env-file .env.production -f compose.prod.yml exec -T postgres \
  sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' \
  > "hubby-before-$(date +%Y%m%d-%H%M%S).dump"
```

Pull, build, jalankan migration, lalu recreate aplikasi:

```bash
cd /opt/hubby
git pull --ff-only origin main
docker compose --env-file .env.production -f compose.prod.yml build frontend backend migrate
docker compose --env-file .env.production -f compose.prod.yml up -d postgres
docker compose --env-file .env.production -f compose.prod.yml run --rm migrate up
docker compose --env-file .env.production -f compose.prod.yml up -d --force-recreate frontend backend
docker compose --env-file .env.production -f compose.prod.yml up -d --no-deps caddy
```

Frontend dan backend perlu dideploy bersamaan karena backend mewajibkan header
`X-Hubby-Client` yang dikirim oleh frontend versi terbaru.

Caddy kini tercantum dalam Compose aktif. Tetap hindari `--remove-orphans` pada
deploy rutin karena service lokal lain yang belum tercantum dapat ikut terhapus.

## Verifikasi

Halaman utama `https://bilyhakim.site/` kini menampilkan landing page publik Hub.by.
Login berada di `/login`, dan portal pemilihan modul berada di `/hub`.
Landing page tetap tampil sebelum login dan ketika API belum tersedia.
Perubahan ini perlu build dan deployment frontend; tidak membutuhkan migration baru.
Konfigurasi Nginx frontend sudah memakai fallback `index.html` untuk rute Vue.
Setelah deployment, periksa halaman utama tanpa sesi login, lalu masuk dan pastikan
portal `/hub` terbuka. Periksa juga tautan modul pada layar ponsel.

```bash
docker compose --env-file .env.production -f compose.prod.yml ps
docker compose --env-file .env.production -f compose.prod.yml logs --tail=100 caddy backend frontend
curl -fsS https://bilyhakim.site/health
```

Health check publik harus mengembalikan JSON dengan `"status":"ok"`, bukan HTML.
Periksa CORS desktop:

```bash
curl -i -X OPTIONS 'https://bilyhakim.site/api/v1/auth/login' \
  -H 'Origin: http://localhost:1420' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type,x-hubby-client'
```

Respons yang benar adalah `204` dengan header:

```text
Access-Control-Allow-Origin: http://localhost:1420
Access-Control-Allow-Credentials: true
```

Terakhir, login melalui website dan installer desktop terbaru untuk memastikan cookie
lintas situs diterima oleh WebView2.

File `.env.production`, private key, dan dump `*.dump` diabaikan Git. Tetap simpan backup
database di lokasi dengan permission terbatas dan jangan unggah ke artifact publik.

## Sebelum push ke GitHub

```bash
git status --short --ignored
git diff --check
git diff --cached
git ls-files | grep -E '(^|/)(\.env$|.*\.(pem|key|p12|pfx)$)'
```

Perintah terakhir seharusnya tidak menampilkan `.env` asli atau private key. File contoh seperti
`.env.production.example` memang sengaja dicatat Git. `.gitignore` tidak dapat menghapus rahasia
yang pernah masuk commit lama; credential seperti itu harus dirotasi dan histori perlu dibersihkan.
