package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"zhajinhua/game"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// TODO: In production, restrict to specific domains
		return true
	},
}

// Client WebSocket客户端
type Client struct {
	Conn          *websocket.Conn
	PlayerID      string
	RoomID        string
	Authenticated bool // 是否完成认证
	AuthWallet    string
	AuthChain     WalletChain
	AuthNonce     string
	AuthExpiry    time.Time
	Send          chan []byte
}

const authSessionTTL = 12 * time.Hour

var authSessionSecret = []byte("mule-auth-session-secret")

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
	Wallet     string `json:"wallet"` // Web3钱包地址
	Chain      string `json:"chain"`  // "evm", "ton", "sol"
}

type BaseBetData struct {
	BaseBet int `json:"baseBet"`
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

	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	client := &Client{
		Conn:          conn,
		PlayerID:      "", // Not set until authenticated
		Authenticated: false,
		Send:          make(chan []byte, 256),
	}

	hub.mu.Lock()
	hub.Clients[connID] = client
	hub.mu.Unlock()

	sendToClient(client, map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{
			"message": "Please authenticate with your wallet",
			"mode":    string(Mode),
		},
	})

	go clientWriter(client)
	go clientReader(client)
}

func init() {
	if secret := os.Getenv("AUTH_SESSION_SECRET"); secret != "" {
		authSessionSecret = []byte(secret)
	}
}

func clientWriter(c *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg := <-c.Send:
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			// Send ping to detect stale connections
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func clientReader(c *Client) {
	// Set read deadline to detect stale connections (pong timeout)
	c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

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
				player := room.FindPlayer(c.PlayerID)
				spectator := room.FindSpectator(c.PlayerID)
				if spectator != nil {
					room.HandlePlayerLeave(c.PlayerID)
					broadcastRoomState(room)
				} else if player != nil && player.State != game.PlayerFolded {
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

					// Close rooms that no longer have enough players to continue.
					if closeRoomIfInsufficientPlayers(room, "房间人数不足，房间已自动关闭") {
						return
					}
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
		// Reset read deadline on successful message read
		c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))

		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}
		handleMessage(c, msg)
	}
}

// LoginData 登录请求
type LoginData struct {
	Wallet       string `json:"wallet"`     // 钱包地址（PE）或临时ID（TE）
	Chain        string `json:"chain"`      // "evm", "ton", "sol"（可选）
	PlayerName   string `json:"playerName"` // 用户自定义昵称
	Nonce        string `json:"nonce"`
	Signature    string `json:"signature"`
	SessionToken string `json:"sessionToken"`
}

type AuthChallengeRequestData struct {
	Wallet string `json:"wallet"`
	Chain  string `json:"chain"`
}

func buildAuthSessionToken(playerID string, chain WalletChain, expiresAt time.Time) string {
	payload := fmt.Sprintf("%s|%s|%d", playerID, chain, expiresAt.Unix())
	mac := hmac.New(sha256.New, authSessionSecret)
	mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + signature))
}

func validateAuthSessionToken(token string, expectedPlayerID string, expectedChain WalletChain) (time.Time, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid session token")
	}

	parts := strings.Split(string(raw), "|")
	if len(parts) != 4 {
		return time.Time{}, fmt.Errorf("invalid session token payload")
	}
	playerID := parts[0]
	chain := parts[1]
	expiresUnix, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid session token expiry")
	}
	signature := parts[3]

	if playerID != expectedPlayerID || chain != string(expectedChain) {
		return time.Time{}, fmt.Errorf("session token wallet mismatch")
	}

	expiresAt := time.Unix(expiresUnix, 0)
	if time.Now().After(expiresAt) {
		return time.Time{}, fmt.Errorf("session token expired")
	}

	payload := fmt.Sprintf("%s|%s|%d", playerID, chain, expiresUnix)
	mac := hmac.New(sha256.New, authSessionSecret)
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return time.Time{}, fmt.Errorf("invalid session token signature")
	}

	return expiresAt, nil
}

// handleLogin 处理用户认证
func handleLogin(c *Client, data json.RawMessage) {
	var req LoginData
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid login data"},
		})
		return
	}

	// PE mode: require valid wallet address
	var sessionChain WalletChain
	if Mode == ModePE {
		if req.Wallet == "" {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": "wallet address required in PE mode"},
			})
			return
		}
		chain := WalletChain(req.Chain)
		if chain == "" {
			chain = DetectChain(req.Wallet)
		}
		sessionChain = chain
		if !ValidateWalletAddress(req.Wallet, chain) {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": "invalid wallet address"},
			})
			return
		}

		if chain == ChainEVM || chain == ChainSOL {
			if req.SessionToken != "" {
				if _, err := validateAuthSessionToken(req.SessionToken, req.Wallet, chain); err == nil {
					c.AuthNonce = ""
				} else {
					sendToClient(c, map[string]interface{}{
						"type": "session_expired",
						"data": map[string]string{"message": "登录状态已过期，请重新授权"},
					})
					return
				}
			} else {
				if req.Nonce == "" || req.Signature == "" {
					sendToClient(c, map[string]interface{}{
						"type": "error",
						"data": map[string]string{"message": "wallet signature required"},
					})
					return
				}
				if c.AuthNonce == "" || c.AuthNonce != req.Nonce || c.AuthWallet != req.Wallet || c.AuthChain != chain {
					sendToClient(c, map[string]interface{}{
						"type": "error",
						"data": map[string]string{"message": "invalid or expired auth challenge"},
					})
					return
				}
				if time.Now().After(c.AuthExpiry) {
					c.AuthNonce = ""
					sendToClient(c, map[string]interface{}{
						"type": "error",
						"data": map[string]string{"message": "auth challenge expired"},
					})
					return
				}

				message := BuildAuthMessage(req.Wallet, chain, req.Nonce)
				if err := VerifyWalletSignature(req.Wallet, chain, message, req.Signature); err != nil {
					sendToClient(c, map[string]interface{}{
						"type": "error",
						"data": map[string]string{"message": "invalid wallet signature"},
					})
					return
				}
				c.AuthNonce = ""
			}
		}
		// Use wallet address as playerID in PE mode
		c.PlayerID = req.Wallet
	} else {
		// TE mode: use provided ID or generate temporary one
		if req.Wallet != "" {
			c.PlayerID = req.Wallet
		} else {
			c.PlayerID = fmt.Sprintf("te_player_%d", time.Now().UnixNano())
		}
	}

	// Use provided player name or default
	playerName := req.PlayerName
	if playerName == "" {
		playerName = "玩家"
	}

	// Create or get user from DB
	chips, _ := game.GetOrCreateUser(c.PlayerID, playerName)

	// Get the actual stored player name from DB
	storedChips, storedName, err := game.GetUserProfile(c.PlayerID)
	if err == nil {
		chips = storedChips
		if storedName != "" {
			playerName = storedName
		}
	}

	// If wallet provided in PE mode, save wallet info
	if Mode == ModePE && req.Wallet != "" {
		chain := WalletChain(req.Chain)
		if chain == "" {
			chain = DetectChain(req.Wallet)
		}
		game.GetOrCreateUserWithWallet(c.PlayerID, playerName, req.Wallet, string(chain))
	}

	// Mark as authenticated
	c.Authenticated = true
	c.AuthWallet = ""
	c.AuthChain = ""
	c.AuthNonce = ""
	c.AuthExpiry = time.Time{}

	// Update hub clients map to use playerID as key
	hub.mu.Lock()
	// Find and remove old connection entry
	for key, client := range hub.Clients {
		if client == c {
			delete(hub.Clients, key)
			break
		}
	}
	// Add with new playerID
	hub.Clients[c.PlayerID] = c
	hub.mu.Unlock()

	sendToClient(c, map[string]interface{}{
		"type": "authenticated",
		"data": map[string]interface{}{
			"playerId":         c.PlayerID,
			"playerName":       playerName, // NEW: return saved player name
			"chips":            chips,
			"mode":             string(Mode),
			"chain":            string(sessionChain),
			"sessionToken":     buildAuthSessionToken(c.PlayerID, sessionChain, time.Now().Add(authSessionTTL)),
			"sessionExpiresAt": time.Now().Add(authSessionTTL).UnixMilli(),
		},
	})
}

func handleMessage(c *Client, msg Message) {
	// 大部分操作需要认证，除了login/auth
	if !c.Authenticated && msg.Type != "login" && msg.Type != "auth" && msg.Type != "auth_challenge_request" {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "not authenticated"},
		})
		return
	}

	switch msg.Type {
	case "login", "auth":
		handleLogin(c, msg.Data)
	case "auth_challenge_request":
		handleAuthChallengeRequest(c, msg.Data)
	case "create_room":
		handleCreateRoom(c, msg.Data)
	case "join_room":
		handleJoinRoom(c, msg.Data)
	case "ready":
		handleReady(c)
	case "update_base_bet":
		handleUpdateBaseBet(c, msg.Data)
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
	case "check_bonus":
		handleCheckBonus(c)
	case "verify_recharge":
		handleVerifyRecharge(c, msg.Data)
	}
}

func handleAuthChallengeRequest(c *Client, data json.RawMessage) {
	var req AuthChallengeRequestData
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid auth challenge request"},
		})
		return
	}

	if Mode != ModePE {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "auth challenge is only required in PE mode"},
		})
		return
	}

	chain := WalletChain(req.Chain)
	if chain == "" {
		chain = DetectChain(req.Wallet)
	}
	if !ValidateWalletAddress(req.Wallet, chain) {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid wallet address"},
		})
		return
	}
	if chain != ChainEVM && chain != ChainSOL {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "auth challenge is currently supported for EVM and SOL only"},
		})
		return
	}

	nonce, err := GenerateAuthNonce()
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "failed to generate auth challenge"},
		})
		return
	}

	c.AuthWallet = req.Wallet
	c.AuthChain = chain
	c.AuthNonce = nonce
	c.AuthExpiry = time.Now().Add(AuthChallengeTTLSeconds * time.Second)

	sendToClient(c, map[string]interface{}{
		"type": "auth_challenge",
		"data": map[string]interface{}{
			"wallet":    req.Wallet,
			"chain":     chain,
			"nonce":     nonce,
			"message":   BuildAuthMessage(req.Wallet, chain, nonce),
			"expiresIn": AuthChallengeTTLSeconds,
		},
	})
}

func handleCreateRoom(c *Client, data json.RawMessage) {
	var req JoinData
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid request data"},
		})
		return
	}

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
	room.OwnerID = c.PlayerID
	room.IsPrivate = req.BotCount == 0

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
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid request data"},
		})
		return
	}

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
	if room.State == game.RoomPlaying {
		if err := room.AddSpectator(player); err != nil {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": err.Error()},
			})
			return
		}
	} else if err := room.AddPlayer(player); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}

	c.RoomID = req.RoomID
	broadcastRoomState(room)
}

func handleUpdateBaseBet(c *Client, data json.RawMessage) {
	var req BaseBetData
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid request data"},
		})
		return
	}
	if req.BaseBet <= 0 {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "底注必须大于 0"},
		})
		return
	}

	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}
	if room.OwnerID != c.PlayerID {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "只有房主可以修改底注"},
		})
		return
	}
	if room.State == game.RoomPlaying {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "游戏进行中不能修改底注"},
		})
		return
	}

	room.BaseBet = req.BaseBet
	room.CurrentBet = req.BaseBet
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

	// Safeguard: verify player is still in room
	player := room.FindPlayer(c.PlayerID)
	if player == nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "You are not in this room"},
		})
		return
	}
	if room.IsPrivate && room.OwnerID != c.PlayerID {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "只有房主可以开始私有房游戏"},
		})
		return
	}

	if err := room.StartGame(); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}
	syncRoomAntesToDB(room)

	broadcastRoomState(room)
	startTurnTimer(room)
	go processBotTurns(room)
}

func handleAction(c *Client, data json.RawMessage) {
	var req ActionData
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid request data"},
		})
		return
	}

	hub.mu.RLock()
	room, ok := hub.Rooms[c.RoomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}

	// Safeguard: verify player is still in room
	player := room.FindPlayer(c.PlayerID)
	if player == nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "You are not in this room"},
		})
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

	// NEW: Deduct from DB immediately for betting actions
	// Calculate how much was actually deducted from player's room chips
	if action.Type == game.ActionCall || action.Type == game.ActionRaise ||
		action.Type == game.ActionAllIn || action.Type == game.ActionCompare {
		if event != nil && event.Amount > 0 {
			// Deduct from DB account
			game.DeductUserChips(c.PlayerID, event.Amount)
		}
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

	// Safeguard: verify player is still in room
	player := room.FindPlayer(c.PlayerID)
	if player == nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "You are not in this room"},
		})
		return
	}

	// Check bankrupt status
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

		// Remove all unconfirmed human players before the next round starts.
		// At this point the previous hand is already settled, so keeping them
		// in the room would resurrect stale/folded players in the next round.
		unconfirmed := r.UnconfirmedPlayers()
		for _, pid := range unconfirmed {
			player := r.FindPlayer(pid)
			if player == nil {
				continue
			}

			r.RemovePlayer(pid)
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
			closeRoomIfInsufficientPlayers(r, "房间人数不足，房间已自动关闭")
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
	for _, spectator := range room.Spectators {
		if spectator == nil {
			continue
		}
		spectator.State = game.PlayerWaiting
		spectator.BetTotal = 0
		spectator.Looked = false
		spectator.Hand = game.Hand{}
	}

	if err := room.StartGame(); err != nil {
		return
	}
	syncRoomAntesToDB(room)

	broadcastRoomState(room)
	startTurnTimer(room)
	go processBotTurns(room)
}

func closeRoomIfInsufficientPlayers(room *game.Room, message string) bool {
	if len(room.Players) >= 2 {
		return false
	}

	cancelTurnTimer(room.ID)

	hub.mu.RLock()
	for _, player := range room.Players {
		if client, ok := hub.Clients[player.ID]; ok {
			client.RoomID = ""
			sendToClient(client, map[string]interface{}{
				"type": "kicked",
				"data": map[string]interface{}{
					"message": message,
				},
			})
		}
	}
	for _, spectator := range room.Spectators {
		if client, ok := hub.Clients[spectator.ID]; ok {
			client.RoomID = ""
			sendToClient(client, map[string]interface{}{
				"type": "kicked",
				"data": map[string]interface{}{
					"message": message,
				},
			})
		}
	}
	hub.mu.RUnlock()

	hub.mu.Lock()
	delete(hub.Rooms, room.ID)
	hub.mu.Unlock()
	return true
}

func destroyRoomIfAbandoned(room *game.Room) bool {
	if len(room.Players) > 0 || len(room.Spectators) > 0 {
		return false
	}
	cancelTurnTimer(room.ID)
	hub.mu.Lock()
	delete(hub.Rooms, room.ID)
	hub.mu.Unlock()
	return true
}

// broadcastNextRoundStatus broadcasts confirmation status to all players in the room
func broadcastNextRoundStatus(room *game.Room) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	info := room.GetRoomInfo("")
	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			// Safeguard: only broadcast to players still connected to this room
			if client.RoomID != room.ID {
				continue
			}
			sendToClient(client, map[string]interface{}{
				"type": "next_round_status",
				"data": map[string]interface{}{
					"readyForNext":      info["readyForNext"],
					"nextRoundDeadline": info["nextRoundDeadline"],
				},
			})
		}
	}
	for _, p := range room.Spectators {
		if client, ok := hub.Clients[p.ID]; ok {
			if client.RoomID != room.ID {
				continue
			}
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
	// Atomically clear room association to prevent race conditions
	hub.mu.Lock()
	roomID := c.RoomID
	c.RoomID = ""
	hub.mu.Unlock()

	if roomID == "" {
		return
	}

	hub.mu.RLock()
	room, ok := hub.Rooms[roomID]
	hub.mu.RUnlock()
	if !ok {
		return
	}

	wasTurn := room.HandlePlayerLeave(c.PlayerID)

	// Send current chips to leaving player
	// Use the actual chips value from the room, not stale DB value
	player := room.FindPlayer(c.PlayerID)
	var currentChips int
	if player != nil {
		currentChips = player.Chips // Use real-time room chips
	} else {
		currentChips, _ = game.GetOrCreateUser(c.PlayerID, "") // Fallback to DB
	}

	sendToClient(c, map[string]interface{}{
		"type": "left_room",
		"data": map[string]interface{}{
			"message": "已离开房间",
			"chips":   currentChips,
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
			room.RemovePlayer(c.PlayerID)
			broadcastGameEnd(room, winner, rake)
			return
		}

		ended, winner := room.CheckGameEnd()
		if ended {
			rake := room.EndGame(winner)
			settleGameToDB(room, winner, rake)
			cancelTurnTimer(room.ID)
			room.RemovePlayer(c.PlayerID)
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
	if destroyRoomIfAbandoned(room) {
		return
	}
	if closeRoomIfInsufficientPlayers(room, "房间人数不足，房间已自动关闭") {
		return
	}

	broadcastRoomState(room)
	if room.State == game.RoomFinished {
		broadcastNextRoundStatus(room)
	}
}

// handleClaimBonus handles daily free bonus claims (3x per day, 500 chips each)
// After 3 claims, users must recharge via smart contracts
func handleClaimBonus(c *Client) {
	// Check current bonus status first
	claimed, remaining, canClaim, err := game.GetDailyBonusStatus(c.PlayerID)
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "failed to check bonus status: " + err.Error()},
		})
		return
	}

	// Return status even if can't claim (for UI to show recharge button)
	statusResp := map[string]interface{}{
		"type": "bonus_status",
		"data": map[string]interface{}{
			"claimed":   claimed,
			"remaining": remaining,
			"canClaim":  canClaim,
			"amount":    500,
		},
	}

	// If no remaining claims, prompt for recharge instead of claiming
	if !canClaim {
		statusResp["data"].(map[string]interface{})["message"] = "daily free bonus exhausted, please recharge"
		sendToClient(c, statusResp)
		return
	}

	// Claim the bonus
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

// handleCheckBonus checks the current daily bonus status without claiming
func handleCheckBonus(c *Client) {
	claimed, remaining, canClaim, err := game.GetDailyBonusStatus(c.PlayerID)
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "failed to check bonus status: " + err.Error()},
		})
		return
	}

	respData := map[string]interface{}{
		"claimed":   claimed,
		"remaining": remaining,
		"canClaim":  canClaim,
		"amount":    500,
	}

	// If no remaining claims, include recharge message
	if !canClaim {
		respData["message"] = "daily free bonus exhausted, please recharge"
	}

	sendToClient(c, map[string]interface{}{
		"type": "bonus_status",
		"data": respData,
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
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid request data"},
		})
		return
	}

	if req.PlayerName == "" {
		req.PlayerName = "玩家"
	}
	if req.BaseBet <= 0 {
		req.BaseBet = 10
	}

	hub.mu.Lock()
	var room *game.Room
	var roomID string
	for id, existing := range hub.Rooms {
		if existing.IsMatch && existing.BaseBet == req.BaseBet && existing.State == game.RoomWaiting && len(existing.Players) < 2 {
			room = existing
			roomID = id
			break
		}
	}
	if room == nil {
		for id, existing := range hub.Rooms {
			if existing.IsMatch && existing.BaseBet == req.BaseBet && existing.State == game.RoomPlaying {
				room = existing
				roomID = id
				break
			}
		}
	}
	if room == nil {
		roomID = fmt.Sprintf("match_%d", time.Now().UnixNano())
		room = game.NewRoom(roomID, 2)
		room.IsMatch = true
		room.OwnerID = c.PlayerID
		room.BaseBet = req.BaseBet
		room.CurrentBet = req.BaseBet
		hub.Rooms[roomID] = room
	}
	hub.mu.Unlock()

	name := "玩家"
	if _, storedName, err := game.GetUserProfile(c.PlayerID); err == nil && storedName != "" {
		name = storedName
	}
	chips, err := game.GetOrCreateUser(c.PlayerID, name)
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "读取玩家筹码失败"},
		})
		return
	}

	player := game.NewPlayer(c.PlayerID, name, 0)
	player.Chips = chips
	if room.State == game.RoomPlaying {
		if err := room.AddSpectator(player); err != nil {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": err.Error()},
			})
			return
		}
	} else {
		player.State = game.PlayerReady
		if err := room.AddPlayer(player); err != nil {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": err.Error()},
			})
			return
		}
		player.Avatar = player.Seat%2 + 1
	}
	c.RoomID = roomID

	if room.State == game.RoomPlaying {
		sendToClient(c, map[string]interface{}{
			"type": "match_found",
			"data": room.GetRoomInfo(c.PlayerID),
		})
		return
	}

	if len(room.Players) < 2 {
		sendToClient(c, map[string]interface{}{
			"type": "match_status",
			"data": map[string]interface{}{
				"status":  "matching",
				"players": len(room.Players),
			},
		})
		return
	}

	for _, matchedPlayer := range room.Players {
		if client, ok := hub.Clients[matchedPlayer.ID]; ok {
			client.RoomID = roomID
			sendToClient(client, map[string]interface{}{
				"type": "match_found",
				"data": room.GetRoomInfo(matchedPlayer.ID),
			})
		}
	}

	go func() {
		time.Sleep(2 * time.Second)
		if err := room.StartGame(); err == nil {
			syncRoomAntesToDB(room)
			broadcastRoomState(room)
			startTurnTimer(room)
		}
	}()
}

func handleCancelMatch(c *Client) {
	if c.RoomID != "" {
		hub.mu.RLock()
		room, ok := hub.Rooms[c.RoomID]
		hub.mu.RUnlock()
		if ok && room.IsMatch && room.State == game.RoomWaiting {
			room.RemovePlayer(c.PlayerID)
			hub.mu.Lock()
			if len(room.Players) == 0 {
				delete(hub.Rooms, room.ID)
			}
			hub.mu.Unlock()
			c.RoomID = ""
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
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid request data"},
		})
		return
	}

	_, remaining, canClaim, err := game.GetDailyBonusStatus(c.PlayerID)
	if err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "failed to check daily bonus status"},
		})
		return
	}

	if canClaim {
		if req.Amount != 500 {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": "有剩余赠送次数时，每次只能领取 500"},
			})
			return
		}

		newChips, remainingClaims, err := game.ClaimDailyBonus(c.PlayerID)
		if err != nil {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": "领取赠送失败: " + err.Error()},
			})
			return
		}

		// Update player chips in room from DB
		hub.mu.RLock()
		room, ok := hub.Rooms[c.RoomID]
		hub.mu.RUnlock()
		if ok {
			if player := room.FindPlayer(c.PlayerID); player != nil {
				player.Chips = newChips
			}
		}

		sendToClient(c, map[string]interface{}{
			"type": "recharge_success",
			"data": map[string]interface{}{
				"amount":          500,
				"wallet":          req.Wallet,
				"newChips":        newChips,
				"remainingClaims": remainingClaims,
				"usedDailyBonus":  true,
			},
		})
		return
	}

	if remaining == 0 {
		// User has claimed all 3 free bonuses today
		// Contract-based recharge is not yet configured
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{
				"message": "Smart contract for recharge is not yet configured. Please try again later.",
				"code":    "contract_not_configured",
			},
		})
		return
	}
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
	if err := json.Unmarshal(data, &req); err != nil {
		sendToClient(c, map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": "invalid request data"},
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
	if foundRoom.State == game.RoomPlaying {
		if err := foundRoom.AddSpectator(player); err != nil {
			sendToClient(c, map[string]interface{}{
				"type": "error",
				"data": map[string]string{"message": err.Error()},
			})
			return
		}
	} else if err := foundRoom.AddPlayer(player); err != nil {
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
		// Side pot settlement: each pot's winner gets their winnings
		// All player bets are already deducted from DB during the game
		// Only add winnings to winners
		for _, pot := range room.SidePots {
			if pot.WinnerID != "" && !room.FindPlayer(pot.WinnerID).IsBot {
				game.AddUserChips(pot.WinnerID, pot.WinAmount)
				game.AddTransaction(pot.WinnerID, "win", pot.WinAmount, room.ID, "")
			}
		}

		// Record platform rake from the winner's perspective
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
	if winner == nil || winner.IsBot {
		return
	}

	// Bets were already deducted from the DB during play, so only credit
	// the net winnings after rake here.
	winnings := room.Pot - rake
	if winnings < 0 {
		winnings = 0
	}
	game.AddUserChips(winner.ID, winnings)
	game.AddTransaction(winner.ID, "win", winnings, room.ID, "")

	// Record platform rake
	if rake > 0 {
		game.RecordRake(room.ID, winner.ID, rake, len(room.Players), room.BaseBet)
	}
}

func syncRoomAntesToDB(room *game.Room) {
	for _, player := range room.Players {
		if player == nil || player.IsBot || player.BetTotal <= 0 {
			continue
		}

		anteAmount := player.BetTotal
		if err := game.DeductUserChips(player.ID, anteAmount); err != nil {
			// The room state is already authoritative at this point, so fall back
			// to syncing the user's DB balance to the room balance.
			if syncErr := game.UpdateUserChips(player.ID, player.Chips); syncErr != nil {
				log.Printf("failed to sync ante for player %s in room %s: deduct err=%v sync err=%v", player.ID, room.ID, err, syncErr)
				continue
			}
		}

		if err := game.AddTransaction(player.ID, "loss", anteAmount, room.ID, "game ante"); err != nil {
			log.Printf("failed to record ante transaction for player %s in room %s: %v", player.ID, room.ID, err)
		}
	}
}

func broadcastRoomState(room *game.Room) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			// Safeguard: only broadcast to players still connected to this room
			// This prevents players who have already left from being "pulled back"
			if client.RoomID != room.ID {
				continue
			}
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
	for _, p := range room.Spectators {
		if client, ok := hub.Clients[p.ID]; ok {
			if client.RoomID != room.ID {
				continue
			}
			sendToClient(client, map[string]interface{}{
				"type": "room_state",
				"data": room.GetRoomInfo(p.ID),
			})
		}
	}
}

func broadcastEvent(room *game.Room, event *game.GameEvent) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			// Safeguard: only broadcast game events to players still connected to this room
			if client.RoomID != room.ID {
				continue
			}
			sendToClient(client, map[string]interface{}{
				"type": "game_event",
				"data": event,
			})
		}
	}
	for _, p := range room.Spectators {
		if client, ok := hub.Clients[p.ID]; ok {
			sendToClient(client, map[string]interface{}{
				"type": "game_event",
				"data": event,
			})
		}
	}
}

func broadcastGameEnd(room *game.Room, winner *game.Player, rake int) {
	room.PromoteSpectatorsForNextRound()

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	winnerName := "未知"
	winnerHand := "未知"
	winnerHandKey := "hand.unknown"
	winnerID := ""
	if winner != nil {
		winnerID = winner.ID
		winnerName = winner.Name
		winnerRank := game.HandRank(winner.Hand)
		winnerHand = game.HandTypeName(winnerRank)
		winnerHandKey = game.HandTypeKey(winnerRank)
	}

	for _, p := range room.Players {
		if client, ok := hub.Clients[p.ID]; ok {
			// Safeguard: only broadcast game end to players still connected to this room
			// Leave early to avoid sending settlement data to players who already left
			if client.RoomID != room.ID {
				continue
			}
			endData := map[string]interface{}{
				"winnerId":          winnerID,
				"winnerName":        winnerName,
				"pot":               room.Pot,
				"anteTotal":         room.AnteTotal,
				"room":              room.GetRoomInfo(p.ID),
				"handType":          winnerHand,
				"handTypeKey":       winnerHandKey,
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
	for _, p := range room.Spectators {
		if client, ok := hub.Clients[p.ID]; ok {
			if client.RoomID != room.ID {
				continue
			}
			endData := map[string]interface{}{
				"winnerId":          winnerID,
				"winnerName":        winnerName,
				"pot":               room.Pot,
				"anteTotal":         room.AnteTotal,
				"room":              room.GetRoomInfo(p.ID),
				"handType":          winnerHand,
				"handTypeKey":       winnerHandKey,
				"rake":              rake,
				"nextRoundDeadline": room.NextRoundDeadline.UnixMilli(),
			}
			if len(room.SidePots) > 0 {
				endData["sidePots"] = room.SidePots
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
