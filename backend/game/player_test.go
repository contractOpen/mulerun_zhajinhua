package game

import (
	"testing"
)

func TestNewPlayer(t *testing.T) {
	p := NewPlayer("p1", "Alice", 2)
	if p.ID != "p1" {
		t.Errorf("expected ID p1, got %s", p.ID)
	}
	if p.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", p.Name)
	}
	if p.Chips != 1000 {
		t.Errorf("expected 1000 chips, got %d", p.Chips)
	}
	if p.State != PlayerWaiting {
		t.Errorf("expected PlayerWaiting state, got %d", p.State)
	}
	if p.Seat != 2 {
		t.Errorf("expected seat 2, got %d", p.Seat)
	}
	if p.IsBot {
		t.Error("expected IsBot to be false")
	}
	if p.Avatar != 3 { // seat%6 + 1 = 2%6+1 = 3
		t.Errorf("expected avatar 3, got %d", p.Avatar)
	}
}

func TestNewBot(t *testing.T) {
	bot := NewBot(0)
	if !bot.IsBot {
		t.Error("expected IsBot to be true")
	}
	if bot.Name == "" {
		t.Error("bot should have a name")
	}
	if bot.State != PlayerReady {
		t.Errorf("expected PlayerReady state for bot, got %d", bot.State)
	}
	if bot.Chips != 1000 {
		t.Errorf("expected 1000 chips, got %d", bot.Chips)
	}
}

func TestNewBot_DifferentSeats(t *testing.T) {
	names := map[string]bool{}
	for i := 0; i < 5; i++ {
		bot := NewBot(i)
		names[bot.Name] = true
	}
	if len(names) != 5 {
		t.Errorf("expected 5 unique bot names across seats 0-4, got %d", len(names))
	}
}

func TestPlayerReset(t *testing.T) {
	p := NewPlayer("p1", "Alice", 0)
	p.State = PlayerPlaying
	p.BetTotal = 50
	p.Looked = true
	p.Hand = Hand{Card{1, 14}, Card{2, 13}, Card{3, 12}}

	p.Reset()

	if p.State != PlayerWaiting {
		t.Errorf("expected PlayerWaiting after reset, got %d", p.State)
	}
	if p.BetTotal != 0 {
		t.Errorf("expected BetTotal 0 after reset, got %d", p.BetTotal)
	}
	if p.Looked {
		t.Error("expected Looked to be false after reset")
	}
	emptyHand := Hand{}
	if p.Hand != emptyHand {
		t.Error("expected empty hand after reset")
	}
}

func TestBotAction_HasAllIn_GoodHand(t *testing.T) {
	room := NewRoom("r1", 5)
	room.HasAllIn = true
	room.BaseBet = 10
	room.CurrentBet = 10

	bot := NewBot(0)
	bot.Hand = Hand{Card{1, 10}, Card{2, 10}, Card{3, 5}} // Pair
	bot.State = PlayerPlaying

	action := bot.BotAction(room)
	if action.Type != ActionAllIn {
		t.Errorf("bot with pair should all-in when HasAllIn, got action type %d", action.Type)
	}
}

func TestBotAction_HighHand_Raise(t *testing.T) {
	room := NewRoom("r1", 5)
	room.HasAllIn = false
	room.BaseBet = 10
	room.CurrentBet = 10
	room.Round = 1

	bot := NewBot(0)
	bot.Chips = 1000
	bot.Hand = Hand{Card{1, 10}, Card{1, 11}, Card{1, 12}} // Straight Flush (same suit)
	bot.State = PlayerPlaying

	action := bot.BotAction(room)
	// StraightFlush with enough chips should raise
	if action.Type != ActionRaise {
		t.Errorf("bot with straight flush and enough chips should raise, got action type %d", action.Type)
	}
}

func TestBotAction_LowHand_LateRound_Fold(t *testing.T) {
	room := NewRoom("r1", 5)
	room.HasAllIn = false
	room.BaseBet = 10
	room.CurrentBet = 10
	room.Round = 5

	bot := NewBot(0)
	bot.Chips = 1000
	bot.Hand = Hand{Card{1, 2}, Card{2, 5}, Card{3, 10}} // Single
	bot.State = PlayerPlaying

	action := bot.BotAction(room)
	if action.Type != ActionFold {
		t.Errorf("bot with single hand in late round should fold, got action type %d", action.Type)
	}
}

func TestBotAction_LowHand_EarlyRound_Call(t *testing.T) {
	room := NewRoom("r1", 5)
	room.HasAllIn = false
	room.BaseBet = 10
	room.CurrentBet = 10
	room.Round = 1

	bot := NewBot(0)
	bot.Chips = 1000
	bot.Hand = Hand{Card{1, 2}, Card{2, 5}, Card{3, 10}} // Single
	bot.State = PlayerPlaying

	action := bot.BotAction(room)
	if action.Type != ActionCall {
		t.Errorf("bot with single hand in early round should call, got action type %d", action.Type)
	}
}
