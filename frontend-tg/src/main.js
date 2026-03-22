import { createApp } from 'vue'
import App from './App.vue'

// Initialize Telegram Web App
const tg = window.Telegram?.WebApp
if (tg) {
  tg.ready()
  tg.expand()
  // Apply TG theme
  document.documentElement.style.setProperty('--tg-theme-bg-color', tg.themeParams.bg_color || '#0a0e1a')
  document.documentElement.style.setProperty('--tg-theme-text-color', tg.themeParams.text_color || '#eef0f4')
}

createApp(App).mount('#app')
