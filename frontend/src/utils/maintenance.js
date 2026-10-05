export const statusLabels = { good: 'Baik', due_soon: 'Segera jatuh tempo', due_today: 'Jatuh tempo hari ini', overdue: 'Terlambat', usage_unknown: 'Penggunaan belum dicatat', inactive: 'Nonaktif', unscheduled: 'Belum ada jadwal' }
export const statusRank = { overdue: 5, due_today: 4, due_soon: 3, usage_unknown: 2, good: 1, inactive: 0 }
export const usageLabels = { km: 'km', hours: 'jam', cycles: 'siklus' }
export const intervalLabels = { days: 'hari', weeks: 'minggu', months: 'bulan', years: 'tahun' }

export function localToday() {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Jakarta', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date())
}
export function formatMaintenanceDate(value) {
  return value ? new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(`${value}T00:00:00`)) : 'Belum dicatat'
}
export function usageDuration(start, today = localToday()) {
  if (!start) return 'Belum dicatat'
  if (start > today) return 'Belum mulai digunakan'
  const from = new Date(`${start}T00:00:00Z`)
  const to = new Date(`${today}T00:00:00Z`)
  let months = (to.getUTCFullYear() - from.getUTCFullYear()) * 12 + to.getUTCMonth() - from.getUTCMonth()
  const anniversary = new Date(Date.UTC(from.getUTCFullYear(), from.getUTCMonth() + months, 1))
  const lastDay = new Date(Date.UTC(anniversary.getUTCFullYear(), anniversary.getUTCMonth() + 1, 0)).getUTCDate()
  anniversary.setUTCDate(Math.min(from.getUTCDate(), lastDay))
  if (anniversary > to) months--
  if (months < 1) return `${Math.floor((to - from) / 86400000)} hari`
  return [months >= 12 ? `${Math.floor(months / 12)} tahun` : '', months % 12 ? `${months % 12} bulan` : ''].filter(Boolean).join(' ')
}
export function itemStatus(rules) {
  const active = rules.filter(rule => rule.active)
  return active.length ? active.reduce((status, rule) => statusRank[rule.status] > statusRank[status] ? rule.status : status, 'good') : 'unscheduled'
}
export function optionalNumber(value) { return value === '' || value === null || value === undefined ? null : Number(value) }
