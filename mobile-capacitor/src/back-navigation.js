export async function handleAndroidBack(router, app, doc = document, win = window) {
  if (doc.querySelector('.reader-is-fullscreen')) {
    win.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
  } else if (win.history.state?.back) {
    router.back()
  } else if (router.currentRoute.value.path !== '/hub') {
    await router.replace('/hub')
  } else {
    await app.minimizeApp()
  }
}
