package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"zhajinhua/game"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client WebSocket客户端
type Client struct {
	Conn     *websocket.Conn
	PlayerID string
	RoomID   string
	Send     chan []byte
}

// MatchQueue 匹配队列
type MatchQueue struct {
	clients []*Client
	mu      sync.Mutex
}

// Hub 管理所有连接和房间
type Hub struct {
	Rooms      map[string]*game.Room
	Clients    map[string]*Client
	MatchQueue MatchQueue
	TurnTimers map[string]*time.Timer
	mu         sync.RWMutex
}

var hub = &Hub{
	Rooms:      make(map[string]*game.Room),
	Clients:    make(map[string]*Client),
	TurnTimers: make(map[string]*time.Timer),
}

// Message WebSocket消息
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type JoinData struct {
	RoomID     string `json:"roomId"`
	PlayerName string `json:"playerName"`
	BotCount   int    `json:"botCount"`
	BaseBet    int    `json:"baseBet"`
	Wallet     string `json:"wallet"`  // Web3钱包地址
	Chain      string `json:"chain"`   // "evm", "ton", "sol"
}

type ActionData struct {
	Type     int    `json:"type"`
	Amount   int    `json:"amount"`
	TargetID string `json:"targetId"`
}

type RechargeData struct {
	Amount int    `json:"amount"`
	Wallet string `json:"wallet"`
}

type VerifyRechargeData struct {
	TxHash string `json:"txHash"`
	Chain  string `json:"chain"`
	Amount int    `json:"amount"` // native coin amount in smallest unit
	Points int    `json:"points"` // game points to credit
}

// HandleWebSocket WebSocket连接处理
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	playerID := fmt.Sprintf("player_%d", time.Now().UnixNano())
	client := &Client{
		Conn:     conn,
		PlayerID: playerID,
		Send:     make(chan []byte, 256),
	}

	hub.mu.Lock()
	hub.Clients[playerID] = client
	hub.mu.Unlock()

	// Load chips from DB
	chips, _ := game.GetOrCreateUser(playerID, "")

	sendToClient(client, map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{
			"playerId": playerID,
			"chips":    chips,
			"mode":     string(Mode),
		},
	})

	go clientWriter(client)
	go clientReader(client)
}

func clientWriter(c *Client) {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func clientReader(c *Client) {
	defer func() {
		// 从匹配队列移除
		hub.MatchQueue.mu.Lock()
		for i, mc := range hub.MatchQueue.clients {
			if mc.PlayerID == c.PlayerID {
				hub.MatchQueue.clients = append(hub.MatchQueue.clients[:i], hub.MatchQueue.clients[i+1:]...)
				break
			}
		}
		hub.MatchQueue.mu.Unlock()

		hub.mu.Lock()
		delete(hub.Clients, c.PlayerID)
		hub.mu.Unlock()

		if c.RoomID != "" {
			hub.mu.RLock()
			room, ok := hub.Rooms[c.RoomID]
			hub.mu.RUnlock()
			if ok {
				wasTurn := room.HandlePlayerLeave(c.PlayerID)

				if room.State == game.RoomPlaying {
					if room.CheckAllInResolution() {
						winner := room.ResolveAllIn()
						rake := room.EndGame(winner)
						settleGameToDB(room, winner, rake)
						cancelTurnTimer(room.ID)
						broadcastGameEnd(room, winner, rake)
					} else {
						ended, winner := room.CheckGameEnd()
						if ended {
							rake := room.EndGame(winner)
							settleGameToDB(room, winner, rake)
							cancelTurnTimer(room.ID)
							broadcastGameEnd(room, winner, rake)
						} else if wasTurn {
							room.NextTurn()
							broadcastRoomState(room)
							startTurnTimer(room)
							go processBotTurns(room)
						} else {
							broadcastRoomState(room)
						}
					}
				} else {
					broadcastRoomState(room)
				}

				// Clean up empty rooms
				if len(room.Players) == 0 {
					hub.mu.Lock()
					delete(hub.Rooms, c.RoomID)
					hub.mu.Unlock()
				}
			}
		}
		close(c.Send)
		c.Conn.Close()
	}()

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}
		handleMessage(c, msg)
	}
}

func handleMessage(c *Client, msg Message) {
	switch msg.Type {
	case "create_room":
		handleCreateRoom(c, msg.Data)
	case "join_room":
		handleJoinRoom(c, msg.Data)
	case "ready":
		handleReady(c)
	case "start_game":
		handleStartGame(c)
	case "action":
		handleAction(c, msg.Data)
	case "new_round":
		handleNewRound(c)
	case "match":
		handleMatch(c, msg.Data)
	case "cancel_match":
		handleCancelMatch(c)
	case "join_by_code":
		handleJoinByCode(c, msg.Data)
	case "recharge":
		handleRecharge(c, msg.Data)
	case "leave_room":
		handleLeaveRoom(c)
	case "claim_bonus":
		handleClaimBonus(c)
	case "verify_recharge":
		handleVerifyRecharge(c, msg.Data)
	}
}

func handleCreateRoom(c *Client, data json.RawMessage) {
	var req JoinData
	json.Unmarshal(data, &req)

	if req.PlayerName == "" {
		req.PlayerName = "玩家"
	}

	// PE mode: require valid wallet address
	if Mode == ModePE {
		chain := WalletChain(req.Chain)
		if chain == "" {
			chain = DetectChain(req.Wallet)
		}
		if !ValidateWalletAddress(req.Wallet, chain) {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": "PE mode requires a valid wallet address (EVM/TON/SOL)"},
			})
			return
		}
	}

	roomID := fmt.Sprintf("room_%d", time.Now().UnixNano())
	room := game.NewRoom(roomID, 6)
	room.RoomCode = generateRoomCode(6)

	// 设置底分
	if req.BaseBet > 0 {
		room.BaseBet = req.BaseBet
		room.CurrentBet = req.BaseBet
	}

	player := game.NewPlayer(c.PlayerID, req.PlayerName, 0)
	chips, _ := game.GetOrCreateUser(c.PlayerID, req.PlayerName)
	player.Chips = chips
	player.State = game.PlayerReady

	if req.Wallet != "" {
		chain := WalletChain(req.Chain)
		if chain == "" {
			chain = DetectChain(req.Wallet)
		}
		player.WalletAddress = req.Wallet
		player.WalletChain = string(chain)
		game.GetOrCreateUserWithWallet(c.PlayerID, req.PlayerName, req.Wallet, string(chain))
	}

	room.AddPlayer(player)

	if req.BotCount > 0 {
		room.FillBots(req.BotCount)
	}

	hub.mu.Lock()
	hub.Rooms[roomID] = room
	hub.mu.Unlock()

	c.RoomID = roomID

	sendToClient(c, map[string]interface{}{
		"type": "room_created",
		"data": room.GetRoomInfo(c.PlayerID),
	})
}

func handleJoinRoom(c *Client, data json.RawMessage) {
	var req JoinData
	json.Unmarshal(data, &req)

	hub.mu.RLock()
	room, ok := hub.Rooms[req.RoomID]
	hub.mu.RUnlock()

	if !ok {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "房间不存在"},
		})
		return
	}

	if req.PlayerName == "" {
		req.PlayerName = "玩家"
	}

	if Mode == ModePE {
		chain := WalletChain(req.Chain)
		if chain == "" {
			chain = DetectChain(req.Wallet)
		}
		if !ValidateWalletAddress(req.Wallet, chain) {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": "PE mode requires a valid wallet address"},
			})
			return
		}
	}

	player := game.NewPlayer(c.PlayerID, req.PlayerName, 0)
	chips, _ := game.GetOrCreateUser(c.PlayerID, req.PlayerName)
	player.Chips = chips
	if err := room.AddPlayer(player); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}

	c.RoomID = req.RoomID
	broadcastRoomState(room)
}

func handleReady(c *Client) {
	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}
	player := room.FindPlayer(c.PlayerID)
	if player != nil {
		player.State = game.PlayerReady
	}
	broadcastRoomState(room)
}

func handleStartGame(c *Client) {
	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}

	if err := room.StartGame(); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}

	broadcastRoomState(room)
	startTurnTimer(room)
	go processBotTurns(room)
}

func handleAction(c *Client, data json.RawMessage) {
	var req ActionData
	json.Unmarshal(data, &req)

	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}

	action := game.Action{
		Type:     game.ActionType(req.Type),
		Amount:   req.Amount,
		TargetID: req.TargetID,
	}

	event, err := room.ProcessAction(c.PlayerID, action)
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}

	broadcastEvent(room, event)

	if action.Type == game.ActionLook {
		broadcastRoomState(room)
		return
	}

	// Check all-in resolution first
	if room.CheckAllInResolution() {
		winner := room.ResolveAllIn()
		rake := room.EndGame(winner)
		settleGameToDB(room, winner, rake)
		cancelTurnTimer(room.ID)
		broadcastGameEnd(room, winner, rake)
		return
	}

	ended, winner := room.CheckGameEnd()
	if ended {
		rake := room.EndGame(winner)
		settleGameToDB(room, winner, rake)
		cancelTurnTimer(room.ID)
		broadcastGameEnd(room, winner, rake)
		return
	}

	room.NextTurn()
	broadcastRoomState(room)
	startTurnTimer(room)
	go processBotTurns(room)
}

func handleNewRound(c *Client) {
	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}

	// 检查发起者是否破产
	player := room.FindPlayer(c.PlayerID)
	if player != nil {
		// Check DB chips too
		dbChips, _ := game.GetOrCreateUser(c.PlayerID, "")
		if dbChips <= 0 {
			sendToClient(c, map[string]interface{}{
				"type": "bankrupt",
				"data": map[string]interface{}{
					"message": "筹码不足，请充值",
					"chips":   0,
				},
			})
			return
		}
	}

	// Mark player as confirmed for next round
	room.ConfirmNextRound(c.PlayerID)

	// Broadcast confirmation status
	broadcastNextRoundStatus(room)

	// If all confirmed, start immediately
	if room.AllConfirmedNextRound() {
		startNextRound(room)
	}
}

// startNextRoundCountdown starts a 30-second countdown after game ends
func startNextRoundCountdown(room *game.Room) {
	roomID := room.ID
	go func() {
		time.Sleep(30 * time.Second)

		hub.mu.RLock()
		r, ok := hub.Rooms[roomID]
		hub.mu.RUnlock()
		if !ok || r.State != game.RoomFinished {
			return
		}

		// Already started by AllConfirmed
		if r.AllConfirmedNextRound() {
			return
		}

		// Kick unconfirmed players
		unconfirmed := r.UnconfirmedPlayers()
		for _, pid := range unconfirmed {
			r.HandlePlayerLeave(pid)
			hub.mu.RLock()
			if client, ok := hub.Clients[pid]; ok {
				client.RoomID = ""
				sendToClient(client, map[string]interface{}{
					"type": "kicked",
					"data": map[string]interface{}{
						"message": "未确认下一局，已被移出房间",
					},
				})
			}
			hub.mu.RUnlock()
		}

		// Start next round with remaining players
		if len(r.Players) >= 2 {
			startNextRound(r)
		} else {
			broadcastRoomState(r)
		}
	}()
}

// startNextRound resets players and starts a new game round
func startNextRound(room *game.Room) {
	for _, p := range room.Players {
		p.Reset()
		p.State = game.PlayerReady
		// Sync chips from DB for human players
		if !p.IsBot {
			dbChips, err := game.GetOrCreateUser(p.ID, "")
			if err == nil {
				p.Chips = dbChips
			}
		}
	}

	if err := room.StartGame(); err != nil {
		return
	}

	broadcastRoomState(room)
	startTurnTimer(room)
	go processBotTurns(room)
}

// broadcastNextRoundStatus broadcasts confirmation status to all players in the room
func broadcastNextRoundStatus(room *game.Room) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	info := room.GetRoomInfo("")
	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			sendToClient(client, map[string]interface{}{
				"type": "next_round_status",
				"data": map[string]interface{}{
					"readyForNext":      info["readyForNext"],
					"nextRoundDeadline": info["nextRoundDeadline"],
				},
			})
		}
	}
}

// handleLeaveRoom handles a player leaving the room
func handleLeaveRoom(c *Client) {
	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}

	wasTurn := room.HandlePlayerLeave(c.PlayerID)
	roomID := c.RoomID
	c.RoomID = ""

	// Send updated chips to leaving player
	dbChips, _ := game.GetOrCreateUser(c.PlayerID, "")
	sendToClient(c, map[string]interface{}{
		"type": "left_room",
		"data": map[string]interface{}{
			"message": "已离开房间",
			"chips":   dbChips,
		},
	})

	// If game was in progress, check game state
	if room.State == game.RoomPlaying {
		// Check all-in resolution
		if room.CheckAllInResolution() {
			winner := room.ResolveAllIn()
			rake := room.EndGame(winner)
			settleGameToDB(room, winner, rake)
			cancelTurnTimer(room.ID)
			broadcastGameEnd(room, winner, rake)
			return
		}

		ended, winner := room.CheckGameEnd()
		if ended {
			rake := room.EndGame(winner)
			settleGameToDB(room, winner, rake)
			cancelTurnTimer(room.ID)
			broadcastGameEnd(room, winner, rake)
			return
		}

		// If it was the leaving player's turn, advance
		if wasTurn {
			// Find next alive player
			room.NextTurn()
			broadcastRoomState(room)
			startTurnTimer(room)
			go processBotTurns(room)
			return
		}
	}

	// Clean up empty rooms
	if len(room.Players) == 0 {
		hub.mu.Lock()
		delete(hub.Rooms, roomID)
		hub.mu.Unlock()
		return
	}

	broadcastRoomState(room)
}

// handleClaimBonus handles daily free bonus claims (3x per day, 500 chips each)
func handleClaimBonus(c *Client) {
	newChips, remainingClaims, err := game.ClaimDailyBonus(c.PlayerID)
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}

	sendToClient(c, map[string]interface{}{
		"type": "bonus_claimed",
		"data": map[string]interface{}{
			"newChips":        newChips,
			"remainingClaims": remainingClaims,
			"amount":          500,
		},
	})
}

// handleVerifyRecharge handles on-chain recharge verification
func handleVerifyRecharge(c *Client, data json.RawMessage) {
	var req VerifyRechargeData
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid recharge data"},
		})
		return
	}

	// Submit the recharge record
	if err := game.SubmitRecharge(c.PlayerID, req.TxHash, req.Chain, req.Amount, req.Points); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "recharge already submitted or error: " + err.Error()},
		})
		return
	}

	// Auto-confirm (in production you'd verify on-chain first)
	_, points, err := game.ConfirmRecharge(req.TxHash)
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "recharge confirmation failed: " + err.Error()},
		})
		return
	}

	// Get updated chips
	newChips, _ := game.GetOrCreateUser(c.PlayerID, "")

	// Update player chips in current room if applicable
	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if ok {
		player := room.FindPlayer(c.PlayerID)
		if player != nil {
			player.Chips = newChips
		}
	}

	sendToClient(c, map[string]interface{}{
		"type": "recharge_confirmed",
		"data": map[string]interface{}{
			"newChips": newChips,
			"txHash":   req.TxHash,
			"points":   points,
		},
	})
}

// handleMatch 匹配对战
func handleMatch(c *Client, data json.RawMessage) {
	var req JoinData
	json.Unmarshal(data, &req)

	if req.PlayerName == "" {
		req.PlayerName = "玩家"
	}

	hub.MatchQueue.mu.Lock()
	defer hub.MatchQueue.mu.Unlock()

	// 检查是否已在队列
	for _, mc := range hub.MatchQueue.clients {
		if mc.PlayerID == c.PlayerID {
			return
		}
	}

	hub.MatchQueue.clients = append(hub.MatchQueue.clients, c)

	sendToClient(c, map[string]interface{}{
		"type": "match_status",
		"data": map[string]interface{}{
			"status":  "matching",
			"players": len(hub.MatchQueue.clients),
		},
	})

	// 广播匹配人数
	for _, mc := range hub.MatchQueue.clients {
		if mc.PlayerID != c.PlayerID {
			sendToClient(mc, map[string]interface{}{
				"type": "match_status",
				"data": map[string]interface{}{
					"status":  "matching",
					"players": len(hub.MatchQueue.clients),
				},
			})
		}
	}

	// 达到2人以上开始匹配
	if len(hub.MatchQueue.clients) >= 2 {
		matched := hub.MatchQueue.clients[:2]
		hub.MatchQueue.clients = hub.MatchQueue.clients[2:]

		// 创建房间
		roomID := fmt.Sprintf("match_%d", time.Now().UnixNano())
		room := game.NewRoom(roomID, 6)

		for i, mc := range matched {
			name := req.PlayerName
			if i > 0 {
				name = "对手" // 简化处理
			}
			player := game.NewPlayer(mc.PlayerID, name, i)
			player.State = game.PlayerReady
			room.AddPlayer(player)
			mc.RoomID = roomID
		}

		// 加几个机器人凑热闹
		room.FillBots(1)

		hub.mu.Lock()
		hub.Rooms[roomID] = room
		hub.mu.Unlock()

		// 通知匹配成功
		for _, mc := range matched {
			sendToClient(mc, map[string]interface{}{
				"type": "match_found",
				"data": room.GetRoomInfo(mc.PlayerID),
			})
		}

		// 自动开始游戏
		go func() {
			time.Sleep(2 * time.Second)
			if err := room.StartGame(); err == nil {
				broadcastRoomState(room)
				startTurnTimer(room)
				go processBotTurns(room)
			}
		}()
	}
}

func handleCancelMatch(c *Client) {
	hub.MatchQueue.mu.Lock()
	defer hub.MatchQueue.mu.Unlock()
	for i, mc := range hub.MatchQueue.clients {
		if mc.PlayerID == c.PlayerID {
			hub.MatchQueue.clients = append(hub.MatchQueue.clients[:i], hub.MatchQueue.clients[i+1:]...)
			break
		}
	}
	sendToClient(c, map[string]interface{}{
		"type": "match_status",
		"data": map[string]interface{}{
			"status": "cancelled",
		},
	})
}

// handleRecharge 模拟充值
func handleRecharge(c *Client, data json.RawMessage) {
	var req RechargeData
	json.Unmarshal(data, &req)

	// Recharge in DB
	newChips, _ := game.RechargeUser(c.PlayerID, req.Amount)

	// Update player chips in room from DB
	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()

	if ok {
		player := room.FindPlayer(c.PlayerID)
		if player != nil {
			dbChips, err := game.GetOrCreateUser(c.PlayerID, "")
			if err == nil {
				player.Chips = dbChips
			} else {
				player.Chips += req.Amount
			}
		}
	}

	sendToClient(c, map[string]interface{}{
		"type": "recharge_success",
		"data": map[string]interface{}{
			"amount":   req.Amount,
			"wallet":   req.Wallet,
			"newChips": newChips,
		},
	})
}

// generateRoomCode 生成随机房间码
func generateRoomCode(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

// JoinByCodeData 通过房间码加入
type JoinByCodeData struct {
	RoomCode   string `json:"roomCode"`
	PlayerName string `json:"playerName"`
	Wallet     string `json:"wallet"`
	Chain      string `json:"chain"`
}

func handleJoinByCode(c *Client, data json.RawMessage) {
	var req JoinByCodeData
	json.Unmarshal(data, &req)

	if req.PlayerName == "" {
		req.PlayerName = "玩家"
	}

	if Mode == ModePE {
		chain := WalletChain(req.Chain)
		if chain == "" {
			chain = DetectChain(req.Wallet)
		}
		if !ValidateWalletAddress(req.Wallet, chain) {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": "PE mode requires a valid wallet address"},
			})
			return
		}
	}

	// 查找房间码对应的房间
	hub.mu.RLock()
	var foundRoom *game.Room
	var foundRoomID string
	for id, room := range hub.Rooms {
		if room.RoomCode == req.RoomCode {
			foundRoom = room
			foundRoomID = id
			break
		}
	}
	hub.mu.RUnlock()

	if foundRoom == nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "房间不存在"},
		})
		return
	}

	player := game.NewPlayer(c.PlayerID, req.PlayerName, 0)
	chips, _ := game.GetOrCreateUser(c.PlayerID, req.PlayerName)
	player.Chips = chips
	if err := foundRoom.AddPlayer(player); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}

	c.RoomID = foundRoomID
	broadcastRoomState(foundRoom)
}

// startTurnTimer starts a 20-second turn timer for the current player
func startTurnTimer(room *game.Room) {
	hub.mu.Lock()
	// Cancel existing timer
	if t, ok := hub.TurnTimers[room.ID]; ok && t != nil {
		t.Stop()
	}

	current := room.CurrentTurnPlayer()
	if current == nil || current.IsBot || room.State != game.RoomPlaying {
		hub.mu.Unlock()
		return
	}

	playerID := current.ID
	turnStart := room.TurnStart

	hub.TurnTimers[room.ID] = time.AfterFunc(20*time.Second, func() {
		// Verify the turn hasn't changed
		currentNow := room.CurrentTurnPlayer()
		if currentNow == nil || currentNow.ID != playerID || room.State != game.RoomPlaying {
			return
		}
		if room.TurnStart != turnStart {
			return
		}

		// Auto fold
		action := game.Action{Type: game.ActionFold}
		event, err := room.ProcessAction(playerID, action)
		if err != nil {
			return
		}

		broadcastEvent(room, event)

		// Check all-in resolution
		if room.CheckAllInResolution() {
			winner := room.ResolveAllIn()
			rake := room.EndGame(winner)
			settleGameToDB(room, winner, rake)
			cancelTurnTimer(room.ID)
			broadcastGameEnd(room, winner, rake)
			return
		}

		ended, winner := room.CheckGameEnd()
		if ended {
			rake := room.EndGame(winner)
			settleGameToDB(room, winner, rake)
			cancelTurnTimer(room.ID)
			broadcastGameEnd(room, winner, rake)
			return
		}

		room.NextTurn()
		broadcastRoomState(room)
		startTurnTimer(room)
		go processBotTurns(room)
	})
	hub.mu.Unlock()
}

func cancelTurnTimer(roomID string) {
	hub.mu.Lock()
	if t, ok := hub.TurnTimers[roomID]; ok && t != nil {
		t.Stop()
		delete(hub.TurnTimers, roomID)
	}
	hub.mu.Unlock()
}

// processBotTurns 处理机器人回合
func processBotTurns(room *game.Room) {
	for {
		current := room.CurrentTurnPlayer()
		if current == nil || !current.IsBot {
			return
		}
		if room.State != game.RoomPlaying {
			return
		}

		time.Sleep(time.Millisecond * 1200)

		action := current.BotAction(room)
		event, err := room.ProcessAction(current.ID, action)
		if err != nil {
			log.Println("Bot action error:", err)
			room.ProcessAction(current.ID, game.Action{Type: game.ActionFold})
			event = &game.GameEvent{Type: "fold", PlayerID: current.ID, PlayerName: current.Name}
		}

		broadcastEvent(room, event)

		// Check all-in resolution after bot action
		if room.CheckAllInResolution() {
			winner := room.ResolveAllIn()
			rake := room.EndGame(winner)
			settleGameToDB(room, winner, rake)
			cancelTurnTimer(room.ID)
			broadcastGameEnd(room, winner, rake)
			return
		}

		ended, winner := room.CheckGameEnd()
		if ended {
			rake := room.EndGame(winner)
			settleGameToDB(room, winner, rake)
			cancelTurnTimer(room.ID)
			broadcastGameEnd(room, winner, rake)
			return
		}

		room.NextTurn()
		broadcastRoomState(room)
		startTurnTimer(room)
	}
}

// settleGameToDB records game results to the database
func settleGameToDB(room *game.Room, winner *game.Player, rake int) {
	if room.HasAllIn && len(room.SidePots) > 0 {
		// Side pot settlement: each pot may have a different winner
		// Track total winnings per player
		winnings := make(map[string]int)
		for _, pot := range room.SidePots {
			if pot.WinnerID != "" {
				winnings[pot.WinnerID] += pot.WinAmount
			}
		}

		// Update DB chips and record transactions for all players
		for _, p := range room.Players {
			if p.IsBot {
				continue
			}
			game.UpdateUserChips(p.ID, p.Chips)
			if w, ok := winnings[p.ID]; ok && w > 0 {
				game.AddTransaction(p.ID, "win", w, room.ID, "")
			} else {
				// Player lost their bet
				if p.BetTotal > 0 {
					game.AddTransaction(p.ID, "loss", p.BetTotal, room.ID, "")
				}
			}
		}

		// Record platform rake
		if rake > 0 {
			rakeWinnerID := ""
			if len(room.SidePots) > 0 {
				rakeWinnerID = room.SidePots[0].WinnerID
			}
			game.RecordRake(room.ID, rakeWinnerID, rake, len(room.Players), room.BaseBet)
		}
		return
	}

	// Normal single-winner settlement
	if winner == nil {
		return
	}
	var losers []string
	var lossAmounts []int
	for _, p := range room.Players {
		if p.ID != winner.ID && !p.IsBot {
			losers = append(losers, p.ID)
			lossAmounts = append(lossAmounts, p.BetTotal)
		}
		// Update DB chips for non-bot players
		if !p.IsBot {
			game.UpdateUserChips(p.ID, p.Chips)
		}
	}
	// Record winner transaction (net winnings = pot - anteTotal)
	if !winner.IsBot {
		netWin := room.Pot - room.AnteTotal
		if netWin < 0 {
			netWin = 0
		}
		game.AddTransaction(winner.ID, "win", netWin, room.ID, "")
	}
	// Record loser transactions
	for i, loserID := range losers {
		game.AddTransaction(loserID, "loss", lossAmounts[i], room.ID, "")
	}
	// Record platform rake
	if rake > 0 {
		game.RecordRake(room.ID, winner.ID, rake, len(room.Players), room.BaseBet)
	}
}

func broadcastRoomState(room *game.Room) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			roomInfo := room.GetRoomInfo(p.ID)
			// Include the player's current DB chips separately for lobby sync (Item 2)
			if !p.IsBot {
				dbChips, err := game.GetOrCreateUser(p.ID, "")
				if err == nil {
					roomInfo["dbChips"] = dbChips
				}
			}
			sendToClient(client, map[string]interface{}{
				"type": "room_state",
				"data": roomInfo,
			})
		}
	}
}

func broadcastEvent(room *game.Room, event *game.GameEvent) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			sendToClient(client, map[string]interface{}{
				"type": "game_event",
				"data": event,
			})
		}
	}
}

func broadcastGameEnd(room *game.Room, winner *game.Player, rake int) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	winnerName := "未知"
	winnerHand := "未知"
	winnerID := ""
	if winner != nil {
		winnerID = winner.ID
		winnerName = winner.Name
		winnerHand = game.HandTypeName(game.HandRank(winner.Hand))
	}

	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			endData := map[string]interface{}{
				"winnerId":          winnerID,
				"winnerName":        winnerName,
				"pot":               room.Pot,
				"anteTotal":         room.AnteTotal,
				"room":              room.GetRoomInfo(p.ID),
				"handType":          winnerHand,
				"rake":              rake,
				"nextRoundDeadline": room.NextRoundDeadline.UnixMilli(),
			}
			if len(room.SidePots) > 0 {
				endData["sidePots"] = room.SidePots
			}
			if !p.IsBot {
				dbChips, err := game.GetOrCreateUser(p.ID, "")
				if err == nil {
					endData["dbChips"] = dbChips
				}
			}
			sendToClient(client, map[string]interface{}{
				"type": "game_end",
				"data": endData,
			})
		}
	}

	// Start 30-second countdown for next round confirmation
	startNextRoundCountdown(room)
}

func sendToClient(c *Client, data interface{}) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	select {
	case c.Send <- msg:
	default:
	}
}

// HandleHealth 健康检查
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
