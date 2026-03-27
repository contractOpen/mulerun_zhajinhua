import { ref } from 'vue'

const AUTH_SESSION_KEY = 'mule_auth_session'
const DEFAULT_API_BASE_URL = 'https://game.atdl.link'
const runtimeApiBaseUrl = (import.meta.env.VITE_API_BASE_URL || DEFAULT_API_BASE_URL).trim().replace(/\/+$/, '')
const runtimeWsUrl = (import.meta.env.VITE_WS_URL || '').trim()
const ws = ref(null)
const connected = ref(false)
const authenticated = ref(false)
const playerId = ref('')
const userChips = ref(1000)
const roomState = ref(null)
const gameEvents = ref([])
const gameEnd = ref(null)
const error = ref('')
const matchStatus = ref(null)
const nextRoundStatus = ref(null)
const serverMode = ref('te')
const bonusInfo = ref(null)
const authChallenge = ref(null)
let pendingChallengeRequest = null

function getDefaultWsUrl() {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${location.host}/ws`
}

function getWebSocketUrl() {
  if (runtimeWsUrl) return runtimeWsUrl
  if (runtimeApiBaseUrl) {
    const apiUrl = new URL(runtimeApiBaseUrl)
    const wsProtocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${wsProtocol}//${apiUrl.host}/ws`
  }
  return getDefaultWsUrl()
}

function connect() {
  try {
    const url = getWebSocketUrl()
    ws.value = new WebSocket(url)

    ws.value.onopen = () => {
      connected.value = true
      error.value = ''
    }

    ws.value.onclose = () => {
      connected.value = false
      authenticated.value = false
      roomState.value = null
      if (pendingChallengeRequest) {
        pendingChallengeRequest.reject(new Error('连接已关闭'))
        pendingChallengeRequest = null
      }
      setTimeout(connect, 3000)
    }

    ws.value.onerror = () => {
      error.value = '连接失败，正在重试...'
    }

    ws.value.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        handleMessage(msg)
      } catch (e) {}
    }
  } catch (e) {
    error.value = '连接失败'
    setTimeout(connect, 3000)
  }
}

function handleMessage(msg) {
  switch (msg.type) {
    case 'connected':
      if (msg.data.mode) serverMode.value = msg.data.mode
      break
    case 'authenticated':
      authenticated.value = true
      playerId.value = msg.data.playerId
      userChips.value = msg.data.chips || 1000
      if (msg.data.mode) serverMode.value = msg.data.mode
      if (msg.data.playerName) {
        window.localStorage?.setItem('playerName_' + msg.data.playerId, msg.data.playerName)
        window.dispatchEvent(new CustomEvent('playerNameUpdated', {
          detail: { playerId: msg.data.playerId, playerName: msg.data.playerName }
        }))
      }
      if (msg.data.sessionToken && msg.data.sessionExpiresAt) {
        window.localStorage?.setItem(AUTH_SESSION_KEY, JSON.stringify({
          playerId: msg.data.playerId,
          wallet: msg.data.playerId,
          chain: msg.data.chain || '',
          token: msg.data.sessionToken,
          expiresAt: msg.data.sessionExpiresAt
        }))
      }
      error.value = ''
      authChallenge.value = null
      break
    case 'auth_challenge':
      authChallenge.value = msg.data
      if (pendingChallengeRequest) {
        pendingChallengeRequest.resolve(msg.data)
        pendingChallengeRequest = null
      }
      break
    case 'room_created':
    case 'room_state':
      roomState.value = msg.data
      if (msg.type === 'room_created' || msg.data.state !== 2) {
        nextRoundStatus.value = null
        gameEnd.value = null
      } else {
        nextRoundStatus.value = {
          readyForNext: msg.data.readyForNext || [],
          nextRoundDeadline: msg.data.nextRoundDeadline || null
        }
      }
      if (msg.data.dbChips !== undefined) {
        userChips.value = msg.data.dbChips
      }
      break
    case 'game_event':
      gameEvents.value.push(msg.data)
      if (gameEvents.value.length > 50) {
        gameEvents.value = gameEvents.value.slice(-30)
      }
      break
    case 'game_end':
      gameEnd.value = msg.data
      roomState.value = msg.data.room
      if (msg.data.dbChips !== undefined) {
        userChips.value = msg.data.dbChips
      }
      break
    case 'match_status':
      matchStatus.value = msg.data
      break
    case 'match_found':
      matchStatus.value = { status: 'found' }
      roomState.value = msg.data
      break
    case 'recharge_success':
      if (msg.data.newChips !== undefined) {
        userChips.value = msg.data.newChips
      } else {
        userChips.value += msg.data.amount || 0
      }
      if (msg.data.usedDailyBonus) {
        bonusInfo.value = {
          claimed: 3 - (msg.data.remainingClaims ?? 0),
          remaining: msg.data.remainingClaims ?? 0,
          canClaim: (msg.data.remainingClaims ?? 0) > 0,
          amount: msg.data.amount || 500
        }
      }
      break
    case 'next_round_status':
      nextRoundStatus.value = msg.data
      break
    case 'left_room':
      roomState.value = null
      gameEnd.value = null
      nextRoundStatus.value = null
      matchStatus.value = null
      window.dispatchEvent(new CustomEvent('roomClosed', {
        detail: { message: msg.data.message || '已离开房间' }
      }))
      if (msg.data.chips !== undefined) {
        userChips.value = msg.data.chips
      }
      break
    case 'kicked':
      roomState.value = null
      gameEnd.value = null
      nextRoundStatus.value = null
      matchStatus.value = null
      error.value = msg.data.message
      window.dispatchEvent(new CustomEvent('roomClosed', {
        detail: { message: msg.data.message || '房间已关闭' }
      }))
      setTimeout(() => { error.value = '' }, 3000)
      break
    case 'error':
      error.value = msg.data.message
      if (pendingChallengeRequest) {
        pendingChallengeRequest.reject(new Error(msg.data.message || '认证失败'))
        pendingChallengeRequest = null
      }
      setTimeout(() => { error.value = '' }, 3000)
      break
    case 'session_expired':
      authenticated.value = false
      roomState.value = null
      gameEnd.value = null
      nextRoundStatus.value = null
      window.localStorage?.removeItem(AUTH_SESSION_KEY)
      error.value = msg.data.message || '登录状态已过期，请重新授权'
      window.dispatchEvent(new CustomEvent('authSessionExpired', {
        detail: { message: error.value }
      }))
      setTimeout(() => { error.value = '' }, 3000)
      break
    case 'bankrupt':
      if (msg.data.chips !== undefined) {
        userChips.value = msg.data.chips
      }
      break
    case 'recharge_confirmed':
      if (msg.data.newChips !== undefined) {
        userChips.value = msg.data.newChips
      }
      break
    case 'bonus_claimed':
      if (msg.data.newChips !== undefined) {
        userChips.value = msg.data.newChips
      }
      bonusInfo.value = {
        claimed: (bonusInfo.value?.claimed ?? 0) + 1,
        remaining: msg.data.remainingClaims ?? 0,
        canClaim: msg.data.remainingClaims > 0,
        amount: msg.data.amount || 500
      }
      break
    case 'bonus_status':
      bonusInfo.value = {
        claimed: msg.data.claimed ?? 0,
        remaining: msg.data.remaining ?? 0,
        canClaim: msg.data.canClaim ?? false,
        amount: msg.data.amount || 500,
        message: msg.data.message || ''
      }
      break
  }
}

function send(type, data) {
  if (ws.value && ws.value.readyState === WebSocket.OPEN) {
    ws.value.send(JSON.stringify({ type, data }))
  }
}

function login(walletAddress, walletChain, playerName = '', nonce = '', signature = '', sessionToken = '') {
  if (!authenticated.value) {
    send('login', { wallet: walletAddress, chain: walletChain, playerName, nonce, signature, sessionToken })
  }
}

function requestAuthChallenge(walletAddress, walletChain) {
  if (!ws.value || ws.value.readyState !== WebSocket.OPEN) {
    return Promise.reject(new Error('连接尚未建立'))
  }
  if (pendingChallengeRequest) {
    pendingChallengeRequest.reject(new Error('已有认证请求进行中'))
    pendingChallengeRequest = null
  }
  return new Promise((resolve, reject) => {
    pendingChallengeRequest = { resolve, reject }
    send('auth_challenge_request', { wallet: walletAddress, chain: walletChain })
  })
}

function createRoom(playerName, botCount, baseBet, wallet, chain) {
  send('create_room', { playerName, botCount, baseBet, wallet, chain })
}

function updateBaseBet(baseBet) {
  send('update_base_bet', { baseBet })
}

function joinRoom(roomId, playerName, wallet, chain) {
  send('join_room', { roomId, playerName, wallet, chain })
}

function startGame() {
  send('start_game', {})
}

function doAction(type, amount = 0, targetId = '') {
  send('action', { type, amount, targetId })
}

function newRound() {
  gameEvents.value = []
  send('new_round', {})
}

function startMatch(playerName, wallet, chain, baseBet = 10) {
  send('match', { playerName, wallet, chain, baseBet })
}

function cancelMatch() {
  send('cancel_match', {})
  matchStatus.value = null
}

function recharge(amount, wallet) {
  send('recharge', { amount, wallet })
}

function leaveRoom() {
  send('leave_room', {})
}

function claimBonus() {
  send('claim_bonus', {})
}

function verifyRecharge(txHash, chain, amount, points) {
  send('verify_recharge', { txHash, chain, amount, points })
}

function checkBonusStatus() {
  send('check_bonus', {})
}

function joinByCode(code, playerName, wallet, chain) {
  send('join_by_code', { roomCode: code, playerName, wallet, chain })
}

export function useGame() {
  return {
    connected, authenticated, playerId, userChips, roomState, gameEvents, gameEnd, error, matchStatus, nextRoundStatus, serverMode, bonusInfo, authChallenge,
    connect, login, requestAuthChallenge, createRoom, updateBaseBet, joinRoom, startGame, doAction, newRound,
    startMatch, cancelMatch, recharge, verifyRecharge, joinByCode, leaveRoom, claimBonus, checkBonusStatus
  }
}
