package smstates

import (
	"context"
	"strings"
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
	listPrecense := state.GetPresenceNotInteract(1)
	if len(listPrecense) > 0 {
		listUserId := make([]string, len(listPrecense))
		for _, p := range listPrecense {
			listUserId = append(listUserId, p.GetUserId())
		}
		procPkg.GetLogger().Info("Kick %d user from math %s",
			len(listPrecense), strings.Join(listUserId, ","))
		state.AddLeavePresence(listPrecense...)
	}
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
	if state.IsEnoughPlayer() {
		s.Trigger(ctx, TriggerStateFinishSuccess)
	} else {
		s.Trigger(ctx, TriggerStateFinishFailed)
	}
	return nil
}
