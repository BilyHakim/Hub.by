import assert from 'node:assert/strict'
import test from 'node:test'
import { itemStatus, optionalNumber, usageDuration } from './maintenance.js'

test('usage age handles leap days, month ends, absent and future dates', () => {
  assert.equal(usageDuration('2025-01-12', '2026-09-12'), '1 tahun 8 bulan')
  assert.equal(usageDuration('2026-01-31', '2026-02-28'), '1 bulan')
  assert.equal(usageDuration('2024-02-29', '2025-02-28'), '1 tahun')
  assert.equal(usageDuration('2026-10-01', '2026-10-05'), '4 hari')
  assert.equal(usageDuration('', '2026-10-05'), 'Belum dicatat')
  assert.equal(usageDuration('2026-11-01', '2026-10-05'), 'Belum mulai digunakan')
})
test('item status uses the most urgent active rule and excludes unscheduled items', () => {
  assert.equal(itemStatus([]), 'unscheduled')
  assert.equal(itemStatus([{ active: false, status: 'overdue' }]), 'unscheduled')
  assert.equal(itemStatus([{ active: true, status: 'good' }, { active: true, status: 'due_today' }]), 'due_today')
  assert.equal(itemStatus([{ active: true, status: 'overdue' }, { active: true, status: 'good' }]), 'overdue')
  assert.equal(itemStatus([{ active: true, status: 'usage_unknown' }, { active: true, status: 'good' }]), 'usage_unknown')
})
test('optional amounts distinguish unknown from zero', () => {
  assert.equal(optionalNumber(''), null)
  assert.equal(optionalNumber(null), null)
  assert.equal(optionalNumber(0), 0)
  assert.equal(optionalNumber('2000'), 2000)
})
