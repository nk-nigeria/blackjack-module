package smstates

import (
	"context"
	"math"

	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"
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
	state.SetUpCountDown(rewardTimeout)
	procPkg.GetProcessor().NotifyUpdateGameState(
		state,
		procPkg.GetLogger(),
		procPkg.GetDispatcher(),
		&pb.UpdateGameState{
			State:     pb.GameState_GameStateReward,
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
