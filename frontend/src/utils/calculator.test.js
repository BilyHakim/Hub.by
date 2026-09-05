import { test } from 'node:test'
import assert from 'node:assert/strict'
import { calculate } from './calculator.js'

test('calculates amounts with standard arithmetic precedence', () => {
  assert.equal(calculate('25000 + 15000 × 2'), 55000)
  assert.equal(calculate('100000 − 20000 ÷ 2'), 90000)
  assert.equal(calculate('12,5 * 4'), 50)
  assert.equal(calculate('10 / 4'), 2.5)
  assert.equal(calculate('-20 + 5'), -15)
  assert.equal(calculate('5 * -2'), -10)
  assert.equal(calculate('100 / 2 / 5'), 10)
})

test('rejects incomplete, unsafe, and invalid calculations', () => {
  for (const expression of ['', '20 +', '1..2', '2(3)', 'alert(1)', '1/0', '0/0', '99999999999999999', '2 ** 3']) {
    assert.throws(() => calculate(expression), undefined, expression)
  }
})
