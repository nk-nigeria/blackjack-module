package smstates

import (
	"context"
	"math"

	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/pkg/packager"
	"github.com/nk-nigeria/cgp-common/bot"
	pb "github.com/nk-nigeria/cgp-common/proto"
)

type StatePlay struct {
	StateBase
	// lastPrintLog time.Time
}

func NewStatePlay(fn FireFn) StateHandler {
	return &StatePlay{
		StateBase: NewStateBase(fn),
	}
}

func (s *StatePlay) Enter(ctx context.Context, _ ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	// Setup count down
	state.SetUpCountDown(entity.GameStateDuration[state.GetGameState()])
	procPkg.GetProcessor().NotifyUpdateGameState(
		state,
		procPkg.GetLogger(),
		procPkg.GetDispatcher(),
		&pb.UpdateGameState{
			State:     pb.GameState_GAME_STATE_PLAY,
			CountDown: int64(math.Round(state.GetRemainCountDown())),
		},
	)
	state.SetupMatchPresence()
	procPkg.GetProcessor().ProcessNewGame(
		procPkg.GetContext(),
		procPkg.GetLogger(),
		procPkg.GetNK(),
		procPkg.GetDb(),
		procPkg.GetDispatcher(), state)
	return nil
}

func (s *StatePlay) Exit(_ context.Context, _ ...interface{}) error {
	return nil
}

func (s *StatePlay) Process(ctx context.Context, args ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	// if time.Since(s.lastPrintLog) > 2*time.Second {
	// 	procPkg.GetLogger().WithField("state", StatePlay.String()).Debug("Process")
	// 	s.lastPrintLog = time.Now()
	// }
	// remain := state.GetRemainCountDown()
	// if remain <= 0 {
	// 	procPkg.GetLogger().Info("[play] timeout reach %v", remain)
	// 	s.Trigger(ctx, TriggerStateFinishFailed)
	// 	return nil
	// }
	if state.IsGameEnded() {
		procPkg.GetLogger().Info("[play] ended")
		s.Trigger(ctx, TriggerStateFinishFailed)
		return nil
	}

	procPkg.GetProcessor().ProcessTurnbase(ctx,
		procPkg.GetLogger(),
		procPkg.GetNK(),
		procPkg.GetDb(),
		procPkg.GetDispatcher(),
		procPkg.GetState(),
	)

	// Get messages from real users
	message := procPkg.GetMessages()

	// Check if insurance round and trigger bot insurance decisions
	if state.IsAllowInsurance() {
		procPkg.GetLogger().Info("[play] Insurance round - checking bots")
		// Insurance round - all bots decide simultaneously
		for _, presence := range state.GetBotPresences() {
			if botPresence, ok := presence.(*bot.BotPresence); ok {
				userId := botPresence.GetUserId()
				// Check if bot hasn't made insurance decision yet
				if !state.HasInsuranceBet(userId) {
					procPkg.GetLogger().Info("[play] Bot insurance check for: %s", userId)
					// Bot decides whether to take insurance
					state.BotInsuranceAction(botPresence)
				}
			}
		}
	}

	// Check if current turn is a bot and trigger bot action
	currentTurn := state.GetCurrentTurn()
	if currentTurn != "" && state.IsAllowAction() {
		// Check if current turn is a bot
		for _, presence := range state.GetBotPresences() {
			if botPresence, ok := presence.(*bot.BotPresence); ok && botPresence.GetUserId() == currentTurn {
				// Current turn is a bot - trigger bot action
				procPkg.GetLogger().Info("[play] Bot turn detected for: %s", currentTurn)

				// Get legal actions for this bot
				legalActions := state.GetLegalActions()
				if len(legalActions) > 0 {
					// Bot decides action and queues message
					state.BotAction(botPresence, legalActions)
				}
				break
			}
		}
	}

	// Get bot messages from queue and merge with user messages
	botMessages := state.Messages()
	message = append(message, botMessages...)

	if len(message) > 0 {
		procPkg.GetProcessor().ProcessMessageFromUser(
			procPkg.GetContext(),
			procPkg.GetLogger(),
			procPkg.GetNK(),
			procPkg.GetDb(),
			procPkg.GetDispatcher(),
			message,
			state,
		)
	}

	if state.IsNeedNotifyCountDown() {
		remainCountDown := int(math.Round(state.GetRemainCountDown()))
		if remainCountDown < 0 {
			return nil
		}
		procPkg.GetProcessor().NotifyUpdateGameState(
			state,
			procPkg.GetLogger(),
			procPkg.GetDispatcher(),
			&pb.UpdateGameState{
				State:     pb.GameState_GAME_STATE_PLAY,
				CountDown: int64(remainCountDown),
			},
		)
		state.SetLastCountDown(remainCountDown)
	}
	return nil
}
