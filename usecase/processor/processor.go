package processor

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/ciaolink-game-platform/cgp-common/bot"
	"github.com/ciaolink-game-platform/cgp-common/lib"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"

	"github.com/ciaolink-game-platform/blackjack-module/entity"
	"github.com/ciaolink-game-platform/blackjack-module/usecase/engine"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

type Processor struct {
	*BaseProcessor
	turnBaseEngine *TurnBaseEngine
	emitBot        bool
}

func NewMatchProcessor(
	marshaler *protojson.MarshalOptions,
	unmarshaler *protojson.UnmarshalOptions,
	engine engine.UseCase,

) IProcessor {
	return &Processor{
		NewBaseProcessor(marshaler, unmarshaler, engine),
		NewTurnBaseEngine(),
		false,
	}
}

func (p *Processor) ProcessNewGame(
	ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	if !p.emitBot {
		p.emitBot = true
		precenses := make([]runtime.Presence, 0)
		for _, presence := range s.GetPresences() {
			if bot.IsBot(presence.GetUserId()) {
				precenses = append(precenses, presence)
			}
		}
		if len(precenses) > 0 {
			p.ProcessPresencesJoin(ctx, logger, nk, db, dispatcher, s, precenses)
		}
	}
	p.engine.NewGame(s)
	listPlayerId := make([]string, 0)
	// deal
	for _, presence := range s.GetPlayingPresences() {
		s.ResetUserNotInteract(presence.GetUserId())
		listPlayerId = append(listPlayerId, presence.GetUserId())
		s.AddCards(p.engine.Deal(2), presence.GetUserId(), pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	}
	// for {
	cards := p.engine.Deal(2)
	// 	hasRankA := false
	// 	if len(cards) > 1 && cards[0].Rank == pb.CardRank_RANK_A {
	// 		hasRankA = true
	// 	}
	// 	if !hasRankA {
	// 		continue
	// 	}
	s.AddCards(cards, "", pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	// 	break
	// }
	p.notifyInitialDealCard(
		ctx, nk, logger, dispatcher, s,
	)
	if p.turnBaseEngine == nil {
		p.turnBaseEngine = NewTurnBaseEngine()
	}
	p.turnBaseEngine.Config(
		listPlayerId,
		[]*Round{
			{
				code:   "insurance",
				isGlob: true,
				phases: []*Phase{
					{
						code:     "main",
						duration: time.Second * 5,
					},
				},
			},
			{
				code:   "playing",
				isGlob: false,
				phases: []*Phase{
					{
						code:     "main",
						duration: time.Second * 10,
					},
				},
			},
		},
	)

	p.turnBaseEngine.SetCurrentRound("bet")
	p.turnBaseEngine.SetCurrentPlayer(listPlayerId[0])
}

func (p *Processor) ProcessFinishGame(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {

	p.revealDealerHiddenCard(ctx, nk, logger, dispatcher, s)
	for s.IsDealerMustDraw() {
		cards := p.engine.Deal(1)
		s.AddCards(cards, "", pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
		p.notifyDealCard(ctx, nk, logger, dispatcher, s, "", pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	}
	s.SetUpdateFinish(s.CalcGameFinish())

	updateFinish := s.GetUpdateFinish()
	balanceResult, totalFee := p.calcRewardForUserPlaying(
		ctx, nk, logger, db, dispatcher, s, updateFinish,
	)
	s.SetBalanceResult(balanceResult)
	p.updateChipByResultGameFinish(ctx, nk, logger, balanceResult)
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_FINISH),
		updateFinish, nil, nil, true,
	)
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_WALLET),
		balanceResult, nil, nil, true,
	)
	p.report(ctx, logger, nk, balanceResult, totalFee, s)
}

func (p *Processor) ProcessTurnbase(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	var turnInfo *TurnInfo
	if p.turnBaseEngine != nil {
		turnInfo = p.turnBaseEngine.Loop()
	}
	if turnInfo.isNewRound {
		s.SetCurrentTurn(turnInfo.userId)
		switch turnInfo.roundCode {
		case "insurance":
			s.SetAllowBet(false)
			s.SetAllowAction(false)
			if s.DealerPotentialBlackjack() && !s.IsAllowInsurance() {
				s.SetAllowInsurance(true)
				s.SetUpCountDown(time.Duration(turnInfo.countDown) * time.Second)
				p.broadcastMessage(
					logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
					&pb.BlackjackUpdateDesk{
						IsInsuranceTurnEnter: true,
					}, nil, nil, true,
				)
			} else {
				p.turnBaseEngine.NextRound()
				return
			}
		case "playing":
			if s.DealerPotentialBlackjack() {
				if s.GetDealerHand().First.Type == pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BLACKJACK {
					s.SetIsGameEnded(true)
					return
				} else {
					p.broadcastMessage(
						logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
						&pb.BlackjackUpdateDesk{
							IsBankerNotBlackjack: true,
						}, nil, nil, true,
					)
					for _, presence := range s.GetPlayingPresences() {
						bet := s.GetUserBetById(presence.GetUserId())
						if bet.Insurance > 0 {
							bet.Insurance = 0
							s.SetUserBetById(presence.GetUserId(), bet)
							p.broadcastMessage(
								logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
								&pb.BlackjackUpdateDesk{
									IsUpdateBet: true,
									Bet:         bet,
								}, nil, nil, true,
							)
						}
					}
				}
			}
			s.InitVisited()
			s.SetAllowBet(false)
			s.SetAllowInsurance(false)
			s.SetAllowAction(true)
		}
	}
	if turnInfo.isNewTurn && turnInfo.roundCode == "playing" {
		if s.IsAllVisited() {
			s.SetIsGameEnded(true)
			return
		}
	}
	if turnInfo.isNewPhase && turnInfo.roundCode == "playing" {
		s.SetVisited(turnInfo.userId)
		s.SetCurrentTurn(turnInfo.userId)
		if len(s.GetLegalActions()) == 0 {
			switch s.GetCurrentHandN0(turnInfo.userId) {
			case pb.BlackjackHandN0_BLACKJACK_HAND_1ST:
				if len(s.GetPlayerPartOfHand(turnInfo.userId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND).Cards) == 2 {
					s.SetCurrentHandN0(turnInfo.userId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
				} else {
					p.turnBaseEngine.NextPhase()
					return
				}
			default:
				p.turnBaseEngine.NextPhase()
				return
			}
		}
		s.SetUpCountDown(time.Duration(turnInfo.countDown * 1e9))
		p.notifyUpdateTurn(ctx, nk, logger, dispatcher, s)
	}
}

func (p *Processor) ProcessMessageFromUser(
	ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	messages []runtime.MatchData,
	s *entity.MatchState,
) {
	for _, message := range messages {
		lib.HandlerTipInGameEvent(ctx, nk, logger, dispatcher, message)
		switch pb.OpCodeRequest(message.GetOpCode()) {
		case pb.OpCodeRequest_OPCODE_REQUEST_BET:
			if !s.IsAllowBet() {
				continue
			}
			bet := &pb.BlackjackBet{}
			if err := p.unmarshaler.Unmarshal(message.GetData(), bet); err != nil {
				logger.
					WithField("module-game", entity.ModuleName).
					WithField("user-id", message.GetUserId()).
					WithField("request-bet", message.GetData()).
					WithField("error", err).
					Error("error-parse-user-bet-request")
				continue
			}
			bet.UserId = message.GetUserId()
			s.ResetUserNotInteract(bet.UserId)
			wallet, err := entity.ReadWalletUser(ctx, nk, logger, bet.UserId)
			if err != nil {
				logger.Error("error.read-user-wallet")
				continue
			}
			// s.ResetUserNotInteract(bet.UserId)
			switch bet.Code {
			case pb.BlackjackBetCode_BLACKJACK_BET_DOUBLE:
				allow, enoughChip := s.IsCanDoubleBet(bet.UserId, wallet.Chips)
				if allow {
					chip := s.DoubleBet(bet.UserId)
					p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, bet.UserId, chip, pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
				} else if !enoughChip {
					p.notifyNotEnoughChip(ctx, nk, logger, dispatcher, s, bet.UserId)
				}
			case pb.BlackjackBetCode_BLACKJACK_BET_REBET:
				allow, enoughChip := s.IsCanRebet(bet.UserId, wallet.Chips)
				if allow {
					chip := s.Rebet(bet.UserId)
					p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, bet.UserId, chip, pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
				} else if !enoughChip {
					p.notifyNotEnoughChip(ctx, nk, logger, dispatcher, s, bet.UserId)
				}
			case pb.BlackjackBetCode_BLACKJACK_BET_NORMAL:
				if s.IsCanBet(bet.UserId, wallet.Chips, bet) {
					s.AddBet(bet)
					p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, bet.UserId, bet.Chips, pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
				} else {
					p.notifyNotEnoughChip(ctx, nk, logger, dispatcher, s, bet.UserId)
				}
			}
		case pb.OpCodeRequest_OPCODE_REQUEST_DECLARE_CARDS:
			if s.GetGameState() != pb.GameState_GameStatePlay || s.GetCurrentTurn() == "" {
				logger.WithField("user-id", message.GetUserId()).Error("current turn is empty")
				continue
			}
			if s.GetCurrentTurn() != message.GetUserId() {
				logger.WithField("user-id", message.GetUserId()).WithField("current-turn", s.GetCurrentTurn()).Error("current turn is not match")
				continue
			}
			action := &pb.BlackjackAction{}
			if err := p.unmarshaler.Unmarshal(message.GetData(), action); err != nil {
				logger.Error("error.parse-action from [%s]", err.Error())
				continue
			} else {
				s.ResetUserNotInteract(message.GetUserId())
				wallet, err := entity.ReadWalletUser(ctx, nk, logger, action.UserId)
				if err != nil {
					logger.Error("error.read-wallet %v", err.Error())
					continue
				}
				action.UserId = message.GetUserId()
				switch action.Code {
				case pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE:
					if !s.IsAllowAction() {
						continue
					}
					if !s.IsCanDoubleDownBet(action.UserId, wallet.Chips, s.GetCurrentHandN0(action.UserId)) {
						p.notifyNotEnoughChip(ctx, nk, logger, dispatcher, s, message.GetUserId())
						continue
					}
					chip := s.DoubleDownBet(action.UserId, s.GetCurrentHandN0(action.UserId))
					p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, action.UserId, chip, s.GetCurrentHandN0(action.UserId))
					cards := p.engine.Deal(1)
					s.AddCards(cards, action.UserId, s.GetCurrentHandN0(action.UserId))
					p.broadcastMessage(
						logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
						&pb.BlackjackUpdateDeal{
							IsBanker:                 false,
							IsRevealBankerHiddenCard: false,
							UserId:                   action.UserId,
							HandN0:                   s.GetCurrentHandN0(action.UserId),
							NewCards:                 cards,
							Hand:                     s.GetPlayerHand(action.UserId),
						}, nil, nil, true,
					)
					if s.GetCurrentHandN0(action.UserId) == pb.BlackjackHandN0_BLACKJACK_HAND_1ST && len(s.GetPlayerPartOfHand(action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND).Cards) == 2 {
						s.SetCurrentHandN0(action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
						p.turnBaseEngine.RePhase()
					} else {
						p.turnBaseEngine.NextPhase()
					}
				case pb.BlackjackActionCode_BLACKJACK_ACTION_HIT:
					if s.IsAllowAction() && s.IsCanHit(action.UserId, s.GetCurrentHandN0(action.UserId)) {
						cards := p.engine.Deal(1)
						s.AddCards(cards, action.UserId, s.GetCurrentHandN0(action.UserId))
						p.broadcastMessage(
							logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
							&pb.BlackjackUpdateDeal{
								IsBanker:                 false,
								IsRevealBankerHiddenCard: false,
								UserId:                   action.UserId,
								HandN0:                   s.GetCurrentHandN0(action.UserId),
								NewCards:                 cards,
								Hand:                     s.GetPlayerHand(action.UserId),
							}, nil, nil, true,
						)
						// after that hit, player can't hit anymore -> next hand if possible else next turn
						if !s.IsCanHit(action.UserId, s.GetCurrentHandN0(action.UserId)) {
							if s.GetCurrentHandN0(action.UserId) == pb.BlackjackHandN0_BLACKJACK_HAND_1ST && len(s.GetPlayerPartOfHand(action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND).Cards) == 2 {
								s.SetCurrentHandN0(action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
								p.turnBaseEngine.RePhase()
							} else {
								p.turnBaseEngine.NextPhase()
							}
						} else {
							p.turnBaseEngine.RePhase()
						}
					}
				case pb.BlackjackActionCode_BLACKJACK_ACTION_INSURANCE:
					if !s.IsAllowInsurance() {
						logger.WithField("user_id", message.GetUserId()).Info("not allow insurance")
						continue
					}
					if s.IsCanInsuranceBet(action.UserId, wallet.Chips) {
						chip := s.InsuranceBet(action.UserId)
						p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, action.UserId, chip, pb.BlackjackHandN0_BLACKJACK_HAND_UNSPECIFIED) // unspecified mean its not in any of 2 hands slot -> insurance slot
					} else {
						p.notifyNotEnoughChip(ctx, nk, logger, dispatcher, s, message.GetUserId())
					}
				case pb.BlackjackActionCode_BLACKJACK_ACTION_STAY:
					if s.IsAllowAction() && s.GetCurrentHandN0(action.UserId) == pb.BlackjackHandN0_BLACKJACK_HAND_1ST && len(s.GetPlayerPartOfHand(action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND).Cards) == 2 {
						s.SetCurrentHandN0(action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
						p.turnBaseEngine.RePhase()
						logger.Info("SWITCH TO 2ND HAND, ACTION_STAY")
					} else {
						p.turnBaseEngine.NextPhase()
						logger.Info("SWITCH TO NEXT PHASE, ACTION_STAY")
					}
				case pb.BlackjackActionCode_BLACKJACK_ACTION_SPLIT:
					if !s.IsAllowAction() {
						continue
					}
					allow, enoughChip := s.IsCanSplitHand(action.UserId, wallet.Chips)
					if !enoughChip {
						p.notifyNotEnoughChip(ctx, nk, logger, dispatcher, s, message.GetUserId())
						continue
					}
					if !allow {
						continue
					}
					chip := s.SplitHand(action.UserId)
					p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, action.UserId, chip, s.GetCurrentHandN0(action.UserId))
					p.broadcastMessage(
						logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
						&pb.BlackjackUpdateDesk{
							IsSplitHand: true,
							Hand:        s.GetPlayerHand(action.UserId),
						}, nil, nil, true,
					)
					cards := p.engine.Deal(2)
					s.AddCards([]*pb.Card{cards[0]}, action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
					p.notifyDealCard(ctx, nk, logger, dispatcher, s, action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
					s.AddCards([]*pb.Card{cards[1]}, action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
					p.notifyDealCard(ctx, nk, logger, dispatcher, s, action.UserId, pb.BlackjackHandN0_BLACKJACK_HAND_2ND)
					p.turnBaseEngine.RePhase()
				}
			}
		case pb.OpCodeRequest_OPCODE_REQUEST_INFO_TABLE:
			p.broadcastMessage(
				logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
				&pb.BlackjackUpdateDesk{
					IsNewTurn:            false,
					IsInsuranceTurnEnter: s.IsAllowInsurance(),
					InTurn:               s.GetCurrentTurn(),
				}, []runtime.Presence{s.GetPresence(message.GetUserId())}, nil, true,
			)
		case pb.OpCodeRequest_OPCODE_REQUEST_SYNC_TABLE:
			msgs := p.engine.RejoinUserMessage(s, message.GetUserId())
			if msgs == nil {
				continue
			}
			for k, msg := range msgs {
				p.broadcastMessage(logger, dispatcher, int64(k), msg, []runtime.Presence{s.GetPresence(message.GetUserId())}, nil, true)
			}
		}
	}
}

func (p *Processor) ProcessMatchKick(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	// kick user not interact
	presenseNotInteract := make(map[string]runtime.Presence)
	{
		precenses := s.GetPresences()
		for _, precense := range precenses {
			countNoInteract := s.PresencesNoInteract[precense.GetUserId()]
			if countNoInteract >= 1 {
				presenseNotInteract[precense.GetUserId()] = precense
			}
		}
		if len(presenseNotInteract) > 0 {
			list := make([]runtime.Presence, 0)
			for _, v := range presenseNotInteract {
				list = append(list, v)
			}
			p.broadcastMessage(logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_KICK_OFF_THE_TABLE), nil, list, nil, true)
			dispatcher.MatchKick(list)
		}
	}
	// kick by not enough chip
	{
		list := make([]runtime.Presence, 0)
		for _, precense := range s.GetPresences() {
			if _, exist := presenseNotInteract[precense.GetUserId()]; !exist {
				list = append(list, precense)
			}
		}
		minChipRequire := s.Label.Bet.AgLeave
		lib.MatchKick(ctx, logger, nk, dispatcher, minChipRequire, list...)
	}
}

//********************* Private functions *************************

func (p *Processor) notifyUpdateTurn(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	legalActions := &pb.BlackjackLegalActions{
		UserId:  s.GetCurrentTurn(),
		Actions: s.GetLegalActions(),
	}
	msg := &pb.BlackjackUpdateDesk{
		IsInsuranceTurnEnter: false,
		IsNewTurn:            true,
		InTurn:               s.GetCurrentTurn(),
		Hand_N0:              s.GetCurrentHandN0(s.GetCurrentTurn()),
		IsUpdateBet:          false,
		Actions:              nil,
		IsSplitHand:          false,
	}
	for _, presence := range s.GetPresences() {
		if presence.GetUserId() == s.GetCurrentTurn() {
			msg.Actions = legalActions
		} else {
			msg.Actions = nil
		}
		p.broadcastMessage(
			logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
			msg, []runtime.Presence{presence}, nil, true,
		)
	}
}

func (p *Processor) notifyUpdateBet(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	userId string,
	chip int64,
	pos pb.BlackjackHandN0,
) {
	bet := s.GetUserBetById(userId)
	updateDesk := &pb.BlackjackUpdateDesk{
		IsInsuranceTurnEnter: false,
		IsNewTurn:            false,
		IsUpdateBet:          true,
		IsUpdateLegalAction:  false,
		IsSplitHand:          false,
		Bet:                  bet,
	}
	wallet, err := entity.ReadWalletUser(ctx, nk, logger, userId)
	if err != nil {
		logger.Error("error.read-wallet [%v]", userId)
		updateDesk.Error = &pb.Error{
			Code:      int64(pb.ErrorType_ERROR_TYPE_UNSPECIFIED),
			Error:     pb.ErrorType_ERROR_TYPE_UNSPECIFIED.String(),
			ErrorType: pb.ErrorType_ERROR_TYPE_UNSPECIFIED,
		}
	}
	if wallet.Chips-chip <= 0 {
		p.notifyNotEnoughChip(ctx, nk, logger, dispatcher, s, userId)
		return
	}
	// logger.WithField("user-id", userId).Info("update-bet")
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
		updateDesk, nil, nil, true,
	)
	if updateDesk.Error != nil {
		return
	}
	p.updateChipByResultGameFinish(
		ctx, nk, logger, &pb.BalanceResult{
			Updates: []*pb.BalanceUpdate{
				{
					UserId:            userId,
					AmountChipBefore:  wallet.Chips,
					AmountChipAdd:     -chip,
					AmountChipCurrent: wallet.Chips - chip,
				},
			},
		},
	)
}

func (p *Processor) notifyNotEnoughChip(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	userId string) {
	logger.WithField("user-id", userId).Error("error.chip-not-enough")
	updateDesk := &pb.BlackjackUpdateDesk{
		IsInsuranceTurnEnter: false,
		IsNewTurn:            false,
		IsUpdateBet:          true,
		IsUpdateLegalAction:  false,
		IsSplitHand:          false,
		Bet:                  nil,
	}
	updateDesk.Error = &pb.Error{
		Code:      int64(pb.ErrorType_ERROR_TYPE_CHIP_NOT_ENOUGH),
		Error:     pb.ErrorType_ERROR_TYPE_CHIP_NOT_ENOUGH.String(),
		ErrorType: pb.ErrorType_ERROR_TYPE_CHIP_NOT_ENOUGH,
	}
	precenses := []runtime.Presence{}
	if len(userId) > 0 {
		precenses = append(precenses, s.GetPresence(userId))
	}
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
		updateDesk, precenses, nil, true,
	)
}
func (p *Processor) updateChipByResultGameFinish(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	balanceResult *pb.BalanceResult,
) {
	walletUpdates := make([]*runtime.WalletUpdate, 0, len(balanceResult.Updates))
	for _, update := range balanceResult.Updates {
		amountChip := update.AmountChipCurrent - update.AmountChipBefore
		changeset := map[string]int64{
			"chips": amountChip,
		}
		metadata := map[string]any{"game_reward": entity.ModuleName}
		walletUpdates = append(walletUpdates, &runtime.WalletUpdate{
			UserID:    update.UserId,
			Changeset: changeset,
			Metadata:  metadata,
		})
	}
	if _, err := nk.WalletsUpdate(ctx, walletUpdates, true); err != nil {
		payload, _ := json.Marshal(walletUpdates)
		logger.WithField("payload", string(payload)).
			WithField("err", err).
			Error("wallet-update-error")
	}
}

func (p *Processor) calcRewardForUserPlaying(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	updateFinish *pb.BlackjackUpdateFinish,
) (*pb.BalanceResult, int64) {
	listUserPlaying := s.GetPlayingPresences()
	listUserId := make([]string, 0)
	mapUserIdCalcReward := make(map[string]bool, 0)
	for _, u := range listUserPlaying {
		if s.IsBet(u.GetUserId()) {
			listUserId = append(listUserId, u.GetUserId())
			mapUserIdCalcReward[u.GetUserId()] = false
		}
	}
	mapUserWallet := make(map[string]entity.Wallet)
	wallets, err := entity.ReadWalletUsers(
		ctx, nk, logger, listUserId...,
	)
	if err != nil {
		data, _ := p.marshaler.Marshal(updateFinish)
		logger.
			WithField("users", strings.Join(listUserId, ", ")).
			WithField("data", string(data)).
			WithField("err", err).
			Error("error.read-wallet")
		return nil, 0
	}
	for _, w := range wallets {
		v := w
		mapUserWallet[v.UserId] = v
	}
	balanceResult := pb.BalanceResult{}
	listFeeGame := make([]entity.FeeGame, 0)
	for _, betResult := range updateFinish.BetResults {
		balance := &pb.BalanceUpdate{
			UserId:           betResult.UserId,
			AmountChipBefore: mapUserWallet[betResult.UserId].Chips,
		}
		balance.AmountChipAdd = betResult.First.Total + betResult.Second.Total + betResult.Insurance.Total
		if balance.AmountChipAdd > 0 {
			fee := int64(0)
			presence, ok := s.GetPresence(betResult.UserId).(entity.MyPrecense)
			percentFeeGame := entity.GetFeeGameByLevel(0)
			if ok {
				percentFeeGame = entity.GetFeeGameByLevel(int(presence.VipLevel))
			}
			fee = balance.AmountChipAdd / 100 * int64(percentFeeGame)
			balance.AmountChipCurrent = balance.AmountChipBefore + balance.AmountChipAdd - fee
			listFeeGame = append(listFeeGame, entity.FeeGame{
				UserID: balance.UserId,
				Fee:    fee,
			})
		} else {
			balance.AmountChipCurrent = balance.AmountChipBefore
		}
		mapUserIdCalcReward[betResult.UserId] = true
		balanceResult.Updates = append(balanceResult.Updates, balance)
	}
	for uid, isChange := range mapUserIdCalcReward {
		if isChange {
			continue
		}
		wallet := mapUserWallet[uid]
		balanceResult.Updates = append(balanceResult.Updates, &pb.BalanceUpdate{
			UserId:            uid,
			AmountChipBefore:  wallet.Chips,
			AmountChipCurrent: wallet.Chips,
			AmountChipAdd:     0,
		})
	}
	totalFee := int64(0)
	for _, fee := range listFeeGame {
		totalFee += fee.Fee
	}
	return &balanceResult, totalFee
}

func (p *Processor) notifyInitialDealCard(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) error {
	for _, presence := range s.GetPlayingPresences() {
		p.broadcastMessage(
			logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
			&pb.BlackjackUpdateDeal{
				IsBanker:                 false,
				IsRevealBankerHiddenCard: false,
				UserId:                   presence.GetUserId(),
				NewCards:                 s.GetPlayerHand(presence.GetUserId()).First.Cards,
				Hand:                     s.GetPlayerHand(presence.GetUserId()),
				HandN0:                   pb.BlackjackHandN0_BLACKJACK_HAND_1ST,
			}, nil, nil, true,
		)
		// initial legal actions for all user
		p.broadcastMessage(
			logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
			&pb.BlackjackUpdateDesk{
				IsInsuranceTurnEnter: false,
				IsNewTurn:            false,
				Hand_N0:              pb.BlackjackHandN0_BLACKJACK_HAND_1ST,
				IsUpdateBet:          false,
				Actions: &pb.BlackjackLegalActions{
					UserId:  presence.GetUserId(),
					Actions: s.GetLegalActionsByUserId(presence.GetUserId()),
				},
				IsSplitHand: false,
			}, []runtime.Presence{presence}, nil, true,
		)
	}
	dealerCards := []*pb.Card{
		s.GetDealerHand().First.GetCards()[0],
		{
			Rank: pb.CardRank_RANK_UNSPECIFIED,
			Suit: pb.CardSuit_SUIT_UNSPECIFIED,
		},
	}
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
		&pb.BlackjackUpdateDeal{
			IsBanker:                 true,
			IsRevealBankerHiddenCard: false,
			UserId:                   "",
			NewCards:                 dealerCards,
			HandN0:                   pb.BlackjackHandN0_BLACKJACK_HAND_1ST,
			Hand: &pb.BlackjackPlayerHand{
				First: &pb.BlackjackHand{
					Cards: dealerCards,
				},
			},
		}, nil, nil, true,
	)
	return nil
}

func (p *Processor) revealDealerHiddenCard(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
		&pb.BlackjackUpdateDeal{
			IsBanker:                 true,
			IsRevealBankerHiddenCard: true,
			UserId:                   "",
			NewCards:                 []*pb.Card{s.GetDealerHand().First.Cards[1]},
			HandN0:                   pb.BlackjackHandN0_BLACKJACK_HAND_1ST,
			Hand:                     s.GetDealerHand(),
		}, nil, nil, true,
	)
}

func (p *Processor) notifyDealCard(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	userId string,
	handN0 pb.BlackjackHandN0,
) error {
	isBanker := false
	var hands *pb.BlackjackPlayerHand
	if userId == "" {
		isBanker = true
		hands = s.GetDealerHand()
	} else {
		hands = s.GetPlayerHand(userId)
	}
	var hand *pb.BlackjackHand
	if handN0 == pb.BlackjackHandN0_BLACKJACK_HAND_1ST {
		hand = hands.First
	} else {
		hand = hands.Second
	}
	msg := &pb.BlackjackUpdateDeal{
		UserId:                   userId,
		IsBanker:                 isBanker,
		IsRevealBankerHiddenCard: false,
		HandN0:                   handN0,
		NewCards: []*pb.Card{
			hand.Cards[len(hand.Cards)-1],
		},
		Hand: hands,
	}
	return p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
		msg,
		nil, nil, true,
	)
}
