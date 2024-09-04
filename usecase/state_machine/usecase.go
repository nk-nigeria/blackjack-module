package state_machine

import (
	"context"

	pb "github.com/nakamaFramework/cgp-common/proto"
	"github.com/qmuntal/stateless"
)

type UseCase interface {
	FireProcessEvent(ctx context.Context, args ...interface{}) error
	MustState() stateless.State
	Trigger(ctx context.Context, trigger stateless.Trigger, args ...interface{}) error
	TriggerIdle(ctx context.Context, args ...interface{}) error

	//
	GetPbState() pb.GameState
	IsPlayingState() bool
	IsReward() bool
}
