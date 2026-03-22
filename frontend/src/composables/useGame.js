import { ref } from 'vue'

const ws = ref(null)
const connected = ref(false)
const playerId = ref('')
const userChips = ref(1000) // synced from server
const roomState = ref(null)
const gameEvents = ref([])
const gameEnd = ref(null)
const error = ref('')
const matchStatus = ref(null) // { status: 'matching'|'found'|'cancelled', players: N }
const nextRoundStatus = ref(null) // { readyForNext: [], nextRoundDeadline: ms }
const serverMode = ref('te') // 'te' or 'pe'
const bonusInfo = ref(null) // { remaining: number, nextReset: string, amount: number }

function connect() {
  try {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/ws`
    ws.value = new WebSocket(url)

    ws.value.onopen = () => {
      connected.value = true
      error.value = ''
    }

    ws.value.onclose = () => {
      connected.value = false
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
      playerId.value = msg.data.playerId
      userChips.value = msg.data.chips || 1000
      if (msg.data.mode) serverMode.value = msg.data.mode
      break
    case 'room_created':
    case 'room_state':
      roomState.value = msg.data
      // Sync user chips from server dbChips field
      if (msg.data.dbChips !== undefined) {
        userChips.value = msg.data.dbChips
      }
      gameEnd.value = null
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
      // Sync chips from game end data
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
      // Refresh chips from server
      if (msg.data.newChips !== undefined) {
        userChips.value = msg.data.newChips
      } else {
        userChips.value += msg.data.amount || 0
      }
      break
    case 'next_round_status':
      nextRoundStatus.value = msg.data
      break
    case 'left_room':
      roomState.value = null
      gameEnd.value = null
      nextRoundStatus.value = null
      // Sync chips when leaving room
      if (msg.data.chips !== undefined) {
        userChips.value = msg.data.chips
      }
      break
    case 'kicked':
      roomState.value = null
      gameEnd.value = null
      nextRoundStatus.value = null
      error.value = msg.data.message
      setTimeout(() => { error.value = '' }, 3000)
      break
    case 'error':
      error.value = msg.data.message
      setTimeout(() => { error.value = '' }, 3000)
      break
    case 'bankrupt':
      // Don't clear roomState, stay in game to see result
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
      // Server confirms a daily bonus claim
      if (msg.data.newChips !== undefined) {
        userChips.value = msg.data.newChips
      }
      bonusInfo.value = {
        remaining: msg.data.remaining ?? 0,
        nextReset: msg.data.nextReset || '',
        amount: msg.data.amount || 0
      }
      break
  }
}

function send(type, data) {
  if (ws.value && ws.value.readyState === WebSocket.OPEN) {
    ws.value.send(JSON.stringify({ type, data }))
  }
}

function createRoom(playerName, botCount, baseBet, wallet, chain) {
  send('create_room', { playerName, botCount, baseBet, wallet, chain })
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

function startMatch(playerName, wallet, chain) {
  send('match', { playerName, wallet, chain })
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

function joinByCode(code, playerName, wallet, chain) {
  send('join_by_code', { code, playerName, wallet, chain })
}

export function useGame() {
  return {
    connected, playerId, userChips, roomState, gameEvents, gameEnd, error, matchStatus, nextRoundStatus, serverMode, bonusInfo,
    connect, createRoom, joinRoom, startGame, doAction, newRound,
    startMatch, cancelMatch, recharge, verifyRecharge, joinByCode, leaveRoom, claimBonus
  }
}
