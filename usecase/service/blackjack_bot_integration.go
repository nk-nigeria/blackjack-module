package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/pkg/packager"
	"github.com/nk-nigeria/cgp-common/bot"
)

// BlackjackBotIntegration implements BotIntegration for Blackjack game
type BlackjackBotIntegration struct {
	db           *sql.DB
	matchID      string
	betAmount    int64
	playerCount  int
	botCount     int
	maxPlayers   int
	minPlayers   int
	lastResult   int
	activeTables int
	botHelper    *bot.BotIntegrationHelper
}

// NewBlackjackBotIntegration creates a new Blackjack bot integration
func NewBlackjackBotIntegration(db *sql.DB) *BlackjackBotIntegration {
	integration := &BlackjackBotIntegration{
		db:           db,
		maxPlayers:   3, // Blackjack typically has 5 players max
		minPlayers:   1, // Minimum 1 player to start
		activeTables: 0, // Will be updated from game state
	}
	integration.botHelper = bot.NewBotIntegrationHelper(db, integration, entity.BotLoader)
	integration.LoadBotConfig(context.Background())

	return integration
}

// GetGameCode returns the game code for Blackjack
func (b *BlackjackBotIntegration) GetGameCode() string {
	gameCode := entity.ModuleName
	return gameCode
}

// GetMinChipBalance returns minimum chip balance for bots
func (b *BlackjackBotIntegration) GetMinChipBalance() int64 {
	return 100000
}

// GetMatchInfo returns current match information
func (b *BlackjackBotIntegration) GetMatchInfo(ctx context.Context) *bot.MatchInfo {
	return &bot.MatchInfo{
		MatchID:           b.matchID,
		BetAmount:         b.betAmount,
		PlayerCount:       b.playerCount,
		BotCount:          b.botCount,
		MaxPlayers:        b.maxPlayers,
		MinPlayers:        b.minPlayers,
		IsFull:            b.IsMatchFull(),
		LastGameResult:    b.lastResult,
		ActiveTablesCount: b.activeTables,
	}
}

// AddBotToMatch adds bots to the current match
func (b *BlackjackBotIntegration) AddBotToMatch(ctx context.Context, numBots int) error {
	// Add bots to match using the existing processor logic
	// This requires access to the processor and state from context
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	if procPkg == nil {
		return fmt.Errorf("processor package not found in context")
	}

	state := procPkg.GetState()

	// Add bots to match using existing processor method
	err := procPkg.GetProcessor().AddBotToMatch(
		ctx,
		procPkg.GetLogger(),
		procPkg.GetNK(),
		procPkg.GetDb(),
		procPkg.GetDispatcher(),
		state,
		numBots,
	)
	if err != nil {
		return err
	}

	// Update player count
	b.playerCount = state.GetPresenceSize()
	b.botCount = b.playerCount - state.GetPresenceNotBotSize()

	return nil
}

// RemoveBotFromMatch removes bots from the current match
func (b *BlackjackBotIntegration) RemoveBotFromMatch(ctx context.Context, botUserID string) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	if procPkg == nil {
		return fmt.Errorf("processor package not found in context")
	}

	botLeftCount, ok := ctx.Value("bot_left_count").(int)
	if !ok {
		botLeftCount = 1
	}

	state := procPkg.GetState()

	botPresenceList := state.GetBotPresences()

	botUserIDs := make([]string, len(botPresenceList))
	for i, presence := range botPresenceList {
		botUserIDs[i] = presence.GetUserId()
	}

	if len(botUserIDs) == 0 {
		fmt.Printf("[DEBUG] [BlackjackBotIntegration] No bots in presence list\n")
		return nil
	}

	// Ensure we don't try to remove more bots than available
	if botLeftCount > len(botUserIDs) {
		botLeftCount = len(botUserIDs)
		fmt.Printf("[DEBUG] [BlackjackBotIntegration] Adjusted botLeftCount to %d (available bots: %d)\n",
			botLeftCount, len(botUserIDs))
	}

	// Random select bot userIDs to remove
	selectedBotUserIDs := make([]string, 0, botLeftCount)
	availableBots := make([]string, len(botUserIDs))
	copy(availableBots, botUserIDs)

	for i := 0; i < botLeftCount; i++ {
		if len(availableBots) == 0 {
			break
		}

		// Random select index
		randomIndex := rand.Intn(len(availableBots))
		selectedBotUserID := availableBots[randomIndex]

		// Add to selected list
		selectedBotUserIDs = append(selectedBotUserIDs, selectedBotUserID)

		// Remove from available list to avoid duplicate selection
		availableBots = append(availableBots[:randomIndex], availableBots[randomIndex+1:]...)
	}

	fmt.Printf("[DEBUG] [BlackjackBotIntegration] Random selected %d bots to remove: %v\n",
		len(selectedBotUserIDs), selectedBotUserIDs)

	for _, selectedBotUserID := range selectedBotUserIDs {
		fmt.Printf("[DEBUG] [BlackjackBotIntegration] Removing bot %s from match\n", selectedBotUserID)

		err := procPkg.GetProcessor().RemoveBotFromMatch(
			ctx,
			procPkg.GetLogger(),
			procPkg.GetNK(),
			procPkg.GetDb(),
			procPkg.GetDispatcher(),
			state,
			selectedBotUserID,
		)
		if err != nil {
			fmt.Printf("[ERROR] [BlackjackBotIntegration] Failed to remove bot %s: %v\n", selectedBotUserID, err)
			return err
		}

		fmt.Printf("[DEBUG] [BlackjackBotIntegration] Successfully removed bot %s\n", selectedBotUserID)
	}

	b.playerCount = state.GetPresenceSize()
	b.botCount -= botLeftCount
	return nil
}

// GetMaxPlayers returns maximum players allowed
func (b *BlackjackBotIntegration) GetMaxPlayers() int {
	return b.maxPlayers
}

// GetMinPlayers returns minimum players required
func (b *BlackjackBotIntegration) GetMinPlayers() int {
	return b.minPlayers
}

// IsMatchFull returns true if match is full
func (b *BlackjackBotIntegration) IsMatchFull() bool {
	return b.playerCount >= b.maxPlayers
}

// GetCurrentPlayerCount returns current player count
func (b *BlackjackBotIntegration) GetCurrentPlayerCount() int {
	return b.playerCount
}

// GetCurrentBetAmount returns current bet amount
func (b *BlackjackBotIntegration) GetCurrentBetAmount() int64 {
	return b.betAmount
}

// GetLastGameResult returns last game result
func (b *BlackjackBotIntegration) GetLastGameResult() int {
	return b.lastResult
}

// GetActiveTablesCount returns active tables count
func (b *BlackjackBotIntegration) GetActiveTablesCount() int {
	return b.activeTables
}

// SetMatchState updates the match state for bot decision making
func (b *BlackjackBotIntegration) SetMatchState(matchID string, betAmount int64, playerCount int, botCount int, activeTables int) {
	b.matchID = matchID
	b.betAmount = betAmount
	b.playerCount = playerCount
	b.botCount = botCount
	b.activeTables = activeTables
}

// ProcessBotLogic processes all bot-related logic
func (b *BlackjackBotIntegration) ProcessJoinBotLogic(ctx context.Context) error {
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] ProcessJoinBotLogic called for matchID=%s, betAmount=%d, playerCount=%d, botCount=%d\n",
		b.matchID, b.betAmount, b.playerCount, b.botCount)

	return b.botHelper.ProcessJoinBotLogic(ctx)
}

// ProcessBotLeaveLogic processes bot leave logic for a specific bot
func (b *BlackjackBotIntegration) ProcessBotLeaveLogic(ctx context.Context) error {
	return b.botHelper.ProcessBotLeaveLogic(ctx, "")
}

// CheckAndJoinExpiredBots checks if any bots should join based on their join time
func (b *BlackjackBotIntegration) CheckAndJoinExpiredBots(ctx context.Context) (bool, error) {
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] CheckAndJoinExpiredBots called for matchID=%s, betAmount=%d, playerCount=%d\n",
		b.matchID, b.betAmount, b.playerCount)

	// Debug current bot config
	config := b.botHelper.GetBotConfig()
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] Current bot config has %d join rules\n", len(config.BotJoinRules))

	result, err := b.botHelper.CheckAndJoinExpiredBots(ctx)
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] CheckAndJoinExpiredBots result: joined=%v, err=%v\n", result, err)

	return result, err
}

// GetBotHelper returns the bot helper for direct access
func (b *BlackjackBotIntegration) GetBotHelper() *bot.BotIntegrationHelper {
	return b.botHelper
}

// LoadBotConfig loads bot configuration from database
func (b *BlackjackBotIntegration) LoadBotConfig(ctx context.Context) error {
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] Loading bot config for game: %s\n", b.GetGameCode())

	configLoader := bot.NewConfigLoader(b.db)
	config, err := configLoader.LoadConfigFromDatabase(ctx, b.GetGameCode())
	if err != nil {
		fmt.Printf("[ERROR] [BlackjackBotIntegration] Failed to load bot config: %v\n", err)
		return fmt.Errorf("failed to load bot config: %w", err)
	}

	fmt.Printf("[DEBUG] [BlackjackBotIntegration] Successfully loaded bot config:\n")
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] - BotJoinRules count: %d\n", len(config.BotJoinRules))
	for i, rule := range config.BotJoinRules {
		fmt.Printf("[DEBUG] [BlackjackBotIntegration] - Rule[%d]: minBet=%d, maxBet=%d, minUsers=%d, maxUsers=%d, joinPercent=%d\n",
			i, rule.MinBet, rule.MaxBet, rule.MinUsers, rule.MaxUsers, rule.JoinPercent)
	}
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] - BotLeaveRules count: %d\n", len(config.BotLeaveRules))
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] - BotCreateTableRules count: %d\n", len(config.BotCreateTableRules))

	b.botHelper.SetBotConfig(config)
	fmt.Printf("[DEBUG] [BlackjackBotIntegration] Bot config set to bot helper\n")
	return nil
}

// SaveBotConfig saves bot configuration to database
func (b *BlackjackBotIntegration) SaveBotConfig(ctx context.Context) error {
	config := b.botHelper.GetBotConfig()
	configLoader := bot.NewConfigLoader(b.db)

	err := configLoader.SaveConfigToDatabase(ctx, b.GetGameCode(), config)
	if err != nil {
		return fmt.Errorf("failed to save bot config: %w", err)
	}

	return nil
}

// DebugPendingRequests prints pending requests for debugging
func (b *BlackjackBotIntegration) DebugPendingRequests() {
	b.botHelper.DebugPendingRequests()
}
