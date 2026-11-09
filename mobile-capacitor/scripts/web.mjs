import { pathToFileURL } from 'node:url'
import path from 'node:path'
import { frontendRoot, mobileConfig, productionAPI } from './config.mjs'

const { build, createServer, preview } = await import(pathToFileURL(path.join(frontendRoot, 'node_modules/vite/dist/node/index.js')))
const { default: vue } = await import(pathToFileURL(path.join(frontendRoot, 'node_modules/@vitejs/plugin-vue/dist/index.mjs')))
const config = mobileConfig(vue)
console.log(`Hubby mobile API: ${productionAPI}`)
switch (process.argv[2]) {
  case 'build': await build(config); break
  case 'dev': {
    const server = await createServer(config)
    await server.listen()
    server.printUrls()
    break
  }
  case 'preview': {
    const server = await preview(config)
    server.printUrls()
    break
  }
  default: throw new Error('Use build, dev, or preview.')
}
