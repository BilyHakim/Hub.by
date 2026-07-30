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

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: DashboardView, meta: { title: 'Ringkasan' } },
    { path: '/transactions', name: 'transactions', component: TransactionsView, meta: { title: 'Arus kas' } },
    { path: '/goals', name: 'goals', component: GoalsView, meta: { title: 'Tujuan keuangan' } },
    { path: '/modules', name: 'modules', component: ModulesView, meta: { title: 'Perencanaan' } },
    { path: '/modules/pyramid', name: 'pyramid', component: PyramidView, meta: { title: 'Piramida keuangan' } },
    { path: '/modules/checkup', name: 'checkup', component: CheckupView, meta: { title: 'Financial check-up' } },
    { path: '/modules/emergency-fund', name: 'emergency-fund', component: EmergencyFundView, meta: { title: 'Dana darurat' } },
    { path: '/modules/mortgage', name: 'mortgage', component: MortgageView, meta: { title: 'Simulasi KPR' } },
    { path: '/modules/investments', name: 'investments', component: InvestmentsView, meta: { title: 'Monitor investasi' } },
    { path: '/modules/rebalancing', name: 'rebalancing', component: RebalancingView, meta: { title: 'Rebalancing' } },
    { path: '/modules/retirement', name: 'retirement', component: RetirementView, meta: { title: 'Persiapan pensiun' } },
    { path: '/modules/glossary', name: 'glossary', component: GlossaryView, meta: { title: 'Glosarium finansial' } },
    { path: '/settings', name: 'settings', component: SettingsView, meta: { title: 'Pengaturan' } },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

router.afterEach((to) => {
  document.title = `${to.meta.title || 'Keuangan'} · Hubby Finance`
})

export default router
