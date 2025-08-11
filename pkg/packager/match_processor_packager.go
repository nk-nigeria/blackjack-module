package packager

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nk-nigeria/blackjack-module/entity"
	processor_interface "github.com/nk-nigeria/blackjack-module/usecase/processor"
)

const processorKey = "pd"

type ProcessorPackager struct {
	state      *entity.MatchState
	processor  processor_interface.IProcessor
	logger     runtime.Logger
	nk         runtime.NakamaModule
	dispatcher runtime.MatchDispatcher
	messages   []runtime.MatchData
	ctx        context.Context
	db         *sql.DB
}

func NewProcessorPackage(state *entity.MatchState, processor processor_interface.IProcessor, logger runtime.Logger, nk runtime.NakamaModule, db *sql.DB, dispatcher runtime.MatchDispatcher, messages []runtime.MatchData, ctx context.Context) *ProcessorPackager {
	return &ProcessorPackager{
		state:      state,
		processor:  processor,
		logger:     logger,
		nk:         nk,
		dispatcher: dispatcher,
		messages:   messages,
		ctx:        ctx,
		db:         db,
	}
}

func (p ProcessorPackager) GetState() *entity.MatchState {
	return p.state
}

func (p ProcessorPackager) GetProcessor() processor_interface.IProcessor {
	return p.processor
}

func (p ProcessorPackager) GetLogger() runtime.Logger {
	return p.logger
}

func (p ProcessorPackager) GetNK() runtime.NakamaModule {
	return p.nk
}

func (p ProcessorPackager) GetDb() *sql.DB {
	return p.db
}

func (p ProcessorPackager) GetDispatcher() runtime.MatchDispatcher {
	return p.dispatcher
}

func (p ProcessorPackager) GetMessages() []runtime.MatchData {
	return p.messages
}

func (p ProcessorPackager) GetContext() context.Context {
	return p.ctx
}

func GetProcessorPackagerFromContext(ctx context.Context) *ProcessorPackager {
	return ctx.Value(processorKey).(*ProcessorPackager)
}

func GetContextWithProcessorPackager(procPkg *ProcessorPackager) context.Context {
	return context.WithValue(context.TODO(), processorKey, procPkg)
}
