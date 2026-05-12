import { fileURLToPath, URL } from 'node:url'
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig } from 'vite'
import { cloudflare } from "@cloudflare/vite-plugin";
import type { Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'


/** Copies src/sw.js into the build output root as an unprocessed asset. */
function copyServiceWorker(): Plugin {
  return {
    name: 'copy-service-worker',
    writeBundle(_options, _bundle) {
      const swSrc = readFileSync(new URL('./src/sw.js', import.meta.url), 'utf-8')
      const outDir = resolve(__dirname, 'dist')
      mkdirSync(outDir, { recursive: true })
      writeFileSync(resolve(outDir, 'sw.js'), swSrc, 'utf-8')
    },
  }
}

export default defineConfig({
  plugins: [
    vue(),
    copyServiceWorker(),
    cloudflare()
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        manualChunks: {
          'vue-core': ['vue', 'vue-router', 'pinia', 'pinia-plugin-persistedstate'],
          'naive-ui': ['naive-ui'],
          'echarts': ['echarts', 'vue-echarts'],
        },
      },
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})