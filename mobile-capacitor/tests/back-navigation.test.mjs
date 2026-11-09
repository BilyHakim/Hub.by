import test from 'node:test'
import assert from 'node:assert/strict'
import { handleAndroidBack } from '../src/back-navigation.js'

test('Android Back navigates history, returns to hub, or minimizes at root', async () => {
  const calls = []
  const router = { currentRoute: { value: { path: '/reader' } }, back: () => calls.push('back'), replace: async path => calls.push(path) }
  const app = { minimizeApp: async () => calls.push('minimize') }
  const doc = { querySelector: () => null }
  const win = { history: { state: { back: '/hub' } } }
  await handleAndroidBack(router, app, doc, win)
  assert.deepEqual(calls, ['back'])
  win.history.state = null
  await handleAndroidBack(router, app, doc, win)
  assert.deepEqual(calls, ['back', '/hub'])
  router.currentRoute.value.path = '/hub'
  await handleAndroidBack(router, app, doc, win)
  assert.deepEqual(calls, ['back', '/hub', 'minimize'])
})

test('Android Back exits the reader before navigating away', async () => {
  const previousKeyboardEvent = globalThis.KeyboardEvent
  globalThis.KeyboardEvent = class { constructor(type, options) { this.type = type; Object.assign(this, options) } }
  try {
    const events = []
    await handleAndroidBack({ back: () => assert.fail('must not leave reader') }, {}, { querySelector: () => ({}) }, { dispatchEvent: event => events.push(event) })
    assert.equal(events[0].type, 'keydown')
    assert.equal(events[0].key, 'Escape')
  } finally { globalThis.KeyboardEvent = previousKeyboardEvent }
})
