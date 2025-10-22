package smstates

import (
	"context"
	"math"

	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/pkg/global"
	"github.com/nk-nigeria/blackjack-module/pkg/packager"
	"github.com/nk-nigeria/blackjack-module/usecase/service"
	pb "github.com/nk-nigeria/cgp-common/proto"
)

type StateReward struct {
	StateBase
}

func NewStateReward(fn FireFn) StateHandler {
	return &StateReward{
		StateBase: NewStateBase(fn),
	}
}
func (s *StateReward) Enter(ctx context.Context, _ ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	procPkg.GetLogger().Info("[reward] enter")
	// setup reward timeout
	state := procPkg.GetState()
	state.SetUpCountDown(entity.GameStateDuration[state.GetGameState()])

	// Initialize bot integration if not already done
	botIntegration := global.GetGlobalBotIntegration()
	if botIntegration == nil {
		botIntegration = service.NewBlackjackBotIntegration(procPkg.GetDb())
		global.SetGlobalBotIntegration(botIntegration)
	}

	// Type assert to get the actual bot integration
	if blackjackBotIntegration, ok := botIntegration.(*service.BlackjackBotIntegration); ok {
		// Set match state for bot decision making
		blackjackBotIntegration.SetMatchState(
			state.GetMatchID(),
			state.GetBetAmount(),
			state.GetPresenceSize(),
			state.GetLastResult(), // Use actual game result
			1,                     // activeTables
		)

		// Process bot leave logic during reward phase
		if err := blackjackBotIntegration.ProcessBotLeaveLogic(ctx); err != nil {
			procPkg.GetLogger().Error("Failed to process bot leave logic: %v", err)
		}
	}

	procPkg.GetProcessor().NotifyUpdateGameState(
		state,
		procPkg.GetLogger(),
		procPkg.GetDispatcher(),
		&pb.UpdateGameState{
			State:     pb.GameState_GAME_STATE_REWARD,
			CountDown: int64(math.Round(state.GetRemainCountDown())),
		},
	)
	// process finish
	procPkg.GetProcessor().ProcessFinishGame(
		procPkg.GetContext(),
		procPkg.GetLogger(),
		procPkg.GetNK(),
		procPkg.GetDb(),
		procPkg.GetDispatcher(),
		state)

	return nil
}

func (s *StateReward) Exit(ctx context.Context, _ ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	state.ResetBalanceResult()
	procPkg.GetProcessor().ProcessMatchKick(procPkg.GetContext(), procPkg.GetLogger(), procPkg.GetNK(), procPkg.GetDb(), procPkg.GetDispatcher(), state)
	return nil
}

func (s *StateReward) Process(ctx context.Context, args ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	message := procPkg.GetMessages()
	if len(message) > 0 {
		procPkg.GetProcessor().ProcessMessageFromUser(ctx,
			procPkg.GetLogger(),
			procPkg.GetNK(),
			procPkg.GetDb(),
			procPkg.GetDispatcher(),
			message, procPkg.GetState())
	}
	if remain := state.GetRemainCountDown(); remain <= 0 {
		s.Trigger(ctx, TriggerStateFinishSuccess)
	}
	return nil
}
