package smstates

import (
	"context"
	"time"

	"github.com/qmuntal/stateless"
)

const (
	TriggerInit               = "TriggerInit"
	TriggerStateFinishSuccess = "TriggerStateFinishSuccess"
	TriggerStateFinishFailed  = "TriggerStateFinishFailed"
	TriggerExit               = "TriggerExit"

	// internal transition
	TriggerProcess = "TriggerProcess"
)

const (
	idleTimeout      = time.Second * 15
	preparingTimeout = time.Second * 5
	playTimeout      = time.Second * 15
	//playTimeout      = time.Second * 10
	rewardTimeout = time.Second * 10
	//rewardTimeout    = time.Second * 10
)

type StateHandler interface {
	Trigger(ctx context.Context, trigger stateless.Trigger, args ...interface{}) error
	Process(ctx context.Context, args ...interface{}) error

	Enter(ctx context.Context, _ ...interface{}) error
	Exit(_ context.Context, _ ...interface{}) error
}

type FireFn func(ctx context.Context, trigger stateless.Trigger, args ...interface{}) error

type StateBase struct {
	fireFn FireFn
}

func NewStateBase(fn FireFn) StateBase {
	return StateBase{
		fireFn: fn,
	}
}

func (s *StateBase) Trigger(ctx context.Context, trigger stateless.Trigger, args ...interface{}) error {
	return s.fireFn(ctx, trigger, args...)
}
