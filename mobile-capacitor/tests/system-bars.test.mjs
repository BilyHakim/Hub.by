import test from 'node:test'
import assert from 'node:assert/strict'
import { systemBarTheme, syncSystemBars } from '../src/system-bars.mjs'

test('system icons remain light against every dark module frame', () => {
  for (const color of ['#304c42', '#4e3f68', '#624125', '#733e38', '#354f59']) {
    assert.deepEqual(systemBarTheme(color), { color, darkIcons: false })
  }
})

test('portal, login, and fullscreen Reader use dark icons on paper', () => {
  assert.deepEqual(systemBarTheme('#f6f4ef'), { color: '#f6f4ef', darkIcons: true })
})

test('computed CSS colors match their native hex color and contrast', () => {
  assert.deepEqual(systemBarTheme('rgb(48, 76, 66)'), systemBarTheme('#304c42'))
  assert.deepEqual(systemBarTheme('rgb(246, 244, 239)'), systemBarTheme('#f6f4ef'))
})

test('system bars follow module, fullscreen, and portal transitions without repeated native calls', async t => {
  let mutation
  let fullscreen = null
  let shell = { backgroundColor: 'rgb(48, 76, 66)' }
  let disconnected = false
  const frames = []
  const calls = []
  const backgrounds = []
  const root = { style: { setProperty: (name, value) => backgrounds.push(value) } }
  const globals = {
    document: {
      documentElement: root,
      getElementById: () => ({}),
      querySelector: selector => selector === '.reader-is-fullscreen' ? fullscreen : shell,
    },
    getComputedStyle: element => element === root ? { getPropertyValue: () => '#f6f4ef' } : element,
    MutationObserver: class {
      constructor(callback) { mutation = callback }
      observe() {}
      disconnect() { disconnected = true }
    },
    requestAnimationFrame: callback => { frames.push(callback); return frames.length },
    cancelAnimationFrame: () => {},
  }
  for (const [name, value] of Object.entries(globals)) {
    const original = Object.getOwnPropertyDescriptor(globalThis, name)
    Object.defineProperty(globalThis, name, { configurable: true, writable: true, value })
    t.after(() => {
      if (original) Object.defineProperty(globalThis, name, original)
      else delete globalThis[name]
    })
  }
  const stop = syncSystemBars({ setTheme: async theme => calls.push(theme) })
  await Promise.resolve()
  async function render() { mutation(); await frames.shift()() }
  shell = { backgroundColor: 'rgb(78, 63, 104)' }
  await render()
  await render() // Updating other page content should not touch native bars.
  shell = { backgroundColor: 'rgb(98, 65, 37)' }
  await render()
  fullscreen = { backgroundColor: 'rgb(246, 244, 239)' }
  await render() // Fullscreen takes precedence over its enclosing module shell.
  fullscreen = null
  await render()
  shell = null
  await render()
  assert.deepEqual(calls.map(theme => theme.color), ['#304c42', '#4e3f68', '#624125', '#f6f4ef', '#624125', '#f6f4ef'])
  assert.deepEqual(calls.map(theme => theme.darkIcons), [false, false, false, true, false, true])
  assert.deepEqual(backgrounds, calls.map(theme => theme.color))
  stop()
  assert.equal(disconnected, true)
})
