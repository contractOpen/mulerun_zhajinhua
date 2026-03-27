/*
 * @Author: small ant shuo582@163.com
 * @Time: 2026-03-22 22:29:12
 * @LastAuthor: small ant shuo582@163.com
 * @lastTime: 2026-03-24 22:34:14
 * @FileName: vite.config
 * @Desc: 
 * 
 * Copyright (c) 2026 by small ant, All Rights Reserved. 
 */
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig(({ mode }) => {
  // 检测是否在 Vercel 环境
  const isVercel = process.env.VERCEL === '1' || process.env.NOW_REGION
    
  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@shared': path.resolve(__dirname, '../frontend/src/composables'),
      },
    },
    build: {
      // 如果是 Vercel 环境，输出到默认 dist 目录
      outDir: isVercel ? 'dist' : '../backend/static-tg',
      // 确保生成的文件可被正确访问
      assetsDir: 'assets',
      rollupOptions: {
        output: {
          manualChunks: undefined,
        },
      },
    },
    define: {
      __APP_MODE__: JSON.stringify(mode === 'tg' ? 'tg' : 'te'),
    },
    // TG 前端需要作为独立静态站部署，不能把资源地址绑死到某个 Vercel 临时域名
    base: '/',
  }
})
