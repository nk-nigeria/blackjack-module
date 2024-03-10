package api

import (
	"context"
	"database/sql"

	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
	"github.com/ciaolink-game-platform/blackjack-module/usecase/state_machine"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (m *MatchHandler) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presence runtime.Presence, metadata map[string]string) (interface{}, bool, string) {
	s := state.(*entity.MatchState)
	logger.Info("match join attempt, state=%v, meta=%v", s, metadata)

	// check password
	if s.Label.Open == 0 {
		logger.Info("match protect with password, check password")
		joinPassword := metadata["password"]
		if joinPassword != s.Label.Password {
			return s, false, "wrong password"
		}
	}
	// Check if it's a user attempting to rejoin after a disconnect.
	if p, _ := s.Presences.Get(presence.GetUserId()); p != nil {
		// 	// User rejoining after a disconnect.
		logger.Info("user %s rejoin after disconnect", presence.GetUserId())
		s.RemoveLeavePresence(presence.GetUserId())

		s.JoinsInProgress++
		return s, true, ""

	}

	// join as new user

	// Check if match is full.
	if s.Presences.Size()+s.JoinsInProgress >= entity.MaxPresences {
		return s, false, "match full"
	}
	// check chip balance in wallet before allow join
	wallet, err := entity.ReadWalletUser(ctx, nk, logger, presence.GetUserId())
	if err != nil {
		return s, false, status.Error(codes.Internal, "read chip balance failed").Error()
	}
	if wallet.Chips < int64(s.Label.Bet) {
		logger.Warn("[Reject] reject allow user %s join game, not enough chip join game, balance user chip %d , game bet %d",
			presence.GetUserId(), wallet.Chips, s.Label.Bet)
		return s, false, status.Error(codes.Internal, "chip balance not enough").Error()
	}

	// New player attempting to connect.
	s.JoinsInProgress++
	return s, true, ""
}

func (m *MatchHandler) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	s := state.(*entity.MatchState)
	logger.Info("match join, state=%v, presences=%v", s, presences)

	m.processor.ProcessPresencesJoin(ctx, logger, nk, db, dispatcher, s, presences)
	return s
}

func (m *MatchHandler) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	s := state.(*entity.MatchState)

	logger.Info("match leave, state=%v, presences=%v", s, presences)

	if m.machine.IsPlayingState() || m.machine.IsReward() {
		m.processor.ProcessPresencesLeavePending(ctx, logger, nk, db, dispatcher, s, presences)
		return s
	}
	m.processor.ProcessPresencesLeave(ctx, logger, nk, db, dispatcher, s, presences)
	return s
}

func (m *MatchHandler) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
	s := state.(*entity.MatchState)

	err := m.machine.FireProcessEvent(packager.GetContextWithProcessorPackager(
		packager.NewProcessorPackage(
			s, m.processor,
			logger,
			nk,
			db,
			dispatcher,
			messages,
			ctx),
	))
	if err == state_machine.ErrStateMachineFinish {
		logger.Info("match need finish")

		return nil
	}

	return s
}

func (m *MatchHandler) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	logger.Info("match terminate, state=%v")
	s := state.(*entity.MatchState)
	m.processor.ProcessMatchTerminate(ctx, logger, nk, db, dispatcher, s)
	return state
}
