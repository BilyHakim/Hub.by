import { Capacitor } from '@capacitor/core'
import { App } from '@capacitor/app'
import router from '../../frontend/src/router'
import './mobile.css'
import { handleAndroidBack } from './back-navigation'

document.documentElement.classList.add('mobile-capacitor')

async function start() {
  if (Capacitor.isNativePlatform()) {
    await App.addListener('backButton', () => handleAndroidBack(router, App))
  }
  if (window.location.pathname === '/' || window.location.pathname === '/index.html') await router.replace('/hub')
  await import('../../frontend/src/main.js')
}

start().catch(error => {
  console.error('Mobile initialization failed', error)
  const message = document.createElement('p')
  message.setAttribute('role', 'alert')
  message.textContent = 'Aplikasi belum dapat dibuka. Tutup lalu buka kembali Hubby.'
  document.getElementById('app').replaceChildren(message)
})
