import { ref, computed, watch } from 'vue'
import { useTelegram } from './useTelegram.js'

const WALLET_SESSION_KEY = 'mule_tg_wallet_session'

const storedWalletAddress = ref('')
const walletConnecting = ref(false)
const walletError = ref('')

const hasMetaMask = computed(() => false)
const hasPhantom = computed(() => false)
const hasTonConnect = computed(() => true)

const {
  tonAddress,
  getTonAddress,
  showAlert
} = useTelegram()

function persistWalletSession(address) {
  if (!address) return
  window.localStorage?.setItem(WALLET_SESSION_KEY, JSON.stringify({
    wallet: address,
    chain: 'ton'
  }))
}

function restoreWalletSession() {
  try {
    const raw = window.localStorage?.getItem(WALLET_SESSION_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw)
    if (!parsed?.wallet) return
    storedWalletAddress.value = parsed.wallet
  } catch (_) {}
}

restoreWalletSession()

watch(tonAddress, (address) => {
  if (!address) return
  storedWalletAddress.value = address
  persistWalletSession(address)
}, { immediate: true })

const walletAddress = computed(() => tonAddress.value || storedWalletAddress.value)
const walletChain = computed(() => walletAddress.value ? 'ton' : '')
const walletConnected = computed(() => !!walletAddress.value)

async function connectTON(preferredWalletAppName = '') {
  walletConnecting.value = true
  walletError.value = ''
  try {
    const address = await getTonAddress(preferredWalletAppName)
    if (!address) {
      throw new Error('Please connect a TON wallet first')
    }
    storedWalletAddress.value = address
    persistWalletSession(address)
    return address
  } catch (e) {
    walletError.value = e.message || 'Failed to connect TON wallet'
    showAlert(walletError.value)
    throw e
  } finally {
    walletConnecting.value = false
  }
}

async function connectEVM() {
  return await connectTON()
}

async function connectSOL() {
  return await connectTON()
}

async function signAuthMessage() {
  throw new Error('TON signed login is not supported yet')
}

function setManualAddress(address) {
  if (!address) return
  storedWalletAddress.value = address
  persistWalletSession(address)
  walletError.value = ''
}

function validateAddress(address, chain) {
  if (chain !== 'ton' || !address) return false
  if (address.length >= 48 && (address.startsWith('EQ') || address.startsWith('UQ'))) return true
  if (/^0:[0-9a-fA-F]{64}$/.test(address)) return true
  return /^[0-9a-fA-F]{64}$/.test(address)
}

function detectChain(address) {
  if (!address) return ''
  if (address.startsWith('EQ') || address.startsWith('UQ') || address.startsWith('0:')) return 'ton'
  return ''
}

function disconnect() {
  storedWalletAddress.value = ''
  walletError.value = ''
  window.localStorage?.removeItem(WALLET_SESSION_KEY)
}

export function useWallet() {
  return {
    walletAddress,
    walletChain,
    walletConnected,
    walletConnecting,
    walletError,
    hasMetaMask,
    hasPhantom,
    hasTonConnect,
    connectEVM,
    connectTON,
    connectSOL,
    signAuthMessage,
    setManualAddress,
    validateAddress,
    detectChain,
    disconnect
  }
}
