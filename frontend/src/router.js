import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from './views/DashboardView.vue'
import TransactionsView from './views/TransactionsView.vue'
import GoalsView from './views/GoalsView.vue'
import ModulesView from './views/ModulesView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: DashboardView, meta: { title: 'Ringkasan' } },
    { path: '/transactions', name: 'transactions', component: TransactionsView, meta: { title: 'Arus kas' } },
    { path: '/goals', name: 'goals', component: GoalsView, meta: { title: 'Tujuan keuangan' } },
    { path: '/modules', name: 'modules', component: ModulesView, meta: { title: 'Perencanaan' } },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

export default router

