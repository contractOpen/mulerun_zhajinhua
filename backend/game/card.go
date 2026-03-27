package game

import (
	"math/rand"
	"sort"
	"time"
)

// 花色
const (
	SuitSpade   = 4 // 黑桃
	SuitHeart   = 3 // 红心
	SuitClub    = 2 // 梅花
	SuitDiamond = 1 // 方块
)

// 牌型
const (
	HandTypeSingle        = 1 // 散牌（高牌）
	HandTypePair          = 2 // 对子
	HandTypeStraight      = 3 // 顺子
	HandTypeFlush         = 4 // 金花（同花）
	HandTypeStraightFlush = 5 // 顺金（同花顺）
	HandTypeThreeOfAKind  = 6 // 豹子（三条）
)

// Card 一张牌
type Card struct {
	Suit  int `json:"suit"`  // 花色 1-4
	Value int `json:"value"` // 点数 2-14(A=14)
}

// Hand 一手牌（3张）
type Hand [3]Card

// NewDeck 创建一副牌（52张）
func NewDeck() []Card {
	deck := make([]Card, 0, 52)
	for suit := 1; suit <= 4; suit++ {
		for value := 2; value <= 14; value++ {
			deck = append(deck, Card{Suit: suit, Value: value})
		}
	}
	return deck
}

// ShuffleDeck 洗牌
func ShuffleDeck(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

// DealHands 普通发牌
func DealHands(n int) []Hand {
	deck := NewDeck()
	ShuffleDeck(deck)
	hands := make([]Hand, n)
	for i := 0; i < n; i++ {
		hands[i] = Hand{deck[i*3], deck[i*3+1], deck[i*3+2]}
	}
	return hands
}

// DealWeightedHands 加权发牌 - 大幅提高好牌概率
// 目标: 约10局只有1局散牌, 豹子约10局出1次, 金花/顺子频繁
func DealWeightedHands(n int, goodBias int) []Hand {
	if goodBias <= 0 {
		return DealHands(n)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hands := make([]Hand, n)

	for i := 0; i < n; i++ {
		hands[i] = dealOneWeightedHand(r, goodBias)
	}
	return hands
}

// dealOneWeightedHand 按权重为单个玩家生成一手牌
func dealOneWeightedHand(r *rand.Rand, bias int) Hand {
	// 目标牌型分布 (bias=3 时约):
	// 豹子 ~10%, 同花顺 ~10%, 金花 ~15%, 顺子 ~20%, 对子 ~35%, 散牌 ~10%
	// bias越高好牌越多
	roll := r.Float64() * 100

	// 根据bias调整阈值
	leopardChance := float64(bias) * 3.5   // bias3 = 10.5%
	sfChance := float64(bias) * 3.5        // bias3 = 10.5%
	flushChance := float64(bias) * 5.0     // bias3 = 15%
	straightChance := float64(bias) * 7.0  // bias3 = 21%
	pairChance := 35.0                     // 固定35%
	// 剩余 = 散牌

	cumulative := 0.0

	cumulative += leopardChance
	if roll < cumulative {
		return generateThreeOfAKind(r)
	}

	cumulative += sfChance
	if roll < cumulative {
		return generateStraightFlush(r)
	}

	cumulative += flushChance
	if roll < cumulative {
		return generateFlush(r)
	}

	cumulative += straightChance
	if roll < cumulative {
		return generateStraight(r)
	}

	cumulative += pairChance
	if roll < cumulative {
		return generatePair(r)
	}

	// 散牌 - 随机发
	return generateRandom(r)
}

func generateThreeOfAKind(r *rand.Rand) Hand {
	val := r.Intn(13) + 2 // 2-14
	suits := []int{1, 2, 3, 4}
	r.Shuffle(4, func(i, j int) { suits[i], suits[j] = suits[j], suits[i] })
	return Hand{
		Card{suits[0], val},
		Card{suits[1], val},
		Card{suits[2], val},
	}
}

func generateStraightFlush(r *rand.Rand) Hand {
	suit := r.Intn(4) + 1
	// 起始值 2-12 (顺子最高 Q-K-A), 加上 A-2-3
	start := r.Intn(12) + 2 // 2-13
	if start == 13 {
		// Q-K-A
		return Hand{Card{suit, 12}, Card{suit, 13}, Card{suit, 14}}
	}
	return Hand{
		Card{suit, start},
		Card{suit, start + 1},
		Card{suit, start + 2},
	}
}

func generateFlush(r *rand.Rand) Hand {
	suit := r.Intn(4) + 1
	// 3个不同且不构成顺子的同花色牌
	for attempts := 0; attempts < 50; attempts++ {
		vals := []int{r.Intn(13) + 2, r.Intn(13) + 2, r.Intn(13) + 2}
		if vals[0] == vals[1] || vals[1] == vals[2] || vals[0] == vals[2] {
			continue
		}
		h := Hand{Card{suit, vals[0]}, Card{suit, vals[1]}, Card{suit, vals[2]}}
		if !isStraight(h) {
			return h
		}
	}
	// fallback
	return Hand{Card{suit, 2}, Card{suit, 5}, Card{suit, 9}}
}

func generateStraight(r *rand.Rand) Hand {
	start := r.Intn(12) + 2
	suits := []int{r.Intn(4) + 1, r.Intn(4) + 1, r.Intn(4) + 1}
	// 确保不是同花
	if suits[0] == suits[1] && suits[1] == suits[2] {
		suits[2] = (suits[2] % 4) + 1
	}
	if start == 13 {
		return Hand{Card{suits[0], 12}, Card{suits[1], 13}, Card{suits[2], 14}}
	}
	return Hand{
		Card{suits[0], start},
		Card{suits[1], start + 1},
		Card{suits[2], start + 2},
	}
}

func generatePair(r *rand.Rand) Hand {
	pairVal := r.Intn(13) + 2
	suits := []int{1, 2, 3, 4}
	r.Shuffle(4, func(i, j int) { suits[i], suits[j] = suits[j], suits[i] })
	// 单牌不同于对子值
	single := r.Intn(13) + 2
	for single == pairVal {
		single = r.Intn(13) + 2
	}
	singleSuit := r.Intn(4) + 1
	return Hand{
		Card{suits[0], pairVal},
		Card{suits[1], pairVal},
		Card{singleSuit, single},
	}
}

func generateRandom(r *rand.Rand) Hand {
	deck := NewDeck()
	r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return Hand{deck[0], deck[1], deck[2]}
}

func sortedValues(h Hand) []int {
	vals := []int{h[0].Value, h[1].Value, h[2].Value}
	sort.Sort(sort.Reverse(sort.IntSlice(vals)))
	return vals
}

func isFlush(h Hand) bool {
	return h[0].Suit == h[1].Suit && h[1].Suit == h[2].Suit
}

func isStraight(h Hand) bool {
	vals := sortedValues(h)
	if vals[0] == 14 && vals[1] == 3 && vals[2] == 2 {
		return true
	}
	return vals[0]-vals[1] == 1 && vals[1]-vals[2] == 1
}

func isThreeOfAKind(h Hand) bool {
	return h[0].Value == h[1].Value && h[1].Value == h[2].Value
}

func isPair(h Hand) (bool, int, int) {
	vals := sortedValues(h)
	if vals[0] == vals[1] {
		return true, vals[0], vals[2]
	}
	if vals[1] == vals[2] {
		return true, vals[1], vals[0]
	}
	return false, 0, 0
}

// HandRank 计算牌型
func HandRank(h Hand) int {
	if isThreeOfAKind(h) {
		return HandTypeThreeOfAKind
	}
	flush := isFlush(h)
	straight := isStraight(h)
	if flush && straight {
		return HandTypeStraightFlush
	}
	if flush {
		return HandTypeFlush
	}
	if straight {
		return HandTypeStraight
	}
	ok, _, _ := isPair(h)
	if ok {
		return HandTypePair
	}
	return HandTypeSingle
}

// CompareHands 比较两手牌
func CompareHands(a, b Hand) int {
	rankA := HandRank(a)
	rankB := HandRank(b)

	if rankA != rankB {
		if rankA > rankB {
			return 1
		}
		return -1
	}

	switch rankA {
	case HandTypeThreeOfAKind:
		if a[0].Value > b[0].Value {
			return 1
		} else if a[0].Value < b[0].Value {
			return -1
		}
		return 0

	case HandTypeStraightFlush, HandTypeStraight:
		va := sortedValues(a)
		vb := sortedValues(b)
		maxA, maxB := va[0], vb[0]
		if va[0] == 14 && va[1] == 3 {
			maxA = 3
		}
		if vb[0] == 14 && vb[1] == 3 {
			maxB = 3
		}
		if maxA > maxB {
			return 1
		} else if maxA < maxB {
			return -1
		}
		return 0

	case HandTypeFlush, HandTypeSingle:
		va := sortedValues(a)
		vb := sortedValues(b)
		for i := 0; i < 3; i++ {
			if va[i] > vb[i] {
				return 1
			} else if va[i] < vb[i] {
				return -1
			}
		}
		return 0

	case HandTypePair:
		_, pairA, singleA := isPair(a)
		_, pairB, singleB := isPair(b)
		if pairA != pairB {
			if pairA > pairB {
				return 1
			}
			return -1
		}
		if singleA != singleB {
			if singleA > singleB {
				return 1
			}
			return -1
		}
		return 0
	}
	return 0
}

// HandTypeName 牌型名称
func HandTypeName(rank int) string {
	switch rank {
	case HandTypeThreeOfAKind:
		return "豹子"
	case HandTypeStraightFlush:
		return "同花顺"
	case HandTypeFlush:
		return "金花"
	case HandTypeStraight:
		return "顺子"
	case HandTypePair:
		return "对子"
	case HandTypeSingle:
		return "散牌"
	}
	return "未知"
}

// HandTypeKey returns a stable translation key for the hand rank.
func HandTypeKey(rank int) string {
	switch rank {
	case HandTypeThreeOfAKind:
		return "hand.threeOfAKind"
	case HandTypeStraightFlush:
		return "hand.straightFlush"
	case HandTypeFlush:
		return "hand.flush"
	case HandTypeStraight:
		return "hand.straight"
	case HandTypePair:
		return "hand.pair"
	case HandTypeSingle:
		return "hand.highCard"
	}
	return "hand.unknown"
}
