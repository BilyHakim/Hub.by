# Deployment VPS

Panduan ini menjalankan PostgreSQL, migration, Go API, dan frontend melalui
`compose.prod.yml`. Hanya frontend yang diterbitkan ke loopback VPS pada port 8080;
PostgreSQL dan backend tidak membuka port langsung ke internet.

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

Arahkan domain ke frontend Compose pada `127.0.0.1:8080`. Contoh blok Nginx:

```nginx
server {
    listen 443 ssl http2;
    server_name bilyhakim.site;

    # ssl_certificate dan ssl_certificate_key dikelola di VPS, bukan di repository.

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Port 8080 harus diblokir dari jaringan publik. `TRUST_PROXY=true` aman hanya jika
request ke aplikasi selalu melewati reverse proxy tersebut.

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
docker compose --env-file .env.production -f compose.prod.yml up -d --force-recreate --remove-orphans frontend backend
```

Frontend dan backend perlu dideploy bersamaan karena backend mewajibkan header
`X-Hubby-Client` yang dikirim oleh frontend versi terbaru.

## Verifikasi

```bash
docker compose --env-file .env.production -f compose.prod.yml ps
docker compose --env-file .env.production -f compose.prod.yml logs --tail=100 backend frontend
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
