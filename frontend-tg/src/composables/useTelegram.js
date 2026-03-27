import { ref, computed } from 'vue'

const tg = typeof window !== 'undefined' ? window.Telegram?.WebApp : null
const TON_CONNECT_MANIFEST_PATH = '/tonconnect-manifest.json'
const isReady = ref(false)
const tgUser = ref(null)
const initData = ref('')
const tonAddress = ref('')
let tonConnectUI = null
let tonConnectSubscribed = false

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

function getTonConnectUI() {
  if (typeof window === 'undefined') return null
  if (tonConnectUI) return tonConnectUI

  const TonConnectUI = window.TON_CONNECT_UI?.TonConnectUI
  if (!TonConnectUI) return null

  tonConnectUI = new TonConnectUI({
    manifestUrl: new URL(TON_CONNECT_MANIFEST_PATH, window.location.origin).toString()
  })

  if (!tonConnectSubscribed && typeof tonConnectUI.onStatusChange === 'function') {
    tonConnectSubscribed = true
    tonConnectUI.onStatusChange((wallet) => {
      tonAddress.value = wallet?.account?.address || ''
    })
  }

  return tonConnectUI
}

function waitForTonWalletConnection(tonConnect, appName = '') {
  return new Promise((resolve, reject) => {
    let finished = false
    let unsubscribeStatus = null
    let unsubscribeModal = null
    let timeoutId = null

    const cleanup = () => {
      if (unsubscribeStatus) unsubscribeStatus()
      if (unsubscribeModal) unsubscribeModal()
      if (timeoutId) clearTimeout(timeoutId)
    }

    const finish = (callback) => {
      if (finished) return
      finished = true
      cleanup()
      callback()
    }

    unsubscribeStatus = tonConnect.onStatusChange((wallet) => {
      if (wallet?.account?.address) {
        tonAddress.value = wallet.account.address
        finish(() => resolve(tonAddress.value))
      }
    })

    const modalListener = appName && typeof tonConnect.onSingleWalletModalStateChange === 'function'
      ? tonConnect.onSingleWalletModalStateChange.bind(tonConnect)
      : typeof tonConnect.onModalStateChange === 'function'
        ? tonConnect.onModalStateChange.bind(tonConnect)
        : null

    if (modalListener) {
      unsubscribeModal = modalListener((state) => {
        if (state?.status === 'closed' && !tonAddress.value) {
          finish(() => reject(new Error('Wallet connection was cancelled')))
        }
      })
    }

    timeoutId = setTimeout(() => {
      finish(() => reject(new Error('Wallet connection timed out')))
    }, 120000)
  })
}

/**
 * Attempt to get a TON wallet address.
 * Tries TonConnect if available, otherwise falls back to initDataUnsafe wallet info.
 * Returns the address string or empty string.
 */
async function getTonAddress(preferredWalletAppName = '') {
  const tonConnect = getTonConnectUI()
  if (tonConnect?.wallet?.account?.address) {
    tonAddress.value = tonConnect.wallet.account.address
    return tonAddress.value
  }

  // Try TonConnect (if the TonConnect UI SDK is loaded on the page)
  if (tonConnect && typeof tonConnect.connectWallet === 'function') {
    try {
      const connectionPromise = waitForTonWalletConnection(tonConnect, preferredWalletAppName)
      if (preferredWalletAppName && typeof tonConnect.openSingleWalletModal === 'function') {
        await tonConnect.openSingleWalletModal(preferredWalletAppName)
      } else {
        await tonConnect.connectWallet()
      }
      const wallet = await connectionPromise
      if (wallet) {
        return wallet
      }
    } catch (e) {
      if (e?.message) {
        throw e
      }
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
