import { ref, computed } from 'vue'

const tg = typeof window !== 'undefined' ? window.Telegram?.WebApp : null
const isReady = ref(false)
const tgUser = ref(null)
const initData = ref('')
const tonAddress = ref('')

function init() {
  if (!tg) return

  tg.ready()
  tg.expand()
  isReady.value = true

  if (tg.initDataUnsafe?.user) {
    tgUser.value = tg.initDataUnsafe.user
  }
  initData.value = tg.initData || ''

  // Set color scheme
  if (tg.colorScheme === 'dark') {
    document.documentElement.classList.add('tg-dark')
  }
}

const userId = computed(() => tgUser.value?.id?.toString() || '')
const userName = computed(() => {
  if (!tgUser.value) return ''
  return tgUser.value.first_name + (tgUser.value.last_name ? ' ' + tgUser.value.last_name : '')
})
const userLanguage = computed(() => tgUser.value?.language_code || 'en')

function hapticFeedback(type = 'impact') {
  if (!tg?.HapticFeedback) return
  switch (type) {
    case 'impact': tg.HapticFeedback.impactOccurred('medium'); break
    case 'success': tg.HapticFeedback.notificationOccurred('success'); break
    case 'error': tg.HapticFeedback.notificationOccurred('error'); break
    case 'selection': tg.HapticFeedback.selectionChanged(); break
  }
}

function showAlert(message) {
  if (tg) tg.showAlert(message)
  else alert(message)
}

function showConfirm(message) {
  return new Promise(resolve => {
    if (tg) tg.showConfirm(message, resolve)
    else resolve(confirm(message))
  })
}

function setMainButton(text, onClick) {
  if (!tg?.MainButton) return
  tg.MainButton.text = text
  tg.MainButton.onClick(onClick)
  tg.MainButton.show()
}

function hideMainButton() {
  if (tg?.MainButton) tg.MainButton.hide()
}

function close() {
  if (tg) tg.close()
}

/**
 * Attempt to get a TON wallet address.
 * Tries TonConnect if available, otherwise falls back to initDataUnsafe wallet info.
 * Returns the address string or empty string.
 */
async function getTonAddress() {
  // Try TonConnect (if the TonConnect SDK is loaded on the page)
  if (typeof window !== 'undefined' && window.tonConnectUI) {
    try {
      const wallet = await window.tonConnectUI.connectWallet()
      if (wallet?.account?.address) {
        tonAddress.value = wallet.account.address
        return tonAddress.value
      }
    } catch (e) {
      // TonConnect not available or user rejected — fall through
    }
  }

  // Fallback: check if initDataUnsafe carries wallet info (future TG API)
  if (tg?.initDataUnsafe?.wallet?.address) {
    tonAddress.value = tg.initDataUnsafe.wallet.address
    return tonAddress.value
  }

  // No wallet found — leave tonAddress empty
  return ''
}

export function useTelegram() {
  return {
    tg, isReady, tgUser, initData, userId, userName, userLanguage, tonAddress,
    init, hapticFeedback, showAlert, showConfirm,
    setMainButton, hideMainButton, close, getTonAddress
  }
}
