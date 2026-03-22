package game

import (
	"sync"
	"time"
)

// PlayerState 玩家状态
type PlayerState int

const (
	PlayerWaiting  PlayerState = 0 // 等待中
	PlayerReady    PlayerState = 1 // 已准备
	PlayerPlaying  PlayerState = 2 // 游戏中
	PlayerFolded   PlayerState = 3 // 已弃牌
	PlayerAllIn    PlayerState = 4 // 已全押
	PlayerCompared PlayerState = 5 // 已比牌出局
)

// Player 玩家
type Player struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Avatar   int         `json:"avatar"`
	Chips    int         `json:"chips"`
	Hand     Hand        `json:"-"`
	State    PlayerState `json:"state"`
	BetTotal int         `json:"betTotal"`
	Looked   bool        `json:"looked"` // 是否看牌
	Seat     int         `json:"seat"`
	IsBot         bool        `json:"isBot"`
	WalletAddress string      `json:"walletAddress,omitempty"`
	WalletChain   string      `json:"walletChain,omitempty"`
	mu            sync.Mutex
}

// NewPlayer 创建玩家
func NewPlayer(id, name string, seat int) *Player {
	return &Player{
		ID:     id,
		Name:   name,
		Avatar: seat%6 + 1,
		Chips:  1000,
		State:  PlayerWaiting,
		Seat:   seat,
	}
}

// NewBot 创建机器人
func NewBot(seat int) *Player {
	names := []string{"机器人A", "机器人B", "机器人C", "机器人D", "机器人E"}
	name := names[seat%len(names)]
	p := NewPlayer("bot_"+name, name, seat)
	p.IsBot = true
	p.State = PlayerReady
	return p
}

// Reset 重置玩家状态（新一局）
func (p *Player) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.State = PlayerWaiting
	p.BetTotal = 0
	p.Looked = false
	p.Hand = Hand{}
}

// BotAction 机器人AI决策
func (p *Player) BotAction(room *Room) Action {
	time.Sleep(time.Millisecond * 500) // 模拟思考

	if room.HasAllIn {
		rank := HandRank(p.Hand)
		if rank >= HandTypePair {
			return Action{Type: ActionAllIn}
		}
		// 30% chance to bluff all-in with bad hand
		if time.Now().UnixNano()%10 < 3 {
			return Action{Type: ActionAllIn}
		}
		return Action{Type: ActionFold}
	}

	rank := HandRank(p.Hand)
	round := room.Round

	// 简单AI策略
	switch {
	case rank >= HandTypeStraightFlush:
		// 同花顺或豹子：加注/全押
		if p.Chips <= room.CurrentBet*2 {
			return Action{Type: ActionAllIn}
		}
		return Action{Type: ActionRaise, Amount: room.CurrentBet * 2}

	case rank >= HandTypeFlush:
		// 金花：跟注或加注
		if round > 3 && p.Looked {
			return Action{Type: ActionRaise, Amount: room.CurrentBet + room.BaseBet}
		}
		return Action{Type: ActionCall}

	case rank >= HandTypeStraight:
		// 顺子：跟注
		if round > 5 {
			// 后期可能比牌
			alive := room.AlivePlayers()
			if len(alive) == 2 {
				for _, ap := range alive {
					if ap.ID != p.ID {
						return Action{Type: ActionCompare, TargetID: ap.ID}
					}
				}
			}
		}
		return Action{Type: ActionCall}

	case rank >= HandTypePair:
		// 对子：前几轮跟注，后面可能弃牌
		if round > 4 && room.CurrentBet > room.BaseBet*4 {
			return Action{Type: ActionFold}
		}
		return Action{Type: ActionCall}

	default:
		// 散牌：前两轮跟注看看，之后弃牌
		if round <= 2 {
			return Action{Type: ActionCall}
		}
		if round <= 3 && room.CurrentBet <= room.BaseBet*2 {
			return Action{Type: ActionCall}
		}
		return Action{Type: ActionFold}
	}
}
