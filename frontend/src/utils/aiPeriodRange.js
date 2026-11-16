export function aiRangeError(startMonth, month) {
  const valid = value => /^(19\d{2}|20\d{2}|21\d{2}|2200)-(0[1-9]|1[0-2])$/.test(value)
  if (!valid(startMonth) || !valid(month)) return 'Pilih periode awal dan akhir yang valid.'
  const [startYear, start] = startMonth.split('-').map(Number)
  const [endYear, end] = month.split('-').map(Number)
  const count = (endYear - startYear) * 12 + end - start + 1
  if (count < 1) return 'Periode akhir harus sama atau setelah periode awal.'
  if (count > 12) return 'Pilih rentang maksimal 12 bulan.'
  return ''
}

export function defaultAIStartMonth(month) {
  const [year, value] = month.split('-').map(Number)
  const index = Math.max(1900 * 12, year * 12 + value - 1 - 3)
  return `${Math.floor(index / 12)}-${String(index % 12 + 1).padStart(2, '0')}`
}
