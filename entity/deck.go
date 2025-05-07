package entity

import (
	"errors"
	"math/rand"

	pb "github.com/nakamaFramework/cgp-common/proto"
)

const MaxCard = 312

type Deck struct {
	ListCard *pb.ListCard
	Dealt    int
}

func NewDeck() *Deck {
	ranks := []pb.CardRank{
		pb.CardRank_RANK_A,
		pb.CardRank_RANK_2,
		pb.CardRank_RANK_3,
		pb.CardRank_RANK_4,
		pb.CardRank_RANK_5,
		pb.CardRank_RANK_6,
		pb.CardRank_RANK_7,
		pb.CardRank_RANK_8,
		pb.CardRank_RANK_9,
		pb.CardRank_RANK_10,
		pb.CardRank_RANK_J,
		pb.CardRank_RANK_Q,
		pb.CardRank_RANK_K,
	}

	suits := []pb.CardSuit{
		pb.CardSuit_SUIT_CLUBS,
		pb.CardSuit_SUIT_DIAMONDS,
		pb.CardSuit_SUIT_HEARTS,
		pb.CardSuit_SUIT_SPADES,
	}

	cards := &pb.ListCard{}
	for i := 0; i < 8; i++ {
		for _, r := range ranks {
			for _, s := range suits {
				cards.Cards = append(cards.Cards, &pb.Card{
					Rank: r,
					Suit: s,
				})
			}
		}
	}
	return &Deck{
		Dealt:    0,
		ListCard: cards,
	}
}

func (d *Deck) Shuffle() {
	for i := 1; i < len(d.ListCard.Cards); i++ {
		r := rand.Intn(i + 1)
		if i != r {
			d.ListCard.Cards[r], d.ListCard.Cards[i] = d.ListCard.Cards[i], d.ListCard.Cards[r]
		}
	}
	// mock
	// if d.ListCard.Cards[0].Rank != pb.CardRank_RANK_A {
	// 	for idx, card := range d.ListCard.Cards {
	// 		if idx == 0 {
	// 			continue
	// 		}
	// 		if card.Rank == pb.CardRank_RANK_A {
	// 			d.ListCard.Cards[0], d.ListCard.Cards[idx] = d.ListCard.Cards[idx], d.ListCard.Cards[0]
	// 			break
	// 		}
	// 	}
	// }
}

func (d *Deck) Deal(n int) (*pb.ListCard, error) {
	if (MaxCard - d.Dealt) < n {
		return nil, errors.New("deck.deal.error-not-enough")
	}
	var cards pb.ListCard
	for i := 0; i < n; i++ {
		cards.Cards = append(cards.Cards, d.ListCard.Cards[d.Dealt])
		d.Dealt++
	}
	return &cards, nil
}
