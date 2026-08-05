const API_URL = import.meta.env.VITE_API_URL || '/api/v1'

async function request(path, options = {}) {
  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })

  if (!response.ok) {
    const payload = await response.json().catch(() => ({}))
    const error = new Error(payload.error || 'Terjadi masalah saat memuat data.')
    error.status = response.status
    if (response.status === 401 && path !== '/auth/login') {
      window.dispatchEvent(new CustomEvent('hubby:unauthorized'))
    }
    throw error
  }

  if (response.status === 204) return null
  const payload = await response.json()
  return payload.data
}

export const api = {
  login: (payload) => request('/auth/login', { method: 'POST', body: JSON.stringify(payload) }),
  logout: () => request('/auth/logout', { method: 'POST' }),
  dashboard: (month) => request(`/dashboard?month=${month}`),
  transactions: (month) => request(`/transactions?month=${month}`),
  recentTransactions: (afterId = 0) => request(`/transactions/recent?afterId=${afterId}`),
  createTransaction: (payload) => request('/transactions', { method: 'POST', body: JSON.stringify(payload) }),
  deleteTransaction: (id) => request(`/transactions/${id}`, { method: 'DELETE' }),
  createTransfer: (payload) => request('/transfers', { method: 'POST', body: JSON.stringify(payload) }),
  deleteTransfer: (id) => request(`/transfers/${id}`, { method: 'DELETE' }),
  budget: (month) => request(`/budgets?month=${month}`),
  updateBudget: (month, items) => request(`/budgets?month=${month}`, {
    method: 'PUT',
    body: JSON.stringify({ items }),
  }),
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
  obligations: () => request('/obligations'),
  createObligation: (payload) => request('/obligations', { method: 'POST', body: JSON.stringify(payload) }),
  updateObligation: (id, payload) => request(`/obligations/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteObligation: (id) => request(`/obligations/${id}`, { method: 'DELETE' }),
  createObligationPayment: (id, payload) => request(`/obligations/${id}/payments`, { method: 'POST', body: JSON.stringify(payload) }),
  deleteObligationPayment: (id) => request(`/obligation-payments/${id}`, { method: 'DELETE' }),
  watch: () => request('/watch'),
  watchTitle: (id) => request(`/watch/titles/${id}`),
  createWatchTitle: (payload) => request('/watch/titles', { method: 'POST', body: JSON.stringify(payload) }),
  updateWatchTitleStatus: (id, status) => request(`/watch/titles/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  }),
  deleteWatchTitle: (id) => request(`/watch/titles/${id}`, { method: 'DELETE' }),
  createWatchSession: (payload) => request('/watch/sessions', { method: 'POST', body: JSON.stringify(payload) }),
  deleteWatchSession: (id) => request(`/watch/sessions/${id}`, { method: 'DELETE' }),
  createWatchSessionBatch: (payload) => request('/watch/sessions/batch', { method: 'POST', body: JSON.stringify(payload) }),
  watchProgress: (id) => request(`/watch/titles/${id}/progress`),
  searchWatchCatalog: (query, type = '') => request(`/watch/catalog/search?q=${encodeURIComponent(query)}${type ? `&type=${type}` : ''}`),
  watchCatalogTitle: (catalogId, mediaType) => request(`/watch/catalog/titles/${encodeURIComponent(catalogId)}?type=${encodeURIComponent(mediaType)}`),
  watchCatalogSeason: (catalogId, season) => request(`/watch/catalog/titles/${encodeURIComponent(catalogId)}/seasons/${season}`),
  books: () => request('/books'),
  bookTitle: (id) => request(`/books/titles/${id}`),
  createBookTitle: (payload) => request('/books/titles', { method: 'POST', body: JSON.stringify(payload) }),
  updateBookTitleStatus: (id, status) => request(`/books/titles/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  deleteBookTitle: (id) => request(`/books/titles/${id}`, { method: 'DELETE' }),
  createReadingSession: (payload) => request('/books/sessions', { method: 'POST', body: JSON.stringify(payload) }),
  deleteReadingSession: (id) => request(`/books/sessions/${id}`, { method: 'DELETE' }),
  searchBookCatalog: (query) => request(`/books/catalog/search?q=${encodeURIComponent(query)}`),
  bookCatalogWork: (workId) => request(`/books/catalog/works/${encodeURIComponent(workId)}`),
  me: () => request('/me'),
  updateMe: (payload) => request('/me', { method: 'PATCH', body: JSON.stringify(payload) }),
  workspaces: () => request('/workspaces'),
  createWorkspace: (name) => request('/workspaces', { method: 'POST', body: JSON.stringify({ name }) }),
  deleteWorkspace: (id) => request(`/workspaces/${id}`, { method: 'DELETE' }),
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
