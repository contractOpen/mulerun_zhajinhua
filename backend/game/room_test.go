package game

import (
	"testing"
)

func TestNewRoom(t *testing.T) {
	r := NewRoom("room1", 5)
	if r.ID != "room1" {
		t.Errorf("expected ID room1, got %s", r.ID)
	}
	if r.MaxPlayers != 5 {
		t.Errorf("expected MaxPlayers 5, got %d", r.MaxPlayers)
	}
	if r.State != RoomWaiting {
		t.Errorf("expected RoomWaiting, got %d", r.State)
	}
	if r.BaseBet != 10 {
		t.Errorf("expected BaseBet 10, got %d", r.BaseBet)
	}
	if r.CurrentBet != 10 {
		t.Errorf("expected CurrentBet 10, got %d", r.CurrentBet)
	}
	if r.GoodBias != 3 {
		t.Errorf("expected GoodBias 3, got %d", r.GoodBias)
	}
	if len(r.Players) != 0 {
		t.Errorf("expected 0 players, got %d", len(r.Players))
	}
}

func TestAddPlayer(t *testing.T) {
	r := NewRoom("room1", 2)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)

	if err := r.AddPlayer(p1); err != nil {
		t.Fatalf("failed to add p1: %v", err)
	}
	if err := r.AddPlayer(p2); err != nil {
		t.Fatalf("failed to add p2: %v", err)
	}
	if len(r.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(r.Players))
	}
}

func TestAddPlayer_RoomFull(t *testing.T) {
	r := NewRoom("room1", 1)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)

	_ = r.AddPlayer(p1)
	err := r.AddPlayer(p2)
	if err == nil {
		t.Error("expected error when room is full")
	}
}

func TestAddPlayer_GameInProgress(t *testing.T) {
	r := NewRoom("room1", 5)
	r.State = RoomPlaying

	p := NewPlayer("p1", "Alice", 0)
	err := r.AddPlayer(p)
	if err == nil {
		t.Error("expected error when game is in progress")
	}
}

func TestRemovePlayer(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p3 := NewPlayer("p3", "Charlie", 2)
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.AddPlayer(p3)

	r.RemovePlayer("p2")

	if len(r.Players) != 2 {
		t.Fatalf("expected 2 players after removal, got %d", len(r.Players))
	}
	// Check seat reassignment
	if r.Players[0].ID != "p1" || r.Players[0].Seat != 0 {
		t.Errorf("p1 seat incorrect: got seat %d", r.Players[0].Seat)
	}
	if r.Players[1].ID != "p3" || r.Players[1].Seat != 1 {
		t.Errorf("p3 should have seat 1 after reassignment, got seat %d", r.Players[1].Seat)
	}
}

func TestFindPlayer_Found(t *testing.T) {
	r := NewRoom("room1", 5)
	p := NewPlayer("p1", "Alice", 0)
	_ = r.AddPlayer(p)

	found := r.FindPlayer("p1")
	if found == nil {
		t.Fatal("expected to find player p1")
	}
	if found.ID != "p1" {
		t.Errorf("expected ID p1, got %s", found.ID)
	}
}

func TestFindPlayer_NotFound(t *testing.T) {
	r := NewRoom("room1", 5)
	found := r.FindPlayer("nonexistent")
	if found != nil {
		t.Error("expected nil for non-existent player")
	}
}

func TestAlivePlayers(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p1.State = PlayerPlaying
	p2 := NewPlayer("p2", "Bob", 1)
	p2.State = PlayerFolded
	p3 := NewPlayer("p3", "Charlie", 2)
	p3.State = PlayerAllIn
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.AddPlayer(p3)

	alive := r.AlivePlayers()
	if len(alive) != 2 {
		t.Errorf("expected 2 alive players, got %d", len(alive))
	}
}

func TestStartGame(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.Chips = 500
	p2.Chips = 500
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)

	err := r.StartGame()
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	if r.State != RoomPlaying {
		t.Errorf("expected RoomPlaying, got %d", r.State)
	}
	if r.Round != 1 {
		t.Errorf("expected round 1, got %d", r.Round)
	}
	// Each player should have been dealt a hand
	for _, p := range r.Players {
		emptyHand := Hand{}
		if p.Hand == emptyHand {
			t.Errorf("player %s should have a hand dealt", p.ID)
		}
		if p.State != PlayerPlaying {
			t.Errorf("player %s should be PlayerPlaying, got %d", p.ID, p.State)
		}
	}
	// Ante should be deducted
	if r.Pot != 20 { // 2 players * 10 base bet
		t.Errorf("expected pot 20, got %d", r.Pot)
	}
	if p1.Chips != 490 {
		t.Errorf("expected p1 chips 490, got %d", p1.Chips)
	}
}

func TestStartGame_NotEnoughPlayers(t *testing.T) {
	r := NewRoom("room1", 5)
	p := NewPlayer("p1", "Alice", 0)
	_ = r.AddPlayer(p)

	err := r.StartGame()
	if err == nil {
		t.Error("expected error with fewer than 2 players")
	}
}

func TestProcessAction_Call(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.Chips = 500
	p2.Chips = 500
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.StartGame()

	current := r.CurrentTurnPlayer()
	initialChips := current.Chips
	initialPot := r.Pot

	event, err := r.ProcessAction(current.ID, Action{Type: ActionCall})
	if err != nil {
		t.Fatalf("call action failed: %v", err)
	}
	if event.Type != "call" {
		t.Errorf("expected event type 'call', got %s", event.Type)
	}
	if current.Chips >= initialChips {
		t.Error("chips should decrease after call")
	}
	if r.Pot <= initialPot {
		t.Error("pot should increase after call")
	}
}

func TestProcessAction_Fold(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.Chips = 500
	p2.Chips = 500
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.StartGame()

	current := r.CurrentTurnPlayer()

	event, err := r.ProcessAction(current.ID, Action{Type: ActionFold})
	if err != nil {
		t.Fatalf("fold action failed: %v", err)
	}
	if event.Type != "fold" {
		t.Errorf("expected event type 'fold', got %s", event.Type)
	}
	if current.State != PlayerFolded {
		t.Errorf("expected PlayerFolded, got %d", current.State)
	}
}

func TestProcessAction_AllIn(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.Chips = 500
	p2.Chips = 500
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.StartGame()

	current := r.CurrentTurnPlayer()
	chipsBeforeAnte := current.Chips

	event, err := r.ProcessAction(current.ID, Action{Type: ActionAllIn})
	if err != nil {
		t.Fatalf("all-in action failed: %v", err)
	}
	if event.Type != "allin" {
		t.Errorf("expected event type 'allin', got %s", event.Type)
	}
	if current.State != PlayerAllIn {
		t.Errorf("expected PlayerAllIn, got %d", current.State)
	}
	if current.Chips != 0 {
		t.Errorf("expected 0 chips after all-in, got %d", current.Chips)
	}
	if !r.HasAllIn {
		t.Error("room HasAllIn should be true")
	}
	if event.Amount != chipsBeforeAnte {
		t.Errorf("all-in amount should be %d, got %d", chipsBeforeAnte, event.Amount)
	}
}

func TestProcessAction_Raise(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.Chips = 500
	p2.Chips = 500
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.StartGame()

	current := r.CurrentTurnPlayer()
	initialChips := current.Chips

	event, err := r.ProcessAction(current.ID, Action{Type: ActionRaise, Amount: 40})
	if err != nil {
		t.Fatalf("raise action failed: %v", err)
	}
	if event.Type != "raise" {
		t.Errorf("expected event type 'raise', got %s", event.Type)
	}
	if current.Chips >= initialChips {
		t.Error("chips should decrease after raise")
	}
	if event.Amount < 20 { // minimum raise is prevBet*2
		t.Errorf("raise amount should be at least 20, got %d", event.Amount)
	}
}

func TestProcessAction_Compare(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.Chips = 500
	p2.Chips = 500
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.StartGame()

	current := r.CurrentTurnPlayer()
	var targetID string
	if current.ID == "p1" {
		targetID = "p2"
	} else {
		targetID = "p1"
	}

	event, err := r.ProcessAction(current.ID, Action{Type: ActionCompare, TargetID: targetID})
	if err != nil {
		t.Fatalf("compare action failed: %v", err)
	}
	if event.Type != "compare" {
		t.Errorf("expected event type 'compare', got %s", event.Type)
	}
	if event.WinnerID == "" {
		t.Error("compare should produce a winner")
	}
	if event.TargetID != targetID {
		t.Errorf("expected target %s, got %s", targetID, event.TargetID)
	}
}

func TestCheckGameEnd_OneAlive(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.State = PlayerPlaying
	p2.State = PlayerFolded
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)

	ended, winner := r.CheckGameEnd()
	if !ended {
		t.Error("game should end with one alive player")
	}
	if winner == nil || winner.ID != "p1" {
		t.Error("winner should be p1")
	}
}

func TestCheckGameEnd_RoundOver20(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.State = PlayerPlaying
	p1.Hand = Hand{Card{1, 14}, Card{2, 14}, Card{3, 14}} // Three Aces
	p2.State = PlayerPlaying
	p2.Hand = Hand{Card{1, 2}, Card{2, 5}, Card{3, 9}} // Single
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	r.Round = 21

	ended, winner := r.CheckGameEnd()
	if !ended {
		t.Error("game should end when round > 20")
	}
	if winner == nil || winner.ID != "p1" {
		t.Errorf("winner should be p1 (best hand), got %v", winner)
	}
}

func TestCheckGameEnd_NotEnded(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.State = PlayerPlaying
	p2.State = PlayerPlaying
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	r.Round = 5

	ended, _ := r.CheckGameEnd()
	if ended {
		t.Error("game should not end with multiple alive players and round <= 20")
	}
}

func TestSidePots(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p3 := NewPlayer("p3", "Charlie", 2)

	p1.State = PlayerAllIn
	p1.BetTotal = 100
	p2.State = PlayerAllIn
	p2.BetTotal = 200
	p3.State = PlayerFolded
	p3.BetTotal = 50

	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	_ = r.AddPlayer(p3)

	r.HasAllIn = true
	r.Pot = 350

	pots := r.calculateSidePots()
	if len(pots) < 1 {
		t.Fatalf("expected at least 1 side pot, got %d", len(pots))
	}

	// First pot: min bet level = 100, contributions: p1=100, p2=100, p3=50 => 250
	if pots[0].Amount != 250 {
		t.Errorf("first pot amount: expected 250, got %d", pots[0].Amount)
	}
	// Second pot: level 200, contributions above 100: p2 has 100 more => 100
	if len(pots) >= 2 && pots[1].Amount != 100 {
		t.Errorf("second pot amount: expected 100, got %d", pots[1].Amount)
	}
}

func TestEndGame_Normal(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	p1.State = PlayerPlaying
	p1.Chips = 100
	p2.State = PlayerFolded
	p2.Chips = 100
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)
	r.Pot = 200
	r.BaseBet = 10
	r.HasAllIn = false

	rake := r.EndGame(p1)

	expectedRake := 10 * 2 // BaseBet * playerCount
	if rake != expectedRake {
		t.Errorf("expected rake %d, got %d", expectedRake, rake)
	}
	expectedWinnings := 200 - expectedRake
	if p1.Chips != 100+expectedWinnings {
		t.Errorf("expected winner chips %d, got %d", 100+expectedWinnings, p1.Chips)
	}
	if r.State != RoomFinished {
		t.Errorf("expected RoomFinished, got %d", r.State)
	}
}

func TestEndGame_AllIn(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)

	// p1 has better hand
	p1.Hand = Hand{Card{1, 14}, Card{2, 14}, Card{3, 14}}
	p2.Hand = Hand{Card{1, 2}, Card{2, 5}, Card{3, 9}}

	p1.State = PlayerAllIn
	p1.BetTotal = 100
	p1.Chips = 0
	p2.State = PlayerAllIn
	p2.BetTotal = 100
	p2.Chips = 0

	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)

	r.Pot = 200
	r.BaseBet = 10
	r.HasAllIn = true

	rake := r.EndGame(nil)

	if rake < 0 {
		t.Errorf("rake should be non-negative, got %d", rake)
	}
	if r.State != RoomFinished {
		t.Errorf("expected RoomFinished, got %d", r.State)
	}
	// p1 should have won something
	if p1.Chips <= 0 {
		t.Errorf("winner should have chips > 0, got %d", p1.Chips)
	}
	if len(r.SidePots) == 0 {
		t.Error("expected side pots to be set")
	}
}

func TestConfirmNextRound(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	p2 := NewPlayer("p2", "Bob", 1)
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(p2)

	r.ReadyForNext = make(map[string]bool)

	r.ConfirmNextRound("p1")
	if !r.ReadyForNext["p1"] {
		t.Error("p1 should be confirmed")
	}
	if r.AllConfirmedNextRound() {
		t.Error("should not be all confirmed yet")
	}

	r.ConfirmNextRound("p2")
	if !r.AllConfirmedNextRound() {
		t.Error("should be all confirmed after both players confirm")
	}
}

func TestConfirmNextRound_BotsIgnored(t *testing.T) {
	r := NewRoom("room1", 5)
	p1 := NewPlayer("p1", "Alice", 0)
	bot := NewBot(1)
	_ = r.AddPlayer(p1)
	_ = r.AddPlayer(bot)
	r.ReadyForNext = make(map[string]bool)

	// Only human needs to confirm
	r.ConfirmNextRound("p1")
	if !r.AllConfirmedNextRound() {
		t.Error("should be all confirmed when only human confirmed (bot ignored)")
	}
}
