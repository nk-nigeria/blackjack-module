package api

import (
	"context"
	"database/sql"

	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
	"github.com/ciaolink-game-platform/blackjack-module/usecase/engine"
	"github.com/ciaolink-game-platform/blackjack-module/usecase/processor"
	gsm "github.com/ciaolink-game-platform/blackjack-module/usecase/state_machine"
	smstates "github.com/ciaolink-game-platform/blackjack-module/usecase/state_machine/sm_states"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ runtime.Match = &MatchHandler{}

const (
	tickRate = 2
)

type MatchHandler struct {
	processor processor.IProcessor
	machine   gsm.UseCase
}

func (m *MatchHandler) MatchSignal(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, data string) (interface{}, string) {
	//panic("implement me")
	s := state.(*entity.MatchState)
	return s, ""
}

func NewMatchHandler(marshaler *protojson.MarshalOptions, unmarshaler *protojson.UnmarshalOptions) *MatchHandler {
	return &MatchHandler{
		processor: processor.NewMatchProcessor(marshaler, unmarshaler, engine.NewGameEngine()),
		machine:   gsm.NewGameStateMachine(smstates.NewStateMachineState()),
	}
}

func (m *MatchHandler) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	logger.Info("match init: %v", params)
	label, ok := params["data"].(string)
	if !ok {
		logger.WithField("params", params).Error("invalid match init parameter \"data\"")
		return nil, entity.TickRate, ""
	}
	matchInfo := &pb.Match{}
	err := entity.DefaulUnmarshaler.Unmarshal([]byte(label), matchInfo)
	if err != nil {
		logger.Error("match init json label failed ", err)
		return nil, entity.TickRate, ""
	}
	matchInfo.MatchId, _ = ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)
	labelJSON, err := entity.DefaultMarshaler.Marshal(matchInfo)

	if err != nil {
		logger.Error("match init json label failed ", err)
		return nil, entity.TickRate, ""
	}

	logger.Info("match init label= %s", string(labelJSON))

	matchState := entity.NewMatchState(matchInfo)
	// init jp treasure
	// jpTreasure, _ := cgbdb.GetJackpot(ctx, logger, db, entity.ModuleName)
	// if jpTreasure != nil {
	// 	matchState.SetJackpotTreasure(&pb.Jackpot{
	// 		GameCode: jpTreasure.GetGameCode(),
	// 		Chips:    jpTreasure.Chips,
	// 	})
	// }
	// fire idle event
	procPkg := packager.NewProcessorPackage(&matchState, m.processor, logger, nil, nil, nil, nil, nil)
	m.machine.TriggerIdle(packager.GetContextWithProcessorPackager(procPkg))

	return &matchState, entity.TickRate, string(labelJSON)
}
