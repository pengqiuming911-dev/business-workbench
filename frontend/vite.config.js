import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api/agent/chat': {
        target: 'http://localhost:3001',
        changeOrigin: true,
      },
      '/api': 'http://localhost:3001',
      '/public': 'http://localhost:3001',
    },
  },
})
