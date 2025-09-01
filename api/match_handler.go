package api

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/pkg/packager"
	"github.com/nk-nigeria/blackjack-module/usecase/engine"
	"github.com/nk-nigeria/blackjack-module/usecase/processor"
	gsm "github.com/nk-nigeria/blackjack-module/usecase/state_machine"
	smstates "github.com/nk-nigeria/blackjack-module/usecase/state_machine/sm_states"
	pb "github.com/nk-nigeria/cgp-common/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

func NewMatchHandler(marshaler *proto.MarshalOptions, unmarshaler *proto.UnmarshalOptions) *MatchHandler {
	return &MatchHandler{
		processor: processor.NewMatchProcessor(marshaler, unmarshaler, engine.NewGameEngine()),
		machine:   gsm.NewGameStateMachine(smstates.NewStateMachineState()),
	}
}

func (m *MatchHandler) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	logger.Info("match init: %v", params)
	label, ok := params["label"].(string)
	if !ok {
		logger.WithField("params", params).Error("invalid match init parameter \"label\"")
		return nil, entity.TickRate, ""
	}
	matchInfo := &pb.Match{}
	err := protojson.Unmarshal([]byte(label), matchInfo)
	if err != nil {
		logger.Error("match init json label failed ", err)
		return nil, entity.TickRate, ""
	}
	matchInfo.MatchId, _ = ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)
	labelJSON, err := protojson.Marshal(matchInfo)

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
	data, err := json.Marshal(matchState)
	if err != nil {
		logger.Error("Marshal matchState failed (likely invalid UTF-8): ", err)
	}
	logger.Info("matchState: %s", string(data))
	return &matchState, entity.TickRate, string(labelJSON)
}
