import { defineManifest } from '@crxjs/vite-plugin'
import { loadEnv } from 'vite'

import pkg from './package.json'

const DEV_EXTENSION_PUBLIC_KEY =
  'MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAttvwkPgXvT5d/M8nK4Q/2PFL4bzr3MZ5Rac3OvED7Pj9lcuYqrRr0Reum12eMfG1ZnDC2Dgfh+STWTS6sZ9SsVwVdZae46wKvxAI0JvVfJvdbm6UCOjxQrXWJ8x2mhhWivdFgt26wH6MLzsMQMK85IJAIUfhQ1FtcBYBuC56ysQfMWgHu3dog9n7nf6obGUmaGWdIfiaCKa0vi6GWrJXh/XkIiUeC1qPLuUEsIshJxpWyGJgp8AWVazlL16Z7T/EzwpPsVIID1ZRsp+JZ+aAyLAnu+h+gB2IXR7b7U+cf4twhsJ0gGSz0I8us+BaP4GozbT0+vYpMeWyB79SPm7WowIDAQAB'

function clerkFrontendApi(publishableKey: string | undefined): string | null {
  if (!publishableKey) return null
  const encoded = publishableKey.replace(/^pk_(test|live)_/, '')
  const host = Buffer.from(encoded, 'base64').toString('utf-8').replace(/\$$/, '')
  return `https://${host}`
}

export default defineManifest(({ mode }) => {
  const env = loadEnv(mode, process.cwd())
  const api = new URL(env.VITE_API_URL || 'http://localhost:8787')
  const syncHost = new URL(env.VITE_CLERK_SYNC_HOST || 'http://localhost:5173')
  const clerkApi = clerkFrontendApi(env.VITE_CLERK_PUBLISHABLE_KEY)
  const development = mode !== 'production' || env.VITE_ENVIRONMENT === 'development'
  return {
    ...(development ? { key: DEV_EXTENSION_PUBLIC_KEY } : {}),
    manifest_version: 3,
    name: 'Fuck Algo',
    description:
      'Your feed. Your rules. Blocks spoilers, repeat content and words you are done with, on YouTube, X, Instagram and any site.',
    version: pkg.version,
    minimum_chrome_version: '116',
    icons: {
      16: 'icons/icon-16.png',
      32: 'icons/icon-32.png',
      48: 'icons/icon-48.png',
      128: 'icons/icon-128.png',
    },
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
    permissions: ['storage', 'unlimitedStorage', 'tabs', 'offscreen', 'alarms', 'cookies'],
    host_permissions: [
      ...new Set([
        'http://*/*',
        'https://*/*',
        `${api.origin}/*`,
        `${syncHost.origin}/*`,
        ...(clerkApi ? [`${clerkApi}/*`] : []),
      ]),
    ],
  }
})
