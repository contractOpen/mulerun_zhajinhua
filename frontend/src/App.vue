<template>
  <div class="app">
    <!-- Toast -->
    <div v-if="error" class="toast error">{{ error }}</div>
    <div v-if="notice" :class="['toast', noticeType]">{{ notice }}</div>

    <!-- LOGIN PAGE -->
    <div v-if="page === 'login'" class="page login-page">
      <div class="login-bg">
        <div class="particle" v-for="i in 15" :key="i" :style="particleStyle(i)"></div>
      </div>
      <div class="login-card">
        <div class="logo">
          <div class="logo-icon"><span class="logo-a">A</span><span class="logo-suit">♠</span></div>
          <h1 class="game-title">{{ t('game.title') }}</h1>
          <p class="game-subtitle">{{ t('game.subtitle') }}</p>
        </div>
        <div class="login-form">
          <div class="input-group">
            <label>{{ t('login.nickname') }}</label>
            <input v-model="playerName" :placeholder="t('login.nicknamePlaceholder')" maxlength="8" @keyup.enter="handleLogin" />
          </div>
          <button class="btn-web3" @click="handleWalletConnect">
            <span class="wallet-icon">🦊</span>
            {{ effectiveWalletConnected ? effectiveWalletAddress.slice(0,6)+'...'+effectiveWalletAddress.slice(-4) : t('login.connectWallet') }}
          </button>
          <button class="btn-enter" @click="handleLogin" :disabled="!connected || (appMode === 'pe' && !effectiveWalletConnected)">
            {{ !connected ? t('login.connecting') : authenticated ? t('login.enter') : t('login.authenticate') }}
          </button>
          <button class="btn-rules" @click="showRulesModal = true">{{ t('login.rules') }}</button>
        </div>
        <div class="login-lang">
          <div class="lang-picker">
            <button v-for="l in locales" :key="l.code" :class="['lang-btn', { active: locale === l.code }]" @click="setLocale(l.code)">{{ l.name }}</button>
          </div>
        </div>
        <div v-if="loginLinks.length" class="login-links">
          <a
            v-for="link in loginLinks"
            :key="link.key"
            :href="link.href"
            class="login-link"
            :title="link.label"
            target="_blank"
            rel="noopener noreferrer"
          >
            <svg v-if="link.key === 'website'" class="login-link-svg mule-icon" viewBox="0 0 24 24" aria-hidden="true">
              <path fill="currentColor" d="M7.1 5.2 9.4 2.8l1.4 3h2.4l1.2-2.9 2.4 2.3-.6 2.8 2.2 2.2v6.4l-2.8 2.8H9l-3.1-3.2V10l2-2.1zm2.2 2.8-1 1v6l1.6 1.6h4.9l1.4-1.4v-4l-1.3.8-.7-1.4-1.8.7-1.6-.7-.9 1.2-1.2-.8.9-1.9 2.7-.9 2.1.6 1.5-1-1-1.2zm1.8 4.4a1 1 0 1 0 0 2.1 1 1 0 0 0 0-2.1zm3.1 0a1 1 0 1 0 0 2.1 1 1 0 0 0 0-2.1z"/>
            </svg>
            <svg v-else-if="link.key === 'telegram'" class="login-link-svg telegram-icon" viewBox="0 0 24 24" aria-hidden="true">
              <path fill="currentColor" d="M20.7 4.2 3.8 10.7c-1.2.5-1.2 1.2-.2 1.5l4.3 1.4 1.7 5.2c.2.6.1.8.8.8.5 0 .7-.2 1-.5l2.4-2.3 4.9 3.6c.9.5 1.5.3 1.7-.8l2.9-13.8c.3-1.4-.5-2-1.6-1.6zM9 13.2l8.4-5.3c.4-.2.7-.1.4.2l-7 6.3-.3 3.3z"/>
            </svg>
            <svg v-else class="login-link-svg discord-icon" viewBox="0 0 24 24" aria-hidden="true">
              <path fill="currentColor" d="M19.7 5.6a15.3 15.3 0 0 0-3.8-1.2l-.2.4-.3 1a14 14 0 0 0-6.8 0l-.3-1-.2-.4a15.3 15.3 0 0 0-3.8 1.2C1.9 9.2 1.2 12.7 1.5 16.1a15.5 15.5 0 0 0 4.7 2.4l1-1.7-1.7-.8.4-.3c3.2 1.5 6.9 1.5 10.1 0l.4.3-1.7.8 1 1.7a15.5 15.5 0 0 0 4.7-2.4c.4-4-.6-7.5-2.7-10.5zm-9.1 8.4c-.9 0-1.6-.8-1.6-1.8s.7-1.8 1.6-1.8 1.6.8 1.6 1.8-.7 1.8-1.6 1.8zm4.8 0c-.9 0-1.6-.8-1.6-1.8s.7-1.8 1.6-1.8 1.6.8 1.6 1.8-.7 1.8-1.6 1.8z"/>
            </svg>
          </a>
        </div>
        <div class="login-footer"><span>{{ t('login.footer') }}</span></div>
      </div>
    </div>

    <!-- LOBBY PAGE -->
    <div v-else-if="page === 'lobby'" class="page lobby-page">
      <div class="lobby-header">
        <div class="user-info">
          <div class="user-avatar" :style="{ backgroundImage: 'url('+identiconSvg(playerId || 'default')+')', backgroundSize: 'cover', background: avatarColor(0) }">{{ playerName[0] }}</div>
          <div class="user-meta">
            <div class="user-name">{{ playerName }}</div>
            <div class="user-chips">
              <span class="chip-icon">🪙</span>{{ userChips }} {{ t('lobby.chips') }}
              <button v-if="bonusInfo?.canClaim" class="btn-bonus" @click="claimBonus()" :disabled="!authenticated">
                {{ bonusInfo?.remaining || 0 }}/3
              </button>
              <button v-else-if="bonusInfo" class="btn-recharge-from-bonus" @click="showRechargeModal = true">
                {{ t('lobby.recharge') }}
              </button>
              <button class="btn-recharge-sm" @click="showRechargeModal = true">+</button>
            </div>
          </div>
        </div>
        <div class="header-actions">
          <div class="lang-picker-inline">
            <button v-for="l in locales" :key="l.code" :class="['lang-btn-sm', { active: locale === l.code }]" @click="setLocale(l.code)">{{ l.code.toUpperCase() }}</button>
          </div>
          <button class="btn-icon" @click="bgmOn = toggleBGMFn()">{{ bgmOn ? '🔊' : '🔇' }}</button>
        </div>
      </div>
      <div class="lobby-body">
        <div class="mode-cards">
          <div class="mode-card" @click="handleQuickStart">
            <div class="mode-icon">⚡</div>
            <h3>{{ t('lobby.quickStart') }}</h3>
            <p>{{ t('lobby.quickStartDesc') }}</p>
            <div class="mode-detail">
              <label>{{ t('lobby.baseBet') }}</label>
              <div class="opt-row">
                <button v-for="b in [5,10,20,50,100]" :key="b" :class="['opt-btn',{active:baseBet===b}]" @click.stop="baseBet=b">{{b}}</button>
              </div>
            </div>
            <div class="mode-detail">
              <label>{{ t('lobby.bots') }}</label>
              <div class="opt-row">
                <button v-for="n in [1,2,3,4,5]" :key="n" :class="['opt-btn',{active:botCount===n}]" @click.stop="botCount=n">{{n}}</button>
              </div>
            </div>
          </div>
          <div class="mode-card" @click="handleStartMatch">
            <div class="mode-icon">🎯</div>
            <h3>{{ t('lobby.matchPlay') }}</h3>
            <p>{{ t('lobby.matchDesc') }}</p>
            <div class="mode-detail">
              <label>{{ t('lobby.baseBet') }}</label>
              <div class="opt-row">
                <button v-for="b in [5,10,20,50,100]" :key="`match-${b}`" :class="['opt-btn',{active:baseBet===b}]" @click.stop="baseBet=b">{{b}}</button>
              </div>
            </div>
            <div v-if="matchStatus?.status==='matching'" class="matching-indicator">
              <div class="match-spinner"></div>
              <span>{{ t('lobby.matching') }} ({{ matchStatus.players }})</span>
              <button class="btn-cancel-match" @click.stop="cancelMatch()">{{ t('common.cancel') }}</button>
            </div>
          </div>
          <div class="mode-card">
            <div class="mode-icon">🏠</div>
            <h3>{{ t('lobby.createRoom') }}</h3>
            <p>{{ t('lobby.inviteDesc') }}</p>
            <button class="btn-create-room" @click="handleCreatePrivateRoom">{{ t('lobby.createPrivateRoom') }}</button>
            <div class="mode-detail join-row">
              <input v-model="joinRoomId" :placeholder="t('lobby.enterRoomCode')" class="room-input" />
              <button class="btn-join" @click="handleJoinByCode">{{ t('lobby.join') }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- GAME PAGE -->
    <div v-else-if="page === 'game'" class="page game-page">
      <div class="game-ambience">
        <div class="ambience-glow left"></div>
        <div class="ambience-glow right"></div>
        <div class="ambience-mesh"></div>
      </div>
      <!-- Top bar -->
      <div class="game-topbar">
        <button class="btn-back" @click="handleLeaveRoom">← {{ t('game.leave') }}</button>
        <div class="room-code" v-if="roomState?.roomCode">
          <span @click="copyRoomCode">{{ roomState.roomCode }} 📋</span>
          <button class="btn-share" @click="copyShareLink">🔗</button>
        </div>
        <div class="topbar-center">
          <span class="pot-label">{{ t('game.pot') }}</span>
          <span class="pot-value">{{ roomState?.pot || 0 }}</span>
        </div>
        <div class="topbar-chips">
          <span class="chip-icon">🪙</span>{{ userChips }}
        </div>
        <div class="topbar-meta">
          <span>R{{ roomState?.round || 0 }}</span>
          <span>{{ t('game.currentBet') }}:{{ roomState?.currentBet || 0 }}</span>
          <span v-if="roomState?.spectators?.length">{{ t('game.spectators', { n: roomState.spectators.length }) }}</span>
        </div>
        <div class="topbar-actions">
          <div class="lang-picker-inline">
            <button v-for="l in locales" :key="l.code" :class="['lang-btn-sm',{active:locale===l.code}]" @click="setLocale(l.code)">{{l.code.toUpperCase()}}</button>
          </div>
          <button class="btn-icon" @click="bgmOn = toggleBGMFn()">{{ bgmOn ? '🔊' : '🔇' }}</button>
        </div>
      </div>

      <!-- Table -->
      <div class="table-container">
        <div class="table-felt">
          <div class="table-rail"></div>
          <div class="table-inner-ring"></div>
          <div class="table-chip-bursts">
            <div
              v-for="burst in chipBursts"
              :key="burst.id"
              class="chip-burst"
              :style="burst.style"
            >
              <span v-for="n in burst.count" :key="n" class="burst-chip" :style="{ animationDelay: `${(n - 1) * 0.04}s` }"></span>
            </div>
          </div>
          <div class="table-center-badge">
            <span class="table-badge-label">{{ t('game.pot') }}</span>
            <strong class="table-badge-value">{{ roomState?.pot || 0 }}</strong>
            <span class="table-badge-meta">{{ t('game.anteLabel') }} {{ roomState?.anteTotal || 0 }}</span>
          </div>
          <div class="table-inner">
            <div class="table-logo">{{ t('game.tableTitle') }}</div>
            <div class="table-pot-center">
              <div class="pot-chips-stack" v-if="roomState?.pot > 0">
                <div class="chip-stack" v-for="i in Math.min(Math.ceil(roomState.pot/50),8)" :key="i" :style="{transform:`translateY(${-i*3}px) rotate(${i*15}deg)`}"></div>
                <span class="pot-amount">{{ roomState.pot }}</span>
              </div>
            </div>
            <!-- Other players -->
            <div v-for="(p,idx) in otherPlayers" :key="p.id"
              :class="['table-player',{
                'is-turn':isTurn(p),'is-folded':p.state===3,'is-out':p.state===5
              }]" :style="getPlayerPosition(idx,otherPlayers.length)">
              <div class="tp-plaque"></div>
              <div class="tp-avatar-wrap">
                <!-- Countdown ring around avatar for current turn player -->
                <svg v-if="isTurn(p) && roomState?.state===1" class="tp-countdown-ring" viewBox="0 0 52 52">
                  <circle class="tp-cd-bg" cx="26" cy="26" r="23" />
                  <circle :class="['tp-cd-fg', {'urgent': countdown<=5}]" cx="26" cy="26" r="23" :style="{strokeDashoffset: countdownOffsetOther}" />
                </svg>
                <div class="tp-avatar" :style="{background:avatarColor(p.seat), backgroundImage:'url('+identiconSvg(p.id)+')', backgroundSize:'cover'}">
                  {{ p.name[0] }}
                  <div v-if="p.isBot" class="tp-bot">AI</div>
                </div>
                <span v-if="isTurn(p) && roomState?.state===1" :class="['tp-cd-num', {'urgent': countdown<=5}]">{{ countdown }}</span>
              </div>
              <div class="tp-name">{{ p.name }}</div>
              <div class="tp-chips"><span class="tp-chip-icon">🪙</span>{{ p.chips }}</div>
              <div v-if="p.betTotal>0" class="tp-bet"><span>{{ p.betTotal }}</span></div>
              <div class="tp-cards">
                <template v-if="p.hand && roomState.state===2">
                  <div v-for="(c,ci) in p.hand" :key="ci" :class="['mini-card', 'fan-card', suitClass(c.suit)]" :style="fanCardStyle(ci, 3)">{{ valueName(c.value) }}{{ suitSymbol(c.suit) }}</div>
                </template>
                <template v-else-if="roomState.state===1 && (p.state===2||p.state===4)">
                  <div class="mini-card back fan-card" v-for="ci in 3" :key="ci" :style="fanCardStyle(ci - 1, 3)"></div>
                </template>
              </div>
              <div v-if="p.state===3" class="tp-status folded">{{ t('game.folded') }}</div>
              <div v-if="p.state===5" class="tp-status out">{{ t('game.eliminated') }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Event log -->
      <div class="game-log" ref="logRef">
        <span v-for="(e,i) in recentEvents" :key="i" class="log-item">{{ formatEvent(e) }}</span>
      </div>

      <!-- My panel -->
      <div class="my-panel">
        <div class="my-info-row">
          <div class="my-avatar-wrap">
            <svg v-if="isMyTurn && roomState?.state===1" class="my-countdown-ring" viewBox="0 0 48 48">
              <circle class="my-cd-bg" cx="24" cy="24" r="21" />
              <circle :class="['my-cd-fg', {'urgent': countdown<=5}]" cx="24" cy="24" r="21" :style="{strokeDashoffset: countdownOffsetMy}" />
            </svg>
            <div class="my-avatar" :style="{background:avatarColor(myPlayer?.seat||0)}">
              {{ playerName[0] }}
            </div>
            <span v-if="isMyTurn && roomState?.state===1" :class="['my-cd-num', {'urgent': countdown<=5}]">{{ countdown }}s</span>
          </div>
          <div class="my-detail">
            <span class="my-name">{{ myPlayer?.name }}</span>
            <span class="my-chips"><span class="my-chip-icon">🪙</span>{{ myPlayer?.chips||0 }}</span>
          </div>
          <div class="my-bet-info" v-if="myPlayer?.betTotal>0">{{ t('game.betPlaced') }}: {{ myPlayer.betTotal }}</div>
        </div>

        <!-- Hand -->
        <div class="my-hand">
          <template v-if="myPlayer?.hand && myPlayer?.looked">
            <div v-for="(c,i) in myPlayer.hand" :key="i" :class="['game-card','fanned-game-card',suitClass(c.suit)]" :style="myHandCardStyle(i)">
              <div class="gc-corner top"><div class="gc-value">{{ valueName(c.value) }}</div><div class="gc-suit">{{ suitSymbol(c.suit) }}</div></div>
              <div class="gc-center">{{ suitSymbol(c.suit) }}</div>
              <div class="gc-corner bottom"><div class="gc-value">{{ valueName(c.value) }}</div><div class="gc-suit">{{ suitSymbol(c.suit) }}</div></div>
            </div>
          </template>
          <template v-else-if="roomState?.state===1">
            <div class="game-card card-back fanned-game-card" v-for="i in 3" :key="i" :style="myHandCardStyle(i - 1)" @click="lookCards">
              <div class="card-back-pattern"></div>
              <span class="peek-label">{{ t('game.look') }}</span>
            </div>
          </template>
        </div>
        <div v-if="myPlayer?.hand && myPlayer?.looked && roomState?.state===1" class="hand-rank">{{ handTypeName }}</div>

        <!-- Action bar -->
        <div class="action-bar" v-if="isMyTurn && roomState?.state===1 && myPlayer?.chips > 0">
          <button class="act-btn fold" @click="doFold"><span class="act-icon">✕</span><span class="act-label">{{ t('action.fold') }}</span></button>
          <button v-if="!roomState.hasAllIn" class="act-btn call" @click="doCall"><span class="act-icon">✓</span><span class="act-label">{{ t('action.call') }}</span><small>{{ callAmount }}</small></button>
          <button v-if="!roomState.hasAllIn" class="act-btn raise" @click="doRaise"><span class="act-icon">↑</span><span class="act-label">{{ t('action.raise') }}</span><small>{{ raiseAmount }}</small></button>
          <button v-if="canCompare" class="act-btn compare" @click="showCompareModal=true"><span class="act-icon">⚔</span><span class="act-label">{{ t('action.compare') }}</span><small v-if="roomState?.hasAllIn">{{ allInCompareCost }}</small></button>
          <button class="act-btn allin" @click="doAllIn"><span class="act-icon">🔥</span><span class="act-label">{{ t('action.allIn') }}</span></button>
        </div>

        <!-- Waiting / Start -->
        <div v-if="roomState?.isSpectator" class="spectator-banner">{{ t('game.spectating') }}</div>
        <div v-if="roomState?.state===0" class="waiting-bar">
          <div v-if="isRoomOwner && !roomState?.isMatch" class="waiting-basebet">
            <span class="waiting-basebet-label">{{ t('lobby.baseBet') }}</span>
            <div class="opt-row">
              <button v-for="b in [5,10,20,50,100]" :key="`room-bet-${b}`" :class="['opt-btn',{active:roomState?.baseBet===b}]" @click="handleUpdateBaseBet(b)">{{ b }}</button>
            </div>
          </div>
          <div v-else class="waiting-readonly">{{ t('lobby.baseBet') }} {{ roomState?.baseBet || 0 }}</div>
          <button v-if="canStartWaitingRoom" class="btn-start" @click="startGame()">{{ t('game.start') }}</button>
          <div v-else-if="roomState?.isPrivate && !roomState?.isSpectator" class="waiting-readonly">{{ t('game.ownerOnlyStart') }}</div>
        </div>

        <!-- In-game recharge -->
        <div v-if="myPlayer && myPlayer.chips<=0 && roomState?.state!==1" class="ingame-recharge">
          <span class="ingame-recharge-msg">{{ t('result.bankrupt') }}</span>
          <button class="btn-ingame-recharge" @click="showRechargeModal=true">{{ t('lobby.confirmRecharge') }}</button>
        </div>
      </div>

      <!-- Result overlay -->
      <div v-if="gameEnd" class="result-overlay" @click.self="dismissResult">
        <div :class="['result-popup',isWinner?'win':'lose']">
          <div class="result-glow"></div>
          <div v-if="isWinner" class="result-crown">👑</div>
          <h2>{{ isWinner ? t('result.youWin') : t('result.wins', { name: gameEnd.winnerName }) }}</h2>
          <div class="result-info">
            <div class="result-row"><span>{{ t('result.handType') }}</span><strong>{{ localizedGameEndHandType }}</strong></div>
            <div class="result-row"><span>{{ t('result.netWin') }}</span><strong class="gold">+{{ finalResultPot }}</strong></div>
            <div v-if="gameEnd.rake" class="result-row"><span>{{ t('result.rake') }}</span><strong class="dim">-{{ gameEnd.rake }}</strong></div>
          </div>
          <div class="result-hands">
            <div v-for="p in roomState?.players" :key="p.id" class="result-player">
              <span class="rp-name" :class="{winner:p.id===gameEnd.winnerId}">{{ p.name }}</span>
              <div class="rp-cards" v-if="p.hand">
                <span v-for="(c,ci) in p.hand" :key="ci" :class="['rp-card',suitClass(c.suit)]">{{ valueName(c.value) }}{{ suitSymbol(c.suit) }}</span>
              </div>
            </div>
          </div>
          <!-- Next round confirmation status -->
          <div class="next-round-status">
            <div class="confirm-progress">
              <span>{{ confirmedCount }}/{{ humanPlayerCount }} {{ t('result.confirmed') }}</span>
              <span class="confirm-timer" v-if="nextRoundCountdown < 30">{{ nextRoundCountdown }}s</span>
            </div>
            <div v-if="isConfirmedNextRound" class="confirmed-badge">✓ {{ t('result.confirmed') }}</div>
          </div>
          <div v-if="myPlayer && myPlayer.chips<=0" class="bankrupt-msg">{{ t('result.bankrupt') }}</div>
          <div class="result-btn-row">
            <button class="btn-exit-game" @click="handleLeaveRoom">{{ t('result.exit') }}</button>
            <button v-if="canPlayAgainAfterResult" class="btn-next-round" @click="handleNewRound">{{ t('result.playAgain') }}</button>
            <button v-else-if="myPlayer && myPlayer.chips <= 0" class="btn-next-round recharge" @click="showRechargeModal=true">{{ t('result.recharge') }}</button>
          </div>
        </div>
      </div>

      <!-- Compare modal -->
      <div v-if="showCompareModal" class="modal-overlay" @click.self="showCompareModal=false">
        <div class="modal compare-modal">
          <h3>{{ t('game.selectCompareTarget') }}</h3>
          <div class="compare-targets">
            <button v-for="p in comparablePlayers" :key="p.id" class="compare-item" @click="doCompare(p.id)">
              <div class="ci-avatar" :style="{background:avatarColor(p.seat)}">{{ p.name[0] }}</div>
              <span>{{ p.name }}</span>
              <span class="ci-chips">🪙{{ p.chips }}</span>
            </button>
          </div>
          <button class="btn-close" @click="showCompareModal=false">{{ t('common.cancel') }}</button>
        </div>
      </div>
    </div>

    <!-- Recharge modal (app-level) -->
    <div v-if="showRechargeModal" class="modal-overlay" @click.self="showRechargeModal=false">
      <div class="modal recharge-modal">
        <h3>{{ t('lobby.rechargeTitle') }}</h3>
        <p class="modal-desc">{{ t('lobby.rechargeDesc') }}</p>
        <div class="recharge-options">
          <button v-for="amt in rechargeOptions" :key="amt" :class="['recharge-btn',{active:rechargeAmount===amt}]" @click="rechargeAmount=amt">
            <span class="recharge-value">{{ amt }}</span>
            <span class="recharge-label">{{ t('lobby.chips') }}</span>
          </button>
        </div>
        <div class="recharge-wallet" v-if="effectiveWalletConnected">
          <span>{{ t('common.wallet') }}: {{ effectiveWalletAddress.slice(0,10) }}...</span>
        </div>
        <button class="btn-confirm-recharge" @click="handleRecharge">{{ t('lobby.confirmRecharge') }}</button>
        <button class="btn-close" @click="showRechargeModal=false">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <!-- Rules modal -->
    <div v-if="showRulesModal" class="modal-overlay" @click="showRulesModal=false">
      <div class="modal rules-modal" @click.stop>
        <h3>{{ t('login.rulesTitle') }}</h3>
        <div class="rules-content">
          <p v-for="(line, i) in rulesLines" :key="i">{{ line }}</p>
        </div>
        <button class="btn-close" @click="showRulesModal=false">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <!-- Wallet connect modal (PE mode) -->
    <div v-if="showWalletModal" class="modal-overlay" @click.self="showWalletModal=false">
      <div class="modal wallet-modal">
        <h3>{{ t('wallet.connectTitle') }}</h3>
        <p class="modal-desc">{{ t('wallet.connectDescription') }}</p>
        <div v-if="walletError" class="wallet-error">{{ walletError }}</div>
        <div class="wallet-options">
          <button class="wallet-opt" @click="connectEVM(); showWalletModal=false" :disabled="walletConnecting">
            <span class="wallet-opt-icon">🦊</span>
            <span class="wallet-opt-name">{{ t('wallet.metamask') }}</span>
            <span class="wallet-opt-chain">EVM</span>
          </button>
          <button class="wallet-opt" @click="connectSOL(); showWalletModal=false" :disabled="walletConnecting">
            <span class="wallet-opt-icon">👻</span>
            <span class="wallet-opt-name">{{ t('wallet.phantom') }}</span>
            <span class="wallet-opt-chain">SOL</span>
          </button>
        </div>
        <button class="btn-close" @click="showWalletModal=false">{{ t('common.cancel') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useGame } from './composables/useGame.js'
import { playSound, toggleBGM, startActionBGM, stopActionBGM } from './composables/useAudio.js'
import { useI18n } from './composables/useI18n.js'
import { useWallet } from './composables/useWallet.js'
const { t, locale, setLocale, locales } = useI18n()

const {
  walletAddress: realWalletAddress, walletChain, walletConnected: realWalletConnected,
  walletConnecting, walletError,
  hasMetaMask, hasPhantom,
  connectEVM, connectSOL,
  signAuthMessage,
  disconnect: disconnectWallet
} = useWallet()

const appMode = import.meta.env.VITE_APP_MODE || 'te'

const {
  connected, authenticated, playerId, userChips, roomState, gameEvents, gameEnd, error, matchStatus, nextRoundStatus, serverMode, bonusInfo,
  connect, login, requestAuthChallenge, createRoom, updateBaseBet: wsUpdateBaseBet, joinRoom, startGame: wsStartGame, doAction: wsDoAction,
  newRound: wsNewRound, startMatch, cancelMatch, recharge, joinByCode, leaveRoom: wsLeaveRoom, claimBonus, checkBonusStatus
} = useGame()

const page = ref('login')
const playerName = ref('')
const botCount = ref(2)
const baseBet = ref(10)
const walletConnected = ref(false)
const walletAddress = ref('')
const bgmOn = ref(false)
const showRechargeModal = ref(false)
const rechargeAmount = ref(500)
const showCompareModal = ref(false)
const showRulesModal = ref(false)
const showWalletModal = ref(false)
const joinRoomId = ref('')
const logRef = ref(null)
const pendingLogin = ref(false) // NEW: flag to auto-enter lobby after auth
const notice = ref('')
const noticeType = ref('info')
const chipBursts = ref([])
const publicConfig = ref({
  websiteUrl: '',
  telegramUrl: '',
  discordUrl: ''
})
const DEFAULT_API_BASE_URL = 'https://game.atdl.link'
const runtimeApiBaseUrl = (import.meta.env.VITE_API_BASE_URL || DEFAULT_API_BASE_URL).trim().replace(/\/+$/, '')

const LOW_CHIPS_THRESHOLD = 50

let noticeTimer = null
let chipBurstId = 0
let handleAuthSessionExpired = null
let handleRoomClosed = null
let bonusDayKey = getUtcDayKey()
let bonusDayInterval = null
let lowChipsStatusRequested = false
let lowChipsAutoClaimPending = false
let lowChipsNoBonusReminderKey = ''

// Global countdown timer (visible to all players for whoever's turn it is)
const countdown = ref(20)
let countdownInterval = null

// Next round confirmation countdown
const nextRoundCountdown = ref(30)
let nextRoundInterval = null

function startCountdown() {
  stopCountdown()
  if (!roomState.value?.turnDeadline) { countdown.value = 20; return }
  function tick() {
    if (!roomState.value?.turnDeadline) return
    const now = Date.now()
    const remaining = Math.max(0, Math.ceil((roomState.value.turnDeadline - now) / 1000))
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

// Which player ID is currently in turn
const currentTurnPlayerId = computed(() => {
  if (!roomState.value || roomState.value.state !== 1) return null
  return roomState.value.players[roomState.value.turnIndex]?.id || null
})

let handlePlayerNameUpdate = null

function getUtcDayKey() {
  return new Date().toISOString().slice(0, 10)
}

function showNotice(message, type = 'info', duration = 3200) {
  if (!message) return
  notice.value = message
  noticeType.value = type
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => {
    notice.value = ''
    noticeTimer = null
  }, duration)
}

function resetLowChipsBonusFlow() {
  lowChipsStatusRequested = false
  lowChipsAutoClaimPending = false
  lowChipsNoBonusReminderKey = ''
}

function refreshBonusDayIfNeeded() {
  const nextDayKey = getUtcDayKey()
  if (nextDayKey === bonusDayKey) return
  bonusDayKey = nextDayKey
  resetLowChipsBonusFlow()
  if (authenticated.value && userChips.value < LOW_CHIPS_THRESHOLD) {
    lowChipsStatusRequested = true
    checkBonusStatus()
  }
}

onMounted(() => {
  connect()
  playerName.value = t('common.player') + Math.floor(Math.random() * 1000)
  bonusDayInterval = setInterval(refreshBonusDayIfNeeded, 60 * 1000)
  loadPublicConfig()

  // NEW: Listen for player name updates from backend
  handlePlayerNameUpdate = (e) => {
    playerName.value = e.detail.playerName
  }
  window.addEventListener('playerNameUpdated', handlePlayerNameUpdate)
  handleAuthSessionExpired = (e) => {
    page.value = 'login'
    pendingLogin.value = false
    showNotice(e?.detail?.message || t('notice.authExpired'), 'info', 4200)
  }
  window.addEventListener('authSessionExpired', handleAuthSessionExpired)
  handleRoomClosed = () => {
    if (authenticated.value) {
      page.value = 'lobby'
    }
  }
  window.addEventListener('roomClosed', handleRoomClosed)

  // Check URL for room code to auto-join
  const urlParams = new URLSearchParams(window.location.search)
  const roomCode = urlParams.get('room')
  if (roomCode) {
    joinRoomId.value = roomCode
    // Wait for authentication, then auto-join
    const authUnwatch = watch(authenticated, (auth) => {
      if (auth) {
        setTimeout(() => {
          joinByCode(roomCode, playerName.value, effectiveWalletAddress.value, walletChain.value)
        }, 500)
        authUnwatch()
      }
    }, { immediate: true })
  }
})

onUnmounted(() => {
  stopCountdown()
  stopNextRoundCountdown()
  if (bonusDayInterval) clearInterval(bonusDayInterval)
  if (noticeTimer) clearTimeout(noticeTimer)
  // Cleanup listener
  if (handlePlayerNameUpdate) {
    window.removeEventListener('playerNameUpdated', handlePlayerNameUpdate)
  }
  if (handleAuthSessionExpired) {
    window.removeEventListener('authSessionExpired', handleAuthSessionExpired)
  }
  if (handleRoomClosed) {
    window.removeEventListener('roomClosed', handleRoomClosed)
  }
})

// Page navigation
watch(authenticated, (auth) => {
  if (auth && pendingLogin.value) {
    pendingLogin.value = false
    page.value = 'lobby'
    // Check bonus status after authentication
    setTimeout(() => checkBonusStatus(), 100)
  }
  if (!auth) {
    resetLowChipsBonusFlow()
    if (page.value !== 'login') page.value = 'login'
  }
})
watch(connected, (isConnected) => {
  if (!isConnected || authenticated.value || appMode !== 'pe') return
  const session = getStoredAuthSession()
  if (!session) return
  if (!effectiveWalletAddress.value || !walletChain.value) return
  if (session.wallet !== effectiveWalletAddress.value || session.chain !== walletChain.value) return
  pendingLogin.value = true
  login(effectiveWalletAddress.value, walletChain.value, playerName.value, '', '', session.token)
}, { immediate: true })
watch(roomState, (rs) => {
  if (rs) {
    if (page.value !== 'game') page.value = 'game'
    return
  }
  if (page.value === 'game' && authenticated.value) {
    page.value = 'lobby'
  }
})
watch(matchStatus, (ms) => { if (ms?.status === 'found') page.value = 'game' })

// Auto-scroll log
watch(gameEvents, async () => {
  await nextTick()
  if (logRef.value) logRef.value.scrollLeft = logRef.value.scrollWidth
}, { deep: true })

// Sound effects
watch(gameEvents, (evts) => {
  if (evts.length === 0) return
  const last = evts[evts.length - 1]
  switch (last.type) {
    case 'call': case 'raise': case 'allin': playSound('chip'); break
    case 'fold': playSound('fold'); break
    case 'compare': playSound('card'); break
  }
  if (['call', 'raise', 'allin', 'compare'].includes(last.type)) {
    spawnChipBurst(last)
  }
}, { deep: true })

watch(gameEnd, (ge) => {
  if (!ge) return
  stopActionBGM()
  stopCountdown()
  if (ge.winnerId === playerId.value) playSound('win')
  else playSound('lose')
  // Start next round confirmation countdown
  if (ge.nextRoundDeadline) startNextRoundCountdown(ge.nextRoundDeadline)
})

// Watch next round status updates
watch(nextRoundStatus, (nrs) => {
  if (nrs?.nextRoundDeadline) startNextRoundCountdown(nrs.nextRoundDeadline)
})

watch([authenticated, userChips], ([auth, chips]) => {
  refreshBonusDayIfNeeded()
  if (!auth) return

  if (chips >= LOW_CHIPS_THRESHOLD) {
    lowChipsStatusRequested = false
    lowChipsAutoClaimPending = false
    lowChipsNoBonusReminderKey = ''
    return
  }

  if (!lowChipsStatusRequested) {
    lowChipsStatusRequested = true
    checkBonusStatus()
  }
}, { immediate: true })

watch(showRechargeModal, (visible) => {
  if (!visible) return
  rechargeAmount.value = bonusInfo.value?.canClaim ? 500 : 100
})

watch(bonusInfo, (info) => {
  refreshBonusDayIfNeeded()
  if (!authenticated.value || !info || userChips.value >= LOW_CHIPS_THRESHOLD) return
  const inRoom = !!roomState.value || page.value === 'game'

  if (info.canClaim) {
    if (inRoom) {
      lowChipsAutoClaimPending = false
      return
    }
    if (!lowChipsAutoClaimPending) {
      lowChipsAutoClaimPending = true
      showNotice(t('notice.lowChipsAutoClaim'), 'info')
      claimBonus()
    }
    return
  }

  lowChipsAutoClaimPending = false
  const reminderKey = `${bonusDayKey}:${info.claimed}:${info.remaining}:${info.message || ''}`
  if (lowChipsNoBonusReminderKey === reminderKey) return
  lowChipsNoBonusReminderKey = reminderKey
  showNotice(info.message || t('notice.lowChipsExhausted'), 'info', 4200)
})

watch(userChips, (chips, prevChips) => {
  if (lowChipsAutoClaimPending && prevChips < LOW_CHIPS_THRESHOLD && chips >= LOW_CHIPS_THRESHOLD) {
    lowChipsAutoClaimPending = false
    lowChipsStatusRequested = false
    showNotice(t('notice.lowChipsClaimed'), 'success')
  }
})

// Is the current player confirmed for next round
const isConfirmedNextRound = computed(() => {
  const nrs = nextRoundStatus.value || gameEnd.value
  if (!nrs?.readyForNext) {
    // Check roomState
    if (roomState.value?.readyForNext) {
      return roomState.value.readyForNext.includes(playerId.value)
    }
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
const totalPlayerCount = computed(() => roomState.value?.players?.length || 0)
const canPlayAgainAfterResult = computed(() => {
  if (!myPlayer.value || myPlayer.value.chips <= 0 || isConfirmedNextRound.value) return false
  return totalPlayerCount.value >= 2
})

// Computed
const myPlayer = computed(() => {
  if (!roomState.value) return null
  return roomState.value.players.find(p => p.id === playerId.value)
})
const otherPlayers = computed(() => {
  if (!roomState.value) return []
  return roomState.value.players.filter(p => p.id !== playerId.value)
})
const isMyTurn = computed(() => {
  if (!roomState.value || !myPlayer.value) return false
  return roomState.value.players[roomState.value.turnIndex]?.id === playerId.value
})

// Countdown & action BGM on turn change
watch(isMyTurn, (myTurn) => {
  if (myTurn && roomState.value?.state === 1) {
    if (bgmOn.value) startActionBGM()
  } else {
    stopActionBGM()
  }
})

// Start countdown for ALL turns (visible to everyone)
watch(() => roomState.value?.turnDeadline, (td) => {
  if (td && roomState.value?.state === 1) startCountdown()
  else stopCountdown()
})

const isWinner = computed(() => gameEnd.value?.winnerId === playerId.value)
const finalResultPot = computed(() => {
  if (!gameEnd.value) return 0
  return Math.max(0, (gameEnd.value.pot || 0) - (gameEnd.value.rake || 0))
})
const handTypeFallbackMap = {
  豹子: 'hand.threeOfAKind',
  同花顺: 'hand.straightFlush',
  金花: 'hand.flush',
  顺子: 'hand.straight',
  对子: 'hand.pair',
  散牌: 'hand.highCard',
  未知: 'hand.unknown',
  ThreeOfAKind: 'hand.threeOfAKind',
  StraightFlush: 'hand.straightFlush',
  Flush: 'hand.flush',
  Straight: 'hand.straight',
  Pair: 'hand.pair',
  HighCard: 'hand.highCard',
  Single: 'hand.highCard',
  Unknown: 'hand.unknown',
}
const localizedGameEndHandType = computed(() => {
  if (!gameEnd.value) return ''
  const key = gameEnd.value.handTypeKey || handTypeFallbackMap[gameEnd.value.handType]
  return key ? t(key) : (gameEnd.value.handType || t('hand.unknown'))
})
const isRoomOwner = computed(() => roomState.value?.ownerId === playerId.value)
const canStartWaitingRoom = computed(() => {
  if (!roomState.value || roomState.value.state !== 0 || roomState.value.isSpectator) return false
  if (roomState.value.isPrivate) return isRoomOwner.value
  return true
})

// SVG countdown offsets for different ring sizes
const countdownOffsetOther = computed(() => {
  const circ = 2 * Math.PI * 23 // r=23, ~144.51
  return circ * (1 - countdown.value / 20)
})
const countdownOffsetMy = computed(() => {
  const circ = 2 * Math.PI * 21 // r=21, ~131.95
  return circ * (1 - countdown.value / 20)
})

const callAmount = computed(() => {
  if (!roomState.value) return 0
  const lastBet = roomState.value.lastBet || roomState.value.baseBet
  return myPlayer.value?.looked ? lastBet * 2 : lastBet
})
const allInCompareCost = computed(() => roomState.value?.lastAllInAmount || 0)
const raiseAmount = computed(() => {
  if (!roomState.value) return 0
  const lastBet = roomState.value.lastBet || roomState.value.baseBet
  const amount = lastBet * 2
  return myPlayer.value?.looked ? amount * 2 : amount
})
const canCompare = computed(() => comparablePlayers.value.length > 0)
const comparablePlayers = computed(() => {
  if (!roomState.value) return []
  if (roomState.value.hasAllIn) {
    const compareCost = roomState.value.lastAllInAmount || 0
    if (!myPlayer.value || compareCost <= 0 || myPlayer.value.chips < compareCost) return []
    return roomState.value.players.filter(p => p.id !== playerId.value && p.state === 4)
  }
  return roomState.value.players.filter(p => p.id !== playerId.value && (p.state === 2 || p.state === 4))
})
const recentEvents = computed(() => gameEvents.value.slice(-8))
const effectiveWalletConnected = computed(() => {
  if (appMode === 'pe') return realWalletConnected.value
  return walletConnected.value
})
const effectiveWalletAddress = computed(() => {
  if (appMode === 'pe') return realWalletAddress.value
  return walletAddress.value
})
const rulesLines = computed(() => {
    const content = t('login.rulesContent')
    // Replace escaped newlines with actual newlines, split, and filter empty lines
    return content
      .replace(/\\n/g, '\n')
      .split('\n')
      .map(l => l.trim())
      .filter(l => l.length > 0)
})
const loginLinks = computed(() => {
  const links = [
    { key: 'website', label: 'Website', href: normalizeExternalUrl(publicConfig.value.websiteUrl) },
    { key: 'telegram', label: 'Telegram', href: normalizeExternalUrl(publicConfig.value.telegramUrl) },
    { key: 'discord', label: 'Discord', href: normalizeExternalUrl(publicConfig.value.discordUrl) }
  ]
  return links.filter(link => !!link.href)
})
const rechargeOptions = computed(() => bonusInfo.value?.canClaim ? [500] : [100, 500, 1000, 5000])

const handTypeName = computed(() => {
  if (!myPlayer.value?.hand) return ''
  const h = myPlayer.value.hand
  const ranks = [
    { key: 'hand.threeOfAKind', check: h => h[0].value===h[1].value && h[1].value===h[2].value },
    { key: 'hand.straightFlush', check: h => isFlushFn(h) && isStraightFn(h) },
    { key: 'hand.flush', check: h => isFlushFn(h) },
    { key: 'hand.straight', check: h => isStraightFn(h) },
    { key: 'hand.pair', check: h => hasPair(h) },
    { key: 'hand.highCard', check: () => true }
  ]
  for (const r of ranks) if (r.check(h)) return t(r.key)
  return t('hand.highCard')
})

function isFlushFn(h) { return h[0].suit===h[1].suit && h[1].suit===h[2].suit }
function isStraightFn(h) {
  const v = h.map(c=>c.value).sort((a,b)=>b-a)
  if (v[0]===14 && v[1]===3 && v[2]===2) return true
  return v[0]-v[1]===1 && v[1]-v[2]===1
}
function hasPair(h) { return h[0].value===h[1].value || h[1].value===h[2].value || h[0].value===h[2].value }

function getApiUrl(path) {
  if (!runtimeApiBaseUrl) return path
  return `${runtimeApiBaseUrl}${path.startsWith('/') ? path : `/${path}`}`
}

async function loadPublicConfig() {
  try {
    const response = await fetch(getApiUrl('/api/config/public'))
    if (!response.ok) throw new Error('failed to load public config')
    const data = await response.json()
    publicConfig.value = {
      websiteUrl: data['social.websiteUrl'] || '',
      telegramUrl: data['social.telegramUrl'] || '',
      discordUrl: data['social.discordUrl'] || ''
    }
  } catch {
    publicConfig.value = {
      websiteUrl: '',
      telegramUrl: '',
      discordUrl: ''
    }
  }
}

function normalizeExternalUrl(raw) {
  if (!raw) return ''
  const value = raw.trim()
  if (!value) return ''
  if (/^(https?:\/\/|tg:\/\/|discord:\/\/)/i.test(value)) return value
  return `https://${value}`
}

// Actions
function handleWalletConnect() {
  playSound('click')
  if (appMode === 'te') {
    // TE mode: simulate wallet
    walletConnected.value = true
    walletAddress.value = '0x'+Array.from({length:40},()=>'0123456789abcdef'[Math.floor(Math.random()*16)]).join('')
    // Auto-generate nickname from wallet suffix
    playerName.value = 'player_' + walletAddress.value.slice(-6)
  }
  // PE mode: use the wallet connect modal
  if (appMode === 'pe') {
    showWalletModal.value = true
  }
}
async function handleLogin() {
  if (!playerName.value.trim()) playerName.value = t('common.player')+Math.floor(Math.random()*1000)
  playSound('click')

  // NEW: If not authenticated, send login request with playerName
  if (!authenticated.value) {
    pendingLogin.value = true
    try {
      const storedSession = getStoredAuthSession()
      if (appMode === 'pe' && storedSession && storedSession.wallet === effectiveWalletAddress.value && storedSession.chain === walletChain.value) {
        login(effectiveWalletAddress.value, walletChain.value, playerName.value, '', '', storedSession.token)
      } else if (appMode === 'pe' && (walletChain.value === 'evm' || walletChain.value === 'sol')) {
        const challenge = await requestAuthChallenge(effectiveWalletAddress.value, walletChain.value)
        const signature = await signAuthMessage(challenge.message, walletChain.value)
        login(effectiveWalletAddress.value, walletChain.value, playerName.value, challenge.nonce, signature)
      } else {
        login(effectiveWalletAddress.value, walletChain.value, playerName.value)
      }
    } catch (e) {
      pendingLogin.value = false
      const message = e?.message || t('notice.walletAuthFailed')
      walletError.value = message
      showNotice(message, 'info', 4200)
    }
    return
  }

  // After authentication, enter lobby
  pendingLogin.value = false
  page.value = 'lobby'
}
function handleQuickStart() { playSound('click'); createRoom(playerName.value,botCount.value,baseBet.value, effectiveWalletAddress.value, walletChain.value) }
function handleStartMatch() { playSound('click'); startMatch(playerName.value, effectiveWalletAddress.value, walletChain.value, baseBet.value) }
function handleCreatePrivateRoom() { playSound('click'); createRoom(playerName.value,0,baseBet.value, effectiveWalletAddress.value, walletChain.value) }
function handleJoinByCode() {
  if (joinRoomId.value.trim()) { playSound('click'); joinByCode(joinRoomId.value.trim(),playerName.value, effectiveWalletAddress.value, walletChain.value) }
}
function copyRoomCode() {
  if (roomState.value?.roomCode) {
    navigator.clipboard.writeText(roomState.value.roomCode).then(() => {
      error.value = t('toast.roomCodeCopied')
      setTimeout(() => { error.value = '' }, 2000)
    }).catch(() => {
      error.value = t('toast.copyFailed')+': '+roomState.value.roomCode
      setTimeout(() => { error.value = '' }, 3000)
    })
  }
}
function copyShareLink() {
  if (roomState.value?.roomCode) {
    const url = new URL(window.location.href.split('?')[0])
    url.searchParams.set('room', roomState.value.roomCode)
    navigator.clipboard.writeText(url.toString()).then(() => {
      error.value = t('toast.linkCopied')
      setTimeout(() => { error.value = '' }, 2000)
    }).catch(() => {
      error.value = t('toast.copyFailed')
      setTimeout(() => { error.value = '' }, 3000)
    })
  }
}
// Blockchain hash identicon (Item 9)
function identiconSvg(id) {
    // Generate a deterministic hash from the ID
    let hash = 0
    for (let i = 0; i < id.length; i++) {
        hash = ((hash << 5) - hash + id.charCodeAt(i)) | 0
    }
    let hash2 = hash
    for (let i = 0; i < id.length; i++) {
        hash2 = ((hash2 << 7) - hash2 + id.charCodeAt(i)) | 0
    }

    const hue1 = Math.abs(hash) % 360
    const hue2 = (hue1 + 120 + Math.abs(hash2) % 120) % 360
    const bg = `hsl(${hue1},65%,25%)`
    const fg = `hsl(${hue2},70%,55%)`

    let rects = ''
    for (let y = 0; y < 5; y++) {
        for (let x = 0; x < 3; x++) {
            const bit = (Math.abs(hash >> (y * 3 + x + 1)) & 1)
            if (bit) {
                // Left side
                rects += `<rect x="${x+1}" y="${y}" width="1" height="1" fill="${fg}" opacity="0.9"/>`
                // Mirror right side (symmetric)
                rects += `<rect x="${5-x}" y="${y}" width="1" height="1" fill="${fg}" opacity="0.9"/>`
            }
        }
        // Center column
        const centerBit = (Math.abs(hash2 >> (y + 3)) & 1)
        if (centerBit) {
            rects += `<rect x="3" y="${y}" width="1" height="1" fill="${fg}" opacity="0.85"/>`
        }
    }

    return `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 7 5"><rect width="7" height="5" fill="${bg}"/>${rects}</svg>`)}`
}
function handleRecharge() {
  playSound('chip')
  recharge(rechargeAmount.value, effectiveWalletAddress.value)
  showRechargeModal.value = false
}
function handleLeaveRoom() {
  stopCountdown()
  stopNextRoundCountdown()
  wsLeaveRoom()
  roomState.value = null; gameEnd.value = null; gameEvents.value = []
  nextRoundStatus.value = null
  page.value = 'lobby'
}
function startGame() { playSound('card'); wsStartGame() }
function handleUpdateBaseBet(baseBetValue) { playSound('click'); wsUpdateBaseBet(baseBetValue) }
function lookCards() { playSound('card'); wsDoAction(6) }
function doFold() { stopCountdown(); playSound('fold'); wsDoAction(3) }
function doCall() { stopCountdown(); playSound('chip'); wsDoAction(1) }
function doRaise() {
  stopCountdown(); playSound('chip')
  const lastBet = roomState.value.lastBet || roomState.value.baseBet
  const amount = lastBet * 2
  const mult = myPlayer.value?.looked ? 2 : 1
  wsDoAction(2, amount * mult)
}
function doAllIn() { stopCountdown(); playSound('chip'); wsDoAction(4) }
function doCompare(targetId) { showCompareModal.value=false; stopCountdown(); playSound('card'); wsDoAction(5,0,targetId) }
function handleNewRound() { playSound('card'); wsNewRound() }
function dismissResult() {}
function toggleBGMFn() { return toggleBGM() }

function isTurn(p) {
  if (!roomState.value) return false
  return roomState.value.players[roomState.value.turnIndex]?.id === p.id
}
function suitSymbol(s) { return ['','♦','♣','♥','♠'][s]||'?' }
function suitClass(s) { return ['','diamond','club','heart','spade'][s]||'' }
function valueName(v) { return v===14?'A':v===13?'K':v===12?'Q':v===11?'J':String(v) }

const avatarColors = ['#e74c3c','#3498db','#2ecc71','#f39c12','#9b59b6','#1abc9c']
function avatarColor(seat) { return avatarColors[seat%avatarColors.length] }

function particleStyle(i) {
  const x=Math.random()*100,y=Math.random()*100
  const size=2+Math.random()*4,delay=Math.random()*5
  return {left:x+'%',top:y+'%',width:size+'px',height:size+'px',animationDelay:delay+'s'}
}

function getStoredAuthSession() {
  try {
    const raw = window.localStorage?.getItem('mule_auth_session')
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (!parsed?.token || !parsed?.wallet || !parsed?.chain || !parsed?.expiresAt) return null
    if (Date.now() >= parsed.expiresAt) {
      window.localStorage?.removeItem('mule_auth_session')
      window.dispatchEvent(new CustomEvent('authSessionExpired', {
        detail: { message: t('notice.authExpired') }
      }))
      return null
    }
    return parsed
  } catch (_) {
    window.localStorage?.removeItem('mule_auth_session')
    return null
  }
}

function getPlayerTablePosition(playerIdForEvent) {
  if (!playerIdForEvent || playerIdForEvent === playerId.value) {
    return { x: 50, y: 86 }
  }
  const players = otherPlayers.value || []
  const idx = players.findIndex(p => p.id === playerIdForEvent)
  if (idx === -1) return { x: 50, y: 50 }
  const angleStart = -180
  const angleRange = 180
  const angle = angleStart + (angleRange / (players.length + 1)) * (idx + 1)
  const rad = angle * Math.PI / 180
  const rx = 42
  const ry = 38
  return {
    x: 50 + rx * Math.cos(rad),
    y: 50 + ry * Math.sin(rad)
  }
}

function spawnChipBurst(event) {
  if (!roomState.value || page.value !== 'game') return
  const origin = getPlayerTablePosition(event.playerId)
  const amount = Number(event.amount) || 0
  const count = Math.max(2, Math.min(5, Math.ceil(amount / 20)))
  const id = ++chipBurstId
  chipBursts.value.push({
    id,
    count,
    style: {
      left: `${origin.x}%`,
      top: `${origin.y}%`,
      '--chip-dx': `${50 - origin.x}%`,
      '--chip-dy': `${50 - origin.y}%`
    }
  })
  window.setTimeout(() => {
    chipBursts.value = chipBursts.value.filter(burst => burst.id !== id)
  }, 900)
}

function fanCardStyle(index, total) {
  const center = (total - 1) / 2
  const offset = index - center
  return {
    '--fan-rotate': `${offset * 7}deg`,
    '--fan-shift': `${offset * 6}px`,
    '--fan-lift': `${Math.abs(offset) * 2}px`
  }
}

function myHandCardStyle(index) {
  const center = 1
  const offset = index - center
  return {
    animationDelay: `${(index + 1) * 0.1}s`,
    '--card-deal-delay': `${(index + 1) * 0.1}s`,
    '--hand-rotate': `${offset * 8}deg`,
    '--hand-shift': `${offset * 10}px`,
    '--hand-lift': `${Math.abs(offset) * 3}px`
  }
}

function getPlayerPosition(idx,total) {
  const angleStart=-180,angleRange=180
  const angle=angleStart+(angleRange/(total+1))*(idx+1)
  const rad=angle*Math.PI/180
  const rx=42,ry=38
  const x=50+rx*Math.cos(rad),y=50+ry*Math.sin(rad)
  return {left:x+'%',top:y+'%',transform:'translate(-50%,-50%)'}
}

function formatEvent(e) {
  const n = e.playerName||'?'
  switch(e.type) {
    case 'call': return `${n} ${t('action.call')} ${e.amount}`
    case 'raise': return `${n} ${t('action.raise')} ${e.amount}`
    case 'fold': return `${n} ${t('action.fold')}`
    case 'allin': return `${n} ${t('action.allIn')} ${e.amount}`
    case 'look': return `${n} ${t('game.look')}`
    case 'compare': return `${n} vs ${e.targetName} → ${e.winnerId===e.playerId?n:e.targetName} ${t('result.wins')}`
    default: return `${n}: ${e.type}`
  }
}
</script>

<style>
:root {
  --bg-dark: #0a0e1a;
  --bg-card: rgba(255,255,255,0.04);
  --felt: #0d6b3d;
  --felt-dark: #084a2a;
  --gold: #ffd700;
  --gold-dim: #b8960f;
  --red: #ff4757;
  --text: #eef0f4;
  --text-dim: #7a8599;
  --accent: #4facfe;
}
* { margin:0; padding:0; box-sizing:border-box; }
html, body {
  font-family: 'SF Pro Display',-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
  background: var(--bg-dark); color: var(--text);
  overflow: hidden;
  height: 100%; width: 100%;
  -webkit-user-select: none; user-select: none;
  -webkit-tap-highlight-color: transparent;
  -webkit-text-size-adjust: 100%;
}
.app {
  height: 100vh; height: 100dvh;
  width: 100vw;
  display: flex; flex-direction: column;
  overflow: hidden;
  position: fixed; inset: 0;
}
.page { flex:1; display:flex; flex-direction:column; overflow:hidden; min-height:0; }

/* Toast */
.toast { position:fixed; top:max(12px,env(safe-area-inset-top)); left:50%; transform:translateX(-50%); background:var(--red); color:#fff; padding:6px 18px; border-radius:20px; z-index:200; font-size:12px; animation:slideDown .3s ease; white-space:nowrap; }
.toast.info { background:rgba(79, 172, 254, 0.96); color:#08111f; }
.toast.success { background:rgba(46, 213, 115, 0.96); color:#04140b; }
@keyframes slideDown { from{opacity:0;transform:translateX(-50%) translateY(-20px);} }

/* ===== Lang picker (replaces <select>) ===== */
.lang-picker { display:flex; gap:4px; justify-content:center; flex-wrap:wrap; }
.lang-btn { padding:5px 12px; border-radius:8px; border:1px solid rgba(255,255,255,0.1); background:transparent; color:var(--text-dim); font-size:12px; cursor:pointer; transition:all .15s; }
.lang-btn.active { background:rgba(255,215,0,0.15); color:var(--gold); border-color:var(--gold); font-weight:600; }
.lang-picker-inline { display:flex; gap:2px; }
.lang-btn-sm { padding:2px 6px; border-radius:4px; border:1px solid rgba(255,255,255,0.08); background:transparent; color:var(--text-dim); font-size:10px; cursor:pointer; transition:all .15s; }
.lang-btn-sm.active { background:rgba(255,215,0,0.15); color:var(--gold); border-color:rgba(255,215,0,0.3); }

/* ===== LOGIN ===== */
.login-page { align-items:center; justify-content:center; background:linear-gradient(135deg,#0a0e1a 0%,#1a1a3e 50%,#0a0e1a 100%); position:relative; padding:20px; overflow-y:auto; }
.login-bg { position:absolute; inset:0; overflow:hidden; pointer-events:none; }
.particle { position:absolute; border-radius:50%; background:rgba(255,215,0,0.3); animation:float 6s ease-in-out infinite; }
@keyframes float { 0%,100%{transform:translateY(0) scale(1);opacity:.3;} 50%{transform:translateY(-30px) scale(1.5);opacity:.6;} }
.login-card { position:relative; background:rgba(15,20,40,0.88); border:1px solid rgba(255,255,255,0.08); border-radius:20px; padding:32px 24px; max-width:360px; width:100%; backdrop-filter:blur(40px); z-index:1; }
.logo { text-align:center; margin-bottom:24px; }
.logo-icon { display:inline-flex; align-items:center; justify-content:center; width:60px; height:60px; border-radius:16px; background:linear-gradient(135deg,#ffd700,#ff8c00); margin-bottom:12px; position:relative; }
.logo-a { font-size:28px; font-weight:800; color:#1a0a00; }
.logo-suit { position:absolute; top:4px; right:6px; font-size:14px; color:rgba(0,0,0,0.4); }
.game-title { font-size:24px; letter-spacing:6px; color:var(--gold); text-shadow:0 0 20px rgba(255,215,0,0.3); }
.game-subtitle { font-size:11px; color:var(--text-dim); margin-top:4px; letter-spacing:2px; }
.login-form { display:flex; flex-direction:column; gap:10px; }
.login-links { display:flex; justify-content:center; gap:14px; margin:18px 0 10px; flex-wrap:wrap; }
.login-link { display:inline-flex; align-items:center; justify-content:center; width:48px; height:48px; border-radius:50%; background:linear-gradient(180deg,rgba(255,255,255,0.12),rgba(255,255,255,0.05)); border:1px solid rgba(255,255,255,0.14); color:#f6e7c8; text-decoration:none; box-shadow:0 10px 24px rgba(0,0,0,0.18); transition:transform .2s ease, box-shadow .2s ease, border-color .2s ease; }
.login-link:hover { transform:translateY(-2px) scale(1.04); border-color:rgba(255,215,0,0.4); box-shadow:0 14px 28px rgba(0,0,0,0.24); }
.login-link-svg { width:22px; height:22px; display:block; }
.mule-icon { width:20px; height:20px; }
.telegram-icon { color:#7fd6ff; }
.discord-icon { color:#c7b9ff; }
.input-group { display:flex; flex-direction:column; gap:4px; }
.input-group label { font-size:11px; color:var(--text-dim); }
.input-group input { padding:10px 14px; border-radius:10px; border:1px solid rgba(255,255,255,0.1); background:rgba(255,255,255,0.06); color:var(--text); font-size:15px; outline:none; transition:border .2s; }
.input-group input:focus { border-color:var(--gold); }
.btn-web3 { display:flex; align-items:center; justify-content:center; gap:8px; padding:10px; border-radius:10px; border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.06); color:var(--text); font-size:13px; cursor:pointer; }
.wallet-icon { font-size:18px; }
.btn-enter { padding:12px; border:none; border-radius:10px; background:linear-gradient(135deg,#ffd700,#ff8c00); color:#1a0a00; font-size:16px; font-weight:700; cursor:pointer; transition:transform .1s; }
.btn-enter:active { transform:scale(.97); }
.btn-enter:disabled { opacity:.5; cursor:default; }
.login-lang { margin-top:16px; }
.login-footer { text-align:center; margin-top:14px; font-size:10px; color:var(--text-dim); opacity:.5; }

/* ===== LOBBY ===== */
.lobby-page { background:var(--bg-dark); }
.lobby-header { display:flex; justify-content:space-between; align-items:center; padding:10px 16px; padding-top:max(10px,env(safe-area-inset-top)); background:rgba(255,255,255,0.03); border-bottom:1px solid rgba(255,255,255,0.06); flex-shrink:0; gap:8px; }
.user-info { display:flex; align-items:center; gap:10px; min-width:0; }
.user-avatar { width:36px; height:36px; border-radius:10px; display:flex; align-items:center; justify-content:center; color:#fff; font-weight:700; font-size:16px; flex-shrink:0; }
.user-meta { min-width:0; }
.user-name { font-weight:600; font-size:14px; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.user-chips { font-size:12px; color:var(--gold); display:flex; align-items:center; gap:3px; }
.chip-icon { font-size:13px; }
.btn-recharge-sm { width:20px; height:20px; border-radius:5px; border:1px solid var(--gold); background:transparent; color:var(--gold); font-size:13px; cursor:pointer; display:flex; align-items:center; justify-content:center; margin-left:4px; flex-shrink:0; }
.header-actions { display:flex; align-items:center; gap:6px; flex-shrink:0; }
.btn-icon { background:none; border:none; font-size:18px; cursor:pointer; padding:4px; }

.lobby-body { flex:1; padding:16px; overflow-y:auto; -webkit-overflow-scrolling:touch; }
.mode-cards { display:flex; flex-direction:column; gap:12px; max-width:480px; margin:0 auto; }
.mode-card { background:rgba(255,255,255,0.04); border:1px solid rgba(255,255,255,0.08); border-radius:14px; padding:16px; cursor:pointer; transition:all .15s; }
.mode-card:active { transform:scale(.98); }
.mode-icon { font-size:24px; margin-bottom:6px; }
.mode-card h3 { font-size:15px; margin-bottom:3px; }
.mode-card p { font-size:12px; color:var(--text-dim); margin-bottom:10px; }
.mode-detail { display:flex; gap:6px; align-items:center; flex-wrap:wrap; margin-top:6px; }
.mode-detail label { font-size:11px; color:var(--text-dim); flex-shrink:0; }
.opt-row { display:flex; gap:4px; }
.opt-btn { width:34px; height:34px; border-radius:7px; border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.05); color:var(--text); font-size:13px; cursor:pointer; transition:all .1s; }
.opt-btn.active { background:var(--gold); color:#000; border-color:var(--gold); font-weight:700; }
.matching-indicator { display:flex; align-items:center; gap:8px; margin-top:6px; }
.match-spinner { width:14px; height:14px; border:2px solid var(--gold); border-top-color:transparent; border-radius:50%; animation:spin 1s linear infinite; }
@keyframes spin { to{transform:rotate(360deg);} }
.btn-cancel-match { padding:3px 8px; border-radius:5px; border:1px solid var(--red); background:transparent; color:var(--red); font-size:11px; cursor:pointer; }
.btn-create-room { width:100%; padding:8px 14px; border:none; border-radius:8px; background:linear-gradient(135deg,var(--gold),#ff8c00); color:#000; font-size:13px; font-weight:700; cursor:pointer; margin-bottom:4px; }
.join-row { display:flex; gap:6px; }
.room-input { flex:1; padding:7px 10px; border-radius:7px; border:1px solid rgba(255,255,255,0.1); background:rgba(255,255,255,0.06); color:var(--text); font-size:13px; outline:none; min-width:0; }
.btn-join { padding:7px 14px; border-radius:7px; border:none; background:var(--accent); color:#fff; font-size:12px; font-weight:600; cursor:pointer; flex-shrink:0; }

/* ===== MODALS ===== */
.modal-overlay { position:fixed; inset:0; background:rgba(0,0,0,0.6); display:flex; align-items:center; justify-content:center; z-index:100; padding:16px; backdrop-filter:blur(4px); }
.modal { background:#151c2e; border:1px solid rgba(255,255,255,0.1); border-radius:16px; padding:20px; max-width:340px; width:100%; max-height:90vh; overflow-y:auto; }
.modal h3 { text-align:center; margin-bottom:6px; font-size:16px; }
.modal-desc { text-align:center; color:var(--text-dim); font-size:12px; margin-bottom:12px; }
.recharge-options { display:grid; grid-template-columns:1fr 1fr; gap:8px; margin-bottom:12px; }
.recharge-btn { padding:14px 10px; border-radius:10px; border:1px solid rgba(255,255,255,0.1); background:rgba(255,255,255,0.04); cursor:pointer; text-align:center; transition:all .15s; color:var(--text); }
.recharge-btn.active { border-color:var(--gold); background:rgba(255,215,0,0.1); }
.recharge-value { display:block; font-size:20px; font-weight:700; color:var(--gold); }
.recharge-label { font-size:10px; color:var(--text-dim); }
.recharge-wallet { font-size:11px; color:var(--text-dim); margin-bottom:10px; text-align:center; }
.btn-confirm-recharge { width:100%; padding:10px; border:none; border-radius:8px; background:linear-gradient(135deg,var(--gold),#ff8c00); color:#000; font-weight:700; font-size:14px; cursor:pointer; margin-bottom:6px; }
.btn-close { width:100%; padding:8px; border:1px solid rgba(255,255,255,0.1); border-radius:8px; background:transparent; color:var(--text-dim); font-size:13px; cursor:pointer; }

/* Rules modal */
.btn-rules { width:100%; padding:8px; border:1px solid rgba(255,255,255,0.12); border-radius:10px; background:transparent; color:var(--text-dim); font-size:13px; cursor:pointer; transition:border-color .2s; }
.btn-rules:hover { border-color:var(--gold); color:var(--gold); }
.rules-modal { max-height:80vh; }
.rules-content { text-align:left; font-size:13px; color:var(--text); line-height:1.8; margin-bottom:14px; max-height:60vh; overflow-y:auto; padding:10px 0; }
.rules-content p { margin:10px 0; padding:0 8px; }
.rules-content p:empty { display:none; }
/* Wallet modal */
.wallet-modal { max-height:85vh; }
.wallet-options { display:flex; flex-direction:column; gap:8px; margin-bottom:12px; }
.wallet-opt { display:flex; align-items:center; gap:10px; padding:12px 16px; border-radius:10px; border:1px solid rgba(255,255,255,0.1); background:rgba(255,255,255,0.04); color:var(--text); cursor:pointer; transition:all .15s; font-size:14px; }
.wallet-opt:hover { border-color:var(--gold); background:rgba(255,215,0,0.08); }
.wallet-opt:disabled { opacity:.5; cursor:default; }
.wallet-opt-icon { font-size:24px; }
.wallet-opt-name { flex:1; font-weight:600; }
.wallet-opt-chain { font-size:11px; color:var(--text-dim); padding:2px 8px; background:rgba(255,255,255,0.06); border-radius:4px; }
.wallet-error { color:var(--red); font-size:12px; text-align:center; margin-bottom:8px; padding:6px; background:rgba(255,71,87,0.1); border-radius:6px; }

/* Share button */
.btn-share { background:none; border:none; font-size:12px; cursor:pointer; padding:0 4px; opacity:.7; }
.btn-share:active { opacity:1; }

/* Next round confirmation */
.next-round-status { margin-bottom:10px; }
.confirm-progress { display:flex; justify-content:space-between; align-items:center; font-size:12px; color:var(--text-dim); padding:6px 10px; background:rgba(255,255,255,0.04); border-radius:8px; margin-bottom:6px; }
.confirm-timer { color:var(--red); font-weight:700; font-size:14px; }
.confirmed-badge { text-align:center; color:var(--accent); font-size:13px; font-weight:600; padding:4px; }
.btn-next-round.recharge { background:linear-gradient(135deg,var(--accent),#3498db); }
.result-row .dim { color:var(--text-dim); }
.compare-targets { display:flex; flex-direction:column; gap:6px; margin-bottom:10px; }
.compare-item { display:flex; align-items:center; gap:8px; padding:8px 12px; border-radius:8px; border:1px solid rgba(255,255,255,0.08); background:rgba(255,255,255,0.04); color:var(--text); cursor:pointer; font-size:13px; }
.ci-avatar { width:28px; height:28px; border-radius:7px; display:flex; align-items:center; justify-content:center; color:#fff; font-weight:700; font-size:13px; }
.ci-chips { margin-left:auto; font-size:11px; color:var(--gold); }

/* ===== GAME PAGE ===== */
.game-page {
  background:
    radial-gradient(circle at top, rgba(255,196,76,0.10), transparent 30%),
    radial-gradient(circle at 20% 30%, rgba(64,153,255,0.12), transparent 28%),
    linear-gradient(180deg, #07111d 0%, #08131e 42%, #050b14 100%);
  position: relative;
}
.game-ambience {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}
.ambience-glow {
  position: absolute;
  top: 10%;
  width: 32vw;
  height: 32vw;
  border-radius: 50%;
  filter: blur(60px);
  opacity: .35;
}
.ambience-glow.left {
  left: -8vw;
  background: radial-gradient(circle, rgba(79,172,254,0.6), transparent 70%);
}
.ambience-glow.right {
  right: -8vw;
  background: radial-gradient(circle, rgba(255,184,0,0.5), transparent 70%);
}
.ambience-mesh {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255,255,255,0.025) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255,255,255,0.025) 1px, transparent 1px);
  background-size: 34px 34px;
  mask-image: linear-gradient(180deg, rgba(0,0,0,0.55), transparent 70%);
}
.game-topbar {
  display:flex; align-items:center; gap:8px;
  padding:8px 12px;
  padding-top:max(8px,env(safe-area-inset-top));
  padding-left:max(12px,env(safe-area-inset-left));
  padding-right:max(12px,env(safe-area-inset-right));
  background:rgba(8,14,24,0.72);
  border-bottom:1px solid rgba(255,255,255,0.06);
  backdrop-filter: blur(18px);
  box-shadow: 0 12px 30px rgba(0,0,0,0.18);
  flex-shrink:0; min-height:0;
  position: relative;
  z-index: 2;
}
.btn-back {
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.08);
  color: var(--text);
  font-size: 12px;
  cursor: pointer;
  padding: 7px 10px;
  border-radius: 999px;
  white-space:nowrap;
}
.room-code {
  font-size:10px;
  color:var(--gold);
  cursor:pointer;
  padding:6px 10px;
  background:rgba(255,215,0,0.08);
  border-radius:999px;
  border:1px solid rgba(255,215,0,0.2);
  white-space:nowrap;
  display:flex;
  align-items:center;
  gap:3px;
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.06);
}
.topbar-center {
  display:flex;
  align-items:center;
  gap:6px;
  background: linear-gradient(180deg, rgba(255,215,0,0.10), rgba(255,215,0,0.04));
  border: 1px solid rgba(255,215,0,0.18);
  border-radius: 999px;
  padding: 7px 12px;
}
.pot-label { font-size:10px; color:var(--text-dim); text-transform: uppercase; letter-spacing: .8px; }
.pot-value { font-size:18px; font-weight:800; color:var(--gold); text-shadow:0 0 12px rgba(255,215,0,0.3); }
.topbar-meta {
  font-size:10px;
  color:var(--text-dim);
  display:flex;
  gap:8px;
  margin-left:auto;
  white-space:nowrap;
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 999px;
  padding: 6px 10px;
}
.topbar-chips {
  font-size:11px;
  color:var(--gold);
  display:flex;
  align-items:center;
  gap:4px;
  white-space:nowrap;
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 999px;
  padding: 6px 10px;
}
.topbar-actions { display:flex; align-items:center; gap:4px; flex-shrink:0; }

/* ===== TABLE ===== */
.table-container {
  flex:1;
  display:flex;
  align-items:center;
  justify-content:center;
  padding:16px 8px 10px;
  min-height:0;
  overflow:hidden;
  position: relative;
  z-index: 1;
}
.table-felt {
  width:min(100%,540px); aspect-ratio:5/4;
  max-height:100%;
  border-radius:50%;
  background:
    radial-gradient(circle at 50% 46%, rgba(255,255,255,0.08), transparent 18%),
    radial-gradient(ellipse at 50% 40%, #2aa56b 0%, #157a48 28%, var(--felt) 58%, #093923 100%);
  border:6px solid #4d3620;
  box-shadow:
    0 0 0 2px #1a1208,
    0 0 0 9px rgba(83,54,24,0.92),
    0 30px 60px rgba(0,0,0,0.45),
    inset 0 0 50px rgba(0,0,0,0.22),
    inset 0 -14px 38px rgba(0,0,0,0.18);
  position:relative; overflow:visible;
}
.table-rail {
  position: absolute;
  inset: -18px;
  border-radius: 50%;
  background:
    radial-gradient(circle at 50% 0%, rgba(255,214,127,0.20), transparent 28%),
    linear-gradient(135deg, #6a4b29, #3b2816 55%, #1e150d);
  z-index: -2;
  box-shadow: inset 0 2px 6px rgba(255,255,255,0.08), 0 18px 40px rgba(0,0,0,0.45);
}
.table-chip-bursts {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 3;
}
.chip-burst {
  position: absolute;
  width: 0;
  height: 0;
}
.burst-chip {
  position: absolute;
  width: 18px;
  height: 18px;
  margin-left: -9px;
  margin-top: -9px;
  border-radius: 50%;
  background:
    radial-gradient(circle at 35% 32%, rgba(255,255,255,0.45), transparent 28%),
    linear-gradient(135deg, #ffe48a, #d8a214 70%, #8f6400);
  border: 2px solid rgba(132, 92, 0, 0.78);
  box-shadow: 0 6px 14px rgba(0,0,0,0.22);
  animation: chipFlyToPot .82s cubic-bezier(.18,.7,.18,1) forwards;
}
.burst-chip:nth-child(2n) { width: 16px; height: 16px; margin-left: -8px; margin-top: -8px; }
.burst-chip:nth-child(3n) { width: 14px; height: 14px; margin-left: -7px; margin-top: -7px; }
.table-inner-ring {
  position: absolute;
  inset: 26px;
  border-radius: 50%;
  border: 1px dashed rgba(241, 215, 136, 0.18);
  box-shadow: inset 0 0 0 1px rgba(255,255,255,0.03);
}
.table-center-badge {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -58%);
  width: 118px;
  height: 118px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at top, rgba(255,255,255,0.10), transparent 46%),
    linear-gradient(180deg, rgba(8,31,23,0.78), rgba(5,16,12,0.94));
  border: 1px solid rgba(255,215,0,0.18);
  box-shadow: 0 10px 40px rgba(0,0,0,0.26), inset 0 0 0 8px rgba(255,255,255,0.03);
  z-index: 1;
  animation: tableBadgePulse 4s ease-in-out infinite;
}
.table-badge-label {
  font-size: 10px;
  letter-spacing: 1.1px;
  text-transform: uppercase;
  color: rgba(255,255,255,0.55);
}
.table-badge-value {
  font-size: 24px;
  line-height: 1.05;
  color: var(--gold);
  text-shadow: 0 0 16px rgba(255,215,0,0.22);
}
.table-badge-meta {
  margin-top: 3px;
  font-size: 10px;
  color: rgba(255,255,255,0.55);
}
.table-inner { position:absolute; inset:0; }
.table-logo {
  position:absolute;
  top:50%;
  left:50%;
  transform:translate(-50%,48px);
  font-size:13px;
  letter-spacing:6px;
  color:rgba(255,255,255,0.06);
  font-weight:800;
  pointer-events:none;
  text-transform:uppercase;
}
.table-pot-center { position:absolute; top:50%; left:50%; transform:translate(-50%,18px); }
.pot-chips-stack { position:relative; display:flex; flex-direction:column; align-items:center; }
.chip-stack {
  width:28px;
  height:28px;
  border-radius:50%;
  background:
    radial-gradient(circle at 35% 32%, rgba(255,255,255,0.4), transparent 28%),
    linear-gradient(135deg, #ffe690, #d6a617 65%, #8f6500);
  border:2px solid #a67c00;
  position:absolute;
  box-shadow:0 4px 10px rgba(0,0,0,0.42);
  animation: chipFloat 2.8s ease-in-out infinite;
}
.pot-amount {
  position:relative;
  top:34px;
  font-size:12px;
  font-weight:800;
  color:rgba(255,255,255,0.78);
  text-shadow:0 1px 6px rgba(0,0,0,0.9);
  white-space:nowrap;
  animation: potAmountPulse 2.8s ease-in-out infinite;
}

/* Table players */
.table-player {
  position:absolute;
  display:flex;
  flex-direction:column;
  align-items:center;
  gap:2px;
  z-index:10;
  transition:opacity .3s, transform .2s ease;
  min-width: 76px;
}
.table-player.is-folded,.table-player.is-out { opacity:.35; }
.table-player.is-turn { transform: translate(-50%,-52%); }
.table-player.is-turn .tp-plaque {
  background:
    radial-gradient(circle at top, rgba(255,215,0,0.16), transparent 40%),
    linear-gradient(180deg, rgba(20,29,16,0.82), rgba(9,23,37,0.42));
  border-color: rgba(255,215,0,0.20);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.07),
    0 0 24px rgba(255,215,0,0.14);
  animation: turnPlaquePulse 1.8s ease-in-out infinite;
}

.tp-plaque {
  position: absolute;
  top: 18px;
  width: 88px;
  height: 104px;
  border-radius: 26px;
  background: linear-gradient(180deg, rgba(6,14,24,0.72), rgba(9,23,37,0.36));
  border: 1px solid rgba(255,255,255,0.05);
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.05);
  z-index: -1;
}

.tp-avatar-wrap {
  position:relative;
  display:flex;
  align-items:center;
  justify-content:center;
  width: 58px;
  height: 58px;
  border-radius: 20px;
  background: linear-gradient(180deg, rgba(5,11,19,0.88), rgba(12,25,40,0.70));
  border: 1px solid rgba(255,255,255,0.08);
  box-shadow: 0 8px 20px rgba(0,0,0,0.26);
}
.tp-avatar { width:44px; height:44px; border-radius:50%; display:flex; align-items:center; justify-content:center; color:#fff; font-weight:700; font-size:17px; position:relative; box-shadow:0 2px 10px rgba(0,0,0,0.5); border:2px solid rgba(255,255,255,0.15); }
.table-player.is-turn .tp-avatar-wrap { border-color: rgba(255,215,0,0.28); box-shadow: 0 0 0 1px rgba(255,215,0,0.14), 0 10px 24px rgba(0,0,0,0.3); }
.table-player.is-turn .tp-avatar { border-color:var(--gold); box-shadow:0 0 16px rgba(255,215,0,0.34); }
.table-player.is-turn .tp-avatar::after {
  content:'';
  position:absolute;
  inset:-7px;
  border-radius:50%;
  border:1px solid rgba(255,215,0,0.30);
  animation: turnHalo 1.8s ease-out infinite;
}
.tp-bot { position:absolute; bottom:-2px; right:-4px; background:rgba(0,0,0,0.8); font-size:7px; padding:1px 4px; border-radius:3px; color:var(--accent); font-weight:600; }

/* Opponent countdown ring */
.tp-countdown-ring { position:absolute; inset:-4px; width:52px; height:52px; transform:rotate(-90deg); pointer-events:none; }
.tp-cd-bg { fill:none; stroke:rgba(255,255,255,0.1); stroke-width:2.5; }
.tp-cd-fg { fill:none; stroke:var(--accent); stroke-width:3; stroke-dasharray:144.51; stroke-linecap:round; transition:stroke-dashoffset .3s linear; }
.tp-cd-fg.urgent { stroke:var(--red); }
.tp-cd-num { position:absolute; bottom:-14px; font-size:10px; font-weight:700; color:var(--accent); background:rgba(0,0,0,0.7); padding:0 5px; border-radius:4px; white-space:nowrap; }
.tp-cd-num.urgent { color:var(--red); animation:countdownPulse .5s ease-in-out infinite; }

.tp-name { font-size:10px; font-weight:700; white-space:nowrap; text-shadow:0 1px 3px rgba(0,0,0,0.8); margin-top:3px; }
.tp-chips {
  font-size:9px;
  color:var(--gold);
  white-space:nowrap;
  text-shadow:0 1px 3px rgba(0,0,0,0.8);
  display:flex;
  align-items:center;
  gap:1px;
  padding: 2px 7px;
  border-radius: 999px;
  background: rgba(0,0,0,0.28);
  border: 1px solid rgba(255,255,255,0.06);
}
.tp-chip-icon { font-size:9px; }
.tp-bet { position:absolute; bottom:-22px; }
.tp-bet span {
  background:linear-gradient(180deg, rgba(255,215,0,0.18), rgba(255,215,0,0.08));
  color:var(--gold);
  font-size:9px;
  padding:2px 7px;
  border-radius:999px;
  border:1px solid rgba(255,215,0,0.22);
  white-space:nowrap;
  font-weight:700;
}
.tp-cards {
  display:flex;
  gap:0;
  margin-top:4px;
  min-height: 30px;
  justify-content: center;
}
.fan-card {
  transform: translateX(var(--fan-shift, 0)) translateY(var(--fan-lift, 0)) rotate(var(--fan-rotate, 0));
  transform-origin: bottom center;
  transition: transform .25s ease, box-shadow .25s ease;
}
.mini-card { width:20px; height:28px; border-radius:3px; background:#fff; display:flex; align-items:center; justify-content:center; font-size:8px; font-weight:700; color:#333; box-shadow:0 1px 2px rgba(0,0,0,0.3); }
.mini-card.heart,.mini-card.diamond { color:var(--red); }
.mini-card.back { background:linear-gradient(135deg,#1a3a5c 25%,#0d2840 25%,#0d2840 50%,#1a3a5c 50%,#1a3a5c 75%,#0d2840 75%); background-size:6px 6px; }
.tp-status { font-size:8px; padding:1px 5px; border-radius:3px; margin-top:1px; }
.tp-status.folded { background:rgba(255,71,87,0.25); color:var(--red); }
.tp-status.out { background:rgba(100,100,100,0.3); color:#888; }

/* Event log */
.game-log {
  display:flex;
  gap:6px;
  padding:6px 12px;
  overflow-x:auto;
  background:rgba(5,10,18,0.62);
  border-top:1px solid rgba(255,255,255,0.04);
  border-bottom:1px solid rgba(255,255,255,0.04);
  flex-shrink:0;
  scrollbar-width:none;
  position: relative;
  z-index: 1;
}
.game-log::-webkit-scrollbar { display:none; }
.log-item {
  font-size:10px;
  color:var(--text-dim);
  white-space:nowrap;
  padding:4px 8px;
  background:rgba(255,255,255,0.05);
  border-radius:999px;
  border:1px solid rgba(255,255,255,0.05);
  flex-shrink:0;
}

/* ===== MY PANEL ===== */
.my-panel {
  background:
    radial-gradient(circle at top, rgba(255,215,0,0.08), transparent 45%),
    linear-gradient(180deg, rgba(10,15,30,0.96) 0%, rgba(8,12,24,0.99) 100%);
  border-top:1px solid rgba(255,255,255,0.08);
  box-shadow: 0 -14px 36px rgba(0,0,0,0.28);
  padding:10px 12px 8px;
  padding-left:max(12px,env(safe-area-inset-left));
  padding-right:max(12px,env(safe-area-inset-right));
  padding-bottom:max(8px,env(safe-area-inset-bottom));
  flex-shrink:0; overflow:hidden;
}
.my-info-row { display:flex; align-items:center; gap:10px; margin-bottom:8px; }

.my-avatar-wrap { position:relative; display:flex; align-items:center; justify-content:center; flex-shrink:0; }
.my-countdown-ring { position:absolute; inset:-4px; width:48px; height:48px; transform:rotate(-90deg); pointer-events:none; }
.my-cd-bg { fill:none; stroke:rgba(255,255,255,0.08); stroke-width:2.5; }
.my-cd-fg { fill:none; stroke:var(--accent); stroke-width:3; stroke-dasharray:131.95; stroke-linecap:round; transition:stroke-dashoffset .3s linear; }
.my-cd-fg.urgent { stroke:var(--red); }
.my-cd-num { position:absolute; top:-10px; right:-14px; font-size:10px; font-weight:700; color:var(--accent); background:rgba(0,0,0,0.8); padding:1px 4px; border-radius:4px; white-space:nowrap; }
.my-cd-num.urgent { color:var(--red); animation:countdownPulse .5s ease-in-out infinite; }

.my-avatar {
  width:38px;
  height:38px;
  border-radius:50%;
  display:flex;
  align-items:center;
  justify-content:center;
  color:#fff;
  font-weight:700;
  font-size:15px;
  position:relative;
  border:2px solid rgba(255,255,255,0.15);
  box-shadow: 0 6px 18px rgba(0,0,0,0.28);
}
.my-detail {
  display:flex;
  align-items:center;
  gap:10px;
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.05);
  border-radius: 999px;
  padding: 6px 12px;
}
.my-name { font-weight:700; font-size:14px; }
.my-chips { font-size:13px; color:var(--gold); display:flex; align-items:center; gap:2px; }
.my-chip-icon { font-size:12px; }
.my-bet-info {
  margin-left:auto;
  font-size:11px;
  color:var(--text-dim);
  background:rgba(255,255,255,0.04);
  padding:6px 10px;
  border-radius:999px;
  border: 1px solid rgba(255,255,255,0.05);
}

@keyframes countdownPulse { 0%,100%{transform:scale(1);} 50%{transform:scale(1.15);} }
@keyframes tableBadgePulse {
  0%,100% { box-shadow: 0 10px 40px rgba(0,0,0,0.26), inset 0 0 0 8px rgba(255,255,255,0.03); }
  50% { box-shadow: 0 14px 48px rgba(0,0,0,0.3), 0 0 24px rgba(255,215,0,0.10), inset 0 0 0 8px rgba(255,255,255,0.04); }
}
@keyframes chipFloat {
  0%,100% { transform: translateY(0) rotate(0deg); }
  50% { transform: translateY(-2px) rotate(4deg); }
}
@keyframes potAmountPulse {
  0%,100% { opacity: .82; transform: translateY(0); }
  50% { opacity: 1; transform: translateY(-1px); }
}
@keyframes turnPlaquePulse {
  0%,100% { box-shadow: inset 0 1px 0 rgba(255,255,255,0.07), 0 0 24px rgba(255,215,0,0.14); }
  50% { box-shadow: inset 0 1px 0 rgba(255,255,255,0.08), 0 0 30px rgba(255,215,0,0.22); }
}
@keyframes turnHalo {
  0% { transform: scale(.92); opacity: .0; }
  20% { opacity: .55; }
  100% { transform: scale(1.18); opacity: 0; }
}
@keyframes chipFlyToPot {
  0% {
    opacity: 0;
    transform: translate(0, 0) scale(.45) rotate(0deg);
  }
  18% {
    opacity: 1;
    transform: translate(calc(var(--chip-dx) * 0.18), calc(var(--chip-dy) * 0.18 - 6px)) scale(1) rotate(90deg);
  }
  72% {
    opacity: 1;
    transform: translate(calc(var(--chip-dx) * 0.82), calc(var(--chip-dy) * 0.82 - 10px)) scale(.92) rotate(220deg);
  }
  100% {
    opacity: 0;
    transform: translate(var(--chip-dx), var(--chip-dy)) scale(.48) rotate(300deg);
  }
}

/* Hand */
.my-hand { display:flex; gap:7px; justify-content:center; margin-bottom:5px; min-height: 76px; }
.game-card {
  width:50px;
  height:72px;
  border-radius:9px;
  background:#fff;
  position:relative;
  box-shadow:0 10px 24px rgba(0,0,0,0.32);
  animation:dealCard .36s cubic-bezier(.2,.7,.2,1) backwards;
  cursor:default;
  transition:transform .14s ease, box-shadow .14s ease;
  border: 1px solid rgba(0,0,0,0.08);
}
.game-card:active { transform:translateY(-3px); }
.game-card:hover { transform:translateY(-5px) rotate(var(--hand-rotate, 0)); box-shadow:0 14px 26px rgba(0,0,0,0.36); }
@keyframes dealCard { from{transform:translateY(30px) rotate(8deg) scale(.94);opacity:0;} }
.fanned-game-card {
  transform: translateX(var(--hand-shift, 0)) translateY(var(--hand-lift, 0)) rotate(var(--hand-rotate, 0));
  transform-origin: bottom center;
}
.my-hand .fanned-game-card {
  animation:
    dealCard .36s cubic-bezier(.2,.7,.2,1) backwards,
    handHoverIdle 4.2s ease-in-out infinite;
}
.my-hand .fanned-game-card:nth-child(2) {
  animation-delay: var(--card-deal-delay, 0s), .12s;
}
.my-hand .fanned-game-card:nth-child(3) {
  animation-delay: var(--card-deal-delay, 0s), .24s;
}
@keyframes handHoverIdle {
  0%,100% { filter: drop-shadow(0 8px 18px rgba(0,0,0,0.12)); }
  50% { filter: drop-shadow(0 12px 22px rgba(0,0,0,0.20)); }
}
.gc-corner { position:absolute; display:flex; flex-direction:column; align-items:center; line-height:1; }
.gc-corner.top { top:3px; left:5px; }
.gc-corner.bottom { bottom:3px; right:5px; transform:rotate(180deg); }
.gc-value { font-size:14px; font-weight:800; color:#333; }
.gc-suit { font-size:11px; }
.gc-center { position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); font-size:22px; }
.game-card.heart .gc-value,.game-card.heart .gc-suit,.game-card.heart .gc-center,
.game-card.diamond .gc-value,.game-card.diamond .gc-suit,.game-card.diamond .gc-center { color:var(--red); }
.game-card.spade .gc-value,.game-card.spade .gc-suit,.game-card.spade .gc-center,
.game-card.club .gc-value,.game-card.club .gc-suit,.game-card.club .gc-center { color:#222; }
.game-card.card-back {
  background:
    radial-gradient(circle at top, rgba(255,255,255,0.10), transparent 42%),
    linear-gradient(180deg, #1a2a44, #102136);
  cursor:pointer;
  display:flex;
  align-items:center;
  justify-content:center;
  flex-direction:column;
}
.card-back-pattern {
  position:absolute;
  inset:3px;
  border-radius:6px;
  border:1px solid rgba(255,255,255,0.12);
  background:
    radial-gradient(circle at center, rgba(255,215,0,0.08), transparent 38%),
    repeating-linear-gradient(45deg,transparent,transparent 3px,rgba(255,255,255,0.04) 3px,rgba(255,255,255,0.04) 6px);
}
.peek-label { position:relative; z-index:1; font-size:10px; color:rgba(255,255,255,0.5); font-weight:600; }
.hand-rank {
  text-align:center;
  font-size:12px;
  color:var(--gold);
  font-weight:700;
  margin-bottom:8px;
  letter-spacing:2px;
  text-shadow: 0 0 10px rgba(255,215,0,0.22);
}

/* Action bar */
.action-bar {
  display:flex;
  gap:8px;
  justify-content:center;
  flex-wrap:nowrap;
  padding:4px 0 2px;
}
.act-btn {
  display:flex;
  flex-direction:column;
  align-items:center;
  gap:2px;
  padding:10px 14px;
  border-radius:14px;
  border:1px solid rgba(255,255,255,0.10);
  font-size:11px;
  font-weight:700;
  cursor:pointer;
  min-width:56px;
  transition:transform .08s, box-shadow .15s, filter .15s;
  color:#fff;
  position:relative;
  overflow:hidden;
  backdrop-filter: blur(10px);
}
.act-btn::after { content:''; position:absolute; inset:0; background:linear-gradient(180deg, rgba(255,255,255,0.1) 0%, transparent 50%); pointer-events:none; }
.act-btn:active { transform:scale(.92); }
.act-btn:hover { filter: brightness(1.03); }
.act-btn .act-icon { font-size:16px; line-height:1; }
.act-btn .act-label { font-size:10px; opacity:.9; }
.act-btn small { font-size:9px; opacity:.7; }
.act-btn.fold { background:linear-gradient(180deg,#4a4a4a,#333); }
.act-btn.call { background:linear-gradient(180deg,#2ecc71,#1fa85a); box-shadow:0 2px 8px rgba(46,204,113,0.3); }
.act-btn.raise { background:linear-gradient(180deg,#3498db,#2176ad); box-shadow:0 2px 8px rgba(52,152,219,0.3); }
.act-btn.compare { background:linear-gradient(180deg,#e67e22,#c96b15); box-shadow:0 2px 8px rgba(230,126,34,0.3); }
.act-btn.allin { background:linear-gradient(180deg,#e74c3c,#b83227); box-shadow:0 2px 8px rgba(231,76,60,0.3); animation:allInGlow 2s ease-in-out infinite; }
@keyframes allInGlow { 0%,100%{box-shadow:0 2px 8px rgba(231,76,60,0.3);} 50%{box-shadow:0 2px 16px rgba(231,76,60,0.6);} }

.waiting-bar { text-align:center; padding:6px 0; }
.waiting-basebet { display:flex; flex-direction:column; gap:8px; align-items:center; margin-bottom:10px; }
.waiting-basebet-label, .waiting-readonly, .spectator-banner { font-size:12px; color:var(--text-dim); }
.waiting-readonly { margin-bottom:10px; }
.spectator-banner { text-align:center; padding:8px 12px; background:rgba(255,255,255,0.06); border:1px solid rgba(255,255,255,0.08); border-radius:10px; margin-bottom:10px; }
.btn-start { padding:10px 36px; border:none; border-radius:10px; background:linear-gradient(135deg,var(--gold),#ff8c00); color:#000; font-size:15px; font-weight:700; cursor:pointer; }

/* Result overlay */
.result-overlay { position:fixed; inset:0; background:rgba(0,0,0,0.7); display:flex; align-items:center; justify-content:center; z-index:90; padding:16px; animation:fadeIn .3s ease; }
@keyframes fadeIn { from{opacity:0;} }
.result-popup { position:relative; background:#111827; border-radius:20px; padding:24px 20px; max-width:340px; width:100%; text-align:center; border:2px solid rgba(255,255,255,0.08); overflow:hidden; max-height:90vh; overflow-y:auto; }
.result-popup.win { border-color:var(--gold); }
.result-popup.win .result-glow { position:absolute; top:-50px; left:50%; transform:translateX(-50%); width:180px; height:180px; border-radius:50%; background:radial-gradient(circle,rgba(255,215,0,0.15),transparent 70%); }
.result-crown { font-size:42px; margin-bottom:6px; animation:bounce .6s ease; }
@keyframes bounce { 0%,100%{transform:translateY(0);} 50%{transform:translateY(-10px);} }
.result-popup h2 { font-size:20px; margin-bottom:12px; position:relative; }
.result-popup.win h2 { color:var(--gold); }
.result-info { display:flex; flex-direction:column; gap:6px; margin-bottom:12px; }
.result-row { display:flex; justify-content:space-between; font-size:13px; }
.result-row span { color:var(--text-dim); }
.result-row .gold { color:var(--gold); }
.result-hands { margin-bottom:16px; display:flex; flex-direction:column; gap:4px; }
.result-player { display:flex; align-items:center; gap:6px; padding:5px 8px; background:rgba(255,255,255,0.04); border-radius:6px; font-size:12px; }
.rp-name { min-width:50px; }
.rp-name.winner { color:var(--gold); font-weight:700; }
.rp-cards { display:flex; gap:3px; }
.rp-card { font-size:11px; padding:1px 3px; background:rgba(255,255,255,0.08); border-radius:3px; font-weight:600; }
.rp-card.heart,.rp-card.diamond { color:var(--red); }
.result-btn-row { display:flex; gap:8px; width:100%; }
.result-btn-row .btn-next-round { flex:1; }
.btn-next-round { width:100%; padding:10px; border:none; border-radius:10px; background:linear-gradient(135deg,var(--gold),#ff8c00); color:#000; font-size:15px; font-weight:700; cursor:pointer; }
.btn-exit-game { flex:1; padding:10px; border:1px solid rgba(255,255,255,0.2); border-radius:10px; background:rgba(255,255,255,0.08); color:var(--text); font-size:15px; font-weight:600; cursor:pointer; }
.bankrupt-msg { color:var(--red); font-size:14px; font-weight:700; margin-bottom:10px; }

/* In-game recharge */
.ingame-recharge { text-align:center; padding:6px; }
.ingame-recharge-msg { color:var(--red); font-size:12px; font-weight:600; display:block; margin-bottom:4px; }
.btn-ingame-recharge { padding:6px 20px; border:none; border-radius:7px; background:linear-gradient(135deg,var(--gold),#ff8c00); color:#000; font-size:13px; font-weight:700; cursor:pointer; }

/* ===== RESPONSIVE - PORTRAIT MOBILE ===== */
@media (max-width:480px) and (orientation:portrait) {
  .game-topbar { gap:4px; padding:3px 8px; flex-wrap:wrap; }
  .topbar-meta { margin-left:0; }
  .table-center-badge { width: 96px; height: 96px; transform: translate(-50%, -58%); }
  .table-badge-value { font-size: 20px; }
  .table-badge-meta { font-size: 9px; }
  .table-felt { width:min(100%,85vw); }
  .game-card { width:44px; height:64px; }
  .gc-value { font-size:12px; }
  .gc-center { font-size:18px; }
  .gc-suit { font-size:9px; }
  .gc-corner.top { top:2px; left:4px; }
  .gc-corner.bottom { bottom:2px; right:4px; }
  .act-btn { padding:6px 10px; font-size:10px; min-width:44px; }
  .act-btn .act-icon { font-size:14px; }
  .act-btn .act-label { font-size:9px; }
  .tp-avatar { width:38px; height:38px; font-size:15px; }
  .tp-countdown-ring { width:46px; height:46px; inset:-4px; }
  .tp-name { font-size:9px; }
  .tp-chips { font-size:8px; }
  .mini-card { width:16px; height:22px; font-size:7px; }
  .lobby-body { padding:10px; }
}

/* ===== RESPONSIVE - LANDSCAPE MOBILE (short screens) ===== */
@media (max-height:440px) and (orientation:landscape) {
  .game-topbar { padding:2px 10px; gap:6px; }
  .pot-value { font-size:15px; }
  .table-center-badge { width: 86px; height: 86px; transform: translate(-50%, -58%); }
  .table-badge-value { font-size: 17px; }
  .table-badge-label, .table-badge-meta { font-size: 8px; }
  .table-felt { width:min(100%,42vh); border-width:3px; }
  .table-logo { font-size:12px; }
  .tp-avatar { width:30px; height:30px; font-size:12px; }
  .tp-countdown-ring { width:38px; height:38px; }
  .tp-cd-num { font-size:8px; bottom:-12px; }
  .tp-name { font-size:8px; }
  .tp-chips { font-size:7px; }
  .mini-card { width:14px; height:20px; font-size:6px; }
  .game-card { width:38px; height:54px; }
  .gc-value { font-size:10px; }
  .gc-center { font-size:14px; }
  .gc-corner.top { top:1px; left:2px; }
  .gc-corner.bottom { bottom:1px; right:2px; }
  .gc-suit { font-size:8px; }
  .my-panel { padding:3px 8px; }
  .my-avatar { width:26px; height:26px; font-size:11px; }
  .my-countdown-ring { width:34px; height:34px; }
  .my-cd-num { font-size:8px; top:-8px; right:-10px; }
  .my-name { font-size:11px; }
  .my-chips { font-size:10px; }
  .act-btn { padding:4px 8px; font-size:9px; min-width:42px; border-radius:7px; }
  .act-btn .act-icon { font-size:12px; }
  .act-btn .act-label { font-size:8px; }
  .game-log { padding:2px 8px; }
  .log-item { font-size:8px; padding:1px 4px; }
  .hand-rank { font-size:10px; margin-bottom:2px; }
  .my-hand { gap:3px; margin-bottom:2px; }
  .chip-stack { width:18px; height:18px; }
  .pot-amount { font-size:10px; top:20px; }
  .my-info-row { gap:6px; margin-bottom:2px; }
  .tp-bet span { font-size:7px; padding:1px 3px; }
}

/* ===== MEDIUM LANDSCAPE ===== */
@media (max-height:600px) and (min-height:441px) and (orientation:landscape) {
  .table-felt { width:min(100%,50vh); }
  .table-center-badge { width: 96px; height: 96px; }
  .game-card { width:46px; height:66px; }
  .gc-value { font-size:12px; }
  .gc-center { font-size:18px; }
  .my-panel { padding:4px 12px; }
  .act-btn { padding:6px 10px; font-size:10px; }
  .tp-avatar { width:36px; height:36px; font-size:14px; }
  .tp-countdown-ring { width:44px; height:44px; }
}

/* ===== TABLET / DESKTOP ===== */
@media (min-width:768px) {
  .table-felt { width:min(100%,650px); }
  .table-center-badge { width: 132px; height: 132px; }
  .table-badge-value { font-size: 28px; }
  .game-card { width:60px; height:86px; }
  .gc-value { font-size:16px; }
  .gc-center { font-size:28px; }
  .act-btn { padding:10px 18px; font-size:13px; min-width:64px; border-radius:12px; }
  .act-btn .act-icon { font-size:18px; }
  .act-btn .act-label { font-size:12px; }
  .mode-cards { flex-direction:row; max-width:800px; }
  .mode-card { flex:1; }
  .tp-avatar { width:50px; height:50px; font-size:20px; }
  .tp-countdown-ring { width:58px; height:58px; }
  .tp-cd-num { font-size:11px; }
  .mini-card { width:24px; height:32px; font-size:9px; }
  .my-avatar { width:40px; height:40px; font-size:17px; }
  .my-countdown-ring { width:48px; height:48px; }
}

@media (min-width:1024px) {
  .table-felt { width:min(100%,750px); }
  .table-center-badge { width: 144px; height: 144px; }
  .table-badge-value { font-size: 32px; }
  .game-card { width:68px; height:96px; }
  .gc-value { font-size:18px; }
  .gc-center { font-size:32px; }
}

/* Landscape layout */
@media (orientation:landscape) {
  .game-page { flex-direction:column; }
  .game-topbar { width:100%; flex-shrink:0; }
  .table-container { flex:1; min-width:0; min-height:0; }
  .my-panel { width:100%; }
  .game-log { width:100%; }
  .lobby-page .mode-cards { flex-direction:row; max-width:900px; }
  .lobby-page .mode-card { flex:1; }
  .login-card { max-width:380px; padding:24px 20px; }
  .login-page { padding:12px; }
  .logo-icon { width:48px; height:48px; border-radius:12px; }
  .logo-a { font-size:22px; }
  .game-title { font-size:20px; letter-spacing:4px; }
  .logo { margin-bottom:16px; }
  .login-links { gap:10px; margin-top:14px; }
  .login-link { width:44px; height:44px; }
  .login-link-svg { width:20px; height:20px; }
}
</style>
