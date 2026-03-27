package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"zhajinhua/game"
)

func setupHandlerTestDB(t *testing.T) func() {
	t.Helper()
	tmpFile := filepath.Join(os.TempDir(), "zhajinhua_handler_test.db")
	_ = os.Remove(tmpFile)
	_ = os.Remove(tmpFile + "-wal")
	_ = os.Remove(tmpFile + "-shm")
	game.InitDB(tmpFile)
	return func() {
		if game.DB != nil {
			game.DB.Close()
		}
		_ = os.Remove(tmpFile)
		_ = os.Remove(tmpFile + "-wal")
		_ = os.Remove(tmpFile + "-shm")
	}
}

func resetHubForTest() {
	hub = &Hub{
		Rooms:      make(map[string]*game.Room),
		Clients:    make(map[string]*Client),
		TurnTimers: make(map[string]*time.Timer),
	}
}

func TestSettleGameToDB_CreditsNetWinningsOnly(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	startChips, err := game.GetOrCreateUser("winner1", "Winner")
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}
	if startChips != 1500 {
		t.Fatalf("expected signup chips 1500, got %d", startChips)
	}

	room := game.NewRoom("room1", 2)
	winner := game.NewPlayer("winner1", "Winner", 0)
	loser := game.NewPlayer("loser1", "Loser", 1)
	room.Players = []*game.Player{winner, loser}
	room.Pot = 100
	room.BaseBet = 10

	settleGameToDB(room, winner, 20)

	finalChips, _, err := game.GetUserProfile("winner1")
	if err != nil {
		t.Fatalf("GetUserProfile failed: %v", err)
	}
	if finalChips != 1580 {
		t.Fatalf("expected winner to receive only net pot 80, got %d", finalChips)
	}
}

func TestSyncRoomAntesToDB_DeductsHumanAntes(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	aliceChips, err := game.GetOrCreateUser("p1", "Alice")
	if err != nil {
		t.Fatalf("GetOrCreateUser failed for p1: %v", err)
	}
	bobChips, err := game.GetOrCreateUser("p2", "Bob")
	if err != nil {
		t.Fatalf("GetOrCreateUser failed for p2: %v", err)
	}

	room := game.NewRoom("room1", 2)
	p1 := game.NewPlayer("p1", "Alice", 0)
	p2 := game.NewPlayer("p2", "Bob", 1)
	p1.Chips = aliceChips
	p2.Chips = bobChips
	if err := room.AddPlayer(p1); err != nil {
		t.Fatalf("AddPlayer p1 failed: %v", err)
	}
	if err := room.AddPlayer(p2); err != nil {
		t.Fatalf("AddPlayer p2 failed: %v", err)
	}

	if err := room.StartGame(); err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	syncRoomAntesToDB(room)

	finalAlice, _, err := game.GetUserProfile("p1")
	if err != nil {
		t.Fatalf("GetUserProfile p1 failed: %v", err)
	}
	finalBob, _, err := game.GetUserProfile("p2")
	if err != nil {
		t.Fatalf("GetUserProfile p2 failed: %v", err)
	}

	if finalAlice != aliceChips-room.BaseBet {
		t.Fatalf("expected p1 DB chips %d, got %d", aliceChips-room.BaseBet, finalAlice)
	}
	if finalBob != bobChips-room.BaseBet {
		t.Fatalf("expected p2 DB chips %d, got %d", bobChips-room.BaseBet, finalBob)
	}
}

func TestHandleRecharge_UsesDailyBonusClaim(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	resetHubForTest()

	if _, err := game.GetOrCreateUser("p1", "Alice"); err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	client := &Client{PlayerID: "p1", Send: make(chan []byte, 1)}
	payload := []byte(`{"amount":500,"wallet":"0xabc"}`)

	handleRecharge(client, payload)

	var msg map[string]interface{}
	select {
	case raw := <-client.Send:
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal recharge response failed: %v", err)
		}
	default:
		t.Fatal("expected recharge response")
	}

	if msg["type"] != "recharge_success" {
		t.Fatalf("expected recharge_success, got %v", msg["type"])
	}

	claimed, remaining, canClaim, err := game.GetDailyBonusStatus("p1")
	if err != nil {
		t.Fatalf("GetDailyBonusStatus failed: %v", err)
	}
	if claimed != 1 || remaining != 2 || !canClaim {
		t.Fatalf("expected daily bonus to be consumed once, got claimed=%d remaining=%d canClaim=%v", claimed, remaining, canClaim)
	}
}

func TestHandleMatch_ThirdPlayerBecomesSpectator(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	resetHubForTest()

	room := game.NewRoom("match_room", 2)
	room.IsMatch = true
	room.State = game.RoomPlaying
	room.OwnerID = "p1"
	room.RoomCode = "MATCH1"
	room.Players = []*game.Player{
		game.NewPlayer("p1", "Alice", 0),
		game.NewPlayer("p2", "Bob", 1),
	}
	hub.Rooms[room.ID] = room

	client := &Client{PlayerID: "p3", Send: make(chan []byte, 1)}
	hub.Clients["p3"] = client

	payload := []byte(`{"playerName":"Carol","wallet":"0x123","chain":"evm"}`)
	handleMatch(client, payload)

	if client.RoomID != room.ID {
		t.Fatalf("expected third client to join existing match room %s, got %s", room.ID, client.RoomID)
	}
	if room.FindSpectator("p3") == nil {
		t.Fatal("expected third client to become spectator")
	}
	if len(room.Players) != 2 {
		t.Fatalf("expected playing match room to remain 2 players, got %d", len(room.Players))
	}
}

func TestStartNextRound_KeepsSpectators(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	room := game.NewRoom("room1", 2)
	p1 := game.NewPlayer("p1", "Alice", 0)
	p2 := game.NewPlayer("p2", "Bob", 1)
	spectator := game.NewPlayer("s1", "Spectator", 0)
	spectator.IsSpectator = true
	spectator.State = game.PlayerFolded
	spectator.BetTotal = 20

	room.Players = []*game.Player{p1, p2}
	room.Spectators = []*game.Player{spectator}

	if _, err := game.GetOrCreateUser("p1", "Alice"); err != nil {
		t.Fatalf("GetOrCreateUser p1 failed: %v", err)
	}
	if _, err := game.GetOrCreateUser("p2", "Bob"); err != nil {
		t.Fatalf("GetOrCreateUser p2 failed: %v", err)
	}

	startNextRound(room)

	if len(room.Spectators) != 1 {
		t.Fatalf("expected spectator to remain attached, got %d spectators", len(room.Spectators))
	}
	if room.Spectators[0].ID != "s1" {
		t.Fatalf("expected spectator s1 to remain, got %s", room.Spectators[0].ID)
	}
	if room.Spectators[0].BetTotal != 0 || room.Spectators[0].Looked {
		t.Fatalf("expected spectator transient state to reset, got betTotal=%d looked=%v", room.Spectators[0].BetTotal, room.Spectators[0].Looked)
	}
}

func TestHandleLeaveRoom_DestroysAbandonedFinishedRoom(t *testing.T) {
	resetHubForTest()

	room := game.NewRoom("room_finished", 2)
	room.State = game.RoomFinished
	room.Players = []*game.Player{
		game.NewPlayer("p1", "Alice", 0),
	}
	hub.Rooms[room.ID] = room

	client := &Client{PlayerID: "p1", RoomID: room.ID, Send: make(chan []byte, 1)}

	handleLeaveRoom(client)

	if _, ok := hub.Rooms[room.ID]; ok {
		t.Fatalf("expected finished room %s to be destroyed after everyone left", room.ID)
	}
}

func TestBroadcastGameEnd_PromotesSpectatorsToPlayers(t *testing.T) {
	resetHubForTest()

	room := game.NewRoom("room_end", 2)
	room.State = game.RoomFinished
	p1 := game.NewPlayer("p1", "Alice", 0)
	p2 := game.NewPlayer("p2", "Bob", 1)
	spec := game.NewPlayer("s1", "Carol", 0)
	spec.IsSpectator = true
	room.Players = []*game.Player{p1, p2}
	room.Spectators = []*game.Player{spec}

	broadcastGameEnd(room, p1, 0)

	if len(room.Spectators) != 0 {
		t.Fatalf("expected spectators to be promoted before game_end broadcast, got %d spectators", len(room.Spectators))
	}
	if room.FindPlayer("s1") == nil {
		t.Fatal("expected promoted spectator to become a normal player")
	}
	if room.FindPlayer("s1").IsSpectator {
		t.Fatal("expected promoted spectator to no longer be marked as spectator")
	}
}

func TestHandleMatch_UsesBaseBetSpecificWaitingRoom(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	resetHubForTest()

	room10 := game.NewRoom("match10", 2)
	room10.IsMatch = true
	room10.BaseBet = 10
	room10.CurrentBet = 10
	room10.Players = []*game.Player{game.NewPlayer("p1", "Alice", 0)}
	hub.Rooms[room10.ID] = room10

	room20 := game.NewRoom("match20", 2)
	room20.IsMatch = true
	room20.BaseBet = 20
	room20.CurrentBet = 20
	room20.Players = []*game.Player{game.NewPlayer("p2", "Bob", 0)}
	hub.Rooms[room20.ID] = room20

	client := &Client{PlayerID: "p3", Send: make(chan []byte, 2)}
	hub.Clients["p3"] = client

	payload := []byte(`{"playerName":"Carol","wallet":"0x123","chain":"evm","baseBet":20}`)
	handleMatch(client, payload)

	if client.RoomID != room20.ID {
		t.Fatalf("expected player to join baseBet=20 room, got %s", client.RoomID)
	}
}
