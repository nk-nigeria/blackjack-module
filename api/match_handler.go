package api

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/ciaolink-game-platform/blackjack-module/pkg/packager"
	"github.com/ciaolink-game-platform/blackjack-module/usecase/engine"
	"github.com/ciaolink-game-platform/blackjack-module/usecase/processor"
	gsm "github.com/ciaolink-game-platform/blackjack-module/usecase/state_machine"
	smstates "github.com/ciaolink-game-platform/blackjack-module/usecase/state_machine/sm_states"
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
	bet, ok := params["bet"].(int32)
	if !ok {
		logger.Error("invalid match init parameter \"bet\"")
		return nil, 0, ""
	}

	name, ok := params["name"].(string)
	if !ok {
		logger.Warn("invalid match init parameter \"name\"")
		//return nil, 0, ""
	}

	password, ok := params["password"].(string)
	if !ok {
		logger.Warn("invalid match init parameter \"password\"")
		//return nil, 0, ""
	}

	open := int32(1)
	if password != "" {
		open = 0
	}

	mockCodeCard, _ := params["mock_code_card"].(int32)

	label := &entity.MatchLabel{
		Open:         open,
		Bet:          bet,
		Code:         entity.ModuleName,
		Name:         name,
		Password:     password,
		MaxSize:      entity.MaxPresences,
		MockCodeCard: mockCodeCard,
	}

	labelJSON, err := json.Marshal(label)
	if err != nil {
		logger.Error("match init json label failed ", err)
		return nil, tickRate, ""
	}

	logger.Info("match init label= %s", string(labelJSON))

	matchState := entity.NewMatchState(label)
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

	return &matchState, tickRate, string(labelJSON)
}
