package entity

import (
	"strconv"

	pb "github.com/nk-nigeria/cgp-common/proto"
)

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
	point1, pointAce1, hand1Type := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	point2, pointAce2, hand2Type := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
	return &pb.BlackjackPlayerHand{
		UserId: h.userId,
		First: &pb.BlackjackHand{
			Cards:      h.first,
			Point:      int32(point1.Point),
			Type:       hand1Type,
			PointCardA: pointAce1,
			MinPoint:   int32(point1.MinPoint),
			MaxPoint:   int32(point1.MaxPoint),
		},
		Second: &pb.BlackjackHand{
			Cards:      h.second,
			Point:      int32(point2.Point),
			Type:       hand2Type,
			PointCardA: pointAce2,
			MinPoint:   int32(point2.MinPoint),
			MaxPoint:   int32(point2.MaxPoint),
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

type CPoint struct {
	Point    int
	MinPoint int
	MaxPoint int
}

func calculatePoint(cards []*pb.Card) (*CPoint, string) {
	if cards == nil {
		return &CPoint{}, ""
	}
	pointAce := ""
	haveAce := false
	point := int32(0)
	cPoint := &CPoint{}
	for _, c := range cards {
		v := getCardPoint(c.Rank)
		if v == 1 {
			haveAce = true
		}
		point += v
	}
	cPoint.MinPoint = int(point)
	cPoint.MaxPoint = int(point)
	if haveAce {
		pointAce = strconv.Itoa(int(point))
		if point <= 11 {
			point += 10
			pointAce += "/" + strconv.Itoa(int(point))
		}
		// blackjack
		if len(cards) == 2 && point == 21 {
			pointAce = ""
		}
		cPoint.MaxPoint = int(point)
	} else {
		pointAce = strconv.Itoa(int(point))
	}
	cPoint.Point = int(point)
	return cPoint, pointAce
}

// Eval(1) if want to evaluate 1st hand, any else for 2nd hand
func (h *Hand) Eval(pos pb.BlackjackHandN0) (*CPoint, string, pb.BlackjackHandType) {
	var point *CPoint
	pointAce := ""
	if pos == 1 {
		point, pointAce = calculatePoint(h.first)
	} else {
		point, pointAce = calculatePoint(h.second)
	}
	if point.Point == 0 {
		return point, pointAce, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED
	}
	if point.Point == 21 {
		if pos == 1 && len(h.first) == 2 && len(h.second) == 0 {
			return point, pointAce, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BLACKJACK
		} else {
			return point, pointAce, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_21P
		}
	} else if point.Point > 21 {
		return point, pointAce, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED
	}
	return point, pointAce, pb.BlackjackHandType_BLACKJACK_HAND_TYPE_NORMAL
}

// Dealer must draw on lower than 17 and stand on >= 17
func (h *Hand) DealerMustDraw() bool {
	point, _ := calculatePoint(h.first)
	return point.Point < 17
}

func (h *Hand) DealerPotentialBlackjack() bool {
	return h.first[0].Rank == pb.CardRank_RANK_A
}

// Check if player can draw on current hand, call with pos=1 for 1st hand, else 2nd hand
func (h *Hand) PlayerCanDraw(pos pb.BlackjackHandN0) bool {
	var point *CPoint
	if pos == pb.BlackjackHandN0_BLACKJACK_HAND_1ST {
		point, _ = calculatePoint(h.first)
	} else {
		point, _ = calculatePoint(h.second)
	}
	return point.Point < 21
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
	hp1, _, ht1 := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	hp2, _, ht2 := h.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
	dp, _, dt := d.Eval(pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	r1 := 0
	r2 := 0
	if ht1 != pb.BlackjackHandType_BLACKJACK_HAND_TYPE_UNSPECIFIED {
		if ht1 == pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BUSTED {
			r1 = -1
		} else {
			if int(ht1) > int(dt) {
				r1 = 1
			} else if int(ht1) == int(dt) {
				if hp1.Point > dp.Point {
					r1 = 1
				} else if hp1.Point < dp.Point {
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
				if hp2.Point > dp.Point {
					r2 = 1
				} else if hp2.Point < dp.Point {
					r2 = -1
				}
			} else {
				r2 = -1
			}
		}
	}
	return r1, r2
}
