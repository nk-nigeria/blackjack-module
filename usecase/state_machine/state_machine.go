package state_machine

import (
	"context"

	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
	lib "github.com/ciaolink-game-platform/blackjack-module/usecase/state_machine/sm_states"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/qmuntal/stateless"
)

const (
	StateInit      = pb.GameState_GameStateUnknown // Only for initialize
	StateIdle      = pb.GameState_GameStateIdle
	StateMatching  = pb.GameState_GameStateMatching
	StatePreparing = pb.GameState_GameStatePreparing
	StatePlay      = pb.GameState_GameStatePlay
	StateReward    = pb.GameState_GameStateReward
	StateFinish    = pb.GameState_GameStateFinish
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
		return pb.GameState_GameStateIdle
	case StateMatching:
		return pb.GameState_GameStateMatching
	case StatePreparing:
		return pb.GameState_GameStatePreparing
	case StatePlay:
		return pb.GameState_GameStatePlay
	case StateReward:
		return pb.GameState_GameStateReward
	default:
		return pb.GameState_GameStateUnknown
	}
}

func (m *Machine) IsPlayingState() bool {
	return m.GetPbState() == pb.GameState_GameStatePlay
}

func (m *Machine) IsReward() bool {
	return m.GetPbState() == pb.GameState_GameStateReward
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
