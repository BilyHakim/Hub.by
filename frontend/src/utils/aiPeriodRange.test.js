import assert from 'node:assert/strict'
import test from 'node:test'
import { aiRangeError, defaultAIStartMonth } from './aiPeriodRange.js'

test('accepts a single month, cross-year ranges, and twelve inclusive months', () => {
  assert.equal(aiRangeError('2026-11', '2026-11'), '')
  assert.equal(aiRangeError('2025-12', '2026-02'), '')
  assert.equal(aiRangeError('2025-12', '2026-11'), '')
})

test('rejects reversed, oversized, malformed, and unsupported periods', () => {
  for (const [start, end] of [['2026-11', '2026-10'], ['2025-11', '2026-11'], ['', '2026-11'], ['2026-00', '2026-11'], ['2026-01', '2026-13'], ['1899-12', '1900-01'], ['2200-12', '2201-01'], ['2026-1', '2026-11']]) {
    assert.notEqual(aiRangeError(start, end), '', `${start} to ${end}`)
  }
})

test('default range includes the last four months and clamps the lower boundary', () => {
  assert.equal(defaultAIStartMonth('2026-11'), '2026-08')
  assert.equal(defaultAIStartMonth('2026-02'), '2025-11')
  assert.equal(defaultAIStartMonth('1900-01'), '1900-01')
})
