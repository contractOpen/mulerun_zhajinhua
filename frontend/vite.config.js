import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/ws': {
        target: 'http://localhost:8080',
        ws: true
      }
    }
  },
  build: {
    outDir: '../backend/static'
  },
  define: {
    __APP_MODE__: JSON.stringify(process.env.VITE_APP_MODE || 'te'),
  },
})
