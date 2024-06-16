package entity

import (
	pb "github.com/ciaolink-game-platform/cgp-common/proto"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/heroiclabs/nakama-common/runtime"
)

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
}

func NewMatchState(label *pb.Match) MatchState {
	return MatchState{
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
	}
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
