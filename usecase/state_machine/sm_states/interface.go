package smstates

import (
	"context"

	"github.com/qmuntal/stateless"
)

type StateMachineState interface {
	NewIdleState(FireFn) StateHandler
	NewStateMatching(FireFn) StateHandler
	NewStatePreparing(FireFn) StateHandler
	NewStatePlay(FireFn) StateHandler
	NewStateReward(FireFn) StateHandler

	OnTransitioning(ctx context.Context, t stateless.Transition)
}
