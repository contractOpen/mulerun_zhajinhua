import { ref, computed } from 'vue'

const walletAddress = ref('')
const walletChain = ref('') // 'evm', 'ton', 'sol'
const walletConnected = ref(false)
const walletConnecting = ref(false)
const walletError = ref('')

// Detect available wallets
const hasMetaMask = computed(() => typeof window !== 'undefined' && !!window.ethereum)
const hasPhantom = computed(() => typeof window !== 'undefined' && !!window.solana?.isPhantom)
const hasTonConnect = ref(false) // Set after TonConnect init

let tonConnector = null

// EVM (MetaMask) connection
async function connectEVM() {
  walletConnecting.value = true
  walletError.value = ''
  try {
    if (!window.ethereum) {
      throw new Error('MetaMask not detected. Please install MetaMask.')
    }
    const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' })
    if (accounts && accounts.length > 0) {
      walletAddress.value = accounts[0]
      walletChain.value = 'evm'
      walletConnected.value = true
    }
  } catch (e) {
    walletError.value = e.message || 'Failed to connect MetaMask'
  } finally {
    walletConnecting.value = false
  }
}

// TON (TonConnect) connection
async function connectTON() {
  walletConnecting.value = true
  walletError.value = ''
  try {
    // TonConnect requires @tonconnect/sdk - check if available
    if (window.TonConnectSDK) {
      tonConnector = new window.TonConnectSDK.TonConnect({
        manifestUrl: window.location.origin + '/manifest.json'
      })
      const wallets = await tonConnector.getWallets()
      if (wallets.length > 0) {
        await tonConnector.connect({ jsBridgeKey: wallets[0].jsBridgeKey })
        const account = tonConnector.account
        if (account) {
          walletAddress.value = account.address
          walletChain.value = 'ton'
          walletConnected.value = true
        }
      }
    } else {
      // Fallback: prompt for manual address input
      walletError.value = 'TonConnect SDK not loaded. Enter address manually.'
    }
  } catch (e) {
    walletError.value = e.message || 'Failed to connect TON wallet'
  } finally {
    walletConnecting.value = false
  }
}

// SOL (Phantom) connection
async function connectSOL() {
  walletConnecting.value = true
  walletError.value = ''
  try {
    const provider = window.solana
    if (!provider?.isPhantom) {
      throw new Error('Phantom wallet not detected. Please install Phantom.')
    }
    const resp = await provider.connect()
    walletAddress.value = resp.publicKey.toString()
    walletChain.value = 'sol'
    walletConnected.value = true
  } catch (e) {
    walletError.value = e.message || 'Failed to connect Phantom'
  } finally {
    walletConnecting.value = false
  }
}

// Manual address input (for PE mode when SDK not available)
function setManualAddress(address, chain) {
  walletAddress.value = address
  walletChain.value = chain
  walletConnected.value = true
  walletError.value = ''
}

// Validate address format locally
function validateAddress(address, chain) {
  if (!address) return false
  switch (chain) {
    case 'evm':
      return /^0x[0-9a-fA-F]{40}$/.test(address)
    case 'ton':
      if (address.length >= 48 && (address.startsWith('EQ') || address.startsWith('UQ'))) return true
      if (/^0:[0-9a-fA-F]{64}$/.test(address)) return true
      return /^[0-9a-fA-F]{64}$/.test(address)
    case 'sol':
      return /^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(address)
    default:
      return false
  }
}

// Detect chain from address
function detectChain(address) {
  if (/^0x[0-9a-fA-F]{40}$/.test(address)) return 'evm'
  if (address.startsWith('EQ') || address.startsWith('UQ') || address.startsWith('0:')) return 'ton'
  if (/^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(address)) return 'sol'
  return ''
}

function disconnect() {
  walletAddress.value = ''
  walletChain.value = ''
  walletConnected.value = false
  walletError.value = ''
  if (tonConnector) {
    try { tonConnector.disconnect() } catch(e) {}
    tonConnector = null
  }
}

export function useWallet() {
  return {
    walletAddress, walletChain, walletConnected, walletConnecting, walletError,
    hasMetaMask, hasPhantom, hasTonConnect,
    connectEVM, connectTON, connectSOL,
    setManualAddress, validateAddress, detectChain, disconnect
  }
}
