package smstates

import (
	"context"

	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
)

type StateIdle struct {
	StateBase
}

func NewIdleState(fn FireFn) StateHandler {
	return &StateIdle{
		StateBase: NewStateBase(fn),
	}
}

func (s *StateIdle) Enter(ctx context.Context, _ ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	state.SetUpCountDown(idleTimeout)
	dispatcher := procPkg.GetDispatcher()
	if dispatcher == nil {
		procPkg.GetLogger().Warn("missing dispatcher don't broadcast")
		return nil
	}
	return nil
}

func (s *StateIdle) Exit(_ context.Context, _ ...interface{}) error {
	return nil
}

func (s *StateIdle) Process(ctx context.Context, args ...interface{}) error {
	procPkg := packager.GetProcessorPackagerFromContext(ctx)
	state := procPkg.GetState()
	if state.GetPresenceSize() > 0 {
		s.Trigger(ctx, TriggerStateFinishSuccess)
		return nil
	}
	if remain := state.GetRemainCountDown(); remain < 0 {
		// Do finish here
		procPkg.GetLogger().Info("[idle] idle timeout => exit")
		s.Trigger(ctx, TriggerExit)
		return entity.ErrGameFinish
	}
	return nil
}
