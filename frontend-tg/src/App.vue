<template>
  <div class="app tg-app">
    <div v-if="error" class="toast error">{{ error }}</div>

    <!-- LOBBY (TG skips login) -->
    <div v-if="page === 'lobby'" class="page lobby-page">
      <!-- TG lobby header -->
      <div class="lobby-header">
        <div class="user-info">
          <div class="user-avatar" :style="{background:avatarColor(0)}">{{ playerName[0] || 'T' }}</div>
          <div class="user-meta">
            <div class="user-name">{{ playerName }}</div>
            <div class="user-chips"><span>&#x1FA99;</span>{{ userChips }}</div>
          </div>
        </div>
      </div>
      <div class="lobby-body">
        <div class="mode-cards">
          <button class="mode-card" @click="handleQuickStart">&#x26A1; Quick Start</button>
          <button class="mode-card" @click="handleCreatePrivateRoom">&#x1F3E0; Create Room</button>
          <div class="join-row">
            <input v-model="joinRoomId" placeholder="Room code..." class="room-input" />
            <button class="btn-join" @click="handleJoinByCode">Join</button>
          </div>
          <button class="mode-card bonus-card" @click="handleClaimBonus">&#x1F381; Daily Bonus
            <span v-if="bonusInfo" class="bonus-remaining">({{ bonusInfo.remaining }} left)</span>
          </button>
        </div>
      </div>
    </div>

    <!-- GAME PAGE -->
    <div v-else-if="page === 'game'" class="page game-page">
      <!-- Top bar: pot, round, current bet, ante -->
      <div class="game-topbar">
        <button class="topbar-back" @click="handleLeaveRoom">&larr;</button>
        <div class="topbar-pot">&#x1FA99; <strong>{{ roomState?.pot || 0 }}</strong></div>
        <div class="topbar-info">
          <span>R{{ roomState?.round || 0 }}</span>
          <span v-if="roomState?.currentBet">Bet:{{ roomState.currentBet }}</span>
          <span v-if="roomState?.anteTotal">Ante:{{ roomState.anteTotal }}</span>
        </div>
        <div class="topbar-room" v-if="roomState?.roomCode" @click="copyRoomCode">{{ roomState.roomCode }} &#x1F4CB;</div>
      </div>

      <!-- Table with player seats -->
      <div class="table-container">
        <div class="table-felt">
          <!-- Pot center -->
          <div class="table-pot-center" v-if="roomState?.pot > 0">
            <div class="pot-stack">
              <div class="pot-chip" v-for="i in Math.min(Math.ceil((roomState?.pot||0)/50),6)" :key="i" :style="{transform:'translateY('+ (-i*2) +'px)'}"></div>
            </div>
            <span class="pot-amount">{{ roomState.pot }}</span>
          </div>

          <!-- Other players positioned around table -->
          <div v-for="(p,idx) in otherPlayers" :key="p.id"
            :class="['seat', {
              'is-turn': isTurn(p),
              'is-folded': p.state===3,
              'is-out': p.state===5,
              'is-allin': p.state===4
            }]"
            :style="getSeatPosition(idx, otherPlayers.length)">
            <!-- Avatar with countdown ring -->
            <div class="seat-avatar-wrap">
              <svg v-if="isTurn(p) && roomState?.state===1" class="seat-cd-ring" viewBox="0 0 44 44">
                <circle class="cd-bg" cx="22" cy="22" r="19" />
                <circle :class="['cd-fg', {'urgent': countdown<=5}]" cx="22" cy="22" r="19" :style="{strokeDashoffset: countdownOffset}" />
              </svg>
              <div class="seat-avatar" :style="{background:avatarColor(p.seat)}">{{ p.name?.[0] || '?' }}</div>
              <span v-if="isTurn(p) && roomState?.state===1" :class="['seat-cd-num', {'urgent': countdown<=5}]">{{ countdown }}</span>
            </div>
            <div class="seat-name">{{ p.name }}</div>
            <div class="seat-chips">&#x1FA99;{{ p.chips }}</div>
            <div v-if="p.betTotal>0" class="seat-bet">{{ p.betTotal }}</div>
            <!-- Cards -->
            <div class="seat-cards">
              <template v-if="p.hand && roomState?.state===2">
                <div v-for="(c,ci) in p.hand" :key="ci" :class="['mini-card', suitClass(c.suit)]">{{ valueName(c.value) }}{{ suitSymbol(c.suit) }}</div>
              </template>
              <template v-else-if="roomState?.state===1 && (p.state===2||p.state===4)">
                <div class="mini-card back" v-for="ci in 3" :key="ci"></div>
              </template>
            </div>
            <!-- Status badges -->
            <div v-if="p.state===3" class="seat-status folded">Fold</div>
            <div v-else-if="p.state===4" class="seat-status allin">ALL IN</div>
            <div v-else-if="p.state===5" class="seat-status out">Out</div>
          </div>
        </div>
      </div>

      <!-- Event log -->
      <div class="game-log" ref="logRef">
        <span v-for="(e,i) in recentEvents" :key="i" class="log-item">{{ formatEvent(e) }}</span>
      </div>

      <!-- My panel -->
      <div class="my-panel">
        <!-- My info row -->
        <div class="my-info-row">
          <div class="my-avatar-wrap">
            <svg v-if="isMyTurn && roomState?.state===1" class="my-cd-ring" viewBox="0 0 40 40">
              <circle class="cd-bg" cx="20" cy="20" r="17" />
              <circle :class="['cd-fg', {'urgent': countdown<=5}]" cx="20" cy="20" r="17" :style="{strokeDashoffset: countdownOffsetMy}" />
            </svg>
            <div class="my-avatar" :style="{background:avatarColor(myPlayer?.seat||0)}">{{ playerName[0] }}</div>
            <span v-if="isMyTurn && roomState?.state===1" :class="['my-cd-num', {'urgent': countdown<=5}]">{{ countdown }}s</span>
          </div>
          <div class="my-detail">
            <span class="my-name">{{ myPlayer?.name || playerName }}</span>
            <span class="my-chips-val">&#x1FA99;{{ myPlayer?.chips || 0 }}</span>
          </div>
          <div v-if="myPlayer?.betTotal>0" class="my-bet-badge">Bet: {{ myPlayer.betTotal }}</div>
        </div>

        <!-- My hand -->
        <div class="my-hand">
          <template v-if="myPlayer?.hand && myPlayer?.looked">
            <div v-for="(c,i) in myPlayer.hand" :key="i" :class="['game-card', suitClass(c.suit)]" :style="{animationDelay: i*0.08+'s'}">
              <div class="gc-corner top"><div class="gc-val">{{ valueName(c.value) }}</div><div class="gc-suit">{{ suitSymbol(c.suit) }}</div></div>
              <div class="gc-center">{{ suitSymbol(c.suit) }}</div>
              <div class="gc-corner bottom"><div class="gc-val">{{ valueName(c.value) }}</div><div class="gc-suit">{{ suitSymbol(c.suit) }}</div></div>
            </div>
          </template>
          <template v-else-if="roomState?.state===1 && myPlayer?.state!==3 && myPlayer?.state!==5">
            <div class="game-card card-back" v-for="i in 3" :key="i" @click="lookCards">
              <div class="card-back-pattern"></div>
              <span class="peek-label">&#x1F441; Look</span>
            </div>
          </template>
        </div>
        <div v-if="myPlayer?.hand && myPlayer?.looked && roomState?.state===1" class="hand-rank">{{ handTypeName }}</div>

        <!-- Action bar -->
        <div class="action-bar" v-if="isMyTurn && roomState?.state===1 && myPlayer?.chips > 0">
          <button class="act-btn fold" @click="doFold"><span class="act-icon">&#x2715;</span><span class="act-label">Fold</span></button>
          <button v-if="!roomState.hasAllIn" class="act-btn call" @click="doCall"><span class="act-icon">&#x2713;</span><span class="act-label">Call</span><small>{{ callAmount }}</small></button>
          <button v-if="!roomState.hasAllIn" class="act-btn raise" @click="doRaise"><span class="act-icon">&#x2191;</span><span class="act-label">Raise</span><small>{{ raiseAmount }}</small></button>
          <button v-if="canCompare && !roomState.hasAllIn" class="act-btn compare" @click="showCompareModal=true"><span class="act-icon">&#x2694;</span><span class="act-label">Compare</span></button>
          <button class="act-btn allin" @click="doAllIn"><span class="act-icon">&#x1F525;</span><span class="act-label">All In</span></button>
        </div>

        <!-- Waiting / Start -->
        <div v-if="roomState?.state===0" class="waiting-bar">
          <div class="waiting-info">{{ roomState?.players?.length || 0 }} player(s) in room</div>
          <button class="btn-start" @click="startGame">Start Game</button>
        </div>
      </div>

      <!-- Result overlay -->
      <div v-if="gameEnd" class="result-overlay">
        <div :class="['result-popup', isWinner ? 'win' : 'lose']">
          <div v-if="isWinner" class="result-crown">&#x1F451;</div>
          <h2>{{ isWinner ? 'You Win!' : (gameEnd.winnerName || 'Unknown') + ' Wins' }}</h2>
          <div class="result-info">
            <div class="result-row"><span>Hand</span><strong>{{ gameEnd.handType || '---' }}</strong></div>
            <div class="result-row"><span>Pot</span><strong class="gold">+{{ gameEnd.pot }}</strong></div>
            <div v-if="gameEnd.anteTotal" class="result-row"><span>Ante</span><strong class="dim">-{{ gameEnd.anteTotal }}</strong></div>
            <div v-if="gameEnd.rake" class="result-row"><span>Rake</span><strong class="dim">-{{ gameEnd.rake }}</strong></div>
          </div>
          <!-- All players' hands -->
          <div class="result-hands">
            <div v-for="p in roomState?.players" :key="p.id" class="result-player">
              <span :class="['rp-name', {winner: p.id===gameEnd.winnerId}]">{{ p.name }}</span>
              <div class="rp-cards" v-if="p.hand">
                <span v-for="(c,ci) in p.hand" :key="ci" :class="['rp-card', suitClass(c.suit)]">{{ valueName(c.value) }}{{ suitSymbol(c.suit) }}</span>
              </div>
            </div>
          </div>
          <!-- Next round confirmation -->
          <div class="next-round-status">
            <div class="confirm-progress">
              <span>{{ confirmedCount }}/{{ humanPlayerCount }} confirmed</span>
              <span class="confirm-timer" v-if="nextRoundCountdown < 30">{{ nextRoundCountdown }}s</span>
            </div>
            <div v-if="isConfirmedNextRound" class="confirmed-badge">&#x2713; Confirmed</div>
          </div>
          <div class="result-btn-row">
            <button class="btn-exit" @click="handleLeaveRoom">Exit</button>
            <button v-if="myPlayer && myPlayer.chips > 0 && !isConfirmedNextRound" class="btn-again" @click="handleNewRound">Play Again</button>
          </div>
        </div>
      </div>

      <!-- Compare modal -->
      <div v-if="showCompareModal" class="modal-overlay" @click.self="showCompareModal=false">
        <div class="modal compare-modal">
          <h3>Select Target</h3>
          <div class="compare-targets">
            <button v-for="p in comparablePlayers" :key="p.id" class="compare-item" @click="doCompare(p.id)">
              <div class="ci-avatar" :style="{background:avatarColor(p.seat)}">{{ p.name?.[0] }}</div>
              <span class="ci-name">{{ p.name }}</span>
              <span class="ci-chips">&#x1FA99;{{ p.chips }}</span>
            </button>
          </div>
          <button class="btn-modal-close" @click="showCompareModal=false">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useGame } from '@shared/useGame.js'
import { useI18n } from '@shared/useI18n.js'
import { useTelegram } from './composables/useTelegram.js'

const { t, setLocale } = useI18n()
const {
  connected, playerId, userChips, roomState, gameEvents, gameEnd, error, nextRoundStatus, bonusInfo,
  connect, createRoom, joinRoom, startGame: wsStartGame, doAction: wsDoAction,
  newRound: wsNewRound, joinByCode, leaveRoom: wsLeaveRoom, claimBonus: wsClaimBonus,
  verifyRecharge
} = useGame()
const { init: initTg, userName, userId: tgUserId, userLanguage, hapticFeedback, tonAddress } = useTelegram()

const page = ref('lobby')
const playerName = ref('')
const joinRoomId = ref('')
const showCompareModal = ref(false)
const logRef = ref(null)

// TON-compatible player identity
const tonIdentity = computed(() => {
  if (tonAddress.value) return tonAddress.value
  return tgUserId.value ? 'tg_' + tgUserId.value : ''
})

// Countdown timer
const countdown = ref(20)
let countdownInterval = null

// Next round countdown
const nextRoundCountdown = ref(30)
let nextRoundInterval = null

function startCountdown() {
  stopCountdown()
  if (!roomState.value?.turnDeadline) { countdown.value = 20; return }
  function tick() {
    if (!roomState.value?.turnDeadline) return
    const remaining = Math.max(0, Math.ceil((roomState.value.turnDeadline - Date.now()) / 1000))
    countdown.value = Math.min(20, remaining)
    if (remaining <= 0) stopCountdown()
  }
  tick()
  countdownInterval = setInterval(tick, 200)
}

function stopCountdown() {
  if (countdownInterval) { clearInterval(countdownInterval); countdownInterval = null }
  countdown.value = 20
}

function startNextRoundCountdown(deadline) {
  stopNextRoundCountdown()
  if (!deadline) return
  function tick() {
    const remaining = Math.max(0, Math.ceil((deadline - Date.now()) / 1000))
    nextRoundCountdown.value = Math.min(30, remaining)
    if (remaining <= 0) stopNextRoundCountdown()
  }
  tick()
  nextRoundInterval = setInterval(tick, 500)
}

function stopNextRoundCountdown() {
  if (nextRoundInterval) { clearInterval(nextRoundInterval); nextRoundInterval = null }
  nextRoundCountdown.value = 30
}

onMounted(() => {
  initTg()
  connect()
  playerName.value = userName.value || 'TG Player ' + Math.floor(Math.random() * 1000)
  const lang = userLanguage.value
  if (['zh','en','ru','es'].includes(lang)) setLocale(lang)

  // Check URL for room code
  const urlParams = new URLSearchParams(window.location.search)
  const roomCode = urlParams.get('room') || urlParams.get('startapp')
  if (roomCode) {
    joinRoomId.value = roomCode
    const unwatch = watch(connected, (c) => {
      if (c) {
        setTimeout(() => joinByCode(roomCode, playerName.value, tonIdentity.value, 'ton'), 500)
        unwatch()
      }
    }, { immediate: true })
  }
})

onUnmounted(() => { stopCountdown(); stopNextRoundCountdown() })

watch(roomState, (rs) => { if (rs && page.value !== 'game') page.value = 'game' })

// Auto-scroll event log
watch(gameEvents, async () => {
  await nextTick()
  if (logRef.value) logRef.value.scrollLeft = logRef.value.scrollWidth
}, { deep: true })

// Start countdown on turn change
watch(() => roomState.value?.turnDeadline, (td) => {
  if (td && roomState.value?.state === 1) startCountdown()
  else stopCountdown()
})

// Handle game end
watch(gameEnd, (ge) => {
  if (!ge) return
  stopCountdown()
  if (ge.winnerId === playerId.value) hapticFeedback('success')
  else hapticFeedback('notification')
  if (ge.nextRoundDeadline) startNextRoundCountdown(ge.nextRoundDeadline)
})

// Watch next round status updates
watch(nextRoundStatus, (nrs) => {
  if (nrs?.nextRoundDeadline) startNextRoundCountdown(nrs.nextRoundDeadline)
})

// Computed
const myPlayer = computed(() => roomState.value?.players?.find(p => p.id === playerId.value))
const otherPlayers = computed(() => roomState.value?.players?.filter(p => p.id !== playerId.value) || [])
const isMyTurn = computed(() => {
  if (!roomState.value || !myPlayer.value) return false
  return roomState.value.players[roomState.value.turnIndex]?.id === playerId.value
})
const isWinner = computed(() => gameEnd.value?.winnerId === playerId.value)

const callAmount = computed(() => {
  if (!roomState.value) return 0
  const lastBet = roomState.value.lastBet || roomState.value.baseBet
  return myPlayer.value?.looked ? lastBet * 2 : lastBet
})
const raiseAmount = computed(() => {
  if (!roomState.value) return 0
  const lastBet = roomState.value.lastBet || roomState.value.baseBet
  const amount = lastBet * 2
  return myPlayer.value?.looked ? amount * 2 : amount
})
const canCompare = computed(() => comparablePlayers.value.length > 0)
const comparablePlayers = computed(() => {
  if (!roomState.value) return []
  return roomState.value.players.filter(p => p.id !== playerId.value && (p.state === 2 || p.state === 4))
})
const recentEvents = computed(() => gameEvents.value.slice(-6))

// Next round confirmation
const isConfirmedNextRound = computed(() => {
  const nrs = nextRoundStatus.value || gameEnd.value
  if (!nrs?.readyForNext) {
    if (roomState.value?.readyForNext) return roomState.value.readyForNext.includes(playerId.value)
    return false
  }
  return nrs.readyForNext.includes(playerId.value)
})
const confirmedCount = computed(() => {
  const nrs = nextRoundStatus.value || gameEnd.value
  if (!nrs?.readyForNext) {
    if (roomState.value?.readyForNext) return roomState.value.readyForNext.length
    return 0
  }
  return nrs.readyForNext.length
})
const humanPlayerCount = computed(() => {
  if (!roomState.value) return 0
  return roomState.value.players.filter(p => !p.isBot).length
})

// SVG countdown offsets
const countdownOffset = computed(() => {
  const circ = 2 * Math.PI * 19 // r=19
  return circ * (1 - countdown.value / 20)
})
const countdownOffsetMy = computed(() => {
  const circ = 2 * Math.PI * 17 // r=17
  return circ * (1 - countdown.value / 20)
})

// Hand type detection
const handTypeName = computed(() => {
  if (!myPlayer.value?.hand) return ''
  const h = myPlayer.value.hand
  const ranks = [
    { name: 'Three of a Kind', check: h => h[0].value===h[1].value && h[1].value===h[2].value },
    { name: 'Straight Flush', check: h => isFlushFn(h) && isStraightFn(h) },
    { name: 'Flush', check: h => isFlushFn(h) },
    { name: 'Straight', check: h => isStraightFn(h) },
    { name: 'Pair', check: h => hasPair(h) },
    { name: 'High Card', check: () => true }
  ]
  for (const r of ranks) if (r.check(h)) return r.name
  return 'High Card'
})

function isFlushFn(h) { return h[0].suit===h[1].suit && h[1].suit===h[2].suit }
function isStraightFn(h) {
  const v = h.map(c=>c.value).sort((a,b)=>b-a)
  if (v[0]===14 && v[1]===3 && v[2]===2) return true
  return v[0]-v[1]===1 && v[1]-v[2]===1
}
function hasPair(h) { return h[0].value===h[1].value || h[1].value===h[2].value || h[0].value===h[2].value }

// Actions
function handleQuickStart() { hapticFeedback('impact'); createRoom(playerName.value, 2, 10, tonIdentity.value, 'ton') }
function handleCreatePrivateRoom() { hapticFeedback('impact'); createRoom(playerName.value, 0, 10, tonIdentity.value, 'ton') }
function handleJoinByCode() { if (joinRoomId.value.trim()) { hapticFeedback('impact'); joinByCode(joinRoomId.value.trim(), playerName.value, tonIdentity.value, 'ton') } }
function handleLeaveRoom() { stopCountdown(); stopNextRoundCountdown(); wsLeaveRoom(); roomState.value = null; gameEnd.value = null; gameEvents.value = []; page.value = 'lobby' }
function handleClaimBonus() { hapticFeedback('impact'); wsClaimBonus() }
function startGame() { hapticFeedback('impact'); wsStartGame() }
function lookCards() { hapticFeedback('selection'); wsDoAction(6) }
function doFold() { hapticFeedback('impact'); stopCountdown(); wsDoAction(3) }
function doCall() { hapticFeedback('impact'); stopCountdown(); wsDoAction(1) }
function doRaise() {
  hapticFeedback('impact'); stopCountdown()
  const lastBet = roomState.value.lastBet || roomState.value.baseBet
  wsDoAction(2, lastBet * 2 * (myPlayer.value?.looked ? 2 : 1))
}
function doAllIn() { hapticFeedback('impact'); stopCountdown(); wsDoAction(4) }
function doCompare(targetId) { showCompareModal.value = false; hapticFeedback('impact'); stopCountdown(); wsDoAction(5, 0, targetId) }
function handleNewRound() { hapticFeedback('success'); wsNewRound() }

function copyRoomCode() {
  if (roomState.value?.roomCode) {
    navigator.clipboard?.writeText(roomState.value.roomCode).catch(() => {})
    hapticFeedback('selection')
  }
}

function isTurn(p) {
  if (!roomState.value) return false
  return roomState.value.players[roomState.value.turnIndex]?.id === p.id
}

function suitSymbol(s) { return ['','\u2666','\u2663','\u2665','\u2660'][s]||'?' }
function suitClass(s) { return ['','diamond','club','heart','spade'][s]||'' }
function valueName(v) { return v===14?'A':v===13?'K':v===12?'Q':v===11?'J':String(v) }

const avatarColors = ['#e74c3c','#3498db','#2ecc71','#f39c12','#9b59b6','#1abc9c']
function avatarColor(seat) { return avatarColors[seat%avatarColors.length] }

// Position players around the oval table (top half arc for mobile)
function getSeatPosition(idx, total) {
  if (total <= 0) return {}
  // Spread players along top arc of table
  const angleStart = -180
  const angleRange = 180
  const angle = angleStart + (angleRange / (total + 1)) * (idx + 1)
  const rad = angle * Math.PI / 180
  const rx = 42, ry = 36
  const x = 50 + rx * Math.cos(rad)
  const y = 48 + ry * Math.sin(rad)
  return { left: x + '%', top: y + '%', transform: 'translate(-50%,-50%)' }
}

function formatEvent(e) {
  const n = e.playerName || '?'
  switch(e.type) {
    case 'call': return n + ' Call ' + e.amount
    case 'raise': return n + ' Raise ' + e.amount
    case 'fold': return n + ' Fold'
    case 'allin': return n + ' All In ' + e.amount
    case 'look': return n + ' Looked'
    case 'compare': return n + ' vs ' + e.targetName + ' -> ' + (e.winnerId===e.playerId?n:e.targetName) + ' wins'
    default: return n + ': ' + e.type
  }
}
</script>

<style>
/* TG Mini App styles - uses Telegram theme CSS vars */
:root {
  --bg-dark: var(--tg-theme-bg-color, #0a0e1a);
  --bg-secondary: var(--tg-theme-secondary-bg-color, #111827);
  --text: var(--tg-theme-text-color, #eef0f4);
  --text-dim: var(--tg-theme-hint-color, #7a8599);
  --accent: var(--tg-theme-button-color, #4facfe);
  --accent-text: var(--tg-theme-button-text-color, #ffffff);
  --gold: #ffd700;
  --red: #ff4757;
  --green: #2ecc71;
  --felt: #0d6b3d;
  --felt-dark: #084a2a;
}
* { margin:0; padding:0; box-sizing:border-box; }
html, body {
  font-family: -apple-system, 'SF Pro Text', 'Helvetica Neue', sans-serif;
  background: var(--bg-dark); color: var(--text);
  height: 100%; overflow: hidden;
  -webkit-user-select: none; user-select: none;
  -webkit-tap-highlight-color: transparent;
  -webkit-text-size-adjust: 100%;
}
.app {
  height: 100vh; height: 100dvh;
  display: flex; flex-direction: column;
  overflow: hidden; position: fixed; inset: 0;
}
.page { flex:1; display:flex; flex-direction:column; overflow:hidden; min-height:0; }
.toast {
  position: fixed; top: max(8px, env(safe-area-inset-top));
  left: 50%; transform: translateX(-50%);
  background: var(--red); color: #fff;
  padding: 6px 16px; border-radius: 16px;
  z-index: 200; font-size: 12px;
  animation: slideDown .3s ease;
  white-space: nowrap;
}
@keyframes slideDown { from { opacity:0; transform:translateX(-50%) translateY(-20px); } }

/* ===== LOBBY ===== */
.lobby-header { display:flex; align-items:center; padding:12px 16px; padding-top:max(12px,env(safe-area-inset-top)); gap:10px; }
.user-info { display:flex; align-items:center; gap:10px; }
.user-avatar { width:36px; height:36px; border-radius:50%; display:flex; align-items:center; justify-content:center; color:#fff; font-weight:700; flex-shrink:0; }
.user-meta {}
.user-name { font-weight:600; font-size:14px; }
.user-chips { color:var(--gold); font-size:12px; }
.lobby-body { flex:1; padding:16px; overflow-y:auto; -webkit-overflow-scrolling:touch; }
.mode-cards { display:flex; flex-direction:column; gap:10px; max-width:400px; margin:0 auto; }
.mode-card { padding:16px; border-radius:12px; border:1px solid rgba(255,255,255,0.1); background:rgba(255,255,255,0.04); color:var(--text); font-size:15px; cursor:pointer; text-align:center; transition:transform .08s; }
.mode-card:active { transform:scale(.97); }
.join-row { display:flex; gap:8px; }
.room-input { flex:1; padding:10px; border-radius:8px; border:1px solid rgba(255,255,255,0.1); background:rgba(255,255,255,0.06); color:var(--text); font-size:14px; outline:none; }
.btn-join { padding:10px 20px; border-radius:8px; border:none; background:var(--accent); color:var(--accent-text); font-size:14px; cursor:pointer; }
.bonus-card { background:linear-gradient(135deg, rgba(255,215,0,0.15), rgba(255,215,0,0.05)) !important; border-color:rgba(255,215,0,0.3) !important; }
.bonus-remaining { font-size:12px; color:var(--gold); margin-left:6px; }

/* ===== GAME TOP BAR ===== */
.game-topbar {
  display:flex; align-items:center; gap:8px;
  padding:4px 10px;
  padding-top:max(4px, env(safe-area-inset-top));
  background:rgba(0,0,0,0.4);
  border-bottom:1px solid rgba(255,255,255,0.06);
  flex-shrink:0;
}
.topbar-back { background:none; border:none; color:var(--text-dim); font-size:16px; cursor:pointer; padding:4px 6px; }
.topbar-pot { font-size:16px; font-weight:700; color:var(--gold); }
.topbar-info { display:flex; gap:8px; font-size:10px; color:var(--text-dim); margin-left:auto; }
.topbar-room { font-size:10px; color:var(--gold); padding:2px 6px; background:rgba(255,215,0,0.1); border-radius:4px; border:1px solid rgba(255,215,0,0.2); cursor:pointer; white-space:nowrap; }

/* ===== TABLE ===== */
.table-container { flex:1; display:flex; align-items:center; justify-content:center; padding:4px; min-height:0; overflow:hidden; }
.table-felt {
  width: min(100%, 92vw);
  aspect-ratio: 5/4;
  max-height: 100%;
  border-radius: 50%;
  background: radial-gradient(ellipse at 50% 40%, #1f9960 0%, #157a48 30%, var(--felt) 55%, var(--felt-dark) 85%);
  border: 4px solid #4a3520;
  box-shadow: 0 0 0 2px #1a1208, 0 0 0 5px #3a2810, 0 0 20px rgba(0,0,0,0.5), inset 0 0 30px rgba(0,0,0,0.25);
  position: relative;
  overflow: visible;
}

/* Pot center */
.table-pot-center { position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); text-align:center; z-index:5; }
.pot-stack { position:relative; display:flex; flex-direction:column; align-items:center; height:20px; }
.pot-chip { width:20px; height:20px; border-radius:50%; background:linear-gradient(135deg,var(--gold),#d4a800); border:1.5px solid #a67c00; position:absolute; box-shadow:0 1px 3px rgba(0,0,0,0.5); }
.pot-amount { display:block; margin-top:6px; font-size:12px; font-weight:800; color:var(--gold); text-shadow:0 1px 4px rgba(0,0,0,0.9); white-space:nowrap; }

/* ===== PLAYER SEATS ===== */
.seat {
  position:absolute; display:flex; flex-direction:column; align-items:center; gap:1px;
  z-index:10; transition:opacity .3s;
}
.seat.is-folded, .seat.is-out { opacity:.3; }

.seat-avatar-wrap { position:relative; display:flex; align-items:center; justify-content:center; }
.seat-avatar {
  width:36px; height:36px; border-radius:50%;
  display:flex; align-items:center; justify-content:center;
  color:#fff; font-weight:700; font-size:14px;
  box-shadow:0 2px 8px rgba(0,0,0,0.5);
  border:2px solid rgba(255,255,255,0.15);
  position:relative;
}
.seat.is-turn .seat-avatar { border-color:var(--gold); box-shadow:0 0 10px rgba(255,215,0,0.4); }

/* Countdown ring on seats */
.seat-cd-ring { position:absolute; inset:-4px; width:44px; height:44px; transform:rotate(-90deg); pointer-events:none; }
.cd-bg { fill:none; stroke:rgba(255,255,255,0.1); stroke-width:2.5; }
.cd-fg { fill:none; stroke:var(--accent); stroke-width:3; stroke-dasharray:119.38; stroke-linecap:round; transition:stroke-dashoffset .3s linear; }
.cd-fg.urgent { stroke:var(--red); }
.seat-cd-num { position:absolute; bottom:-12px; font-size:9px; font-weight:700; color:var(--accent); background:rgba(0,0,0,0.7); padding:0 4px; border-radius:3px; white-space:nowrap; }
.seat-cd-num.urgent { color:var(--red); animation:cdPulse .5s ease-in-out infinite; }
@keyframes cdPulse { 0%,100%{transform:scale(1);} 50%{transform:scale(1.15);} }

.seat-name { font-size:9px; font-weight:600; white-space:nowrap; text-shadow:0 1px 3px rgba(0,0,0,0.8); max-width:60px; overflow:hidden; text-overflow:ellipsis; }
.seat-chips { font-size:8px; color:var(--gold); white-space:nowrap; text-shadow:0 1px 3px rgba(0,0,0,0.8); }
.seat-bet { font-size:8px; background:rgba(255,215,0,0.15); color:var(--gold); padding:1px 5px; border-radius:6px; border:1px solid rgba(255,215,0,0.25); font-weight:600; margin-top:1px; }
.seat-cards { display:flex; gap:1px; margin-top:1px; }
.mini-card {
  width:16px; height:22px; border-radius:2px; background:#fff;
  display:flex; align-items:center; justify-content:center;
  font-size:7px; font-weight:700; color:#333;
  box-shadow:0 1px 2px rgba(0,0,0,0.3);
}
.mini-card.heart, .mini-card.diamond { color:var(--red); }
.mini-card.back {
  background:linear-gradient(135deg,#1a3a5c 25%,#0d2840 25%,#0d2840 50%,#1a3a5c 50%,#1a3a5c 75%,#0d2840 75%);
  background-size:5px 5px;
}
.seat-status { font-size:7px; padding:1px 4px; border-radius:3px; margin-top:1px; font-weight:600; }
.seat-status.folded { background:rgba(255,71,87,0.25); color:var(--red); }
.seat-status.allin { background:rgba(231,76,60,0.3); color:#ff6b6b; }
.seat-status.out { background:rgba(100,100,100,0.3); color:#888; }

/* ===== EVENT LOG ===== */
.game-log { display:flex; gap:5px; padding:2px 10px; overflow-x:auto; background:rgba(0,0,0,0.3); flex-shrink:0; scrollbar-width:none; }
.game-log::-webkit-scrollbar { display:none; }
.log-item { font-size:9px; color:var(--text-dim); white-space:nowrap; padding:2px 5px; background:rgba(255,255,255,0.04); border-radius:6px; flex-shrink:0; }

/* ===== MY PANEL ===== */
.my-panel {
  background: linear-gradient(180deg, rgba(10,15,30,0.95) 0%, rgba(8,12,24,0.98) 100%);
  border-top: 1px solid rgba(255,255,255,0.08);
  padding: 6px 10px;
  padding-bottom: max(6px, env(safe-area-inset-bottom));
  flex-shrink: 0;
}
.my-info-row { display:flex; align-items:center; gap:8px; margin-bottom:4px; }

.my-avatar-wrap { position:relative; display:flex; align-items:center; justify-content:center; flex-shrink:0; }
.my-cd-ring { position:absolute; inset:-4px; width:40px; height:40px; transform:rotate(-90deg); pointer-events:none; }
/* Re-use .cd-bg / .cd-fg from above, but override dasharray for r=17 */
.my-cd-ring .cd-fg { stroke-dasharray:106.81; }
.my-avatar {
  width:30px; height:30px; border-radius:50%;
  display:flex; align-items:center; justify-content:center;
  color:#fff; font-weight:700; font-size:13px;
  border:2px solid rgba(255,255,255,0.15);
}
.my-cd-num { position:absolute; top:-8px; right:-12px; font-size:9px; font-weight:700; color:var(--accent); background:rgba(0,0,0,0.8); padding:1px 3px; border-radius:3px; white-space:nowrap; }
.my-cd-num.urgent { color:var(--red); animation:cdPulse .5s ease-in-out infinite; }

.my-detail { display:flex; align-items:center; gap:8px; }
.my-name { font-weight:600; font-size:13px; }
.my-chips-val { font-size:12px; color:var(--gold); }
.my-bet-badge { margin-left:auto; font-size:10px; color:var(--text-dim); background:rgba(255,255,255,0.04); padding:2px 7px; border-radius:5px; }

/* ===== MY HAND ===== */
.my-hand { display:flex; gap:4px; justify-content:center; margin-bottom:3px; }
.game-card {
  width:46px; height:66px; border-radius:6px; background:#fff;
  position:relative; box-shadow:0 2px 8px rgba(0,0,0,0.4);
  animation:dealCard .3s ease backwards;
  cursor:default; transition:transform .1s;
}
.game-card:active { transform:translateY(-3px); }
@keyframes dealCard { from { transform:translateY(30px) rotate(8deg); opacity:0; } }
.gc-corner { position:absolute; display:flex; flex-direction:column; align-items:center; line-height:1; }
.gc-corner.top { top:3px; left:4px; }
.gc-corner.bottom { bottom:3px; right:4px; transform:rotate(180deg); }
.gc-val { font-size:13px; font-weight:800; color:#333; }
.gc-suit { font-size:10px; }
.gc-center { position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); font-size:20px; }
.game-card.heart .gc-val, .game-card.heart .gc-suit, .game-card.heart .gc-center,
.game-card.diamond .gc-val, .game-card.diamond .gc-suit, .game-card.diamond .gc-center { color:var(--red); }
.game-card.spade .gc-val, .game-card.spade .gc-suit, .game-card.spade .gc-center,
.game-card.club .gc-val, .game-card.club .gc-suit, .game-card.club .gc-center { color:#222; }
.game-card.card-back { background:#1a2a44; cursor:pointer; display:flex; align-items:center; justify-content:center; flex-direction:column; }
.card-back-pattern { position:absolute; inset:3px; border-radius:4px; border:1px solid rgba(255,255,255,0.1); background:repeating-linear-gradient(45deg,transparent,transparent 3px,rgba(255,255,255,0.03) 3px,rgba(255,255,255,0.03) 6px); }
.peek-label { position:relative; z-index:1; font-size:9px; color:rgba(255,255,255,0.5); font-weight:600; }
.hand-rank { text-align:center; font-size:11px; color:var(--gold); font-weight:700; margin-bottom:3px; letter-spacing:1px; }

/* ===== ACTION BAR ===== */
.action-bar { display:flex; gap:5px; justify-content:center; flex-wrap:nowrap; padding:2px 0; }
.act-btn {
  display:flex; flex-direction:column; align-items:center; gap:1px;
  padding:7px 11px; border-radius:9px; border:none;
  font-size:10px; font-weight:600; cursor:pointer;
  min-width:46px; transition:transform .08s;
  color:#fff; position:relative; overflow:hidden;
}
.act-btn::after { content:''; position:absolute; inset:0; background:linear-gradient(180deg, rgba(255,255,255,0.1) 0%, transparent 50%); pointer-events:none; }
.act-btn:active { transform:scale(.9); }
.act-btn .act-icon { font-size:14px; line-height:1; }
.act-btn .act-label { font-size:9px; opacity:.9; }
.act-btn small { font-size:8px; opacity:.7; }
.act-btn.fold { background:linear-gradient(180deg,#4a4a4a,#333); }
.act-btn.call { background:linear-gradient(180deg,#2ecc71,#1fa85a); box-shadow:0 2px 6px rgba(46,204,113,0.3); }
.act-btn.raise { background:linear-gradient(180deg,#3498db,#2176ad); box-shadow:0 2px 6px rgba(52,152,219,0.3); }
.act-btn.compare { background:linear-gradient(180deg,#e67e22,#c96b15); box-shadow:0 2px 6px rgba(230,126,34,0.3); }
.act-btn.allin { background:linear-gradient(180deg,#e74c3c,#b83227); box-shadow:0 2px 6px rgba(231,76,60,0.3); animation:allInGlow 2s ease-in-out infinite; }
@keyframes allInGlow { 0%,100%{box-shadow:0 2px 6px rgba(231,76,60,0.3);} 50%{box-shadow:0 2px 14px rgba(231,76,60,0.6);} }

/* ===== WAITING BAR ===== */
.waiting-bar { text-align:center; padding:6px 0; }
.waiting-info { font-size:12px; color:var(--text-dim); margin-bottom:6px; }
.btn-start { padding:10px 32px; border:none; border-radius:10px; background:linear-gradient(135deg,var(--gold),#ff8c00); color:#000; font-size:15px; font-weight:700; cursor:pointer; transition:transform .08s; }
.btn-start:active { transform:scale(.95); }

/* ===== RESULT OVERLAY ===== */
.result-overlay { position:fixed; inset:0; background:rgba(0,0,0,0.75); display:flex; align-items:center; justify-content:center; z-index:90; padding:16px; animation:fadeIn .3s ease; }
@keyframes fadeIn { from{opacity:0;} }
.result-popup {
  background:var(--bg-secondary); border-radius:18px; padding:20px 16px;
  max-width:320px; width:100%; text-align:center;
  border:2px solid rgba(255,255,255,0.08);
  max-height:85vh; overflow-y:auto;
}
.result-popup.win { border-color:var(--gold); }
.result-crown { font-size:36px; margin-bottom:4px; animation:bounce .6s ease; }
@keyframes bounce { 0%,100%{transform:translateY(0);} 50%{transform:translateY(-8px);} }
.result-popup h2 { font-size:18px; margin-bottom:10px; }
.result-popup.win h2 { color:var(--gold); }
.result-info { display:flex; flex-direction:column; gap:5px; margin-bottom:10px; }
.result-row { display:flex; justify-content:space-between; font-size:12px; padding:0 4px; }
.result-row span { color:var(--text-dim); }
.result-row .gold { color:var(--gold); }
.result-row .dim { color:var(--text-dim); }
.result-hands { margin-bottom:12px; display:flex; flex-direction:column; gap:3px; }
.result-player { display:flex; align-items:center; gap:5px; padding:4px 6px; background:rgba(255,255,255,0.04); border-radius:5px; font-size:11px; }
.rp-name { min-width:44px; }
.rp-name.winner { color:var(--gold); font-weight:700; }
.rp-cards { display:flex; gap:2px; }
.rp-card { font-size:10px; padding:1px 3px; background:rgba(255,255,255,0.08); border-radius:2px; font-weight:600; }
.rp-card.heart, .rp-card.diamond { color:var(--red); }

/* Next round confirmation */
.next-round-status { margin-bottom:10px; }
.confirm-progress { display:flex; justify-content:space-between; align-items:center; font-size:11px; color:var(--text-dim); padding:5px 8px; background:rgba(255,255,255,0.04); border-radius:6px; margin-bottom:4px; }
.confirm-timer { color:var(--red); font-weight:700; font-size:13px; }
.confirmed-badge { text-align:center; color:var(--accent); font-size:12px; font-weight:600; padding:3px; }

.result-btn-row { display:flex; gap:8px; }
.btn-exit { flex:1; padding:10px; border:1px solid rgba(255,255,255,0.15); border-radius:10px; background:rgba(255,255,255,0.06); color:var(--text); font-size:14px; font-weight:600; cursor:pointer; }
.btn-again { flex:1; padding:10px; border:none; border-radius:10px; background:linear-gradient(135deg,var(--gold),#ff8c00); color:#000; font-size:14px; font-weight:700; cursor:pointer; }
.btn-exit:active, .btn-again:active { transform:scale(.95); }

/* ===== COMPARE MODAL ===== */
.modal-overlay { position:fixed; inset:0; background:rgba(0,0,0,0.6); display:flex; align-items:flex-end; justify-content:center; z-index:100; padding:0; backdrop-filter:blur(3px); }
.modal {
  background:var(--bg-secondary); border:1px solid rgba(255,255,255,0.1);
  border-radius:16px 16px 0 0; padding:16px;
  max-width:100%; width:100%;
  max-height:60vh; overflow-y:auto;
  animation:slideUp .25s ease;
}
@keyframes slideUp { from { transform:translateY(100%); } }
.modal h3 { text-align:center; margin-bottom:10px; font-size:15px; }
.compare-targets { display:flex; flex-direction:column; gap:6px; margin-bottom:10px; }
.compare-item {
  display:flex; align-items:center; gap:8px; padding:10px 12px;
  border-radius:10px; border:1px solid rgba(255,255,255,0.08);
  background:rgba(255,255,255,0.04); color:var(--text); cursor:pointer;
  font-size:14px; transition:background .1s;
}
.compare-item:active { background:rgba(255,255,255,0.1); }
.ci-avatar { width:30px; height:30px; border-radius:8px; display:flex; align-items:center; justify-content:center; color:#fff; font-weight:700; font-size:14px; }
.ci-name { flex:1; }
.ci-chips { font-size:12px; color:var(--gold); }
.btn-modal-close { width:100%; padding:10px; border:1px solid rgba(255,255,255,0.1); border-radius:10px; background:transparent; color:var(--text-dim); font-size:14px; cursor:pointer; }
</style>
