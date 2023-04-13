package smstates

import (
	"context"
	"time"

	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
)

type StateMatching struct {
	StateBase
}

func NewStateMatching(fn FireFn) StateHandler {
	return &StateMatching{
		StateBase: NewStateBase(fn),
	}
}

func (s *StateMatching) Enter(ctx context.Context, _ ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	procPkg.GetLogger().Info("[matching] enter")
	state := procPkg.GetState()
	state.SetUpCountDown(1 * time.Second)
	procPkg.GetLogger().Info("apply leave presence")
	procPkg.GetProcessor().ProcessApplyPresencesLeave(
		procPkg.GetContext(),
		procPkg.GetLogger(),
		procPkg.GetNK(),
		procPkg.GetDb(),
		procPkg.GetDispatcher(),
		state)
	return nil
}

func (s *StateMatching) Exit(_ context.Context, _ ...interface{}) error {
	return nil
}

func (s *StateMatching) Process(ctx context.Context, args ...interface{}) error {
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
	remain := state.GetRemainCountDown()
	if remain > 0 {
		return nil
	}
	if state.IsReadyToPlay() {
		s.Trigger(ctx, TriggerStateFinishSuccess)
	} else {
		s.Trigger(ctx, TriggerStateFinishFailed)
	}
	return nil
}
