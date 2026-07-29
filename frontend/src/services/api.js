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
  createTransaction: (payload) => request('/transactions', { method: 'POST', body: JSON.stringify(payload) }),
  deleteTransaction: (id) => request(`/transactions/${id}`, { method: 'DELETE' }),
  accounts: () => request('/accounts'),
  categories: () => request('/categories'),
  goals: () => request('/goals'),
  updateGoal: (id, currentAmount) => request(`/goals/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ currentAmount }),
  }),
  investments: () => request('/investments'),
  me: () => request('/me'),
  updateMe: (payload) => request('/me', { method: 'PATCH', body: JSON.stringify(payload) }),
  workspaces: () => request('/workspaces'),
  createWorkspace: (name) => request('/workspaces', { method: 'POST', body: JSON.stringify({ name }) }),
  selectWorkspace: (workspaceId) => request('/me/workspace', {
    method: 'PATCH',
    body: JSON.stringify({ workspaceId }),
  }),
}
