import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import { resolve } from 'path'

export default defineConfig({
  plugins: [
    vue(),
    vuetify({ autoImport: true }),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.TASK_MANAGER_DEV_BACKEND_URL || 'http://localhost:8080',
        changeOrigin: false,
      },
      '/health': {
        target: process.env.TASK_MANAGER_DEV_BACKEND_URL || 'http://localhost:8080',
        changeOrigin: false,
      },
    },
  },
})
