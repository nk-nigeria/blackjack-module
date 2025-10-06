package entity

import (
	"math/rand"
	"time"

	pb "github.com/nk-nigeria/cgp-common/proto"
)

// BlackjackBotLogic handles bot decision making for blackjack game
type BlackjackBotLogic struct {
	// Betting strategy configuration
	bettingStrategy BettingStrategy
	// Risk tolerance level (0-100)
	riskTolerance int
	// Previous bet history for this bot
	betHistory []*pb.BlackjackPlayerBet
	// Current balance
	currentBalance int64
	// Betting patterns
	bettingPatterns map[string]int
	// Game action history
	actionHistory []*pb.BlackjackAction
}

// BettingStrategy defines how bot should bet
type BettingStrategy struct {
	// Preferred betting types
	PreferredBetTypes []pb.BlackjackBetCode
	// Betting amount strategy
	BetAmountStrategy BetAmountStrategy
	// Risk level (conservative, moderate, aggressive)
	RiskLevel string
}

// BetAmountStrategy defines how much bot should bet
type BetAmountStrategy struct {
	// Base bet amount as percentage of balance
	BaseBetPercentage float64
	// Maximum bet amount as percentage of balance
	MaxBetPercentage float64
	// Progressive betting (increase bet after loss)
	ProgressiveBetting bool
	// Martingale multiplier
	MartingaleMultiplier float64
}

// NewBlackjackBotLogic creates a new bot logic instance
func NewBlackjackBotLogic() *BlackjackBotLogic {
	rand.Seed(time.Now().UnixNano())

	return &BlackjackBotLogic{
		bettingStrategy: BettingStrategy{
			PreferredBetTypes: []pb.BlackjackBetCode{
				pb.BlackjackBetCode_BLACKJACK_BET_NORMAL,
				pb.BlackjackBetCode_BLACKJACK_BET_DOUBLE,
			},
			BetAmountStrategy: BetAmountStrategy{
				BaseBetPercentage:    0.05, // 5% of balance
				MaxBetPercentage:     0.20, // 20% of balance
				ProgressiveBetting:   true,
				MartingaleMultiplier: 2.0,
			},
			RiskLevel: "moderate",
		},
		riskTolerance:   rand.Intn(41) + 30, // 30-70
		betHistory:      make([]*pb.BlackjackPlayerBet, 0),
		bettingPatterns: make(map[string]int),
		actionHistory:   make([]*pb.BlackjackAction, 0),
		currentBalance:  10000, // Default balance
	}
}

// Reset resets the bot logic state
func (b *BlackjackBotLogic) Reset() {
	b.betHistory = make([]*pb.BlackjackPlayerBet, 0)
	b.bettingPatterns = make(map[string]int)
	b.actionHistory = make([]*pb.BlackjackAction, 0)
}

// SetBalance updates the bot's current balance
func (b *BlackjackBotLogic) SetBalance(balance int64) {
	b.currentBalance = balance
}

// GetBalance returns the bot's current balance
func (b *BlackjackBotLogic) GetBalance() int64 {
	return b.currentBalance
}

// AddBetHistory adds a bet to the history
func (b *BlackjackBotLogic) AddBetHistory(bet *pb.BlackjackPlayerBet) {
	b.betHistory = append(b.betHistory, bet)
}

// GetBetHistory returns the bot's bet history
func (b *BlackjackBotLogic) GetBetHistory() []*pb.BlackjackPlayerBet {
	return b.betHistory
}

// AddActionHistory adds an action to the history
func (b *BlackjackBotLogic) AddActionHistory(action *pb.BlackjackAction) {
	b.actionHistory = append(b.actionHistory, action)
}

// GetActionHistory returns the bot's action history
func (b *BlackjackBotLogic) GetActionHistory() []*pb.BlackjackAction {
	return b.actionHistory
}

// DecideBettingType decides which type of bet to make
func (b *BlackjackBotLogic) DecideBettingType() pb.BlackjackBetCode {
	// Analyze betting patterns and make decision
	betType := b.analyzeBettingPatterns()

	// Add some randomness based on risk tolerance
	if rand.Intn(100) < b.riskTolerance {
		// Higher risk tolerance - more likely to double down
		if rand.Intn(100) < 30 { // 30% chance for double down
			betType = pb.BlackjackBetCode_BLACKJACK_BET_DOUBLE
		}
	}

	return betType
}

// DecideBetAmount decides how much to bet
func (b *BlackjackBotLogic) DecideBetAmount() int64 {
	strategy := b.bettingStrategy.BetAmountStrategy

	// Calculate base bet amount
	baseAmount := int64(float64(b.currentBalance) * strategy.BaseBetPercentage)

	// Apply progressive betting if enabled
	if strategy.ProgressiveBetting && len(b.betHistory) > 0 {
		lastBet := b.betHistory[len(b.betHistory)-1]
		// If last bet was a loss, increase bet amount
		if b.wasLastBetLoss() {
			baseAmount = int64(float64(lastBet.First) * strategy.MartingaleMultiplier)
		}
	}

	// Ensure bet amount doesn't exceed maximum
	maxAmount := int64(float64(b.currentBalance) * strategy.MaxBetPercentage)
	if baseAmount > maxAmount {
		baseAmount = maxAmount
	}

	// Ensure bet amount doesn't exceed current balance
	if baseAmount > b.currentBalance {
		baseAmount = b.currentBalance
	}

	// Round to nearest chip value
	baseAmount = b.roundToChipValue(baseAmount)

	return baseAmount
}

// GenerateBotBet generates a complete betting decision for the bot
func (b *BlackjackBotLogic) GenerateBotBet() *pb.BlackjackPlayerBet {
	// Decide betting type
	betType := b.DecideBettingType()

	// Decide bet amount
	amount := b.DecideBetAmount()

	// Create bet based on type
	bet := &pb.BlackjackPlayerBet{
		UserId: "", // Will be set by caller
		First:  amount,
		Second: 0, // Split bet, initially 0
	}

	// If double down, set the bet amount
	if betType == pb.BlackjackBetCode_BLACKJACK_BET_DOUBLE {
		// Double down means we're ready to double our bet
		// The actual doubling will be handled by the game logic
	}

	// Add to history
	b.AddBetHistory(bet)

	// Update betting patterns
	b.updateBettingPatterns(betType)

	return bet
}

// DecideGameAction decides what action to take during the game
func (b *BlackjackBotLogic) DecideGameAction(playerHand *pb.BlackjackHand, dealerUpCard *pb.Card, legalActions []pb.BlackjackActionCode) pb.BlackjackActionCode {
	// Basic strategy implementation
	action := b.basicStrategy(playerHand, dealerUpCard, legalActions)

	// Add some randomness based on risk tolerance
	if rand.Intn(100) < b.riskTolerance {
		// Higher risk tolerance - more likely to take risky actions
		if action == pb.BlackjackActionCode_BLACKJACK_ACTION_STAY && rand.Intn(100) < 20 {
			// 20% chance to hit instead of stay when risk tolerance is high
			if b.containsAction(legalActions, pb.BlackjackActionCode_BLACKJACK_ACTION_HIT) {
				action = pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
			}
		}
	}

	return action
}

// basicStrategy implements basic blackjack strategy
func (b *BlackjackBotLogic) basicStrategy(playerHand *pb.BlackjackHand, dealerUpCard *pb.Card, legalActions []pb.BlackjackActionCode) pb.BlackjackActionCode {
	// Use the hand's calculated point which already considers Ace flexibility
	playerPoints := playerHand.Point
	dealerPoints := b.getCardValue(dealerUpCard)

	// Check for blackjack
	if playerPoints == 21 {
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	}

	// Check for bust
	if playerPoints > 21 {
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	}

	// Check if hand has Ace (soft total)
	hasAce := false
	for _, card := range playerHand.Cards {
		if card.Rank == pb.CardRank_RANK_A {
			hasAce = true
			break
		}
	}

	// Soft totals (with Ace)
	if hasAce {
		return b.softTotalStrategy(playerPoints, dealerPoints, legalActions)
	}

	// Hard totals
	return b.hardTotalStrategy(playerPoints, dealerPoints, legalActions)
}

// softTotalStrategy handles soft totals (with Ace)
func (b *BlackjackBotLogic) softTotalStrategy(playerPoints, dealerPoints int32, legalActions []pb.BlackjackActionCode) pb.BlackjackActionCode {
	// Basic soft total strategy
	switch playerPoints {
	case 20, 21:
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	case 19:
		if dealerPoints >= 6 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
	case 18:
		if dealerPoints >= 9 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	case 17:
		if dealerPoints >= 7 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	default:
		return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
	}
}

// hardTotalStrategy handles hard totals
func (b *BlackjackBotLogic) hardTotalStrategy(playerPoints, dealerPoints int32, legalActions []pb.BlackjackActionCode) pb.BlackjackActionCode {
	// Basic hard total strategy
	switch playerPoints {
	case 17, 18, 19, 20, 21:
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	case 16:
		if dealerPoints >= 7 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	case 15:
		if dealerPoints >= 7 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	case 14:
		if dealerPoints >= 7 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	case 13:
		if dealerPoints >= 7 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
	case 12:
		if dealerPoints >= 4 && dealerPoints <= 6 {
			return pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
		}
		return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
	default:
		return pb.BlackjackActionCode_BLACKJACK_ACTION_HIT
	}
}

// getCardValue returns the value of a card
func (b *BlackjackBotLogic) getCardValue(card *pb.Card) int32 {
	switch card.Rank {
	case pb.CardRank_RANK_A:
		return 11 // Default to 11, will be adjusted in calculateHandValue
	case pb.CardRank_RANK_J, pb.CardRank_RANK_Q, pb.CardRank_RANK_K:
		return 10
	default:
		return int32(card.Rank)
	}
}

// containsAction checks if an action is in the legal actions list
func (b *BlackjackBotLogic) containsAction(legalActions []pb.BlackjackActionCode, action pb.BlackjackActionCode) bool {
	for _, legalAction := range legalActions {
		if legalAction == action {
			return true
		}
	}
	return false
}

// analyzeBettingPatterns analyzes current betting patterns to make better decisions
func (b *BlackjackBotLogic) analyzeBettingPatterns() pb.BlackjackBetCode {
	// If no history, use preferred bet types
	if len(b.betHistory) == 0 {
		return b.bettingStrategy.PreferredBetTypes[rand.Intn(len(b.bettingStrategy.PreferredBetTypes))]
	}

	// Analyze recent bets to avoid patterns
	recentBets := b.getRecentBets(5)

	// Count recent bets by type
	betTypeCounts := make(map[pb.BlackjackBetCode]int)
	for range recentBets {
		// For now, we'll consider all bets as normal bets
		// In a more sophisticated implementation, we'd track the actual bet types
		betTypeCounts[pb.BlackjackBetCode_BLACKJACK_BET_NORMAL]++
	}

	// Find least used bet type in recent history
	var leastUsedBetType pb.BlackjackBetCode
	minCount := 999
	for betType, count := range betTypeCounts {
		if count < minCount {
			minCount = count
			leastUsedBetType = betType
		}
	}

	// If all bet types are equally used, choose randomly
	if minCount == 999 {
		return b.bettingStrategy.PreferredBetTypes[rand.Intn(len(b.bettingStrategy.PreferredBetTypes))]
	}

	return leastUsedBetType
}

// getRecentBets returns the most recent n bets
func (b *BlackjackBotLogic) getRecentBets(n int) []*pb.BlackjackPlayerBet {
	if len(b.betHistory) <= n {
		return b.betHistory
	}
	return b.betHistory[len(b.betHistory)-n:]
}

// wasLastBetLoss checks if the last bet was a loss
func (b *BlackjackBotLogic) wasLastBetLoss() bool {
	// This would need to be implemented based on game results
	// For now, we'll assume 50% chance of loss
	return rand.Intn(2) == 0
}

// updateBettingPatterns updates the betting pattern statistics
func (b *BlackjackBotLogic) updateBettingPatterns(betType pb.BlackjackBetCode) {
	betTypeKey := betType.String()
	b.bettingPatterns[betTypeKey]++
}

// roundToChipValue rounds the bet amount to the nearest valid chip value
func (b *BlackjackBotLogic) roundToChipValue(amount int64) int64 {
	// Define chip values (similar to baccarat)
	chipValues := []int64{100, 500, 1000, 5000, 10000}

	// Find the closest chip value
	closest := int64(0)
	minDiff := int64(999999)

	for _, chipValue := range chipValues {
		diff := abs(amount - chipValue)
		if diff < minDiff {
			minDiff = diff
			closest = chipValue
		}
	}

	return closest
}

// abs returns absolute value
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// SetRiskLevel changes the bot's risk level
func (b *BlackjackBotLogic) SetRiskLevel(level string) {
	b.bettingStrategy.RiskLevel = level

	switch level {
	case "conservative":
		b.riskTolerance = rand.Intn(21) + 10 // 10-30
		b.bettingStrategy.BetAmountStrategy.BaseBetPercentage = 0.02
		b.bettingStrategy.BetAmountStrategy.MaxBetPercentage = 0.10
	case "moderate":
		b.riskTolerance = rand.Intn(41) + 30 // 30-70
		b.bettingStrategy.BetAmountStrategy.BaseBetPercentage = 0.05
		b.bettingStrategy.BetAmountStrategy.MaxBetPercentage = 0.20
	case "aggressive":
		b.riskTolerance = rand.Intn(31) + 70 // 70-100
		b.bettingStrategy.BetAmountStrategy.BaseBetPercentage = 0.10
		b.bettingStrategy.BetAmountStrategy.MaxBetPercentage = 0.40
	}
}

// GetRiskLevel returns the current risk level
func (b *BlackjackBotLogic) GetRiskLevel() string {
	return b.bettingStrategy.RiskLevel
}

// GetRiskTolerance returns the current risk tolerance
func (b *BlackjackBotLogic) GetRiskTolerance() int {
	return b.riskTolerance
}

// GetBettingStrategy returns the current betting strategy
func (b *BlackjackBotLogic) GetBettingStrategy() BettingStrategy {
	return b.bettingStrategy
}

// GetBaseBetPercentage returns the base bet percentage
func (b *BlackjackBotLogic) GetBaseBetPercentage() float64 {
	return b.bettingStrategy.BetAmountStrategy.BaseBetPercentage
}

// GetMaxBetPercentage returns the max bet percentage
func (b *BlackjackBotLogic) GetMaxBetPercentage() float64 {
	return b.bettingStrategy.BetAmountStrategy.MaxBetPercentage
}

// GetProgressiveBetting returns whether progressive betting is enabled
func (b *BlackjackBotLogic) GetProgressiveBetting() bool {
	return b.bettingStrategy.BetAmountStrategy.ProgressiveBetting
}

// GetMartingaleMultiplier returns the martingale multiplier
func (b *BlackjackBotLogic) GetMartingaleMultiplier() float64 {
	return b.bettingStrategy.BetAmountStrategy.MartingaleMultiplier
}

// GetBettingPatterns returns the betting patterns
func (b *BlackjackBotLogic) GetBettingPatterns() map[string]int {
	return b.bettingPatterns
}

// SetBettingStrategy updates the betting strategy
func (b *BlackjackBotLogic) SetBettingStrategy(strategy BettingStrategy) {
	b.bettingStrategy = strategy
}
