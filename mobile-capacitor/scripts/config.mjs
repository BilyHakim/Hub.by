import { fileURLToPath } from 'node:url'
import path from 'node:path'

export const mobileRoot = fileURLToPath(new URL('../', import.meta.url))
export const frontendRoot = path.resolve(mobileRoot, '../frontend')
export const productionAPI = 'https://bilyhakim.site/api/v1'

export function mobileConfig(vue) {
  return {
    configFile: false,
    root: frontendRoot,
    envDir: mobileRoot,
    base: '/',
    define: { 'import.meta.env.VITE_API_URL': JSON.stringify(productionAPI) },
    plugins: [
      vue(),
      {
        name: 'hubby-mobile-entry',
        enforce: 'pre',
        transformIndexHtml: {
          order: 'pre',
          handler: html => html.replace('src="/src/main.js"', 'src="/@mobile/main.js"').replace('width=device-width, initial-scale=1.0', 'width=device-width, initial-scale=1.0, viewport-fit=cover'),
        },
        resolveId: source => source === '/@mobile/main.js' ? path.resolve(mobileRoot, 'src/main.js') : undefined,
      },
    ],
    server: { host: 'localhost', port: 1420, strictPort: true, fs: { allow: [frontendRoot, mobileRoot] } },
    preview: { host: 'localhost', port: 1420, strictPort: true },
    build: {
      outDir: path.resolve(mobileRoot, 'www'),
      emptyOutDir: true,
      rollupOptions: {
        output: {
          assetFileNames: asset => (asset.names?.[0] || asset.name || '').endsWith('.mjs') ? 'assets/[name]-[hash].js' : 'assets/[name]-[hash][extname]',
        },
      },
    },
  }
}
