package smstates

import (
	"context"
	"math"
	"strings"

	"github.com/nakamaFramework/blackjack-module/entity"
	"github.com/nakamaFramework/blackjack-module/pkg/packager"
	pb "github.com/nakamaFramework/cgp-common/proto"
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
	procPkg.GetProcessor().ProcessApplyPresencesLeave(ctx,
		procPkg.GetLogger(),
		procPkg.GetNK(),
		procPkg.GetDb(),
		procPkg.GetDispatcher(),
		state,
	)
	procPkg.GetProcessor().NotifyUpdateGameState(
		state,
		procPkg.GetLogger(),
		procPkg.GetDispatcher(),
		&pb.UpdateGameState{
			State:     pb.GameState_GameStatePreparing,
			CountDown: int64(math.Round(float64(state.GetRemainCountDown()))),
		},
	)
	for _, precense := range state.GetPresences() {
		state.PresencesNoInteract[precense.GetUserId()]++
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
	message := procPkg.GetMessages()
	if len(message) > 0 {
		procPkg.GetProcessor().ProcessMessageFromUser(ctx,
			procPkg.GetLogger(),
			procPkg.GetNK(),
			procPkg.GetDb(),
			procPkg.GetDispatcher(),
			message, procPkg.GetState())
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
					State:     pb.GameState_GameStatePreparing,
					CountDown: int64(remainCountDown),
				},
			)
			state.SetLastCountDown(remainCountDown)
		}
	}
	return nil
}
