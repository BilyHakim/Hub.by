import { spawnSync } from 'node:child_process'
import { copyFileSync, readdirSync, readFileSync, cpSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import assert from 'node:assert/strict'

const desktop = fileURLToPath(new URL('../', import.meta.url))
const root = path.dirname(desktop)
const cli = path.join(desktop, 'node_modules/@tauri-apps/cli/tauri.js')
const icons = path.join(desktop, 'src-tauri/icons')
const web = path.join(root, 'frontend/public')
const android = path.join(root, 'mobile-capacitor/android/app/src/main/res')

function generate(input, output, extra = []) {
  const result = spawnSync(process.execPath, [cli, 'icon', input, '--output', output, ...extra], {
    cwd: desktop,
    stdio: 'inherit',
  })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`Icon generation failed (${result.status})`)
}

generate('app-icon.json', icons)
generate('app-icon.svg', path.join(web, 'icons'), [16, 32, 180, 192, 512, 1024].flatMap(size => ['--png', String(size)]))
copyFileSync(path.join(desktop, 'app-icon.svg'), path.join(web, 'favicon.svg'))
copyFileSync(path.join(icons, 'icon.ico'), path.join(web, 'favicon.ico'))
for (const entry of readdirSync(path.join(icons, 'android'))) {
  if (/^mipmap-(?:mdpi|hdpi|xhdpi|xxhdpi|xxxhdpi)$/.test(entry)) {
    cpSync(path.join(icons, 'android', entry), path.join(android, entry), { recursive: true })
  }
}
for (const name of ['ic_launcher.xml', 'ic_launcher_round.xml']) {
  copyFileSync(path.join(icons, 'android/mipmap-anydpi-v26/ic_launcher.xml'), path.join(android, 'mipmap-anydpi-v26', name))
}
copyFileSync(path.join(icons, 'android/values/ic_launcher_background.xml'), path.join(android, 'values/ic_launcher_background.xml'))

for (const size of [16, 32, 180, 192, 512, 1024]) {
  const png = readFileSync(path.join(web, 'icons', `${size}x${size}.png`))
  assert.equal(png.subarray(1, 4).toString(), 'PNG')
  assert.equal(png.readUInt32BE(16), size)
  assert.equal(png.readUInt32BE(20), size)
}
assert.deepEqual(readFileSync(path.join(web, 'favicon.svg')), readFileSync(path.join(desktop, 'app-icon.svg')))
console.log('Hub.by icons generated and verified for web, desktop, and Android.')
