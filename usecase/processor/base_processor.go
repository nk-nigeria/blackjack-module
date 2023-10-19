package processor

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/ciaolink-game-platform/blackjack-module/cgbdb"
	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/ciaolink-game-platform/blackjack-module/usecase/engine"
	"github.com/ciaolink-game-platform/cgp-common/lib"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type BaseProcessor struct {
	engine      engine.UseCase
	marshaler   *protojson.MarshalOptions
	unmarshaler *protojson.UnmarshalOptions
}

func NewBaseProcessor(marshaler *protojson.MarshalOptions, unmarshaler *protojson.UnmarshalOptions, engine engine.UseCase) *BaseProcessor {
	return &BaseProcessor{
		marshaler:   marshaler,
		unmarshaler: unmarshaler,
		engine:      engine,
	}
}

func (m *BaseProcessor) NotifyUpdateGameState(s *entity.MatchState,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	updateState proto.Message,
) {
	m.broadcastMessage(
		logger, dispatcher,
		int64(pb.OpCodeUpdate_OPCODE_UPDATE_GAME_STATE),
		updateState, nil, nil, true)
}

func (m *BaseProcessor) ProcessApplyPresencesLeave(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState) {
	pendingLeaves := s.GetLeavePresences()
	if len(pendingLeaves) == 0 {
		return
	}
	logger.Info("process apply presences")
	defer m.notifyUserChange(ctx, nk, logger, db, dispatcher, s)

	s.RemovePresences(pendingLeaves...)
	s.ApplyLeavePresence()
	listUserId := make([]string, 0)
	for _, p := range pendingLeaves {
		listUserId = append(listUserId, p.GetUserId())
	}
	cgbdb.UpdateUsersPlayingInMatch(ctx, logger, db, listUserId, "")
	logger.Info("notify to player kick off %s", strings.Join(listUserId, ","))
	m.broadcastMessage(
		logger, dispatcher,
		int64(pb.OpCodeUpdate_OPCODE_KICK_OFF_THE_TABLE),
		nil, pendingLeaves, nil, true)
}

func (m *BaseProcessor) ProcessPresencesJoin(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	presences []runtime.Presence,
) {
	logger.Info("process presences join %v", presences)
	// update new presence
	newJoins := make([]runtime.Presence, 0)
	for _, presence := range presences {
		// check in list leave pending
		{
			_, found := s.LeavePresences.Get(presence.GetUserId())
			if found {
				s.LeavePresences.Remove(presence.GetUserId())
			} else {
				newJoins = append(newJoins, presence)
			}
		}
	}
	s.AddPresence(ctx, nk, newJoins)
	s.JoinsInProgress -= len(newJoins)
	// update match profile user
	{
		var listUserId []string
		for _, p := range newJoins {
			listUserId = append(listUserId, p.GetUserId())
		}
		matchId, _ := ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)
		playingMatch := &pb.PlayingMatch{
			Code:    entity.ModuleName,
			MatchId: matchId,
		}
		playingMatchJson, _ := json.Marshal(playingMatch)
		cgbdb.UpdateUsersPlayingInMatch(ctx, logger, db, listUserId, string(playingMatchJson))
	}
	m.notifyUserChange(ctx, nk, logger, db, dispatcher, s)
	for _, presence := range presences {
		messages := m.engine.RejoinUserMessage(s, presence.GetUserId())
		if messages == nil {
			continue
		}
		for k, msg := range messages {
			m.broadcastMessage(logger, dispatcher, int64(k), msg, []runtime.Presence{presence}, nil, true)
		}
	}
}

func (m *BaseProcessor) ProcessPresencesLeave(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	presences []runtime.Presence,
) {
	s.RemovePresences(presences...)
	var listUserId []string
	for _, p := range presences {
		listUserId = append(listUserId, p.GetUserId())
	}
	cgbdb.UpdateUsersPlayingInMatch(ctx, logger, db, listUserId, "")
	m.notifyUserChange(ctx, nk, logger, db, dispatcher, s)
}

func (m *BaseProcessor) ProcessPresencesLeavePending(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	presences []runtime.Presence,
) {
	logger.Info("process presences leave pending %v", presences)
	for _, presence := range presences {
		_, found := s.PlayingPresences.Get(presence.GetUserId())
		if found {
			s.AddLeavePresence(presence)
		} else {
			s.RemovePresences(presence)
			cgbdb.UpdateUsersPlayingInMatch(ctx, logger, db, []string{presence.GetUserId()}, "")
			m.notifyUserChange(ctx, nk, logger, nil, dispatcher, s)
		}
	}
}

func (m *BaseProcessor) broadcastMessage(logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	opCode int64,
	data proto.Message,
	presences []runtime.Presence,
	sender runtime.Presence,
	reliable bool,
) error {
	dataJson, err := m.marshaler.Marshal(data)
	if err != nil {
		logger.Error("Error when marshaler data for broadcastMessage")
		return err
	}
	err = dispatcher.BroadcastMessage(opCode, dataJson, presences, sender, true)
	if opCode == int64(pb.OpCodeUpdate_OPCODE_UPDATE_GAME_STATE) {
		return nil
	}
	logger.Info("broadcast message opcode %v, to %v, data %v", opCode, presences, string(dataJson))
	if err != nil {
		logger.Error("Error BroadcastMessage, message: %s", string(dataJson))
		return err
	}
	return nil
}

func (m *BaseProcessor) notifyUserChange(ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState) {
	msg := &pb.UpdateTable{
		Players:        entity.NewListPlayer(s.GetPresences()),
		PlayingPlayers: entity.NewListPlayer(s.GetPlayingPresences()),
		LeavePlayers:   entity.NewListPlayer(s.GetLeavePresences()),
	}

	m.broadcastMessage(
		logger, dispatcher,
		int64(pb.OpCodeUpdate_OPCODE_USER_IN_TABLE_INFO),
		msg, nil, nil, true)
}

func (m *BaseProcessor) report(
	ctx context.Context,
	logger runtime.Logger,
	balanceResult *pb.BalanceResult,
	totalFee int64,
	s *entity.MatchState,
) {
	report := lib.NewReportGame(ctx)
	report.AddMatch(&pb.MatchData{
		GameId:   0,
		GameCode: s.Label.Code,
		Mcb:      int64(s.Label.Bet),
		ChipFee:  totalFee,
	})
	for _, b := range balanceResult.Updates {
		report.AddPlayerData(&pb.PlayerData{
			UserId:  b.UserId,
			Chip:    b.AmountChipCurrent,
			ChipAdd: b.AmountChipAdd,
		})
	}
	data, status, err := report.Commit()
	if err != nil || status > 300 {
		if err != nil {
			logger.Error("Report game (%s) operation -> url %s failed, response %s status %d err %s",
				report.ReportEndpoint(), s.Label.Code, string(data), status, err.Error())
		} else {
			logger.Info("Report game (%s) operatio -> %s successful", s.Label.Code)
		}
	}
}
