import { fileURLToPath, URL } from 'node:url'
import { resolve } from 'node:path'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The console is served from /sidebar/ inside an iframe of the host page.
// Everything it loads is resolved relative to that folder (`base: './'`),
// which keeps a single build working under any mount point.
export default defineConfig({
  base: './',
  build: {
    target: 'es2020',
    outDir: resolve(__dirname, 'dist'),
    minify: 'terser',
  },
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 23367,
    proxy: {
      '/api': 'http://127.0.0.1:23366',
      '/dist': 'http://127.0.0.1:23366',
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
