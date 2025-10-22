package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nk-nigeria/blackjack-module/api"
	"github.com/nk-nigeria/blackjack-module/entity"
	"github.com/nk-nigeria/blackjack-module/pkg/global"
	"github.com/nk-nigeria/blackjack-module/usecase/service"
	"github.com/nk-nigeria/cgp-common/bot"
	"github.com/nk-nigeria/cgp-common/define"
	"google.golang.org/protobuf/proto"
)

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	initStart := time.Now()

	marshaler := &proto.MarshalOptions{}
	unmarshaler := &proto.UnmarshalOptions{
		DiscardUnknown: false,
	}
	if err := initializer.RegisterMatch(entity.ModuleName, func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
		return api.NewMatchHandler(marshaler, unmarshaler), nil
	}); err != nil {
		return err
	}

	// Initialize BotLoader for blackjack
	entity.BotLoader = bot.NewBotLoader(db, define.BlackjackName.String(), 100000)

	// Initialize bot integration service and set it globally
	botIntegration := service.NewBlackjackBotIntegration(db)
	// Set the global bot integration in state machine package
	global.SetGlobalBotIntegration(botIntegration)
	// This will be done when the first match starts

	logger.Info("Plugin loaded in '%d' msec.", time.Since(initStart).Milliseconds())
	return nil

}
