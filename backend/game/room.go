package game

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	// MaxRoundsPerGame is the maximum number of rounds before the game ends (determines winner by best hand)
	MaxRoundsPerGame = 20
)

// SidePot represents a pot that only certain players are eligible to win
type SidePot struct {
	Amount    int      `json:"amount"`
	Eligible  []string `json:"eligible"`  // player IDs eligible for this pot
	WinnerID  string   `json:"winnerId"`  // winner of this pot (set after resolution)
	WinAmount int      `json:"winAmount"` // amount awarded after rake (set after resolution)
}

// ActionType 操作类型
type ActionType int

const (
	ActionCall    ActionType = 1
	ActionRaise   ActionType = 2
	ActionFold    ActionType = 3
	ActionAllIn   ActionType = 4
	ActionCompare ActionType = 5
	ActionLook    ActionType = 6
)

// RoomState 房间状态
type RoomState int

const (
	RoomWaiting  RoomState = 0
	RoomPlaying  RoomState = 1
	RoomFinished RoomState = 2
)

// Room 游戏房间
type Room struct {
	ID                string          `json:"id"`
	RoomCode          string          `json:"roomCode"`
	OwnerID           string          `json:"ownerId"`
	IsPrivate         bool            `json:"isPrivate"`
	IsMatch           bool            `json:"isMatch"`
	State             RoomState       `json:"state"`
	Players           []*Player       `json:"players"`
	Spectators        []*Player       `json:"spectators"`
	MaxPlayers        int             `json:"maxPlayers"`
	BaseBet           int             `json:"baseBet"`
	CurrentBet        int             `json:"currentBet"`
	LastBet           int             `json:"lastBet"`
	Pot               int             `json:"pot"`
	AnteTotal         int             `json:"anteTotal"`
	Round             int             `json:"round"`
	TurnIndex         int             `json:"turnIndex"`
	DealerIndex       int             `json:"dealerIndex"`
	GoodBias          int             `json:"goodBias"` // 好牌权重 1-5
	HasAllIn          bool            `json:"hasAllIn"`
	LastAllInAmount   int             `json:"lastAllInAmount"`
	LastAllInPlayerID string          `json:"lastAllInPlayerId"`
	SidePots          []SidePot       `json:"sidePots,omitempty"`
	TurnStart         time.Time       `json:"-"`
	CreatedAt         time.Time       `json:"createdAt"`
	ReadyForNext      map[string]bool `json:"-"`
	NextRoundDeadline time.Time       `json:"-"`
	Broadcast         func(msg interface{})
	mu                sync.RWMutex
}

// Action 玩家操作
type Action struct {
	Type     ActionType `json:"type"`
	Amount   int        `json:"amount,omitempty"`
	TargetID string     `json:"targetId,omitempty"`
}

// GameEvent 游戏事件
type GameEvent struct {
	Type       string `json:"type"`
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	TargetID   string `json:"targetId,omitempty"`
	TargetName string `json:"targetName,omitempty"`
	WinnerID   string `json:"winnerId,omitempty"`
	Amount     int    `json:"amount,omitempty"`
}

// NewRoom 创建房间
func NewRoom(id string, maxPlayers int) *Room {
	return &Room{
		ID:         id,
		State:      RoomWaiting,
		Players:    make([]*Player, 0, maxPlayers),
		Spectators: make([]*Player, 0),
		MaxPlayers: maxPlayers,
		BaseBet:    10,
		CurrentBet: 10,
		GoodBias:   3, // 默认好牌权重
		CreatedAt:  time.Now(),
	}
}

// AddPlayer 添加玩家
func (r *Room) AddPlayer(p *Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.Players {
		if existing.ID == p.ID {
			return fmt.Errorf("你已在房间中")
		}
	}
	if len(r.Players) >= r.MaxPlayers {
		return fmt.Errorf("房间已满")
	}
	if r.State == RoomPlaying {
		return fmt.Errorf("游戏进行中")
	}
	p.Seat = len(r.Players)
	p.IsSpectator = false
	r.Players = append(r.Players, p)
	return nil
}

func (r *Room) AddSpectator(p *Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.Spectators {
		if existing.ID == p.ID {
			return nil
		}
	}
	p.IsSpectator = true
	p.Seat = -1
	p.State = PlayerWaiting
	p.Looked = false
	p.BetTotal = 0
	p.Hand = Hand{}
	r.Spectators = append(r.Spectators, p)
	return nil
}

// RemovePlayer 移除玩家
func (r *Room) RemovePlayer(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.Players {
		if p.ID == id {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			// 重新排座位
			for j := i; j < len(r.Players); j++ {
				r.Players[j].Seat = j
			}
			break
		}
	}
}

func (r *Room) RemoveSpectator(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.Spectators {
		if p.ID == id {
			r.Spectators = append(r.Spectators[:i], r.Spectators[i+1:]...)
			break
		}
	}
}

// FindPlayer 查找玩家
func (r *Room) FindPlayer(id string) *Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (r *Room) FindSpectator(id string) *Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.Spectators {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// AlivePlayers 存活玩家
func (r *Room) AlivePlayers() []*Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	alive := make([]*Player, 0)
	for _, p := range r.Players {
		if p.State == PlayerPlaying || p.State == PlayerAllIn {
			alive = append(alive, p)
		}
	}
	return alive
}

// FillBots 填充机器人
func (r *Room) FillBots(count int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < count && len(r.Players) < r.MaxPlayers; i++ {
		bot := NewBot(len(r.Players))
		r.Players = append(r.Players, bot)
	}
}

// StartGame 开始游戏 - 每局重新洗牌
func (r *Room) StartGame() error {
	r.mu.Lock()
	if len(r.Players) < 2 {
		r.mu.Unlock()
		return fmt.Errorf("至少需要2名玩家")
	}

	// 只给机器人补充筹码
	for _, p := range r.Players {
		if p.IsBot && p.Chips <= 0 {
			p.Chips = 500 // 破产补助
		}
	}

	r.State = RoomPlaying
	r.HasAllIn = false
	r.LastAllInAmount = 0
	r.LastAllInPlayerID = ""
	r.SidePots = nil
	r.Round = 1
	r.CurrentBet = r.BaseBet
	r.LastBet = 0
	r.Pot = 0
	r.AnteTotal = 0
	r.ReadyForNext = make(map[string]bool)
	r.NextRoundDeadline = time.Time{}
	r.DealerIndex = (r.DealerIndex + 1) % len(r.Players)

	for _, p := range r.Players {
		p.State = PlayerPlaying
		p.BetTotal = 0
		p.Looked = false
	}
	r.mu.Unlock()

	// 每局重新洗牌 + 加权发牌（好牌更多）
	hands := DealWeightedHands(len(r.Players), r.GoodBias)
	r.mu.Lock()
	for i, p := range r.Players {
		p.Hand = hands[i]
	}

	// 底注 - 安全扣除
	anteTotal := 0
	for _, p := range r.Players {
		deduct := r.BaseBet
		if deduct > p.Chips {
			deduct = p.Chips
		}
		p.Chips -= deduct
		p.BetTotal += deduct
		r.Pot += deduct
		anteTotal += deduct
	}
	r.AnteTotal = anteTotal

	r.TurnIndex = (r.DealerIndex + 1) % len(r.Players)
	r.TurnStart = time.Now()
	r.mu.Unlock()

	return nil
}

// CurrentTurnPlayer 当前操作玩家
func (r *Room) CurrentTurnPlayer() *Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.State != RoomPlaying || len(r.Players) == 0 {
		return nil
	}
	return r.Players[r.TurnIndex]
}

// NextTurn 切换到下一个存活玩家
func (r *Room) NextTurn() {
	r.mu.Lock()
	defer r.mu.Unlock()

	foundAlive := false
	for i := 0; i < len(r.Players); i++ {
		r.TurnIndex = (r.TurnIndex + 1) % len(r.Players)
		p := r.Players[r.TurnIndex]
		if p.State == PlayerPlaying || p.State == PlayerAllIn {
			foundAlive = true
			break
		}
	}

	// Always update Round and TurnStart, regardless of whether we found an alive player
	// This prevents the game loop from getting stuck if all players are folded
	if foundAlive {
		r.Round++
		r.TurnStart = time.Now()
	}
}

// safeDeduct 安全扣除筹码，不会扣成负数
func safeDeduct(p *Player, amount int) int {
	if amount > p.Chips {
		amount = p.Chips
	}
	if amount < 0 {
		amount = 0
	}
	p.Chips -= amount
	return amount
}

// ProcessAction 处理玩家操作
func (r *Room) ProcessAction(playerID string, action Action) (*GameEvent, error) {
	player := r.FindPlayer(playerID)
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	current := r.CurrentTurnPlayer()
	if current == nil || current.ID != playerID {
		return nil, fmt.Errorf("不是你的回合")
	}

	// Auto-fold when chips insufficient for minimum bet
	minBet := r.BaseBet
	if r.LastBet > 0 {
		minBet = r.LastBet
	}
	if player.Looked {
		minBet = minBet * 2
	}
	if player.Chips < minBet && action.Type != ActionFold && action.Type != ActionAllIn && action.Type != ActionCompare {
		action = Action{Type: ActionFold}
	}

	// 如果有人全押，其他人只能弃牌、全押或比牌
	if r.HasAllIn && action.Type != ActionFold && action.Type != ActionAllIn && action.Type != ActionCompare {
		return nil, fmt.Errorf("有人全押，只能弃牌、比牌或全押")
	}

	event := &GameEvent{
		PlayerID:   playerID,
		PlayerName: player.Name,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	switch action.Type {
	case ActionLook:
		player.Looked = true
		event.Type = "look"
		return event, nil

	case ActionCall:
		prevBet := r.LastBet
		if prevBet <= 0 {
			prevBet = r.BaseBet
		}
		bet := prevBet
		if player.Looked {
			bet = prevBet * 2
		}
		actual := safeDeduct(player, bet)
		player.BetTotal += actual
		r.Pot += actual
		r.LastBet = prevBet // LastBet stays as the blind-equivalent amount
		event.Type = "call"
		event.Amount = actual

	case ActionRaise:
		prevBet := r.LastBet
		if prevBet <= 0 {
			prevBet = r.BaseBet
		}
		minAmount := prevBet * 2
		if player.Looked {
			minAmount = minAmount * 2
		}
		amount := action.Amount
		if amount < minAmount {
			amount = minAmount
		}
		actual := safeDeduct(player, amount)
		// Update LastBet to the blind-equivalent amount
		if player.Looked {
			r.LastBet = actual / 2
		} else {
			r.LastBet = actual
		}
		r.CurrentBet = r.LastBet
		player.BetTotal += actual
		r.Pot += actual
		event.Type = "raise"
		event.Amount = actual

	case ActionFold:
		player.State = PlayerFolded
		event.Type = "fold"

	case ActionAllIn:
		remaining := player.Chips
		if remaining < 0 {
			remaining = 0
		}
		player.Chips = 0
		player.BetTotal += remaining
		player.State = PlayerAllIn
		r.Pot += remaining
		r.HasAllIn = true
		r.LastAllInAmount = remaining
		r.LastAllInPlayerID = player.ID
		event.Type = "allin"
		event.Amount = remaining

	case ActionCompare:
		target := r.findPlayerLocked(action.TargetID)
		if target == nil {
			return nil, fmt.Errorf("比牌目标不存在")
		}
		if target.State != PlayerPlaying && target.State != PlayerAllIn {
			return nil, fmt.Errorf("目标玩家已出局")
		}

		baseCost := r.CurrentBet
		if r.HasAllIn {
			if target.State != PlayerAllIn {
				return nil, fmt.Errorf("全押状态下只能比牌全押玩家")
			}
			baseCost = r.LastAllInAmount
			if baseCost <= 0 {
				baseCost = target.BetTotal
			}
			if player.Chips < baseCost {
				return nil, fmt.Errorf("筹码不足，只能弃牌或全押")
			}
		} else if player.Looked {
			baseCost = r.CurrentBet * 2
		}
		actual := safeDeduct(player, baseCost)
		player.BetTotal += actual
		r.Pot += actual

		result := CompareHands(player.Hand, target.Hand)
		event.Type = "compare"
		event.TargetID = target.ID
		event.TargetName = target.Name
		event.Amount = actual

		if result > 0 {
			// Initiator wins
			target.State = PlayerCompared
			if r.HasAllIn && player.Chips == 0 {
				player.State = PlayerAllIn
			}
			event.WinnerID = player.ID
		} else if result == 0 {
			// Draw: both pay half or full amount back (simplified: just mark as draw)
			if r.HasAllIn && player.Chips == 0 {
				player.State = PlayerAllIn
			}
			event.WinnerID = "draw"
		} else {
			// Initiator loses
			player.State = PlayerCompared
			event.WinnerID = target.ID
		}

	default:
		return nil, fmt.Errorf("未知操作")
	}

	return event, nil
}

func (r *Room) findPlayerLocked(id string) *Player {
	for _, p := range r.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// CheckGameEnd 检查游戏是否结束
func (r *Room) CheckGameEnd() (bool, *Player) {
	alive := r.AlivePlayers()
	if len(alive) == 1 {
		return true, alive[0]
	}
	if len(alive) == 0 {
		// 不应该发生，但安全处理
		if len(r.Players) > 0 {
			return true, r.Players[0]
		}
		return true, nil
	}
	if r.Round > 20 {
		winner := alive[0]
		for i := 1; i < len(alive); i++ {
			if CompareHands(alive[i].Hand, winner.Hand) > 0 {
				winner = alive[i]
			}
		}
		return true, winner
	}
	return false, nil
}

// calculateSidePots computes side pots based on each player's total bet.
// Eligible players are those who are not folded (AllIn or Playing) at game end.
// All players' bets (including folded) contribute to the pots.
func (r *Room) calculateSidePots() []SidePot {
	// Collect alive (non-folded) players and sort by BetTotal ascending
	var alive []*Player
	for _, p := range r.Players {
		if p.State == PlayerAllIn || p.State == PlayerPlaying {
			alive = append(alive, p)
		}
	}
	sort.Slice(alive, func(i, j int) bool {
		return alive[i].BetTotal < alive[j].BetTotal
	})

	var pots []SidePot
	prevLevel := 0

	for _, ap := range alive {
		level := ap.BetTotal
		if level <= prevLevel {
			continue // same bet level, skip duplicates
		}

		potAmount := 0
		for _, p := range r.Players {
			contribution := p.BetTotal
			if contribution > level {
				contribution = level
			}
			prev := p.BetTotal
			if prev > prevLevel {
				prev = prevLevel
			}
			potAmount += contribution - prev
		}

		// Eligible = alive players whose BetTotal >= this level
		var eligible []string
		for _, a := range alive {
			if a.BetTotal >= level {
				eligible = append(eligible, a.ID)
			}
		}

		if potAmount > 0 {
			pots = append(pots, SidePot{
				Amount:   potAmount,
				Eligible: eligible,
			})
		}

		prevLevel = level
	}

	return pots
}

// findBestHandAmongIDs returns the player with the best hand among the given IDs.
func (r *Room) findBestHandAmongIDs(ids []string) *Player {
	var best *Player
	for _, id := range ids {
		p := r.findPlayerLocked(id)
		if p == nil {
			continue
		}
		if best == nil {
			best = p
		} else if CompareHands(p.Hand, best.Hand) > 0 {
			best = p
		}
	}
	return best
}

// EndGame 结束游戏
// When HasAllIn is true, it calculates side pots and distributes winnings per pot.
// The winner param is used only when HasAllIn is false (normal single-winner case).
// Returns total rake.
func (r *Room) EndGame(winner *Player) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	participantCount := len(r.Players)
	rake := r.BaseBet * participantCount
	if rake > r.Pot {
		rake = r.Pot
	}

	if r.HasAllIn {
		// Calculate side pots
		pots := r.calculateSidePots()

		// Distribute rake proportionally across pots
		totalPotAmount := 0
		for _, pot := range pots {
			totalPotAmount += pot.Amount
		}

		rakeRemaining := rake
		for i := range pots {
			// Proportional rake from each pot (use int64 to avoid overflow)
			var potRake int
			if i == len(pots)-1 {
				// Last pot gets remaining rake to avoid rounding issues
				potRake = rakeRemaining
			} else {
				potRake = int(int64(rake) * int64(pots[i].Amount) / int64(totalPotAmount))
			}
			if potRake > pots[i].Amount {
				potRake = pots[i].Amount
			}
			rakeRemaining -= potRake

			winnings := pots[i].Amount - potRake
			if winnings < 0 {
				winnings = 0
			}
			pots[i].WinAmount = winnings

			// Find best hand among eligible players for this pot
			potWinner := r.findBestHandAmongIDs(pots[i].Eligible)
			if potWinner != nil {
				potWinner.Chips += winnings
				pots[i].WinnerID = potWinner.ID
			}
		}

		r.SidePots = pots
	} else {
		// Normal single-winner case
		r.SidePots = nil
		if winner != nil {
			winnings := r.Pot - rake
			if winnings < 0 {
				winnings = 0
				rake = r.Pot
			}
			winner.Chips += winnings
		}
	}

	r.State = RoomFinished
	r.ReadyForNext = make(map[string]bool)
	r.NextRoundDeadline = time.Now().Add(30 * time.Second)
	return rake
}

// CheckAllInResolution checks if all alive players have decided (fold or all-in) after someone went all-in
func (r *Room) CheckAllInResolution() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.HasAllIn {
		return false
	}
	for _, p := range r.Players {
		if p.State == PlayerPlaying {
			// Someone still hasn't decided
			return false
		}
	}
	return true
}

// ResolveAllIn compares all all-in players' hands and returns the winner
func (r *Room) ResolveAllIn() *Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var best *Player
	for _, p := range r.Players {
		if p.State == PlayerAllIn {
			if best == nil {
				best = p
			} else if CompareHands(p.Hand, best.Hand) > 0 {
				best = p
			}
		}
	}
	return best
}

// ConfirmNextRound marks a player as ready for the next round
func (r *Room) ConfirmNextRound(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ReadyForNext == nil {
		r.ReadyForNext = make(map[string]bool)
	}
	r.ReadyForNext[playerID] = true
}

// AllConfirmedNextRound returns true if all human players have confirmed
func (r *Room) AllConfirmedNextRound() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.Players {
		if !p.IsBot && !r.ReadyForNext[p.ID] {
			return false
		}
	}
	return true
}

// UnconfirmedPlayers returns player IDs that have not confirmed next round
func (r *Room) UnconfirmedPlayers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var ids []string
	for _, p := range r.Players {
		if !p.IsBot && !r.ReadyForNext[p.ID] {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

// PromoteSpectatorsForNextRound turns current spectators into normal room users
// after settlement so they can choose to stay for the next round or leave.
func (r *Room) PromoteSpectatorsForNextRound() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.Spectators) == 0 {
		return
	}
	if r.MaxPlayers < len(r.Players)+len(r.Spectators) {
		r.MaxPlayers = len(r.Players) + len(r.Spectators)
	}
	for _, p := range r.Spectators {
		if p == nil {
			continue
		}
		p.IsSpectator = false
		p.Seat = len(r.Players)
		p.State = PlayerReady
		p.Looked = false
		p.BetTotal = 0
		p.Hand = Hand{}
		r.Players = append(r.Players, p)
	}
	r.Spectators = nil
}

// HandlePlayerLeave handles a player leaving the room. Returns true if the
// leaving player was the current turn player (caller should advance turn).
// During an active hand the player is only marked folded so settlement can
// still account for prior bets. Outside active play the player is removed.
func (r *Room) HandlePlayerLeave(playerID string) (wasTurn bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.Spectators {
		if p.ID == playerID {
			r.Spectators = append(r.Spectators[:i], r.Spectators[i+1:]...)
			return false
		}
	}

	if r.State != RoomPlaying {
		for i, p := range r.Players {
			if p.ID == playerID {
				r.Players = append(r.Players[:i], r.Players[i+1:]...)
				for j := i; j < len(r.Players); j++ {
					r.Players[j].Seat = j
				}
				if len(r.Players) > 0 && r.TurnIndex >= len(r.Players) {
					r.TurnIndex = r.TurnIndex % len(r.Players)
				} else if len(r.Players) == 0 {
					r.TurnIndex = 0
				}
				break
			}
		}
		return false
	}

	// Check if it was this player's turn
	if len(r.Players) > 0 {
		if r.TurnIndex < len(r.Players) && r.Players[r.TurnIndex].ID == playerID {
			wasTurn = true
		}
	}
	// Mark as folded (do NOT remove from players list for proper settlement)
	for _, p := range r.Players {
		if p.ID == playerID {
			p.State = PlayerFolded
			break
		}
	}

	// Fix TurnIndex if it's pointing to folded player
	if r.TurnIndex < len(r.Players) && r.Players[r.TurnIndex].ID == playerID {
		// Need to find next alive player, will be done by NextTurn()
		// Just mark that we need to advance
		wasTurn = true
	}
	return wasTurn
}

// GetRoomInfo 获取房间信息
func (r *Room) GetRoomInfo(forPlayerID string) map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]map[string]interface{}, 0)
	for _, p := range r.Players {
		pi := map[string]interface{}{
			"id":       p.ID,
			"name":     p.Name,
			"avatar":   p.Avatar,
			"chips":    p.Chips,
			"state":    p.State,
			"betTotal": p.BetTotal,
			"looked":   p.Looked,
			"seat":     p.Seat,
			"isBot":    p.IsBot,
		}
		if (p.ID == forPlayerID && p.Looked) || r.State == RoomFinished {
			pi["hand"] = p.Hand
		}
		players = append(players, pi)
	}

	spectators := make([]map[string]interface{}, 0, len(r.Spectators))
	for _, p := range r.Spectators {
		spectators = append(spectators, map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"avatar":      p.Avatar,
			"isBot":       p.IsBot,
			"isSpectator": true,
		})
	}

	info := map[string]interface{}{
		"id":                r.ID,
		"roomCode":          r.RoomCode,
		"ownerId":           r.OwnerID,
		"isPrivate":         r.IsPrivate,
		"isMatch":           r.IsMatch,
		"isSpectator":       r.FindSpectator(forPlayerID) != nil,
		"state":             r.State,
		"players":           players,
		"spectators":        spectators,
		"maxPlayers":        r.MaxPlayers,
		"baseBet":           r.BaseBet,
		"currentBet":        r.CurrentBet,
		"lastBet":           r.LastBet,
		"pot":               r.Pot,
		"anteTotal":         r.AnteTotal,
		"round":             r.Round,
		"turnIndex":         r.TurnIndex,
		"dealerIndex":       r.DealerIndex,
		"hasAllIn":          r.HasAllIn,
		"lastAllInAmount":   r.LastAllInAmount,
		"lastAllInPlayerId": r.LastAllInPlayerID,
		"turnDeadline":      r.TurnStart.Add(20 * time.Second).UnixMilli(),
	}

	if r.State == RoomFinished && !r.NextRoundDeadline.IsZero() {
		confirmed := make([]string, 0)
		for pid := range r.ReadyForNext {
			confirmed = append(confirmed, pid)
		}
		info["readyForNext"] = confirmed
		info["nextRoundDeadline"] = r.NextRoundDeadline.UnixMilli()
	}

	return info
}
