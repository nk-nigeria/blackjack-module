package entity

import (
	"testing"

	pb "github.com/nk-nigeria/cgp-common/proto"
)

func TestNewBlackjackBotLogic(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	if botLogic == nil {
		t.Fatal("NewBlackjackBotLogic returned nil")
	}

	if botLogic.GetBalance() != 10000 {
		t.Errorf("Expected default balance 10000, got %d", botLogic.GetBalance())
	}

	if botLogic.GetRiskLevel() != "moderate" {
		t.Errorf("Expected default risk level 'moderate', got %s", botLogic.GetRiskLevel())
	}
}

func TestBlackjackBotLogic_SetBalance(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	botLogic.SetBalance(50000)

	if botLogic.GetBalance() != 50000 {
		t.Errorf("Expected balance 50000, got %d", botLogic.GetBalance())
	}
}

func TestBlackjackBotLogic_SetRiskLevel(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	// Test conservative
	botLogic.SetRiskLevel("conservative")
	if botLogic.GetRiskLevel() != "conservative" {
		t.Errorf("Expected risk level 'conservative', got %s", botLogic.GetRiskLevel())
	}
	if botLogic.GetBaseBetPercentage() != 0.02 {
		t.Errorf("Expected base bet percentage 0.02, got %f", botLogic.GetBaseBetPercentage())
	}

	// Test aggressive
	botLogic.SetRiskLevel("aggressive")
	if botLogic.GetRiskLevel() != "aggressive" {
		t.Errorf("Expected risk level 'aggressive', got %s", botLogic.GetRiskLevel())
	}
	if botLogic.GetBaseBetPercentage() != 0.10 {
		t.Errorf("Expected base bet percentage 0.10, got %f", botLogic.GetBaseBetPercentage())
	}
}

func TestBlackjackBotLogic_DecideBetAmount(t *testing.T) {
	botLogic := NewBlackjackBotLogic()
	botLogic.SetBalance(100000)

	amount := botLogic.DecideBetAmount()

	// Should be between 5% and 20% of balance (5000-20000)
	if amount < 5000 || amount > 20000 {
		t.Errorf("Bet amount %d should be between 5000 and 20000", amount)
	}

	// Should be rounded to chip values
	validChipValues := []int64{100, 500, 1000, 5000, 10000}
	isValid := false
	for _, chipValue := range validChipValues {
		if amount == chipValue {
			isValid = true
			break
		}
	}
	if !isValid {
		t.Errorf("Bet amount %d should be a valid chip value", amount)
	}
}

func TestBlackjackBotLogic_DecideBettingType(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	betType := botLogic.DecideBettingType()

	validTypes := []pb.BlackjackBetCode{
		pb.BlackjackBetCode_BLACKJACK_BET_NORMAL,
		pb.BlackjackBetCode_BLACKJACK_BET_DOUBLE,
	}

	isValid := false
	for _, validType := range validTypes {
		if betType == validType {
			isValid = true
			break
		}
	}
	if !isValid {
		t.Errorf("Bet type %v should be a valid betting type", betType)
	}
}

func TestBlackjackBotLogic_GenerateBotBet(t *testing.T) {
	botLogic := NewBlackjackBotLogic()
	botLogic.SetBalance(100000)

	bet := botLogic.GenerateBotBet()

	if bet == nil {
		t.Fatal("GenerateBotBet returned nil")
	}

	if bet.First <= 0 {
		t.Errorf("Bet amount should be positive, got %d", bet.First)
	}

	if bet.First > 100000 {
		t.Errorf("Bet amount should not exceed balance, got %d", bet.First)
	}
}

func TestBlackjackBotLogic_DecideGameAction(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	// Test with a good hand (20 points)
	playerHand := &pb.BlackjackHand{
		Point: 20,
		Type:  pb.BlackjackHandType_BLACKJACK_HAND_TYPE_NORMAL,
	}

	dealerUpCard := &pb.Card{
		Rank: pb.CardRank_RANK_6,
	}

	legalActions := []pb.BlackjackActionCode{
		pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
		pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
	}

	action := botLogic.DecideGameAction(playerHand, dealerUpCard, legalActions)

	// With 20 points against dealer 6, should stay
	if action != pb.BlackjackActionCode_BLACKJACK_ACTION_STAY {
		t.Errorf("Expected action STAY with 20 points vs dealer 6, got %v", action)
	}
}

func TestBlackjackBotLogic_BasicStrategy(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	// Test blackjack (21 points)
	playerHand := &pb.BlackjackHand{
		Point: 21,
		Type:  pb.BlackjackHandType_BLACKJACK_HAND_TYPE_21P,
	}

	dealerUpCard := &pb.Card{
		Rank: pb.CardRank_RANK_10,
	}

	legalActions := []pb.BlackjackActionCode{
		pb.BlackjackActionCode_BLACKJACK_ACTION_STAY,
		pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
	}

	action := botLogic.DecideGameAction(playerHand, dealerUpCard, legalActions)

	if action != pb.BlackjackActionCode_BLACKJACK_ACTION_STAY {
		t.Errorf("Expected action STAY with blackjack, got %v", action)
	}
}

func TestBlackjackBotLogic_Reset(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	// Add some history
	bet := &pb.BlackjackPlayerBet{
		UserId: "test",
		First:  1000,
	}
	botLogic.AddBetHistory(bet)

	action := &pb.BlackjackAction{
		UserId: "test",
		Code:   pb.BlackjackActionCode_BLACKJACK_ACTION_HIT,
	}
	botLogic.AddActionHistory(action)

	// Reset
	botLogic.Reset()

	if len(botLogic.GetBetHistory()) != 0 {
		t.Errorf("Bet history should be empty after reset, got %d items", len(botLogic.GetBetHistory()))
	}

	if len(botLogic.GetActionHistory()) != 0 {
		t.Errorf("Action history should be empty after reset, got %d items", len(botLogic.GetActionHistory()))
	}
}

func TestBlackjackBotLogic_GetCardValue(t *testing.T) {
	botLogic := NewBlackjackBotLogic()

	// Test Ace
	ace := &pb.Card{Rank: pb.CardRank_RANK_A}
	if botLogic.getCardValue(ace) != 11 {
		t.Errorf("Expected Ace value 11, got %d", botLogic.getCardValue(ace))
	}

	// Test King
	king := &pb.Card{Rank: pb.CardRank_RANK_J}
	if botLogic.getCardValue(king) != 10 {
		t.Errorf("Expected King value 10, got %d", botLogic.getCardValue(king))
	}

	// Test number card
	seven := &pb.Card{Rank: pb.CardRank_RANK_7}
	if botLogic.getCardValue(seven) != 7 {
		t.Errorf("Expected 7 value 7, got %d", botLogic.getCardValue(seven))
	}
}
