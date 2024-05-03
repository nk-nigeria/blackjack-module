package entity

import pb "github.com/ciaolink-game-platform/cgp-common/proto"

type Hand struct {
	userId string
	first  *SubHand
	second *SubHand
}

type SubHand struct {
	cards    []*pb.Card
	point    int
	handType pb.BlackjackHandType
	stay     bool
}

func (h *SubHand) AddCards(c []*pb.Card) {
	h.cards = append(h.cards, c...)
}

func (h *SubHand) ToPb() *pb.BlackjackHand {
	return &pb.BlackjackHand{
		Cards: h.cards,
		Point: int32(h.point),
		Type:  h.handType,
	}
}

func (h *SubHand) Stay() {
	h.stay = true
}

func NewPlayerHand(userId string, first []*pb.Card, second []*pb.Card) *Hand {
	return &Hand{
		userId: userId,
		first: &SubHand{
			cards: first,
		},
		second: &SubHand{
			cards: second,
		},
	}
}

func NewSubHand(cards []*pb.Card) *SubHand {
	return &SubHand{
		cards: cards,
		stay:  false,
	}
}

func NewHandFromPb(v *pb.BlackjackPlayerHand) *Hand {
	return &Hand{
		userId: v.UserId,
		first: &SubHand{
			cards: v.First.Cards,
		},
		second: &SubHand{
			cards: v.Second.Cards,
		},
	}
}

func (h *Hand) ToPb() *pb.BlackjackPlayerHand {
	h.Eval()
	return &pb.BlackjackPlayerHand{
		UserId: h.userId,
		First:  h.first.ToPb(),
		Second: h.second.ToPb(),
	}
}

func getCardPoint(r pb.CardRank) int32 {
	switch v := int32(r); {
	case v <= 9:
		return v
	default:
		return 10
	}
}

func calculatePoint(cards []*pb.Card) int32 {
	if cards == nil {
		return 0
	}
	haveAce := false
	point := int32(0)
	for _, c := range cards {
		v := getCardPoint(c.Rank)
		if v == 1 {
			haveAce = true
		}
		point += v
	}
	if haveAce && point <= 11 {
		point += 10
	}
	return point
}

func (h *Hand) Eval() {
	h.first.point = int(calculatePoint(h.first.cards))
	if h.first.point == 0 {
		h.first.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED
	} else if h.first.point == 21 {
		if len(h.first.cards) == 2 && (h.second == nil || len(h.second.cards) == 0) {
			h.first.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BLACKJACK
		} else {
			h.first.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_21P
		}
	} else if h.first.point > 21 {
		h.first.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED
	} else {
		h.first.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_NORMAL
	}
	if h.second != nil && len(h.second.cards) > 0 {
		h.second.point = int(calculatePoint(h.second.cards))
		if h.second.point == 0 {
			h.second.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED
		} else if h.second.point == 21 {
			h.second.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_21P
		} else if h.second.point > 21 {
			h.second.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED
		} else {
			h.second.handType = pb.BlackjackHandType_BLACKJACK_HAND_TYPE_NORMAL
		}
	}
}

// Dealer must draw on lower than 17 and stand on >= 17
func (h *Hand) DealerMustDraw() bool {
	return calculatePoint(h.first.cards) < 17
}

func (h *Hand) DealerPotentialBlackjack() bool {
	return h.first.cards[0].Rank == pb.CardRank_RANK_A
}

// Check if player can draw on current hand, call with pos=1 for 1st hand, else 2nd hand
func (h *Hand) PlayerCanDraw(pos pb.BlackjackHandN0) bool {
	if pos == pb.BlackjackHandN0_BLACKJACK_HAND_1ST {
		if h.first.stay {
			return false
		}
		return calculatePoint(h.first.cards) < 21
	} else {
		if h.second.stay {
			return false
		}
		return calculatePoint(h.second.cards) < 21
	}
}

func (h *Hand) PlayerCanSplit() bool {
	return (h.second == nil || len(h.second.cards) == 0) &&
		len(h.first.cards) == 2 &&
		getCardPoint(h.first.cards[0].Rank) == getCardPoint(h.first.cards[1].Rank)
}

func (h *Hand) Split() {
	h.second = &SubHand{
		cards: []*pb.Card{
			h.first.cards[1],
		},
	}
	h.first = &SubHand{
		cards: []*pb.Card{
			h.first.cards[0],
		},
	}
}

// comparing player hand with dealer hand, -1 -> lost, 1 -> win, 0 -> tie
func (h *Hand) Compare(dealer *Hand) (int, int) {
	h.Eval()
	dealer.Eval()
	r1 := 0
	r2 := 0
	if h.first.handType != pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED {
		if h.first.handType == pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED {
			r1 = -1
		} else {
			if int(h.first.handType) > int(dealer.first.handType) {
				r1 = 1
			} else if int(h.first.handType) == int(dealer.first.handType) {
				if h.first.point > dealer.first.point {
					r1 = 1
				} else if h.first.point < dealer.first.point {
					r1 = -1
				}
			} else {
				r1 = -1
			}
		}
	}
	if h.second.handType != pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED {
		if h.second.handType == pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED {
			r2 = -1
		} else {
			if int(h.second.handType) > int(dealer.first.handType) {
				r2 = 1
			} else if int(h.second.handType) == int(dealer.first.handType) {
				if h.second.point > dealer.first.point {
					r2 = 1
				} else if h.second.point < dealer.first.point {
					r2 = -1
				}
			} else {
				r2 = -1
			}
		}
	}
	return r1, r2
}
