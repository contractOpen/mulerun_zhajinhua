import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@shared': path.resolve(__dirname, '../frontend/src/composables'),
    },
  },
  build: {
    outDir: '../backend/static-tg',
  },
  define: {
    __APP_MODE__: JSON.stringify('tg'),
  },
})
