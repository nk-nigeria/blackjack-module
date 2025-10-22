package entity

import (
	"testing"

	pb "github.com/nk-nigeria/cgp-common/proto"
)

func TestBlackjackBotLogic(t *testing.T) {
	// Test creating new bot logic
	botLogic := NewBlackjackBotLogic()
	if botLogic == nil {
		t.Fatal("Failed to create BlackjackBotLogic")
	}

	// Test setting balance
	botLogic.SetBalance(10000)
	if botLogic.GetBalance() != 10000 {
		t.Errorf("Expected balance 10000, got %d", botLogic.GetBalance())
	}

	// Test risk level
	botLogic.SetRiskLevel("aggressive")
	if botLogic.GetRiskLevel() != "aggressive" {
		t.Errorf("Expected risk level 'aggressive', got '%s'", botLogic.GetRiskLevel())
	}

	// Test bet amount decision
	amount := botLogic.DecideBetAmount()
	if amount <= 0 {
		t.Errorf("Bet amount should be positive, got %d", amount)
	}

	// Test bet amount doesn't exceed balance
	if amount > botLogic.GetBalance() {
		t.Errorf("Bet amount %d exceeds balance %d", amount, botLogic.GetBalance())
	}

	// Test generating bot bet
	botBet := botLogic.GenerateBotBet()
	if botBet == nil {
		t.Fatal("Failed to generate bot bet")
	}
	if botBet.First <= 0 {
		t.Errorf("Bot bet amount should be positive, got %d", botBet.First)
	}
}

func TestBlackjackBotLogicBasicStrategy(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	// Test basic strategy with different hands
	tests := []struct {
		name           string
		playerCards    []*pb.Card
		dealerCard     *pb.Card
		expectedAction pb.BlackjackActionCode
	}{
		{
			name: "Blackjack - should stay",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_A, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_K, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:     &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			expectedAction: pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
		},
		{
			name: "Hard 17 - should stay",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_10, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_7, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:     &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			expectedAction: pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
		},
		{
			name: "Hard 11 - should double down",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_5, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:     &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			expectedAction: pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE,
		},
		{
			name: "Hard 16 vs 7 - should hit",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_10, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:     &pb.Card{Rank: pb.CardRank_RANK_7, Suit: pb.CardSuit_SUIT_DIAMONDS},
			expectedAction: pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
		},
		{
			name: "Soft 18 vs 9 - should hit",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_A, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_7, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:     &pb.Card{Rank: pb.CardRank_RANK_9, Suit: pb.CardSuit_SUIT_DIAMONDS},
			expectedAction: pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create hand with cards
			hand := &pb.BlackjackHand{
				Cards: test.playerCards,
				Point: calculateHandValue(test.playerCards),
			}

			// Test basic strategy
			action := botLogic.basicStrategy(hand, test.dealerCard, []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
				pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE,
			})

			if action != test.expectedAction {
				t.Errorf("Expected action %v, got %v", test.expectedAction, action)
			}
		})
	}
}

func TestBlackjackBotLogicSplitStrategy(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	tests := []struct {
		name         string
		playerCards  []*pb.Card
		dealerCard   *pb.Card
		shouldSplit  bool
		legalActions []pb.BlackjackActionCode
	}{
		{
			name: "Aces - should split",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_A, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_A, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:  &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			shouldSplit: true,
			legalActions: []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_SPLIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
			},
		},
		{
			name: "8s - should split",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_8, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_8, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:  &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			shouldSplit: true,
			legalActions: []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_SPLIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
			},
		},
		{
			name: "10s - should not split",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_10, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_K, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:  &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			shouldSplit: false,
			legalActions: []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_SPLIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hand := &pb.BlackjackHand{
				Cards: test.playerCards,
				Point: calculateHandValue(test.playerCards),
			}

			shouldSplit := botLogic.ShouldSplit(hand, test.dealerCard, test.legalActions)
			if shouldSplit != test.shouldSplit {
				t.Errorf("Expected should split %v, got %v", test.shouldSplit, shouldSplit)
			}
		})
	}
}

func TestBlackjackBotLogicDoubleDownStrategy(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	tests := []struct {
		name         string
		playerCards  []*pb.Card
		dealerCard   *pb.Card
		shouldDouble bool
		legalActions []pb.BlackjackActionCode
	}{
		{
			name: "11 vs 6 - should double",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_5, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:   &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			shouldDouble: true,
			legalActions: []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE,
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
			},
		},
		{
			name: "10 vs 6 - should double",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_4, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:   &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			shouldDouble: true,
			legalActions: []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE,
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
			},
		},
		{
			name: "9 vs 6 - should double",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_3, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:   &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			shouldDouble: true,
			legalActions: []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE,
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
			},
		},
		{
			name: "8 vs 6 - should not double",
			playerCards: []*pb.Card{
				{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_SPADES},
				{Rank: pb.CardRank_RANK_2, Suit: pb.CardSuit_SUIT_HEARTS},
			},
			dealerCard:   &pb.Card{Rank: pb.CardRank_RANK_6, Suit: pb.CardSuit_SUIT_DIAMONDS},
			shouldDouble: false,
			legalActions: []pb.BlackjackActionCode{
				pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE,
				pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
				pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hand := &pb.BlackjackHand{
				Cards: test.playerCards,
				Point: calculateHandValue(test.playerCards),
			}

			shouldDouble := botLogic.ShouldDoubleDown(hand, test.dealerCard, test.legalActions)
			if shouldDouble != test.shouldDouble {
				t.Errorf("Expected should double %v, got %v", test.shouldDouble, shouldDouble)
			}
		})
	}
}

// Helper function to calculate hand value
func calculateHandValue(cards []*pb.Card) int32 {
	total := int32(0)
	aces := int32(0)

	for _, card := range cards {
		switch card.Rank {
		case pb.CardRank_RANK_A:
			total += 11
			aces++
		case pb.CardRank_RANK_J, pb.CardRank_RANK_Q, pb.CardRank_RANK_K:
			total += 10
		default:
			total += int32(card.Rank)
		}
	}

	// Adjust for aces
	for aces > 0 && total > 21 {
		total -= 10
		aces--
	}

	return total
}

func TestBlackjackBotLogicRiskLevels(t *testing.T) {
	// Test different risk levels
	riskLevels := []string{"conservative", "moderate", "aggressive"}

	for _, level := range riskLevels {
		t.Run(level, func(t *testing.T) {
			botLogic := NewBlackjackBotLogic()
			botLogic.SetRiskLevel(level)

			// Test risk tolerance is within expected range
			riskTolerance := botLogic.GetRiskTolerance()
			switch level {
			case "conservative":
				if riskTolerance < 10 || riskTolerance > 30 {
					t.Errorf("Conservative risk tolerance should be 10-30, got %d", riskTolerance)
				}
			case "moderate":
				if riskTolerance < 30 || riskTolerance > 70 {
					t.Errorf("Moderate risk tolerance should be 30-70, got %d", riskTolerance)
				}
			case "aggressive":
				if riskTolerance < 70 || riskTolerance > 100 {
					t.Errorf("Aggressive risk tolerance should be 70-100, got %d", riskTolerance)
				}
			}

			// Test bet percentages
			basePercentage := botLogic.GetBaseBetPercentage()
			maxPercentage := botLogic.GetMaxBetPercentage()

			if basePercentage <= 0 || basePercentage > 1 {
				t.Errorf("Base bet percentage should be 0-1, got %f", basePercentage)
			}
			if maxPercentage <= 0 || maxPercentage > 1 {
				t.Errorf("Max bet percentage should be 0-1, got %f", maxPercentage)
			}
			if maxPercentage < basePercentage {
				t.Errorf("Max bet percentage %f should be >= base bet percentage %f", maxPercentage, basePercentage)
			}
		})
	}
}
