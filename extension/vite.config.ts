import path from 'node:path'

import { crx } from '@crxjs/vite-plugin'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

import manifest from './manifest.config'

export default defineConfig(({ command }) => ({
  plugins: [react(), crx({ manifest })],
  resolve: { alias: { '@': path.resolve(__dirname, './src') } },
  server: { port: 5174, strictPort: true, hmr: { port: 5174 } },
  build: {
    outDir: command === 'serve' ? 'dist-dev' : 'dist',
    rollupOptions: {
      input: { offscreen: path.resolve(__dirname, 'src/offscreen/index.html') },
    },
  },
}))
