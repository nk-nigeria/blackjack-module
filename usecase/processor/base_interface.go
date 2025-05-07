package processor

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nakamaFramework/blackjack-module/entity"
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
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState,
		presences []runtime.Presence,
	)

	ProcessMatchTerminate(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState,
	)

	ProcessMatchKick(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState,
	)
}
