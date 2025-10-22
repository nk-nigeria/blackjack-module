package smstates

import (
	"context"
	"math"
	"strings"

	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/pkg/global"
	"github.com/nk-nigeria/blackjack-module/pkg/packager"
	"github.com/nk-nigeria/blackjack-module/usecase/service"
	"github.com/nk-nigeria/cgp-common/bot"
	pb "github.com/nk-nigeria/cgp-common/proto"
)

type StatePreparing struct {
	StateBase
}

func NewStatePreparing(fn FireFn) StateHandler {
	return &StatePreparing{
		StateBase: NewStateBase(fn),
	}
}
func (s *StatePreparing) Enter(ctx context.Context, _ ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	state.SetAllowBet(true)
	state.InitUserBet()
	procPkg.GetLogger().Info("state %v", state.Presences)
	state.SetUpCountDown(entity.GameStateDuration[state.GetGameState()])
	// remove all user not interact 2 game conti
	listPrecense := state.GetPresenceNotInteract(2)
	if len(listPrecense) > 0 {
		listUserId := make([]string, len(listPrecense))
		for _, p := range listPrecense {
			listUserId = append(listUserId, p.GetUserId())
		}
		procPkg.GetLogger().Info("Kick %d user from math %s",
			len(listPrecense), strings.Join(listUserId, ","))
		state.AddLeavePresence(listPrecense...)
	}
	procPkg.GetProcessor().ProcessApplyPresencesLeave(ctx,
		procPkg.GetLogger(),
		procPkg.GetNK(),
		procPkg.GetDb(),
		procPkg.GetDispatcher(),
		state,
	)

	// Initialize bot integration if not already done
	botIntegration := global.GetGlobalBotIntegration()
	if botIntegration == nil {
		botIntegration = service.NewBlackjackBotIntegration(procPkg.GetDb())
		global.SetGlobalBotIntegration(botIntegration)
	}

	// Type assert to get the actual bot integration
	if blackjackBotIntegration, ok := botIntegration.(*service.BlackjackBotIntegration); ok {
		// Set match state for bot decision making before processing bot logic
		blackjackBotIntegration.SetMatchState(
			state.GetMatchID(),
			state.GetBetAmount(),
			state.GetPresenceSize(),
			0, // lastResult - chưa có kết quả game
			1, // activeTables
		)

		// Process bot join logic during preparing phase
		if err := blackjackBotIntegration.ProcessJoinBotLogic(ctx); err != nil {
			procPkg.GetLogger().Error("Failed to process bot join logic: %v", err)
		}
	}

	procPkg.GetProcessor().NotifyUpdateGameState(
		state,
		procPkg.GetLogger(),
		procPkg.GetDispatcher(),
		&pb.UpdateGameState{
			State:     pb.GameState_GAME_STATE_PREPARING,
			CountDown: int64(math.Round(float64(state.GetRemainCountDown()))),
		},
	)
	for _, precense := range state.GetPresences() {
		state.PresencesNoInteract[precense.GetUserId()]++
	}

	// Initialize bot betting turns for all bots
	procPkg.GetLogger().Info("[preparing] Initializing bot betting turns")
	for _, presence := range state.GetBotPresences() {
		if botPresence, ok := presence.(*bot.BotPresence); ok {
			procPkg.GetLogger().Info("[preparing] Setting up bot turn for: %s", botPresence.GetUserId())
			state.InitTurnBot(botPresence)
		}
	}

	return nil
}

func (s *StatePreparing) Exit(_ context.Context, _ ...interface{}) error {
	return nil
}

func (s *StatePreparing) Process(ctx context.Context, args ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	remain := state.GetRemainCountDown()
	if state.GetPresenceNotBotSize() == 0 {
		s.Trigger(ctx, TriggerStateFinishFailed)
		return nil
	}
	// Get messages from real users
	message := procPkg.GetMessages()

	// Process bot loops (triggers scheduled bot turns)
	state.BotLoop()

	// Get bot messages from queue and merge with user messages
	botMessages := state.Messages()
	message = append(message, botMessages...)

	if len(message) > 0 {
		procPkg.GetProcessor().ProcessMessageFromUser(ctx,
			procPkg.GetLogger(),
			procPkg.GetNK(),
			procPkg.GetDb(),
			procPkg.GetDispatcher(),
			message, procPkg.GetState())
	}

	// Check if bot should join based on time (similar to Baccarat module)
	if state.GetPresenceSize() < entity.MaxPresences {
		// Check if bot should join based on time
		botCtx := packager.GetContextWithProcessorPackager(procPkg)
		if blackjackBotIntegration, ok := global.GetGlobalBotIntegration().(*service.BlackjackBotIntegration); ok {
			joined, err := blackjackBotIntegration.CheckAndJoinExpiredBots(botCtx)
			if err != nil {
				procPkg.GetLogger().Error("[preparing] Bot join error: %v", err)
			} else if joined {
				procPkg.GetLogger().Info("[preparing] Bot joined based on time")
				// Initialize betting turn for newly joined bots
				for _, presence := range state.GetBotPresences() {
					if botPresence, ok := presence.(*bot.BotPresence); ok {
						if !state.IsBet(botPresence.GetUserId()) {
							state.InitTurnBot(botPresence)
						}
					}
				}
			}
		}
	} else {
		procPkg.GetLogger().Info("[preparing] Skip bot join - maximum players reached (%d)", entity.MaxPresences)
	}

	if remain <= 0 {
		state.SetAllowBet(false)
		if state.IsReadyToPlay() {
			s.Trigger(ctx, TriggerStateFinishSuccess)
		} else {
			// change to wait
			s.Trigger(ctx, TriggerStateFinishFailed)
		}
		return nil
	} else {
		if state.IsNeedNotifyCountDown() {
			remainCountDown := int(math.Round(state.GetRemainCountDown()))
			procPkg.GetProcessor().NotifyUpdateGameState(
				state,
				procPkg.GetLogger(),
				procPkg.GetDispatcher(),
				&pb.UpdateGameState{
					State:     pb.GameState_GAME_STATE_PREPARING,
					CountDown: int64(remainCountDown),
				},
			)
			state.SetLastCountDown(remainCountDown)
		}
	}
	return nil
}
