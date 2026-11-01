import { defineManifest } from '@crxjs/vite-plugin'
import { loadEnv } from 'vite'

import pkg from './package.json'

export default defineManifest(({ mode }) => {
  const env = loadEnv(mode, process.cwd())
  const api = new URL(env.VITE_API_URL || 'http://localhost:8787')
  return {
    manifest_version: 3,
    name: 'Fuck Algo',
    description:
      'Your feed. Your rules. Blocks spoilers, repeat content and words you are done with, on YouTube, X, Instagram and any site.',
    version: pkg.version,
    minimum_chrome_version: '116',
    icons: { 16: 'icons/icon-16.png', 32: 'icons/icon-32.png', 48: 'icons/icon-48.png', 128: 'icons/icon-128.png' },
    action: {
      default_title: 'Fuck Algo',
      default_popup: 'src/popup/index.html',
      default_icon: { 16: 'icons/icon-16.png', 32: 'icons/icon-32.png', 48: 'icons/icon-48.png' },
    },
    options_page: 'src/options/index.html',
    background: { service_worker: 'src/background/worker.ts', type: 'module' },
    content_scripts: [
      {
        matches: ['http://*/*', 'https://*/*'],
        js: ['src/content/main.ts'],
        run_at: 'document_idle',
      },
    ],
    permissions: ['storage', 'unlimitedStorage', 'tabs', 'offscreen', 'alarms'],
    host_permissions: ['http://*/*', 'https://*/*', `${api.origin}/*`],
  }
})
