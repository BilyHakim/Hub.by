import { spawnSync } from 'node:child_process'
import path from 'node:path'
import fs from 'node:fs'
import { mobileRoot } from './config.mjs'

const windows = process.platform === 'win32'
const env = { ...process.env }
const tools = path.join(mobileRoot, '.tools')
const localJdk = fs.existsSync(tools) && fs.readdirSync(tools).find(name => /^jdk-21[.]/.test(name) && fs.existsSync(path.join(tools, name, 'bin', windows ? 'java.exe' : 'java')))
if (localJdk) env.JAVA_HOME = path.join(tools, localJdk)
const defaultSDK = windows && env.LOCALAPPDATA && path.join(env.LOCALAPPDATA, 'Android', 'Sdk')
if (!env.ANDROID_HOME && defaultSDK && fs.existsSync(defaultSDK)) env.ANDROID_HOME = defaultSDK
const result = spawnSync(windows ? 'cmd.exe' : './gradlew', windows ? ['/d', '/s', '/c', 'gradlew.bat assembleDebug'] : ['assembleDebug'], {
  cwd: path.join(mobileRoot, 'android'),
  stdio: 'inherit',
  env,
})
if (result.error) throw result.error
process.exitCode = result.status ?? 1
