// Use the rendered module colors rather than maintaining a second palette.
export function systemBarTheme(color) {
  const channels = color.startsWith('#')
    ? color.slice(1).match(/.{2}/g).map(value => parseInt(value, 16))
    : color.match(/[\d.]+/g).slice(0, 3).map(Number)
  const linear = channels.map(value => {
    const channel = value / 255
    return channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4
  })
  const luminance = linear[0] * 0.2126 + linear[1] * 0.7152 + linear[2] * 0.0722
  return {
    color: `#${channels.map(value => value.toString(16).padStart(2, '0')).join('')}`,
    darkIcons: luminance > 0.179,
  }
}

export function syncSystemBars(nativeTheme) {
  let lastColor
  let frame
  async function update() {
    frame = undefined
    const surface = document.querySelector('.reader-is-fullscreen') || document.querySelector('.app-shell.module-shell')
    const color = surface
      ? getComputedStyle(surface).backgroundColor
      : getComputedStyle(document.documentElement).getPropertyValue('--paper').trim() || '#f6f4ef'
    const theme = systemBarTheme(color)
    if (theme.color === lastColor) return
    document.documentElement.style.setProperty('--mobile-bar-background', theme.color)
    try {
      await nativeTheme.setTheme(theme)
      lastColor = theme.color
    } catch (error) {
      console.warn('Could not update Android system bars', error)
    }
  }
  const observer = new MutationObserver(() => {
    if (frame === undefined) frame = requestAnimationFrame(update)
  })
  observer.observe(document.getElementById('app'), { childList: true, subtree: true, attributes: true, attributeFilter: ['class'] })
  update()
  return () => { observer.disconnect(); cancelAnimationFrame(frame) }
}
