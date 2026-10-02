import { createServer } from 'node:http'
import { readFile, mkdir } from 'node:fs/promises'
import { resolve, extname } from 'node:path'
import assert from 'node:assert/strict'

const { puppeteer } = await import(process.env.CHROME_TOOLS_MODULE)
const root = resolve('frontend/dist')
const server = createServer(async (req, res) => {
  const pathname = new URL(req.url, 'http://localhost').pathname
  const file = pathname.startsWith('/assets/') || pathname === '/favicon.svg' ? resolve(root, `.${pathname}`) : resolve(root, 'index.html')
  try {
    res.setHeader('Content-Type', ({ '.js': 'text/javascript', '.css': 'text/css', '.svg': 'image/svg+xml' })[extname(file)] || 'text/html')
    res.end(await readFile(file))
  } catch { res.writeHead(404).end() }
})
await new Promise(resolve => server.listen(4178, '127.0.0.1', resolve))
const browser = await puppeteer.launch({ executablePath: 'C:/Program Files/Google/Chrome/Application/chrome.exe', headless: true })
const page = await browser.newPage()
const errors = []
page.on('pageerror', error => errors.push(error.message))
let loggedIn = false
let rejectLogin = true
let apiUnavailable = false
await page.setRequestInterception(true)
page.on('request', request => {
  if (request.url().includes('fonts.googleapis') || request.url().includes('fonts.gstatic')) return request.abort()
  if (request.url().includes('/api/v1/')) {
    if (apiUnavailable) return request.abort()
    const headers = {
      'Access-Control-Allow-Origin': 'http://127.0.0.1:4178',
      'Access-Control-Allow-Credentials': 'true',
      'Access-Control-Allow-Methods': 'GET, POST, PATCH, DELETE, OPTIONS',
      'Access-Control-Allow-Headers': 'content-type,x-hubby-client',
    }
    if (request.method() === 'OPTIONS') return request.respond({ status: 204, headers })
    const path = new URL(request.url()).pathname
    let status = 200
    let data = []
    if (path.endsWith('/auth/login')) {
      if (rejectLogin) return request.respond({ status: 401, headers, contentType: 'application/json', body: JSON.stringify({ error: 'Email atau kata sandi salah.' }) })
      loggedIn = true
      data = {}
    } else if (path.endsWith('/me')) {
      if (!loggedIn) status = 401
      data = { id: 1, displayName: 'Akun pengujian', email: 'test@example.com', initials: 'AP', currentWorkspaceId: 1 }
    } else if (path.endsWith('/workspaces')) data = [{ id: 1, name: 'Ruang pengujian', initials: 'RP', role: 'owner' }]
    else if (path.endsWith('/auth/logout')) loggedIn = false
    return request.respond({ status, headers, contentType: 'application/json', body: JSON.stringify(status === 401 ? { error: 'Masuk untuk mengakses akun.' } : { data }) })
  }
  request.continue()
})
const url = 'http://127.0.0.1:4178'
const record = message => console.log(`PASS: ${message}`)
async function open(path = '/') { await page.goto(`${url}${path}`, { waitUntil: 'networkidle0' }) }
try {
  await mkdir('anti-slop/evidence', { recursive: true })
  for (const width of [320, 375, 640, 768, 960, 1280, 1440]) {
    await page.setViewport({ width, height: 900 })
    await open()
    assert.equal(await page.$eval('h1', el => el.textContent), 'Catatan sehari-hari,punya tempat sendiri.')
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
    const smallTargets = await page.$$eval('.landing a', links => links.filter(el => el.getBoundingClientRect().height < 44).map(el => el.textContent))
    assert.deepEqual(smallTargets, [])
    record(`layout ${width}px tanpa overflow, seluruh target minimal 44px`)
    if ([375, 1440].includes(width)) await page.screenshot({ path: `anti-slop/evidence/landing-${width}.png`, fullPage: true })
  }
  for (const width of [320, 768, 1440]) {
    await page.setViewport({ width, height: 900 })
    await open()
    await page.evaluate(() => { document.documentElement.style.fontSize = '200%' })
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
    record(`teks 200% pada ${width}px tanpa overflow`)
  }
  await open()
  await page.keyboard.press('Tab')
  assert.equal(await page.evaluate(() => document.activeElement.textContent), 'Lewati ke konten')
  await page.keyboard.press('Enter')
  assert.equal(await page.evaluate(() => document.activeElement.id), 'konten')
  await open()
  const tabLabels = []
  for (let i = 0; i < 10; i++) {
    await page.keyboard.press('Tab')
    tabLabels.push(await page.evaluate(() => document.activeElement.textContent.trim()))
    assert.equal(await page.evaluate(() => getComputedStyle(document.activeElement).outlineStyle), 'solid')
  }
  assert.deepEqual(tabLabels, ['Lewati ke konten', 'Hub.by', 'Lihat fitur', 'Masuk', 'Masuk ke Hub.by', 'Buka Finance', 'Buka Watch', 'Buka Books', 'Masuk dan pilih modul', 'Kembali ke atas'])
  record('Tab mengikuti urutan konten, focus terlihat, skip link Enter memindahkan fokus')
  await page.keyboard.down('Shift')
  await page.keyboard.press('Tab')
  await page.keyboard.up('Shift')
  assert.equal(await page.evaluate(() => document.activeElement.textContent.trim()), 'Masuk dan pilih modul')
  record('Shift+Tab bekerja')
  for (const width of [375, 1440]) {
  await page.setViewport({ width, height: 900 })
  for (const label of tabLabels.slice(1)) {
    await open()
    await page.$$eval('.landing a', (links, label) => links.find(el => el.textContent.trim() === label).click(), label)
    await new Promise(resolve => setTimeout(resolve, 150))
    if (['Masuk', 'Masuk ke Hub.by', 'Masuk dan pilih modul', 'Buka Finance', 'Buka Watch', 'Buka Books'].includes(label)) assert.ok(await page.$('.login-form'))
    else if (label === 'Lihat fitur') assert.equal(await page.evaluate(() => location.hash), '#fitur')
    else if (label === 'Kembali ke atas') assert.equal(await page.evaluate(() => location.hash), '#landing-title')
    record(`klik ${width}px ${label} → ${new URL(page.url()).pathname}${new URL(page.url()).hash}`)
  }
  }
  await open('/login')
  await page.click('.login-submit')
  assert.equal(await page.$eval('input[type=email]', el => el.validity.valid), false)
  await page.type('input[type=email]', 'test@example.com')
  await page.type('input[autocomplete=current-password]', 'testing-password')
  await page.click('.login-input button')
  assert.equal(await page.$eval('input[autocomplete=current-password]', el => el.type), 'text')
  await page.click('.login-input button')
  assert.equal(await page.$eval('input[autocomplete=current-password]', el => el.type), 'password')
  await page.click('.login-submit')
  await page.waitForSelector('[role=alert]')
  assert.match(await page.$eval('[role=alert]', el => el.textContent), /salah/)
  record('login: validasi kosong, tampil/sembunyikan sandi, pesan error dari API')
  rejectLogin = false
  await page.type('input[autocomplete=current-password]', 'testing-password')
  await page.click('.login-submit')
  await page.waitForSelector('.hub-home')
  assert.equal(new URL(page.url()).pathname, '/hub')
  record('login sukses dengan API uji → /hub, portal tetap tersedia')
  await open()
  assert.ok(await page.$('.landing'))
  for (const selector of ['.landing-entry', '.landing-intro .landing-cta', '.landing-closing .landing-cta']) {
    await open()
    await page.click(selector)
    assert.equal(new URL(page.url()).pathname, '/hub')
    record(`sesi aktif: ${selector} → /hub`)
  }
  await page.click('.hub-portal-profile button')
  await page.waitForSelector('.landing')
  record('keluar → landing publik')
  await open('/login')
  await page.click('.login-back')
  await page.waitForSelector('.landing')
  record('Kembali ke Hub.by → /')
  apiUnavailable = true
  await open()
  assert.ok(await page.$('.landing'))
  record('API tidak tersedia: landing publik tetap tampil')
  await open('/login')
  await page.type('input[type=email]', 'test@example.com')
  await page.type('input[autocomplete=current-password]', 'testing-password')
  await page.click('.login-submit')
  await page.waitForSelector('[role=alert]')
  assert.match(await page.$eval('[role=alert]', el => el.textContent), /Tidak dapat terhubung/)
  record('API tidak tersedia: login menampilkan pesan pemulihan berbahasa Indonesia')
  assert.deepEqual(errors, [])
  record('tidak ada error JavaScript aplikasi')
  const luminance = hex => {
    const values = hex.match(/\w\w/g).map(v => parseInt(v, 16) / 255).map(v => v <= .03928 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4)
    return values.reduce((sum, v, i) => sum + v * [.2126, .7152, .0722][i], 0)
  }
  for (const [fg, bg] of [['24332d','f6f4ef'],['526159','f6f4ef'],['24332d','ebe8df'],['526159','ebe8df'],['24332d','dfe8e2'],['526159','dfe8e2'],['fffefa','49685c'],['fffefa','304c42'],['526159','fffefa'],['853b34','f6e4e1']]) {
    const a = luminance(fg), b = luminance(bg), ratio = (Math.max(a,b)+.05)/(Math.min(a,b)+.05)
    assert.ok(ratio >= 4.5)
    record(`kontras #${fg}/#${bg} ${ratio.toFixed(2)}:1`)
  }
} finally {
  await browser.close()
  await new Promise(resolve => server.close(resolve))
}
