package processor

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nakamaFramework/blackjack-module/entity"
)

type IProcessor interface {
	ProcessNewGame(
		ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState)

	ProcessFinishGame(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState)

	ProcessTurnbase(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		s *entity.MatchState)

	ProcessMessageFromUser(ctx context.Context,
		logger runtime.Logger,
		nk runtime.NakamaModule,
		db *sql.DB,
		dispatcher runtime.MatchDispatcher,
		messages []runtime.MatchData,
		s *entity.MatchState)

	IBaseProcessor
}
