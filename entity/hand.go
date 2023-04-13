package entity

import pb "github.com/ciaolink-game-platform/cgp-common/proto"

type Hand struct {
	userId string
	first  []*pb.Card
	second []*pb.Card
}

func NewHand(userId string, first []*pb.Card, second []*pb.Card) *Hand {
	return &Hand{
		userId: userId,
		first:  first,
		second: second,
	}
}

func NewHandFromPb(v *pb.BlackjackPlayerHand) *Hand {
	return &Hand{
		userId: v.UserId,
		first:  v.First.Cards,
		second: v.Second.Cards,
	}
}

func (h *Hand) ToPb() *pb.BlackjackPlayerHand {
	point1, hand1Type := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	point2, hand2Type := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
	return &pb.BlackjackPlayerHand{
		UserId: h.userId,
		First: &pb.BlackjackHand{
			Cards: h.first,
			Point: point1,
			Type:  hand1Type,
		},
		Second: &pb.BlackjackHand{
			Cards: h.second,
			Point: point2,
			Type:  hand2Type,
		},
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

// Eval(1) if want to evaluate 1st hand, any else for 2nd hand
func (h *Hand) Eval(pos pb.BlackjackHandN0) (int32, pb.BlackjackHandType) {
	point := int32(0)
	if pos == 1 {
		point = calculatePoint(h.first)
	} else {
		point = calculatePoint(h.second)
	}
	if point == 0 {
		return point, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED
	}
	if point == 21 {
		if len(h.first) == 2 {
			return point, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BLACKJACK
		} else {
			return point, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_21P
		}
	} else if point > 21 {
		return point, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED
	}
	return point, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_NORMAL
}

// Dealer must draw on lower than 17 and stand on >= 17
func (h *Hand) DealerMustDraw() bool {
	return calculatePoint(h.first) < 17
}

func (h *Hand) DealerPotentialBlackjack() bool {
	return h.first[0].Rank == pb.CardRank_RANK_A
}

// Check if player can draw on current hand, call with pos=1 for 1st hand, else 2nd hand
func (h *Hand) PlayerCanDraw(pos pb.BlackjackHandN0) bool {
	if pos == pb.BlackjackHandN0_BLACKJACK_HAND_1ST {
		return calculatePoint(h.first) < 21
	} else {
		return calculatePoint(h.second) < 21
	}
}

func (h *Hand) PlayerCanSplit() bool {
	return (h.second == nil || len(h.second) == 0) &&
		len(h.first) == 2 &&
		getCardPoint(h.first[0].Rank) == getCardPoint(h.first[1].Rank)
}

func (h *Hand) Split() {
	h.second = []*pb.Card{
		h.first[1],
	}
	h.first = []*pb.Card{
		h.first[0],
	}
}

func (h *Hand) AddCards(c []*pb.Card, pos pb.BlackjackHandN0) {
	switch pos {
	case pb.BlackjackHandN0_BLACKJACK_HAND_1ST:
		h.first = append(h.first, c...)
	case pb.BlackjackHandN0_BLACKJACK_HAND_2ND:
		h.second = append(h.second, c...)
	case pb.BlackjackHandN0_BLACKJACK_HAND_UNSPECIFIED:
		h.first = append(h.first, c...)
	}
}

// comparing player hand with dealer hand, -1 -> lost, 1 -> win, 0 -> tie
func (h *Hand) Compare(d *Hand) (int, int) {
	hp1, ht1 := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	hp2, ht2 := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
	dp, dt := d.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	r1 := 0
	r2 := 0
	if ht1 != pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED {
		if ht1 == pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED {
			r1 = -1
		} else {
			if int(ht1) > int(dt) {
				r1 = 1
			} else if int(ht1) == int(dt) {
				if hp1 > dp {
					r1 = 1
				} else if hp1 < dp {
					r1 = -1
				}
			} else {
				r1 = -1
			}
		}
	}
	if ht2 != pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED {
		if ht2 == pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED {
			r2 = -1
		} else {
			if int(ht2) > int(dt) {
				r2 = 1
			} else if int(ht2) == int(dt) {
				if hp2 > dp {
					r2 = 1
				} else if hp2 < dp {
					r2 = -1
				}
			} else {
				r2 = -1
			}
		}
	}
	return r1, r2
}
