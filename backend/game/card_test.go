package game

import (
	"testing"
)

func TestHandRank_ThreeOfAKind(t *testing.T) {
	h := Hand{Card{1, 7}, Card{2, 7}, Card{3, 7}}
	if rank := HandRank(h); rank != HandTypeThreeOfAKind {
		t.Errorf("expected ThreeOfAKind (%d), got %d", HandTypeThreeOfAKind, rank)
	}
}

func TestHandRank_StraightFlush(t *testing.T) {
	h := Hand{Card{1, 5}, Card{1, 6}, Card{1, 7}}
	if rank := HandRank(h); rank != HandTypeStraightFlush {
		t.Errorf("expected StraightFlush (%d), got %d", HandTypeStraightFlush, rank)
	}
}

func TestHandRank_Flush(t *testing.T) {
	h := Hand{Card{2, 3}, Card{2, 7}, Card{2, 10}}
	if rank := HandRank(h); rank != HandTypeFlush {
		t.Errorf("expected Flush (%d), got %d", HandTypeFlush, rank)
	}
}

func TestHandRank_Straight(t *testing.T) {
	h := Hand{Card{1, 8}, Card{2, 9}, Card{3, 10}}
	if rank := HandRank(h); rank != HandTypeStraight {
		t.Errorf("expected Straight (%d), got %d", HandTypeStraight, rank)
	}
}

func TestHandRank_Pair(t *testing.T) {
	h := Hand{Card{1, 9}, Card{2, 9}, Card{3, 5}}
	if rank := HandRank(h); rank != HandTypePair {
		t.Errorf("expected Pair (%d), got %d", HandTypePair, rank)
	}
}

func TestHandRank_Single(t *testing.T) {
	h := Hand{Card{1, 2}, Card{2, 5}, Card{3, 10}}
	if rank := HandRank(h); rank != HandTypeSingle {
		t.Errorf("expected Single (%d), got %d", HandTypeSingle, rank)
	}
}

func TestCompareHands_DifferentTypes(t *testing.T) {
	threeOfKind := Hand{Card{1, 7}, Card{2, 7}, Card{3, 7}}
	pair := Hand{Card{1, 9}, Card{2, 9}, Card{3, 5}}

	if result := CompareHands(threeOfKind, pair); result != 1 {
		t.Errorf("ThreeOfAKind should beat Pair, got %d", result)
	}
	if result := CompareHands(pair, threeOfKind); result != -1 {
		t.Errorf("Pair should lose to ThreeOfAKind, got %d", result)
	}
}

func TestCompareHands_SameTypeDifferentValues(t *testing.T) {
	// Two pairs: pair of 10s vs pair of 9s
	pairHigh := Hand{Card{1, 10}, Card{2, 10}, Card{3, 5}}
	pairLow := Hand{Card{1, 9}, Card{2, 9}, Card{3, 5}}

	if result := CompareHands(pairHigh, pairLow); result != 1 {
		t.Errorf("higher pair should win, got %d", result)
	}
}

func TestCompareHands_PairTiebreaker(t *testing.T) {
	// Same pair value, different kicker
	pairHighKicker := Hand{Card{1, 10}, Card{2, 10}, Card{3, 8}}
	pairLowKicker := Hand{Card{1, 10}, Card{2, 10}, Card{3, 5}}

	if result := CompareHands(pairHighKicker, pairLowKicker); result != 1 {
		t.Errorf("same pair higher kicker should win, got %d", result)
	}
}

func TestCompareHands_FlushTiebreaker(t *testing.T) {
	flushHigh := Hand{Card{1, 14}, Card{1, 10}, Card{1, 5}}
	flushLow := Hand{Card{2, 13}, Card{2, 10}, Card{2, 5}}

	if result := CompareHands(flushHigh, flushLow); result != 1 {
		t.Errorf("flush with Ace should beat flush with King, got %d", result)
	}
}

func TestCompareHands_Equal(t *testing.T) {
	a := Hand{Card{1, 10}, Card{2, 10}, Card{3, 5}}
	b := Hand{Card{3, 10}, Card{4, 10}, Card{1, 5}}

	if result := CompareHands(a, b); result != 0 {
		t.Errorf("identical value hands should tie, got %d", result)
	}
}

func TestCompareHands_ThreeOfAKindValues(t *testing.T) {
	high := Hand{Card{1, 14}, Card{2, 14}, Card{3, 14}}
	low := Hand{Card{1, 2}, Card{2, 2}, Card{3, 2}}

	if result := CompareHands(high, low); result != 1 {
		t.Errorf("higher three of a kind should win, got %d", result)
	}
}

func TestCompareHands_StraightValues(t *testing.T) {
	high := Hand{Card{1, 10}, Card{2, 11}, Card{3, 12}}
	low := Hand{Card{1, 5}, Card{2, 6}, Card{3, 7}}

	if result := CompareHands(high, low); result != 1 {
		t.Errorf("higher straight should win, got %d", result)
	}
}

func TestSpecialStraight_A23(t *testing.T) {
	// A-2-3 is the lowest straight (wheel)
	h := Hand{Card{1, 14}, Card{2, 2}, Card{3, 3}}
	if rank := HandRank(h); rank != HandTypeStraight {
		t.Errorf("A-2-3 should be a straight, got %d", rank)
	}
}

func TestSpecialStraight_A23_LosesTo234(t *testing.T) {
	wheel := Hand{Card{1, 14}, Card{2, 2}, Card{3, 3}}
	low := Hand{Card{1, 2}, Card{2, 3}, Card{3, 4}}

	// A-2-3 has effective max 3, 2-3-4 has max 4
	if result := CompareHands(wheel, low); result != -1 {
		t.Errorf("A-2-3 should lose to 2-3-4, got %d", result)
	}
}

func TestSpecialStraight_A23_StraightFlush(t *testing.T) {
	h := Hand{Card{1, 14}, Card{1, 2}, Card{1, 3}}
	if rank := HandRank(h); rank != HandTypeStraightFlush {
		t.Errorf("A-2-3 same suit should be straight flush, got %d", rank)
	}
}

func TestDealWeightedHands_CorrectCount(t *testing.T) {
	for _, n := range []int{2, 3, 5} {
		hands := DealWeightedHands(n, 3)
		if len(hands) != n {
			t.Errorf("DealWeightedHands(%d, 3) returned %d hands", n, len(hands))
		}
	}
}

func TestDealWeightedHands_ThreeCardsEach(t *testing.T) {
	hands := DealWeightedHands(4, 3)
	for i, h := range hands {
		// Each card must have valid suit and value
		for j, c := range h {
			if c.Suit < 1 || c.Suit > 4 {
				t.Errorf("hand %d card %d: invalid suit %d", i, j, c.Suit)
			}
			if c.Value < 2 || c.Value > 14 {
				t.Errorf("hand %d card %d: invalid value %d", i, j, c.Value)
			}
		}
	}
}

func TestDealWeightedHands_ZeroBiasFallsBack(t *testing.T) {
	hands := DealWeightedHands(3, 0)
	if len(hands) != 3 {
		t.Errorf("expected 3 hands with bias 0, got %d", len(hands))
	}
}

func TestDealWeightedHands_HighBias(t *testing.T) {
	hands := DealWeightedHands(5, 5)
	if len(hands) != 5 {
		t.Errorf("expected 5 hands with bias 5, got %d", len(hands))
	}
}

func TestHandTypeName(t *testing.T) {
	tests := []struct {
		rank int
		name string
	}{
		{HandTypeThreeOfAKind, "豹子"},
		{HandTypeStraightFlush, "同花顺"},
		{HandTypeFlush, "金花"},
		{HandTypeStraight, "顺子"},
		{HandTypePair, "对子"},
		{HandTypeSingle, "散牌"},
		{99, "未知"},
	}

	for _, tt := range tests {
		got := HandTypeName(tt.rank)
		if got != tt.name {
			t.Errorf("HandTypeName(%d) = %q, want %q", tt.rank, got, tt.name)
		}
	}
}
