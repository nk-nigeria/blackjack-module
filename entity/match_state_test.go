package entity

import (
	"fmt"
	"testing"

	pb "github.com/nakamaFramework/cgp-common/proto"
)

func TestMatchState(t *testing.T) {
	s := NewMatchState(&pb.Match{
		Open:     false,
		MarkUnit: MaxBetAllowed,
		// Code:     "test",
		Name:     "test_table",
		Password: "",
		MaxSize:  5,
	})
	s.Init()
	s.PlayingPresences.Put("A", FakePrecense{})
	s.PlayingPresences.Put("B", FakePrecense{})
	deck := NewDeck()
	deck.Shuffle()
	// bankerCards, _ := deck.Deal(2)
	s.AddBet(&pb.BlackjackBet{
		UserId: "A",
		Chips:  100,
	})
	s.AddBet(&pb.BlackjackBet{
		UserId: "A",
		Chips:  200,
	})
	fmt.Printf("A is playing: %v\n, B is playing: %v\n", s.IsBet("A"), s.IsBet("B"))

	if cards, err := deck.Deal(2); err != nil {
		t.Fatalf(err.Error())
	} else {
		s.AddCards(cards.Cards, "", pb.BlackjackHandN0_BLACKJACK_HAND_UNSPECIFIED)
	}
	if cards, err := deck.Deal(2); err != nil {
		t.Fatalf(err.Error())
	} else {
		s.AddCards(cards.Cards, "A", pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	}
	fmt.Printf("====END GAME====\n%v\n", s.CalcGameFinish())
}
