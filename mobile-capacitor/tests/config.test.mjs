import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { mobileRoot, frontendRoot, productionAPI, mobileConfig } from '../scripts/config.mjs'

test('mobile always targets the HTTPS production API and shared frontend', () => {
  const config = mobileConfig(() => ({}))
  assert.equal(productionAPI, 'https://bilyhakim.site/api/v1')
  assert.equal(config.define['import.meta.env.VITE_API_URL'], JSON.stringify(productionAPI))
  assert.equal(config.root, frontendRoot)
  assert.equal(config.build.outDir, path.join(mobileRoot, 'www'))
  assert.equal(config.server.proxy, undefined)
  assert.notEqual(config.build.outDir, path.join(frontendRoot, 'dist'))
})

test('native assets stay local and native HTTP handles authenticated API requests', () => {
  const config = JSON.parse(fs.readFileSync(path.join(mobileRoot, 'capacitor.config.json')))
  assert.equal(config.webDir, 'www')
  assert.equal(config.server.url, undefined)
  assert.equal(config.server.cleartext, false)
  assert.equal(config.android.allowMixedContent, false)
  assert.equal(config.plugins.CapacitorHttp.enabled, true)
})

test('mobile entry retains zoom and workers use a JavaScript extension', () => {
  const config = mobileConfig(() => ({}))
  const html = fs.readFileSync(path.join(frontendRoot, 'index.html'), 'utf8')
  const result = config.plugins[1].transformIndexHtml.handler(html)
  assert.match(result, /viewport-fit=cover/)
  assert.match(result, /src="\/@mobile\/main.js"/)
  assert.doesNotMatch(result, /user-scalable=no|maximum-scale=1/)
  assert.equal(config.build.rollupOptions.output.assetFileNames({ names: ['pdf.worker.min.mjs'] }), 'assets/[name]-[hash].js')
})

test('built assets contain the production endpoint and no database connection string', { skip: !fs.existsSync(path.join(mobileRoot, 'www/assets')) }, () => {
  const assets = path.join(mobileRoot, 'www/assets')
  const scripts = fs.readdirSync(assets).filter(name => name.endsWith('.js')).map(name => fs.readFileSync(path.join(assets, name), 'utf8'))
  assert(scripts.some(source => source.includes(productionAPI)))
  assert(scripts.every(source => !source.includes('postgres://') && !source.includes('postgresql://')))
  assert(fs.readdirSync(assets).some(name => /^pdf\.worker\.min-.+\.js$/.test(name)))
})
