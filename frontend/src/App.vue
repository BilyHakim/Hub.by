<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  LayoutDashboard, ArrowLeftRight, Target, Blocks, Settings,
  Bell, Search, Menu, X, HeartHandshake, ChevronDown, Plus,
  Check, Ellipsis, UserRound, SlidersHorizontal, Users, WalletCards,
} from '@lucide/vue'
import { api } from './services/api'

const route = useRoute()
const sidebarOpen = ref(false)
const workspaceMenuOpen = ref(false)
const profileMenuOpen = ref(false)
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

const currentWorkspace = computed(() =>
  workspaces.value.find((item) => item.id === selectedWorkspaceId.value) || workspaces.value[0],
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
}
function toggleWorkspaceMenu() {
  profileMenuOpen.value = false
  workspaceMenuOpen.value = !workspaceMenuOpen.value
}
function toggleProfileMenu() {
  workspaceMenuOpen.value = false
  profileMenuOpen.value = !profileMenuOpen.value
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

onMounted(() => {
  loadIdentity()
  document.addEventListener('click', closeMenus)
  document.addEventListener('keydown', handleKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', closeMenus)
  document.removeEventListener('keydown', handleKeydown)
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
        <RouterLink to="/modules"><Settings :size="19" /> Pengaturan</RouterLink>
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
              <RouterLink class="dropdown-action" to="/modules" @click="closeMenus"><SlidersHorizontal :size="17" /> Pengaturan</RouterLink>
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
          <button class="icon-button has-notification" aria-label="Notifikasi"><Bell :size="20" /></button>
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
