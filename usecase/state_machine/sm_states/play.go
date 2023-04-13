package smstates

import (
	"context"
	"math"

	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"
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
	state.SetUpCountDown(playTimeout)
	procPkg.GetProcessor().NotifyUpdateGameState(
		state,
		procPkg.GetLogger(),
		procPkg.GetDispatcher(),
		&pb.UpdateGameState{
			State:     pb.GameState_GameStatePlay,
			CountDown: int64(math.Round(state.GetRemainCountDown())),
		},
	)
	state.SetupMatchPresence()
	procPkg.GetProcessor().ProcessNewGame(
		procPkg.GetContext(),
		procPkg.GetNK(),
		procPkg.GetLogger(),
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

	message := procPkg.GetMessages()
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
		procPkg.GetProcessor().NotifyUpdateGameState(
			state,
			procPkg.GetLogger(),
			procPkg.GetDispatcher(),
			&pb.UpdateGameState{
				State:     pb.GameState_GameStatePlay,
				CountDown: int64(remainCountDown),
			},
		)
		state.SetLastCountDown(remainCountDown)
	}
	return nil
}
