package smstates

import (
	"context"

	"github.com/qmuntal/stateless"
)

type stateMachine struct{}

func NewStateMachineState() StateMachineState {
	s := stateMachine{}
	return &s
}

func (sm *stateMachine) NewIdleState(fn FireFn) StateHandler {
	return NewIdleState(fn)
}

func (sm *stateMachine) NewStateMatching(fn FireFn) StateHandler {
	return NewStateMatching(fn)
}

func (sm *stateMachine) NewStatePlay(fn FireFn) StateHandler {
	return NewStatePlay(fn)
}

func (sm *stateMachine) NewStatePreparing(fn FireFn) StateHandler {
	return NewStatePreparing(fn)
}

func (sm *stateMachine) NewStateReward(fn FireFn) StateHandler {
	return NewStateReward(fn)
}

func (sm *stateMachine) OnTransitioning(ctx context.Context, t stateless.Transition) {}
