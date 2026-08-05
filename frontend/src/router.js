import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from './views/DashboardView.vue'
import TransactionsView from './views/TransactionsView.vue'
import GoalsView from './views/GoalsView.vue'
import ModulesView from './views/ModulesView.vue'
import PyramidView from './views/PyramidView.vue'
import CheckupView from './views/CheckupView.vue'
import EmergencyFundView from './views/EmergencyFundView.vue'
import SettingsView from './views/SettingsView.vue'
import MortgageView from './views/MortgageView.vue'
import InvestmentsView from './views/InvestmentsView.vue'
import RebalancingView from './views/RebalancingView.vue'
import RetirementView from './views/RetirementView.vue'
import GlossaryView from './views/GlossaryView.vue'
import BudgetView from './views/BudgetView.vue'
import ObligationsView from './views/ObligationsView.vue'
import WatchView from './views/WatchView.vue'
import HubView from './views/HubView.vue'
import WatchDetailView from './views/WatchDetailView.vue'
import BooksView from './views/BooksView.vue'
import BookDetailView from './views/BookDetailView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'hub', component: HubView, meta: { title: 'Pilih modul', product: 'Hubby', layout: 'portal' } },
    { path: '/finance', name: 'dashboard', component: DashboardView, meta: { title: 'Ringkasan', product: 'Finance' } },
    { path: '/finance/transactions', name: 'transactions', component: TransactionsView, meta: { title: 'Arus kas', product: 'Finance' } },
    { path: '/finance/goals', name: 'goals', component: GoalsView, meta: { title: 'Tujuan keuangan', product: 'Finance' } },
    { path: '/finance/modules', name: 'modules', component: ModulesView, meta: { title: 'Perencanaan', product: 'Finance' } },
    { path: '/finance/modules/budget', name: 'budget', component: BudgetView, meta: { title: 'Rencana pengeluaran', product: 'Finance' } },
    { path: '/finance/modules/obligations', name: 'obligations', component: ObligationsView, meta: { title: 'Utang & piutang', product: 'Finance' } },
    { path: '/finance/modules/pyramid', name: 'pyramid', component: PyramidView, meta: { title: 'Piramida keuangan', product: 'Finance' } },
    { path: '/finance/modules/checkup', name: 'checkup', component: CheckupView, meta: { title: 'Financial check-up', product: 'Finance' } },
    { path: '/finance/modules/emergency-fund', name: 'emergency-fund', component: EmergencyFundView, meta: { title: 'Dana darurat', product: 'Finance' } },
    { path: '/finance/modules/mortgage', name: 'mortgage', component: MortgageView, meta: { title: 'Simulasi KPR', product: 'Finance' } },
    { path: '/finance/modules/investments', name: 'investments', component: InvestmentsView, meta: { title: 'Monitor investasi', product: 'Finance' } },
    { path: '/finance/modules/rebalancing', name: 'rebalancing', component: RebalancingView, meta: { title: 'Rebalancing', product: 'Finance' } },
    { path: '/finance/modules/retirement', name: 'retirement', component: RetirementView, meta: { title: 'Persiapan pensiun', product: 'Finance' } },
    { path: '/finance/modules/glossary', name: 'glossary', component: GlossaryView, meta: { title: 'Glosarium finansial', product: 'Finance' } },
    { path: '/finance/settings', name: 'settings', component: SettingsView, meta: { title: 'Pengaturan', product: 'Finance' } },
    { path: '/watch', name: 'watch', component: WatchView, meta: { title: 'Watch', product: 'Watch' } },
    { path: '/watch/:id', name: 'watch-detail', component: WatchDetailView, meta: { title: 'Detail tontonan', product: 'Watch' } },
    { path: '/books', name: 'books', component: BooksView, meta: { title: 'Books', product: 'Books' } },
    { path: '/books/:id', name: 'book-detail', component: BookDetailView, meta: { title: 'Detail buku', product: 'Books' } },
    { path: '/transactions', redirect: '/finance/transactions' },
    { path: '/goals', redirect: '/finance/goals' },
    { path: '/modules', redirect: '/finance/modules' },
    { path: '/modules/:pathMatch(.*)*', redirect: (to) => `/finance/modules/${Array.isArray(to.params.pathMatch) ? to.params.pathMatch.join('/') : to.params.pathMatch}` },
    { path: '/settings', redirect: '/finance/settings' },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

router.afterEach((to) => {
  const product = to.meta.product === 'Watch' ? 'Hubby Watch' : to.meta.product === 'Books' ? 'Hubby Books' : to.meta.product === 'Finance' ? 'Hubby Finance' : 'Hubby'
  document.title = `${to.meta.title || 'Hubby'} · ${product}`
})

export default router
