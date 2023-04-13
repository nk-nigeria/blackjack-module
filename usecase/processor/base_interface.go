package processor

import (
	"context"
	"database/sql"

	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/proto"
)

type IBaseProcessor interface {
	NotifyUpdateGameState(s *entity.MatchState,
		logger runtime.Logger,
		dispatcher runtime.MatchDispatcher,
		updateState proto.Message)

	ProcessApplyPresencesLeave(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState)

	ProcessPresencesJoin(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState,
		presences []runtime.Presence,
	)

	ProcessPresencesLeave(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState,
		presences []runtime.Presence,
	)

	ProcessPresencesLeavePending(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState,
		presences []runtime.Presence,
	)
}
