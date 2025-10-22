package state_machine

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/pkg/packager"
	lib "github.com/nk-nigeria/blackjack-module/usecase/state_machine/sm_states"
	pb "github.com/nk-nigeria/cgp-common/proto"
	"github.com/qmuntal/stateless"
)

const (
	StateInit      = pb.GameState_GAME_STATE_UNKNOWN // Only for initialize
	StateIdle      = pb.GameState_GAME_STATE_IDLE
	StateMatching  = pb.GameState_GAME_STATE_MATCHING
	StatePreparing = pb.GameState_GAME_STATE_PREPARING
	StatePlay      = pb.GameState_GAME_STATE_PLAY
	StateReward    = pb.GameState_GAME_STATE_REWARD
	StateFinish    = pb.GameState_GAME_STATE_FINISH
)

var (
	ErrStateMachineFinish = runtime.NewError("state machine finish", -1)
)

func NewGameStateMachine(stateMachineState lib.StateMachineState) UseCase {
	gs := &Machine{
		state: stateless.NewStateMachine(StateInit),
	}
	gs.configure(stateMachineState)

	return gs
}

var _ UseCase = &Machine{}

type Machine struct {
	state *stateless.StateMachine
}

func (m *Machine) GetPbState() pb.GameState {
	switch m.state.MustState() {
	case StateIdle:
		return pb.GameState_GAME_STATE_IDLE
	case StateMatching:
		return pb.GameState_GAME_STATE_MATCHING
	case StatePreparing:
		return pb.GameState_GAME_STATE_PREPARING
	case StatePlay:
		return pb.GameState_GAME_STATE_PLAY
	case StateReward:
		return pb.GameState_GAME_STATE_REWARD
	default:
		return pb.GameState_GAME_STATE_UNKNOWN
	}
}

func (m *Machine) IsPlayingState() bool {
	return m.GetPbState() == pb.GameState_GAME_STATE_PLAY
}

func (m *Machine) IsReward() bool {
	return m.GetPbState() == pb.GameState_GAME_STATE_REWARD
}

func (m *Machine) FireProcessEvent(ctx context.Context, args ...interface{}) error {
	return m.state.FireCtx(ctx, lib.TriggerProcess, args...)
}

func (m *Machine) MustState() stateless.State {
	return m.state.MustState()
}

func (m *Machine) Trigger(ctx context.Context, trigger stateless.Trigger, args ...interface{}) error {
	return m.state.FireCtx(ctx, trigger, args...)
}

func (m *Machine) TriggerIdle(ctx context.Context, args ...interface{}) error {
	return m.state.FireCtx(ctx, lib.TriggerInit, args...)
}

func (m *Machine) configure(stateMachineState lib.StateMachineState) {
	m.state.Configure(StateInit).
		Permit(lib.TriggerInit, StateIdle)
	fireCtx := m.state.FireCtx
	m.state.OnTransitioning(func(ctx context.Context, t stateless.Transition) {
		procPkg := packager.GetProcessorPackagerFromContext(ctx)
		logger := procPkg.GetLogger()
		logger.WithField("source", t.Source).
			WithField("destination", t.Destination).
			WithField("transition", t.Trigger).
			Info("OnTransitioning")
		stateMachineState.OnTransitioning(ctx, t)
		state := procPkg.GetState()
		state.SetGameState(t.Destination.(pb.GameState))
		if procPkg.GetDispatcher() != nil {
			state.UpdateLabel()
			labelJson, _ := entity.DefaultMarshaler.Marshal(state.Label)
			procPkg.GetDispatcher().MatchLabelUpdate(string(labelJson))
		}
	})

	{
		idle := stateMachineState.NewIdleState(fireCtx)
		m.state.Configure(StateIdle).
			OnEntry(idle.Enter).
			OnExit(idle.Exit).
			InternalTransition(lib.TriggerProcess, idle.Process).
			Permit(lib.TriggerStateFinishSuccess, StateMatching).
			Permit(lib.TriggerStateFinishFailed, StateFinish).
			Permit(lib.TriggerExit, StateFinish)
	}
	{
		state := defaultFinishHandler
		m.state.Configure(StateFinish).
			OnEntry(state.Enter).
			OnExit(state.Exit).
			InternalTransition(lib.TriggerProcess, state.Process)
	}
	{
		matching := stateMachineState.NewStateMatching(fireCtx)
		m.state.Configure(StateMatching).
			OnEntry(matching.Enter).
			OnExit(matching.Exit).
			InternalTransition(lib.TriggerProcess, matching.Process).
			Permit(lib.TriggerStateFinishSuccess, StatePreparing).
			Permit(lib.TriggerStateFinishFailed, StateIdle).
			Permit(lib.TriggerInit, StateIdle)
	}
	{
		preparing := stateMachineState.NewStatePreparing(fireCtx)
		m.state.Configure(StatePreparing).
			OnEntry(preparing.Enter).
			OnExit(preparing.Exit).
			InternalTransition(lib.TriggerProcess, preparing.Process).
			Permit(lib.TriggerStateFinishSuccess, StatePlay).
			Permit(lib.TriggerStateFinishFailed, StateMatching)
	}
	{
		play := stateMachineState.NewStatePlay(fireCtx)
		m.state.Configure(StatePlay).
			OnEntry(play.Enter).
			OnExit(play.Exit).
			InternalTransition(lib.TriggerProcess, play.Process).
			Permit(lib.TriggerStateFinishSuccess, StateReward).
			Permit(lib.TriggerStateFinishFailed, StateReward)
	}
	{
		reward := stateMachineState.NewStateReward(fireCtx)
		m.state.Configure(StateReward).
			OnEntry(reward.Enter).
			OnExit(reward.Exit).
			InternalTransition(lib.TriggerProcess, reward.Process).
			Permit(lib.TriggerStateFinishSuccess, StateMatching).
			Permit(lib.TriggerStateFinishFailed, StatePreparing)
	}

	m.state.ToGraph()
}

var defaultFinishHandler lib.StateHandler = &finishHandler{}

type finishHandler struct{}

func (*finishHandler) Enter(ctx context.Context, _ ...any) error {
	return nil
}

func (*finishHandler) Exit(_ context.Context, _ ...any) error {
	return ErrStateMachineFinish
}

func (*finishHandler) Process(ctx context.Context, args ...any) error {
	return ErrStateMachineFinish
}

func (*finishHandler) Trigger(ctx context.Context, trigger any, args ...any) error {
	return nil
}
