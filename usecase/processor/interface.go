package processor

import (
	"context"
	"database/sql"

	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/heroiclabs/nakama-common/runtime"
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
