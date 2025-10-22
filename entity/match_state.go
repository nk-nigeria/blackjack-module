package entity

import (
	"fmt"
	"math/rand"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/nk-nigeria/cgp-common/bot"
	pb "github.com/nk-nigeria/cgp-common/proto"
	"google.golang.org/protobuf/proto"
)

var BotLoader = bot.NewBotLoader(nil, "", 0)
var marshaler = &proto.MarshalOptions{}

// GetMarshaler returns the global marshaler
func GetMarshaler() *proto.MarshalOptions {
	return marshaler
}

// NewBotMatchData creates a new bot match data
func NewBotMatchData(opCode pb.OpCodeRequest, data []byte, presence *bot.BotPresence) runtime.MatchData {
	return bot.NewBotMatchData(opCode, data, presence)
}

const (
	MinPresences  = 1
	MaxPresences  = 5
	MinBetAllowed = 1
	MaxBetAllowed = 200
	TickRate      = 2
)

type MatchState struct {
	baseMatchState

	allowBet       bool
	allowInsurance bool
	allowAction    bool
	visited        map[string]bool
	userBets       map[string]*pb.BlackjackPlayerBet
	userLastBets   map[string]int64
	userHands      map[string]*Hand
	dealerHand     *Hand
	currentTurn    string
	currentHand    map[string]pb.BlackjackHandN0
	// gameState      pb.GameState
	updateFinish *pb.BlackjackUpdateFinish
	isGameEnded  bool

	// Bot-related fields
	messages   []runtime.MatchData
	Bots       []*bot.BotPresence
	EmitBot    bool
	MatchCount int
	BotResults map[string]int // Create a map to store individual bot results

	// Bot logic for intelligent betting decisions
	BotLogic *BlackjackBotLogic
}

func NewMatchState(label *pb.Match) MatchState {
	m := MatchState{
		baseMatchState: baseMatchState{
			Label:               label,
			MinPresences:        MinPresences,
			MaxPresences:        MaxPresences,
			Presences:           linkedhashmap.New(),
			PlayingPresences:    linkedhashmap.New(),
			LeavePresences:      linkedhashmap.New(),
			PresencesNoInteract: make(map[string]int, 0),
			balanceResult:       nil,
		},
		userBets:     make(map[string]*pb.BlackjackPlayerBet, 0),
		userLastBets: make(map[string]int64, 0),
		userHands:    make(map[string]*Hand, 0),
		dealerHand:   &Hand{},
		currentTurn:  "",
		currentHand:  make(map[string]pb.BlackjackHandN0, 0),
		// gameState:    pb.GameState_GameStateIdle,
		updateFinish: nil,
		isGameEnded:  false,
		BotResults:   make(map[string]int, 0),
		BotLogic:     NewBlackjackBotLogic(),
	}
	// Automatically add bot players
	if bots, err := BotLoader.GetFreeBot(int(label.NumBot)); err != nil {
		fmt.Printf("\r\n load bot failed %s  \r\n", err.Error())
	} else {
		m.Bots = bots
	}
	for _, bot := range m.Bots {
		m.Presences.Put(bot.GetUserId(), bot)
		m.Label.Size += 1
	}
	return m
}

func (s *MatchState) InitUserBet() {
	for k := range s.userBets {
		delete(s.userBets, k)
	}
}

func (s *MatchState) Init() {
	for k := range s.userHands {
		delete(s.userHands, k)
	}
	s.balanceResult = nil
	s.dealerHand = &Hand{
		first: make([]*pb.Card, 0),
	}
	s.currentTurn = ""
	s.updateFinish = nil
	for _, presence := range s.GetPlayingPresences() {
		s.currentHand[presence.GetUserId()] = pb.BlackjackHandN0_BLACKJACK_HAND_1ST
	}
	s.isGameEnded = false
}

// InitTurnBot initializes bot turn for betting in preparing phase
func (s *MatchState) InitTurnBot(botPresence *bot.BotPresence) {
	// Reset bot logic for new game
	if s.BotLogic != nil {
		s.BotLogic.Reset()
	}
	preparingTimeout := GameStateDuration[pb.GameState_GAME_STATE_PREPARING].Seconds()
	opt := bot.TurnOpt{
		MaxOccur: bot.RandomInt(1, 3),                // Bot will bet 1-3 times during preparing
		MinTick:  2 * TickRate,                       // Earliest: 1 second after preparing starts
		MaxTick:  int(preparingTimeout-2) * TickRate, // Latest: 1 second before preparing ends
	}

	botPresence.InitTurnWithOption(opt, func() {
		s.BotTurn(botPresence)
	})
}

func (s *MatchState) InitVisited() {
	s.visited = make(map[string]bool, 0)
	for k := range s.userHands {
		s.visited[k] = false
	}
}

func (s *MatchState) IsAllVisited() bool {
	if s.visited == nil {
		return false
	} else {
		for _, v := range s.visited {
			if !v {
				return false
			}
		}
		return true
	}
}

func (s *MatchState) SetVisited(userId string) {
	s.visited[userId] = true
}

func (s *MatchState) SetCurrentHandN0(userId string, v pb.BlackjackHandN0) { s.currentHand[userId] = v }
func (s *MatchState) GetCurrentHandN0(userId string) pb.BlackjackHandN0    { return s.currentHand[userId] }

func (s *MatchState) SetCurrentTurn(v string) { s.currentTurn = v }
func (s *MatchState) GetCurrentTurn() string  { return s.currentTurn }

func (s *MatchState) GetGameState() pb.GameState  { return s.Label.GameState }
func (s *MatchState) SetGameState(v pb.GameState) { s.Label.GameState = v }

func (s *MatchState) SetIsGameEnded(v bool) { s.isGameEnded = v }
func (s *MatchState) IsGameEnded() bool     { return s.isGameEnded }

func (s *MatchState) GetPlayerHand(userId string) *pb.BlackjackPlayerHand {
	return s.userHands[userId].ToPb()
}

func (s *MatchState) PlayerHand(userId string) *Hand {
	return s.userHands[userId]
}

func (s *MatchState) GetPlayerPartOfHand(userId string, pos pb.BlackjackHandN0) *pb.BlackjackHand {
	if pos == pb.BlackjackHandN0_BLACKJACK_HAND_1ST {
		return s.userHands[userId].ToPb().First
	} else {
		return s.userHands[userId].ToPb().Second
	}
}

func (s *MatchState) GetDealerHand() *pb.BlackjackPlayerHand {
	return s.dealerHand.ToPb()
}

func (s *MatchState) DealerHand() *Hand {
	return s.dealerHand
}

func (s *MatchState) AddCards(cards []*pb.Card, userId string, handN0 pb.BlackjackHandN0) {
	if userId == "" {
		s.dealerHand.AddCards(cards, pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	} else {
		if _, found := s.userHands[userId]; !found {
			s.userHands[userId] = &Hand{
				userId: userId,
				first:  make([]*pb.Card, 0),
				second: make([]*pb.Card, 0),
			}
		}
		s.userHands[userId].AddCards(cards, handN0)
	}
}

func (s *MatchState) SetAllowBet(v bool) { s.allowBet = v }
func (s *MatchState) IsAllowBet() bool   { return s.allowBet }

func (s *MatchState) SetAllowInsurance(v bool) { s.allowInsurance = v }
func (s *MatchState) IsAllowInsurance() bool   { return s.allowInsurance }

func (s *MatchState) SetAllowAction(v bool) { s.allowAction = v }
func (s *MatchState) IsAllowAction() bool   { return s.allowAction }

func (s *MatchState) SetUpdateFinish(v *pb.BlackjackUpdateFinish) { s.updateFinish = v }
func (s *MatchState) GetUpdateFinish() *pb.BlackjackUpdateFinish  { return s.updateFinish }

func (s *MatchState) GetUserBetById(userId string) *pb.BlackjackPlayerBet { return s.userBets[userId] }
func (s *MatchState) SetUserBetById(userId string, bet *pb.BlackjackPlayerBet) {
	s.userBets[userId] = bet
}

func (s *MatchState) IsCanBet(userId string, balance int64, bet *pb.BlackjackBet) bool {
	// fmt.Printf("[LABEL.BET] = %v", s.Label.MarkUnit)
	if _, found := s.userBets[userId]; !found {
		return bet.Chips <= balance
		// && bet.Chips <= int64(MaxBetAllowed*s.Label.Bet)
	}
	if balance < bet.Chips {
		// || bet.Chips+s.userBets[userId].First+s.userBets[userId].Insurance+s.userBets[userId].Second > int64(MaxBetAllowed*s.Label.Bet)
		return false
	}
	return true
}

func (s *MatchState) AddBet(v *pb.BlackjackBet) {
	if _, found := s.userBets[v.UserId]; !found {
		s.userBets[v.UserId] = &pb.BlackjackPlayerBet{
			UserId:    v.UserId,
			Insurance: 0,
			First:     0,
			Second:    0,
		}
	}
	s.userBets[v.UserId].First += v.Chips
	s.userLastBets[v.UserId] = s.userBets[v.UserId].First
	s.allowAction = false
}

func (s *MatchState) IsCanInsuranceBet(userId string, balance int64) bool {
	return balance*2 >= s.userBets[userId].First
}

func (s *MatchState) InsuranceBet(userId string) int64 {
	s.userBets[userId].Insurance = s.userBets[userId].First / 2
	return s.userBets[userId].Insurance
}

func (s *MatchState) IsCanDoubleDownBet(userId string, balance int64, pos pb.BlackjackHandN0) bool {
	if pos == pb.BlackjackHandN0_BLACKJACK_HAND_1ST {
		return balance >= s.userBets[userId].First
	} else {
		return balance >= s.userBets[userId].Second
	}
}

func (s *MatchState) DoubleDownBet(userId string, pos pb.BlackjackHandN0) int64 {
	r := int64(0)
	if pos == pb.BlackjackHandN0_BLACKJACK_HAND_1ST {
		r = s.userBets[userId].First
		s.userBets[userId].First *= 2
	} else if pos == pb.BlackjackHandN0_BLACKJACK_HAND_2ND {
		r = s.userBets[userId].Second
		s.userBets[userId].Second *= 2
	}
	return r
}

func (s *MatchState) IsCanSplitHand(userId string, balance int64) (allow bool, enougChip bool) {
	enougChip = balance >= s.userBets[userId].First
	allow = false
	if !enougChip {
		return allow, enougChip
	}
	allow = s.userHands[userId].PlayerCanSplit()
	return allow, enougChip
}

func (s *MatchState) SplitHand(userId string) int64 {
	s.userBets[userId].Second = s.userBets[userId].First
	s.userHands[userId].Split()
	return s.userBets[userId].Second
}

func (s *MatchState) Rebet(userId string) int64 {
	if _, found := s.userBets[userId]; !found {
		s.userBets[userId] = &pb.BlackjackPlayerBet{
			UserId:    userId,
			Insurance: 0,
			First:     0,
			Second:    0,
		}
	}
	s.userBets[userId].First = s.userLastBets[userId]
	return s.userLastBets[userId]
}

func (s *MatchState) DoubleBet(userId string) int64 {
	if _, found := s.userBets[userId]; found && s.userBets[userId].First >= MinBetAllowed*int64(s.Label.MarkUnit) {
		r := s.userBets[userId].First
		s.userBets[userId].First *= 2
		s.userLastBets[userId] = s.userBets[userId].First
		return r
	} else if _, found := s.userLastBets[userId]; found {
		if _, found := s.userBets[userId]; !found {
			s.userBets[userId] = &pb.BlackjackPlayerBet{
				UserId:    userId,
				Insurance: 0,
				First:     0,
				Second:    0,
			}
		}
		s.userLastBets[userId] *= 2
		s.userBets[userId].First = s.userLastBets[userId]
		return s.userLastBets[userId]
	}
	return 0
}

func (s *MatchState) IsCanRebet(userId string, balance int64) (allow bool, enougChip bool) {
	if _, found := s.userBets[userId]; found {
		return false, true
	}
	if _, found := s.userLastBets[userId]; !found || s.userLastBets[userId] > balance {
		return false, false
	}
	return true, true
}

func (s *MatchState) IsCanDoubleBet(userId string, balance int64) (allow bool, enougChip bool) {
	allow = false
	chipNeed := int64(0)
	if _, found := s.userBets[userId]; found {
		chipNeed = s.userBets[userId].First
		allow = true
	} else if _, found := s.userLastBets[userId]; found {
		allow = true
		chipNeed = s.userLastBets[userId] * 2
	}
	enougChip = chipNeed <= balance
	if !enougChip {
		allow = false
	}
	return allow, enougChip

}

func (s *MatchState) IsCanHit(userId string, pos pb.BlackjackHandN0) bool {
	return s.userHands[userId].PlayerCanDraw(pos)
}

func (s *MatchState) IsBet(userId string) bool {
	if _, found := s.userBets[userId]; found && s.userBets[userId].First > 0 {
		return true
	}
	return false
}

// override
func (s *MatchState) IsReadyToPlay() bool {
	if s.Presences.Size() < s.MinPresences {
		return false
	}
	for _, presence := range s.GetPresences() {
		if s.IsBet(presence.GetUserId()) {
			return true
		}
	}
	return false
}

func (s *MatchState) IsEnoughPlayer() bool {
	return s.Presences.Size() >= s.MinPresences
}

// override
func (s *MatchState) SetupMatchPresence() {
	s.PlayingPresences = linkedhashmap.New()
	p := make([]runtime.Presence, 0, s.GetPresenceSize())
	s.Presences.Each(func(key, value interface{}) {
		if s.IsBet(key.(string)) {
			p = append(p, value.(runtime.Presence))
		}
	})
	s.AddPlayingPresences(p...)
}

func (s *MatchState) CalcGameFinish() *pb.BlackjackUpdateFinish {
	result := &pb.BlackjackUpdateFinish{
		BetResults: make([]*pb.BlackjackPLayerBetResult, 0),
	}
	for _, h := range s.userHands {
		result.BetResults = append(result.BetResults, s.getPlayerBetResult(h.userId))
	}
	return result
}

func (s *MatchState) getPlayerBetResult(userId string) *pb.BlackjackPLayerBetResult {
	defer func() { s.userBets[userId].Insurance = 0 }()
	userBet := s.userBets[userId]
	r1, r2 := s.userHands[userId].Compare(s.dealerHand)
	insurance := &pb.BlackjackBetResult{
		BetAmount: userBet.Insurance,
		WinAmount: 0,
		Total:     0,
	}
	first := &pb.BlackjackBetResult{
		BetAmount: userBet.First,
		WinAmount: 0,
		Total:     userBet.First,
	}
	second := &pb.BlackjackBetResult{
		BetAmount: userBet.Second,
		WinAmount: 0,
		Total:     userBet.Second,
	}
	// meaning that currently in insurance round
	if insurance.BetAmount > 0 {
		// case win bet -> game also ended
		if _, _, dt := s.dealerHand.Eval(1); dt == pb.BlackjackHandType_BLACKJACK_HAND_TYPE_BLACKJACK {
			insurance.WinAmount = insurance.BetAmount * 2
			insurance.Total = insurance.BetAmount + insurance.WinAmount
			insurance.IsWin = 1
			// case not win bet -> game will continue, return result of insurance bet only
		} else {
			insurance.WinAmount = -insurance.BetAmount
			insurance.Total = insurance.BetAmount + insurance.WinAmount
			insurance.IsWin = -1
			return &pb.BlackjackPLayerBetResult{
				UserId:    userId,
				Insurance: insurance,
			}
		}
	}
	if first.BetAmount > 0 {
		first.IsWin = int32(r1)
		if r1 > 0 {
			first.WinAmount = first.BetAmount
			first.Total = first.BetAmount + first.WinAmount
		} else if r1 < 0 {
			first.WinAmount = -first.BetAmount
			first.Total = first.BetAmount + first.WinAmount
		}
	}
	if second.BetAmount > 0 {
		second.IsWin = int32(r2)
		if r2 > 0 {
			second.WinAmount = second.BetAmount
			second.Total = second.BetAmount + second.WinAmount
		} else if r2 < 0 {
			second.WinAmount = -second.BetAmount
			second.Total = second.BetAmount + second.WinAmount
		}
	}
	return &pb.BlackjackPLayerBetResult{
		UserId:    userId,
		Insurance: insurance,
		First:     first,
		Second:    second,
	}
}

func (s *MatchState) GetLegalActions() []pb.BlackjackActionCode {
	result := make([]pb.BlackjackActionCode, 0)
	if s.userHands[s.currentTurn].PlayerCanDraw(s.currentHand[s.currentTurn]) {
		result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_HIT)
		if len(s.GetPlayerPartOfHand(s.currentTurn, s.currentHand[s.currentTurn]).Cards) == 2 {
			result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE)
			if s.userHands[s.currentTurn].PlayerCanSplit() {
				result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_SPLIT)
			}
		}
		result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_STAY)
	}
	return result
}

func (s *MatchState) GetLegalActionsByUserId(userId string) []pb.BlackjackActionCode {
	result := make([]pb.BlackjackActionCode, 0)
	if s.userHands[userId].PlayerCanDraw(s.currentHand[userId]) {
		result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_HIT)
		if len(s.GetPlayerPartOfHand(userId, s.currentHand[userId]).Cards) == 2 {
			result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_DOUBLE)
			if s.userHands[userId].PlayerCanSplit() {
				result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_SPLIT)
			}
		}
		result = append(result, pb.BlackjackActionCode_BLACKJACK_ACTION_STAY)
	}
	return result
}

func (s *MatchState) DealerPotentialBlackjack() bool {
	return s.dealerHand.DealerPotentialBlackjack()
}

func (s *MatchState) IsDealerMustDraw() bool {
	return s.dealerHand.DealerMustDraw()
}

func (s *MatchState) GetPlayersBet() []*pb.BlackjackPlayerBet {
	res := make([]*pb.BlackjackPlayerBet, 0)
	for k, v := range s.userBets {
		res = append(res, &pb.BlackjackPlayerBet{
			UserId:    k,
			Insurance: v.Insurance,
			First:     v.First,
			Second:    v.Second,
		})
	}
	return res
}

// Messages returns and clears the messages queue
func (s *MatchState) Messages() []runtime.MatchData {
	msgs := s.messages
	s.messages = make([]runtime.MatchData, 0)
	return msgs
}

// AddMessages adds messages to the queue
func (s *MatchState) AddMessages(mgs ...runtime.MatchData) {
	s.messages = append(s.messages, mgs...)
}

// BotLoop loops through all bots and calls their Loop method
func (s *MatchState) BotLoop() {
	for _, v := range s.Bots {
		v.Loop()
	}
}

// BotTurn handles bot betting decisions
func (s *MatchState) BotTurn(v *bot.BotPresence) error {
	s.ResetUserNotInteract(v.GetUserId())
	userId := v.GetUserId()

	fmt.Printf("[DEBUG] [BotTurn] Handle bot turn for user: %s\n", userId)

	// Use intelligent bot logic for betting decisions
	if s.BotLogic != nil {
		// Set bot balance based on current game state
		s.BotLogic.SetBalance(int64(s.Label.MarkUnit) * 100) // Assume 100x base unit

		// Generate intelligent bet using bot logic
		botBet := s.BotLogic.GenerateBotBet()
		botBet.UserId = userId

		if botBet.First <= 0 {
			fmt.Printf("[DEBUG] [BotTurn] Bot %s generated 0 bet, skipping\n", userId)
			return nil
		}

		fmt.Printf("[DEBUG] [BotTurn] Bot %s placing intelligent bet: %d chips\n", userId, botBet.First)

		// Create bet message (convert BlackjackPlayerBet to BlackjackBet for opcode)
		betMessage := &pb.BlackjackBet{
			UserId: userId,
			Chips:  botBet.First,
			Code:   pb.BlackjackBetCode_BLACKJACK_BET_NORMAL,
		}

		// Marshal and send via message queue like a real user
		buf, _ := marshaler.Marshal(betMessage)
		data := bot.NewBotMatchData(
			pb.OpCodeRequest_OPCODE_REQUEST_BET, buf, v,
		)
		s.AddMessages(data)

		fmt.Printf("[DEBUG] [BotTurn] Bot %s bet message queued successfully\n", userId)
		return nil
	}

	// Fallback to old random betting logic
	fmt.Printf("[DEBUG] [BotTurn] Using fallback random betting for bot %s\n", userId)
	betAmount := int64(s.Label.Bet.MarkUnit) * int64(rand.Intn(5)+1) // Random bet 1-5x base unit

	bet := &pb.BlackjackBet{
		UserId: userId,
		Chips:  betAmount,
		Code:   pb.BlackjackBetCode_BLACKJACK_BET_NORMAL,
	}

	// Marshal and send via message queue
	buf, _ := marshaler.Marshal(bet)
	data := bot.NewBotMatchData(
		pb.OpCodeRequest_OPCODE_REQUEST_BET, buf, v,
	)
	s.AddMessages(data)

	fmt.Printf("[DEBUG] [BotTurn] Bot %s placed fallback bet: %d chips via message queue\n", userId, betAmount)
	return nil
}

// HasInsuranceBet checks if user has already placed insurance bet
func (s *MatchState) HasInsuranceBet(userId string) bool {
	if bet, found := s.userBets[userId]; found {
		return bet.Insurance > 0
	}
	return false
}

// BotInsuranceAction handles bot insurance decision and sends via message queue
func (s *MatchState) BotInsuranceAction(v *bot.BotPresence) error {
	userId := v.GetUserId()

	fmt.Printf("[DEBUG] [BotInsuranceAction] Handle bot insurance for user: %s\n", userId)

	// Get player hand
	playerHand := s.GetPlayerPartOfHand(userId, pb.BlackjackHandN0_BLACKJACK_HAND_1ST)
	if playerHand == nil {
		fmt.Printf("[DEBUG] [BotInsuranceAction] Hand not found for user %s, skipping insurance\n", userId)
		return nil
	}

	// Get dealer's up card
	var dealerUpCard *pb.Card
	if s.dealerHand != nil && len(s.dealerHand.first) > 0 {
		dealerUpCard = s.dealerHand.first[0] // First card is face up
	}

	if dealerUpCard == nil || dealerUpCard.Rank != pb.CardRank_RANK_A {
		fmt.Printf("[DEBUG] [BotInsuranceAction] Dealer doesn't have Ace, no insurance needed\n")
		return nil
	}

	// Bot decides whether to take insurance
	var shouldTakeInsurance bool
	if s.BotLogic != nil {
		shouldTakeInsurance = s.BotLogic.ShouldTakeInsurance(playerHand, dealerUpCard)
		fmt.Printf("[DEBUG] [BotInsuranceAction] Bot %s intelligent insurance decision: %v\n", userId, shouldTakeInsurance)
	} else {
		// Fallback: random decision (30% chance)
		shouldTakeInsurance = rand.Intn(100) < 30
		fmt.Printf("[DEBUG] [BotInsuranceAction] Bot %s random insurance decision: %v\n", userId, shouldTakeInsurance)
	}

	if !shouldTakeInsurance {
		fmt.Printf("[DEBUG] [BotInsuranceAction] Bot %s decided NOT to take insurance\n", userId)
		return nil
	}

	// Create insurance action message
	insuranceAction := &pb.BlackjackAction{
		UserId: userId,
		Code:   pb.BlackjackActionCode_BLACKJACK_ACTION_INSURANCE,
	}

	// Marshal and send via message queue
	buf, _ := marshaler.Marshal(insuranceAction)
	data := bot.NewBotMatchData(
		pb.OpCodeRequest_OPCODE_REQUEST_DECLARE_CARDS, buf, v,
	)
	s.AddMessages(data)

	fmt.Printf("[DEBUG] [BotInsuranceAction] Bot %s insurance action queued\n", userId)
	return nil
}

// BotAction handles bot action decisions during game play and sends via message queue
func (s *MatchState) BotAction(v *bot.BotPresence, legalActions []pb.BlackjackActionCode) error {
	userId := v.GetUserId()

	fmt.Printf("[DEBUG] [BotAction] Handle bot action for user: %s, legal actions: %v\n", userId, legalActions)

	var action pb.BlackjackActionCode

	// Use intelligent bot logic for action decisions
	if s.BotLogic != nil {
		// Get current player hand
		playerHand := s.GetPlayerPartOfHand(userId, s.currentHand[userId])
		if playerHand == nil {
			fmt.Printf("[DEBUG] [BotAction] Hand not found for user %s, using fallback\n", userId)
			// Use random action if hand not found
			if len(legalActions) > 0 {
				action = legalActions[rand.Intn(len(legalActions))]
			} else {
				action = pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
			}
		} else {
			// Get dealer's up card
			var dealerUpCard *pb.Card
			if s.dealerHand != nil && len(s.dealerHand.first) > 0 {
				dealerUpCard = s.dealerHand.first[0] // First card is face up
			}

			// Decide action using bot logic
			action = s.BotLogic.DecideGameAction(playerHand, dealerUpCard, legalActions)

			fmt.Printf("[DEBUG] [BotAction] Bot %s intelligent action decision: %v\n", userId, action)
		}
	} else {
		// Fallback to random action selection
		fmt.Printf("[DEBUG] [BotAction] Using fallback random action for bot %s\n", userId)
		if len(legalActions) > 0 {
			action = legalActions[rand.Intn(len(legalActions))]
		} else {
			action = pb.BlackjackActionCode_BLACKJACK_ACTION_STAY
		}
	}

	// Create action message
	actionMessage := &pb.BlackjackAction{
		UserId: userId,
		Code:   action,
	}

	// Add action to history
	if s.BotLogic != nil {
		s.BotLogic.AddActionHistory(actionMessage)
	}

	// Marshal and send via message queue like a real user
	buf, _ := marshaler.Marshal(actionMessage)
	data := bot.NewBotMatchData(
		pb.OpCodeRequest_OPCODE_REQUEST_DECLARE_CARDS, buf, v,
	)
	s.AddMessages(data)

	fmt.Printf("[DEBUG] [BotAction] Bot %s action message queued: %v\n", userId, action)
	return nil
}

// GetMatchID returns the match ID
func (s *MatchState) GetMatchID() string {
	return s.Label.MatchId
}

// GetBetAmount returns the bet amount
func (s *MatchState) GetBetAmount() int64 {
	return int64(s.Label.MarkUnit)
}

// GetLastResult returns the last game result (for bot decision making)
func (s *MatchState) GetLastResult() int {
	// For blackjack, we can use a simple result calculation
	// This is a placeholder - you might want to implement more sophisticated logic
	if s.isGameEnded && s.updateFinish != nil {
		// Return a simple result based on game outcome
		// 1 = player win, -1 = dealer win, 0 = tie
		return 1 // Placeholder
	}
	return 0
}
