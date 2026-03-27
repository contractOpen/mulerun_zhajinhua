import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(() => {
  const isVercel = process.env.VERCEL === '1' || process.env.NOW_REGION

  return {
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
      outDir: isVercel ? 'dist' : '../backend/static'
    },
    define: {
      __APP_MODE__: JSON.stringify(process.env.VITE_APP_MODE || 'te'),
    },
  }
})
