const API_URL = import.meta.env.VITE_API_URL || '/api/v1'

async function request(path, options = {}) {
  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })

  if (!response.ok) {
    const payload = await response.json().catch(() => ({}))
    throw new Error(payload.error || 'Terjadi masalah saat memuat data.')
  }

  if (response.status === 204) return null
  const payload = await response.json()
  return payload.data
}

export const api = {
  dashboard: (month) => request(`/dashboard?month=${month}`),
  transactions: (month) => request(`/transactions?month=${month}`),
  recentTransactions: (afterId = 0) => request(`/transactions/recent?afterId=${afterId}`),
  createTransaction: (payload) => request('/transactions', { method: 'POST', body: JSON.stringify(payload) }),
  deleteTransaction: (id) => request(`/transactions/${id}`, { method: 'DELETE' }),
  accounts: () => request('/accounts'),
  createAccount: (payload) => request('/accounts', { method: 'POST', body: JSON.stringify(payload) }),
  updateAccount: (id, payload) => request(`/accounts/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }),
  deleteAccount: (id) => request(`/accounts/${id}`, { method: 'DELETE' }),
  categories: () => request('/categories'),
  createCategory: (payload) => request('/categories', { method: 'POST', body: JSON.stringify(payload) }),
  updateCategory: (id, payload) => request(`/categories/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }),
  deleteCategory: (id) => request(`/categories/${id}`, { method: 'DELETE' }),
  goals: () => request('/goals'),
  createGoal: (payload) => request('/goals', { method: 'POST', body: JSON.stringify(payload) }),
  replaceGoal: (id, payload) => request(`/goals/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  updateGoal: (id, currentAmount) => request(`/goals/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ currentAmount }),
  }),
  deleteGoal: (id) => request(`/goals/${id}`, { method: 'DELETE' }),
  investments: () => request('/investments'),
  createInvestment: (payload) => request('/investments', { method: 'POST', body: JSON.stringify(payload) }),
  updateInvestment: (id, payload) => request(`/investments/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }),
  deleteInvestment: (id) => request(`/investments/${id}`, { method: 'DELETE' }),
  me: () => request('/me'),
  updateMe: (payload) => request('/me', { method: 'PATCH', body: JSON.stringify(payload) }),
  workspaces: () => request('/workspaces'),
  createWorkspace: (name) => request('/workspaces', { method: 'POST', body: JSON.stringify({ name }) }),
  selectWorkspace: (workspaceId) => request('/me/workspace', {
    method: 'PATCH',
    body: JSON.stringify({ workspaceId }),
  }),
  pyramid: () => request('/modules/pyramid'),
  updatePyramidItem: (id, isCompleted) => request(`/modules/pyramid/items/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ isCompleted }),
  }),
  financialCheckup: (month) => request(`/modules/checkup?month=${month}`),
  emergencyFund: (month) => request(`/modules/emergency-fund?month=${month}`),
  updateEmergencyFund: (payload) => request('/modules/emergency-fund', {
    method: 'PATCH',
    body: JSON.stringify(payload),
  }),
  mortgage: () => request('/modules/mortgage'),
  updateMortgage: (payload) => request('/modules/mortgage', { method: 'PUT', body: JSON.stringify(payload) }),
  rebalancing: () => request('/modules/rebalancing'),
  retirement: () => request('/modules/retirement'),
  updateRetirement: (payload) => request('/modules/retirement', { method: 'PUT', body: JSON.stringify(payload) }),
  financePeriodSetting: (month = '') => request(`/settings/finance-period${month ? `?month=${month}` : ''}`),
  updateFinancePeriodSetting: (periodMode, periodStartDay) => request('/settings/finance-period', {
    method: 'PATCH',
    body: JSON.stringify({ periodMode, periodStartDay }),
  }),
}
