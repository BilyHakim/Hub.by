<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  LayoutDashboard, ArrowLeftRight, Target, Blocks, Settings,
  Bell, Search, Menu, X, HeartHandshake, ChevronDown, Plus,
  Check, Ellipsis, UserRound, SlidersHorizontal, Users, WalletCards,
  ArrowDownLeft, ArrowUpRight,
} from '@lucide/vue'
import { api } from './services/api'

const route = useRoute()
const sidebarOpen = ref(false)
const workspaceMenuOpen = ref(false)
const profileMenuOpen = ref(false)
const notificationMenuOpen = ref(false)
const createWorkspaceOpen = ref(false)
const profileModalOpen = ref(false)
const savingWorkspace = ref(false)
const savingProfile = ref(false)
const toast = ref('')
const workspaceName = ref('')
const selectedWorkspaceId = ref(Number(localStorage.getItem('hubby-workspace-id')) || 1)

const profile = reactive({
  id: 1,
  displayName: 'Bily & Ami',
  email: 'bily@hubby.local',
  initials: 'BA',
  subtitle: 'Rencana bersama',
})
const profileForm = reactive({ displayName: '', email: '', subtitle: '' })
const workspaces = ref([
  { id: 1, name: 'Keluarga AmBil', initials: 'AB', role: 'owner' },
])
const guidanceNotifications = [
  {
    id: 'record-transactions',
    title: 'Catat transaksi hari ini',
    message: 'Jaga ringkasan keuangan tetap akurat dengan mencatat pemasukan atau pengeluaran terbaru.',
    time: 'Hari ini',
    to: '/transactions',
    icon: ArrowLeftRight,
    tone: 'sage',
  },
  {
    id: 'review-goals',
    title: 'Saatnya cek tujuan keuangan',
    message: 'Lihat kembali progres tujuan dan sesuaikan target kontribusi bulan ini.',
    time: 'Minggu ini',
    to: '/goals',
    icon: Target,
    tone: 'sand',
  },
  {
    id: 'planning-ready',
    title: 'Fitur perencanaan siap digunakan',
    message: 'Coba alat perencanaan untuk menyusun kondisi keuangan keluarga dengan lebih terarah.',
    time: 'Baru',
    to: '/modules',
    icon: Blocks,
    tone: 'lilac',
  },
]
const transactionNotifications = ref(loadTransactionNotifications())
const lastKnownTransactionIds = loadLastKnownTransactionIds()
const readNotificationIds = ref(loadReadNotificationIds())
let transactionPollTimer
let checkingTransactions = false

const currentWorkspace = computed(() =>
  workspaces.value.find((item) => item.id === selectedWorkspaceId.value) || workspaces.value[0],
)
const notifications = computed(() => [
  ...transactionNotifications.value.filter((item) => item.workspaceId === selectedWorkspaceId.value),
  ...guidanceNotifications,
])
const unreadNotificationCount = computed(() =>
  notifications.value.filter((item) => !readNotificationIds.value.includes(item.id)).length,
)
const nav = [
  { to: '/', label: 'Ringkasan', icon: LayoutDashboard },
  { to: '/transactions', label: 'Arus kas', icon: ArrowLeftRight },
  { to: '/goals', label: 'Tujuan keuangan', icon: Target },
  { to: '/modules', label: 'Perencanaan', icon: Blocks },
]

function closeMenus() {
  workspaceMenuOpen.value = false
  profileMenuOpen.value = false
  notificationMenuOpen.value = false
}
function toggleWorkspaceMenu() {
  profileMenuOpen.value = false
  notificationMenuOpen.value = false
  workspaceMenuOpen.value = !workspaceMenuOpen.value
}
function toggleProfileMenu() {
  workspaceMenuOpen.value = false
  notificationMenuOpen.value = false
  profileMenuOpen.value = !profileMenuOpen.value
}
function loadReadNotificationIds() {
  try {
    const saved = JSON.parse(localStorage.getItem('hubby-read-notifications') || '[]')
    return Array.isArray(saved) ? saved : []
  } catch {
    return []
  }
}
function loadTransactionNotifications() {
  try {
    const saved = JSON.parse(localStorage.getItem('hubby-transaction-notifications') || '[]')
    return Array.isArray(saved)
      ? saved.map((item) => ({
          ...item,
          icon: item.transactionType === 'income' || item.tone === 'sage' ? ArrowDownLeft : ArrowUpRight,
        }))
      : []
  } catch {
    return []
  }
}
function loadLastKnownTransactionIds() {
  try {
    const saved = JSON.parse(localStorage.getItem('hubby-last-transaction-ids') || '{}')
    return saved && typeof saved === 'object' && !Array.isArray(saved) ? saved : {}
  } catch {
    return {}
  }
}
function saveTransactionNotifications(items) {
  transactionNotifications.value = items.slice(0, 30)
  localStorage.setItem('hubby-transaction-notifications', JSON.stringify(transactionNotifications.value))
}
function saveReadNotificationIds(ids) {
  readNotificationIds.value = ids
  localStorage.setItem('hubby-read-notifications', JSON.stringify(ids))
}
function toggleNotificationMenu() {
  workspaceMenuOpen.value = false
  profileMenuOpen.value = false
  notificationMenuOpen.value = !notificationMenuOpen.value
}
function isNotificationUnread(id) {
  return !readNotificationIds.value.includes(id)
}
function markNotificationRead(id) {
  if (isNotificationUnread(id)) {
    saveReadNotificationIds([...readNotificationIds.value, id])
  }
  notificationMenuOpen.value = false
}
function markAllNotificationsRead() {
  saveReadNotificationIds(notifications.value.map((item) => item.id))
}
function formatCurrency(value) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(value)
}
function toTransactionNotification(item, workspaceId) {
  const isIncome = item.type === 'income'
  const detail = item.description?.trim() || item.category
  return {
    id: `transaction-${workspaceId}-${item.id}`,
    transactionId: item.id,
    transactionType: item.type,
    workspaceId,
    title: isIncome ? 'Pemasukan baru tercatat' : 'Pengeluaran baru tercatat',
    message: `${formatCurrency(item.amount)} · ${detail} · ${item.account}`,
    time: 'Baru saja',
    to: '/transactions',
    icon: isIncome ? ArrowDownLeft : ArrowUpRight,
    tone: isIncome ? 'sage' : 'rose',
  }
}
async function checkForNewTransactions() {
  if (checkingTransactions || document.hidden) return
  checkingTransactions = true
  const workspaceId = selectedWorkspaceId.value
  const hasBaseline = Object.prototype.hasOwnProperty.call(lastKnownTransactionIds, workspaceId)
  const knownID = Number(lastKnownTransactionIds[workspaceId] || 0)
  try {
    const items = await api.recentTransactions(knownID)
    if (workspaceId !== selectedWorkspaceId.value) return
    if (!items?.length) {
      if (!hasBaseline) {
        lastKnownTransactionIds[workspaceId] = 0
        localStorage.setItem('hubby-last-transaction-ids', JSON.stringify(lastKnownTransactionIds))
      }
      return
    }
    const latestID = Math.max(...items.map((item) => Number(item.id)))
    lastKnownTransactionIds[workspaceId] = latestID
    localStorage.setItem('hubby-last-transaction-ids', JSON.stringify(lastKnownTransactionIds))

    // Pemeriksaan pertama menetapkan baseline agar transaksi lama tidak muncul sebagai notifikasi baru.
    if (!hasBaseline) return
    const freshNotifications = items.map((item) => toTransactionNotification(item, workspaceId))
    const freshIDs = new Set(freshNotifications.map((item) => item.id))
    saveTransactionNotifications([
      ...freshNotifications,
      ...transactionNotifications.value.filter((item) => !freshIDs.has(item.id)),
    ])
    window.dispatchEvent(new CustomEvent('hubby:transactions-updated'))
    notify(freshNotifications.length === 1
      ? freshNotifications[0].title
      : `${freshNotifications.length} transaksi baru tercatat`)
  } catch {
    // API mungkin belum aktif; pemeriksaan berikutnya akan mencoba kembali.
  } finally {
    checkingTransactions = false
  }
}
function handleVisibilityChange() {
  if (!document.hidden) checkForNewTransactions()
}
function notify(message) {
  toast.value = message
  window.setTimeout(() => {
    if (toast.value === message) toast.value = ''
  }, 2600)
}
async function loadIdentity() {
  const [profileResult, workspaceResult] = await Promise.allSettled([api.me(), api.workspaces()])
  if (profileResult.status === 'fulfilled') {
    Object.assign(profile, profileResult.value)
    selectedWorkspaceId.value = profileResult.value.currentWorkspaceId || selectedWorkspaceId.value
  }
  if (workspaceResult.status === 'fulfilled' && workspaceResult.value.length) {
    workspaces.value = workspaceResult.value
  }
}
async function selectWorkspace(item) {
  workspaceMenuOpen.value = false
  try {
    await api.selectWorkspace(item.id)
  } catch {
    notify(`${item.name} dipilih secara lokal`)
  }
  selectedWorkspaceId.value = item.id
  localStorage.setItem('hubby-workspace-id', String(item.id))
  window.dispatchEvent(new CustomEvent('hubby:workspace-changed', {
    detail: { workspaceId: item.id, workspaceName: item.name },
  }))
  notify(`Berpindah ke ${item.name}`)
  checkForNewTransactions()
}
function openCreateWorkspace() {
  workspaceMenuOpen.value = false
  workspaceName.value = ''
  createWorkspaceOpen.value = true
}
async function createWorkspace() {
  const name = workspaceName.value.trim()
  if (name.length < 2) return
  savingWorkspace.value = true
  let created
  try {
    created = await api.createWorkspace(name)
  } catch {
    created = {
      id: Date.now(),
      name,
      initials: name.split(/\s+/).slice(0, 2).map((word) => word[0]).join('').toUpperCase(),
      role: 'owner',
    }
  } finally {
    savingWorkspace.value = false
  }
  workspaces.value.push(created)
  createWorkspaceOpen.value = false
  await selectWorkspace(created)
  notify(`${name} berhasil dibuat`)
}
function openProfile() {
  profileMenuOpen.value = false
  Object.assign(profileForm, {
    displayName: profile.displayName,
    email: profile.email,
    subtitle: profile.subtitle,
  })
  profileModalOpen.value = true
}
async function saveProfile() {
  savingProfile.value = true
  try {
    const updated = await api.updateMe(profileForm)
    Object.assign(profile, updated)
  } catch {
    Object.assign(profile, profileForm, {
      initials: profileForm.displayName.split(/\s+/).slice(0, 2).map((word) => word[0]).join('').toUpperCase(),
    })
  } finally {
    savingProfile.value = false
  }
  profileModalOpen.value = false
  notify('Profil berhasil diperbarui')
}
function handleKeydown(event) {
  if (event.key === 'Escape') {
    closeMenus()
    createWorkspaceOpen.value = false
    profileModalOpen.value = false
  }
}

onMounted(async () => {
  await loadIdentity()
  document.addEventListener('click', closeMenus)
  document.addEventListener('keydown', handleKeydown)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  await checkForNewTransactions()
  transactionPollTimer = window.setInterval(checkForNewTransactions, 15000)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', closeMenus)
  document.removeEventListener('keydown', handleKeydown)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.clearInterval(transactionPollTimer)
})
</script>

<template>
  <div class="app-shell">
    <div v-if="sidebarOpen" class="sidebar-backdrop" @click="sidebarOpen = false" />
    <aside class="sidebar" :class="{ 'is-open': sidebarOpen }">
      <div class="brand">
        <span class="brand-mark"><HeartHandshake :size="23" stroke-width="1.8" /></span>
        <span>
          <strong>hubby</strong>
          <small>finance</small>
        </span>
        <button class="icon-button mobile-close" aria-label="Tutup menu" @click="sidebarOpen = false"><X :size="19" /></button>
      </div>

      <div class="workspace-wrap" @click.stop>
        <button class="workspace-switcher" type="button" :aria-expanded="workspaceMenuOpen" @click="toggleWorkspaceMenu">
          <span class="avatar avatar-sage">{{ currentWorkspace?.initials }}</span>
          <span><small>Ruang keuangan</small><strong>{{ currentWorkspace?.name }}</strong></span>
          <ChevronDown :size="16" :class="{ rotated: workspaceMenuOpen }" />
        </button>
        <Transition name="dropdown">
          <div v-if="workspaceMenuOpen" class="dropdown-menu workspace-menu">
            <div class="dropdown-title"><span>Ruang keuangan</span><small>{{ workspaces.length }} ruang</small></div>
            <button v-for="item in workspaces" :key="item.id" class="workspace-option" type="button" @click="selectWorkspace(item)">
              <span class="avatar avatar-sage">{{ item.initials }}</span>
              <span><strong>{{ item.name }}</strong><small>{{ item.role === 'owner' ? 'Pemilik' : 'Anggota' }}</small></span>
              <Check v-if="item.id === selectedWorkspaceId" :size="17" />
            </button>
            <div class="dropdown-divider" />
            <button class="dropdown-action" type="button" @click="openCreateWorkspace"><Plus :size="17" /> Buat ruang baru</button>
            <RouterLink class="dropdown-action" to="/modules" @click="closeMenus"><Users :size="17" /> Kelola anggota</RouterLink>
          </div>
        </Transition>
      </div>

      <nav class="main-nav" aria-label="Navigasi utama">
        <p class="nav-label">Keuangan</p>
        <RouterLink v-for="item in nav" :key="item.to" :to="item.to" @click="sidebarOpen = false">
          <component :is="item.icon" :size="19" stroke-width="1.8" />
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="sidebar-bottom">
        <RouterLink to="/settings"><Settings :size="19" /> Pengaturan</RouterLink>
        <div class="profile-menu-wrap" @click.stop>
          <button class="user-card" type="button" :aria-expanded="profileMenuOpen" @click="toggleProfileMenu">
            <span class="avatar">{{ profile.initials }}</span>
            <span><strong>{{ profile.displayName }}</strong><small>{{ profile.subtitle }}</small></span>
            <Ellipsis :size="19" />
          </button>
          <Transition name="dropdown-up">
            <div v-if="profileMenuOpen" class="dropdown-menu profile-menu">
              <div class="profile-summary">
                <span class="avatar">{{ profile.initials }}</span>
                <span><strong>{{ profile.displayName }}</strong><small>{{ profile.email }}</small></span>
              </div>
              <div class="dropdown-divider" />
              <button class="dropdown-action" type="button" @click="openProfile"><UserRound :size="17" /> Profil saya</button>
              <RouterLink class="dropdown-action" to="/settings" @click="closeMenus"><SlidersHorizontal :size="17" /> Pengaturan</RouterLink>
              <div class="dropdown-footer"><WalletCards :size="14" /> Hubby Finance · Lokal</div>
            </div>
          </Transition>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <header class="topbar">
        <button class="icon-button menu-button" aria-label="Buka menu" @click="sidebarOpen = true"><Menu :size="21" /></button>
        <div class="search-box">
          <Search :size="18" />
          <input aria-label="Cari" placeholder="Cari transaksi, tujuan..." />
          <kbd>⌘ K</kbd>
        </div>
        <div class="topbar-actions">
          <div class="notification-wrap" @click.stop>
            <button
              class="icon-button notification-button"
              :class="{ 'has-notification': unreadNotificationCount > 0 }"
              type="button"
              aria-label="Notifikasi"
              aria-haspopup="dialog"
              :aria-expanded="notificationMenuOpen"
              @click="toggleNotificationMenu"
            >
              <Bell :size="20" />
              <span v-if="unreadNotificationCount" class="sr-only">{{ unreadNotificationCount }} notifikasi belum dibaca</span>
            </button>
            <Transition name="dropdown">
              <section v-if="notificationMenuOpen" class="notification-panel" aria-label="Notifikasi">
                <div class="notification-heading">
                  <div>
                    <h2>Notifikasi</h2>
                    <p>{{ unreadNotificationCount ? `${unreadNotificationCount} belum dibaca` : 'Semua sudah dibaca' }}</p>
                  </div>
                  <button
                    v-if="unreadNotificationCount"
                    type="button"
                    class="mark-read-button"
                    @click="markAllNotificationsRead"
                  >
                    <Check :size="14" /> Tandai semua dibaca
                  </button>
                </div>
                <div class="notification-list">
                  <RouterLink
                    v-for="item in notifications"
                    :key="item.id"
                    :to="item.to"
                    class="notification-item"
                    :class="{ unread: isNotificationUnread(item.id) }"
                    @click="markNotificationRead(item.id)"
                  >
                    <span class="notification-icon" :class="`tone-${item.tone}`">
                      <component :is="item.icon" :size="18" stroke-width="1.8" />
                    </span>
                    <span class="notification-copy">
                      <span class="notification-title-row">
                        <strong>{{ item.title }}</strong>
                        <i v-if="isNotificationUnread(item.id)" aria-hidden="true" />
                      </span>
                      <span>{{ item.message }}</span>
                      <small>{{ item.time }}</small>
                    </span>
                  </RouterLink>
                </div>
              </section>
            </Transition>
          </div>
          <span class="avatar top-avatar">{{ profile.initials }}</span>
        </div>
      </header>
      <div class="page-wrap" :key="route.path">
        <RouterView />
      </div>
    </main>

    <Transition name="toast">
      <div v-if="toast" class="app-toast"><Check :size="17" /> {{ toast }}</div>
    </Transition>

    <Teleport to="body">
      <div v-if="createWorkspaceOpen" class="modal-backdrop" @click.self="createWorkspaceOpen = false">
        <form class="modal shell-modal" @submit.prevent="createWorkspace">
          <div class="modal-heading">
            <div><p class="eyebrow">Ruang keuangan</p><h2>Buat ruang baru</h2></div>
            <button type="button" class="icon-button" @click="createWorkspaceOpen = false"><X :size="20" /></button>
          </div>
          <p class="modal-description">Pisahkan rencana keluarga, pribadi, atau bisnis dalam ruang yang berbeda.</p>
          <label>Nama ruang<input v-model="workspaceName" minlength="2" maxlength="80" autofocus placeholder="Contoh: Keuangan Pribadi" required /></label>
          <button class="primary-button full-button" :disabled="savingWorkspace">{{ savingWorkspace ? 'Membuat...' : 'Buat ruang keuangan' }}</button>
        </form>
      </div>

      <div v-if="profileModalOpen" class="modal-backdrop" @click.self="profileModalOpen = false">
        <form class="modal shell-modal" @submit.prevent="saveProfile">
          <div class="modal-heading">
            <div><p class="eyebrow">Akun lokal</p><h2>Profil saya</h2></div>
            <button type="button" class="icon-button" @click="profileModalOpen = false"><X :size="20" /></button>
          </div>
          <label>Nama tampilan<input v-model="profileForm.displayName" maxlength="80" required /></label>
          <label>Email<input v-model="profileForm.email" type="email" required /></label>
          <label>Keterangan singkat<input v-model="profileForm.subtitle" maxlength="80" placeholder="Contoh: Rencana bersama" /></label>
          <button class="primary-button full-button" :disabled="savingProfile">{{ savingProfile ? 'Menyimpan...' : 'Simpan perubahan' }}</button>
        </form>
      </div>
    </Teleport>
  </div>
</template>
