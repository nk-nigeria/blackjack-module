package processor

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nk-nigeria/blackjack-module/cgbdb"
	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/usecase/engine"
	"github.com/nk-nigeria/cgp-common/define"
	"github.com/nk-nigeria/cgp-common/lib"
	pb "github.com/nk-nigeria/cgp-common/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BaseProcessor struct {
	engine      engine.UseCase
	marshaler   *proto.MarshalOptions
	unmarshaler *proto.UnmarshalOptions
}

func NewBaseProcessor(marshaler *proto.MarshalOptions, unmarshaler *proto.UnmarshalOptions, engine engine.UseCase) *BaseProcessor {
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
	defer s.UpdateLabel()
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
	s.AddPresence(ctx, db, newJoins)
	s.JoinsInProgress -= len(newJoins)
	// update match profile user
	userIDs := make([]string, 0)
	for _, presence := range presences {
		userIDs = append(userIDs, presence.GetUserId())
	}
	m.emitNkEvent(ctx, define.NakEventMatchJoin, nk, s, userIDs...)
	// for _, presence := range presences {
	// 	messages := m.engine.RejoinUserMessage(s, presence.GetUserId())
	// 	if messages == nil {
	// 		continue
	// 	}
	// 	for k, msg := range messages {
	// 		m.broadcastMessage(logger, dispatcher, int64(k), msg, []runtime.Presence{presence}, nil, true)
	// 	}
	// }
	m.notifyUserChange(ctx, nk, logger, db, dispatcher, s)
}

func (m *BaseProcessor) ProcessPresencesLeave(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	presences []runtime.Presence,
) {
	defer s.UpdateLabel()
	s.RemovePresences(presences...)
	var listUserId []string
	for _, p := range presences {
		listUserId = append(listUserId, p.GetUserId())
	}
	m.emitNkEvent(ctx, define.NakEventMatchLeave, nk, s, listUserId...)
	// cgbdb.UpdateUsersPlayingInMatch(ctx, logger, db, listUserId, "")
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
	defer s.UpdateLabel()
	logger.Info("process presences leave pending %v", presences)
	userIdsLeave := make([]string, 0)
	for _, presence := range presences {
		_, found := s.PlayingPresences.Get(presence.GetUserId())
		if found {
			s.AddLeavePresence(presence)
		} else {
			s.RemovePresences(presence)
			// cgbdb.UpdateUsersPlayingInMatch(ctx, logger, db, []string{presence.GetUserId()}, "")
			// m.emitNkEvent(ctx, define.NakEventMatchLeave, nk, presence.GetUserId(), s)
			m.notifyUserChange(ctx, nk, logger, nil, dispatcher, s)
			userIdsLeave = append(userIdsLeave, presence.GetUserId())
		}
	}
	m.emitNkEvent(ctx, define.NakEventMatchLeave, nk, s, userIdsLeave...)
}

func (m *BaseProcessor) ProcessMatchTerminate(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	userIds := make([]string, 0)
	for _, presence := range s.GetPresences() {
		userIds = append(userIds, presence.GetUserId())
	}
	m.emitNkEvent(ctx, define.NakEventMatchEnd, nk, s, userIds...)
}

func (m *BaseProcessor) emitNkEvent(ctx context.Context, eventNk define.NakEvent, nk runtime.NakamaModule, s *entity.MatchState, userId ...string) {
	matchId, _ := ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)
	nk.Event(ctx, &api.Event{
		Name:      string(eventNk),
		Timestamp: timestamppb.Now(),
		Properties: map[string]string{
			"user_id":        strings.Join(userId, ","),
			"game_code":      s.Label.Name,
			"end_match_unix": strconv.FormatInt(time.Now().Unix(), 10),
			"match_id":       matchId,
			"mcb":            strconv.FormatInt(int64(s.Label.MarkUnit), 10),
		},
	})
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
	walletByUser := make(map[string]lib.Wallet, 0)
	{
		userIds := make([]string, 0)
		for _, precense := range s.GetPresences() {
			userIds = append(userIds, precense.GetUserId())
		}
		wallets, _ := lib.ReadWalletUsers(ctx, nk, logger, userIds...)
		for _, wallet := range wallets {
			v := wallet
			walletByUser[wallet.UserId] = v
		}
	}
	msg := &pb.UpdateTable{
		Players:        entity.NewListPlayer(s.GetPresences()),
		PlayingPlayers: entity.NewListPlayer(s.GetPlayingPresences()),
		LeavePlayers:   entity.NewListPlayer(s.GetLeavePresences()),
	}
	for _, player := range msg.Players {
		w, exist := walletByUser[player.GetId()]
		if !exist {
			continue
		}
		player.Wallet = strconv.FormatInt(w.Chips, 10)
	}
	for _, player := range msg.PlayingPlayers {
		w, exist := walletByUser[player.GetId()]
		if !exist {
			continue
		}
		player.Wallet = strconv.FormatInt(w.Chips, 10)
	}
	for _, player := range msg.LeavePlayers {
		w, exist := walletByUser[player.GetId()]
		if !exist {
			continue
		}
		player.Wallet = strconv.FormatInt(w.Chips, 10)
	}

	m.broadcastMessage(
		logger, dispatcher,
		int64(pb.OpCodeUpdate_OPCODE_USER_IN_TABLE_INFO),
		msg, nil, nil, true)
}

func (m *BaseProcessor) report(
	ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	balanceResult *pb.BalanceResult,
	totalFee int64,
	s *entity.MatchState,
) {
	report := lib.NewReportGame(ctx)
	report.AddMatch(&pb.MatchData{
		GameId:   0,
		GameCode: s.Label.Name,
		Mcb:      int64(s.Label.MarkUnit),
		ChipFee:  totalFee,
	})
	for _, b := range balanceResult.Updates {
		report.AddPlayerData(&pb.PlayerData{
			UserId:  b.UserId,
			Chip:    b.AmountChipCurrent,
			ChipAdd: b.AmountChipAdd,
		})
	}
	data, status, err := report.Commit(ctx, nk)
	if err != nil || status > 300 {
		if err != nil {
			logger.Error("Report game (%s) operation -> url %s failed, response %s status %d err %s",
				report.ReportEndpoint(), s.Label.Name, string(data), status, err.Error())
		} else {
			logger.Info("Report game (%s) operatio -> %s successful", s.Label.Name)
		}
	}
}

// AddBotToMatch adds bots to the current match
func (p *Processor) AddBotToMatch(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	numBots int) error {

	logger.Info("Adding %d bots to match", numBots)

	bJoin := s.AddBotToMatch(numBots)

	if len(bJoin) > 0 {
		listUserId := make([]string, 0, len(bJoin))
		for _, p := range bJoin {
			listUserId = append(listUserId, p.GetUserId())
		}
		p.emitNkEvent(ctx, define.NakEventMatchJoin, nk, s, listUserId...)
		s.AddPlayingPresences(bJoin...)
		p.notifyUserChange(ctx, nk, logger, db, dispatcher, s)
		matchJson, err := protojson.Marshal(s.Label)
		if err != nil {
			logger.Error("update json label failed ", err)
			return nil
		}
		dispatcher.MatchLabelUpdate(string(matchJson))
		return nil
	}
	return fmt.Errorf("no bot join")

}

// RemoveBotFromMatch removes a bot from the current match
func (p *Processor) RemoveBotFromMatch(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	botUserID string) error {

	logger.Info("RemoveBotFromMatch %s", botUserID)
	if botUserID == "" {
		return nil
	}

	err, botPresence := s.RemoveBotFromMatch(botUserID)
	if err != nil {
		return err
	}

	// Emit leave event
	p.emitNkEvent(ctx, define.NakEventMatchLeave, nk, s, botUserID)

	s.AddLeavePresence(botPresence)

	// Update match label
	matchJson, err := protojson.Marshal(s.Label)
	if err != nil {
		logger.Error("update json label failed ", err)
		return nil
	}
	dispatcher.MatchLabelUpdate(string(matchJson))

	logger.Info("Bot %s removed from match", botUserID)
	return nil
}
