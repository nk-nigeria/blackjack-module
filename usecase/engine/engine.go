package engine

import (
	"github.com/ciaolink-game-platform/blackjack-module/entity"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"
	"google.golang.org/protobuf/proto"
)

type Engine struct {
	deck *entity.Deck
}

func NewGameEngine() UseCase {
	return &Engine{}
}

func (m *Engine) NewGame(s *entity.MatchState) error {
	m.deck = entity.NewDeck()
	m.deck.Shuffle()
	s.Init()
	return nil
}

func (m *Engine) Deal(amount int) []*pb.Card {
	if list, err := m.deck.Deal(amount); err != nil {
		return nil
	} else {
		return list.Cards
	}
}

func (m *Engine) RejoinUserMessage(s *entity.MatchState, userId string) map[pb.OpCodeUpdate]proto.Message {
	messages := make(map[pb.OpCodeUpdate]proto.Message)
	if s.GetGameState() == pb.GameState_GameStatePlay {
		hands := []*pb.BlackjackPlayerHand{}
		dealerHand := &pb.BlackjackPlayerHand{
			UserId: "",
			First: &pb.BlackjackHand{
				Cards: []*pb.Card{
					s.GetDealerHand().First.Cards[0],
					{Rank: pb.CardRank_RANK_UNSPECIFIED, Suit: pb.CardSuit_SUIT_UNSPECIFIED},
				},
			},
		}
		hands = append(hands, dealerHand)
		for _, presence := range s.GetPlayingPresences() {
			hands = append(hands, s.GetPlayerHand(presence.GetUserId()))
		}
		messages[pb.OpCodeUpdate_OPCODE_UPDATE_DEAL] = &pb.BlackjackUpdateDeal{
			AllPlayerHand: hands,
			IsBanker:      false,
		}

		if s.GetCurrentTurn() == userId {
			messages[pb.OpCodeUpdate_OPCODE_UPDATE_TABLE] = &pb.BlackjackUpdateDesk{
				IsInsuranceTurnEnter: s.IsAllowInsurance(),
				InTurn:               s.GetCurrentTurn(),
				Hand_N0:              s.GetCurrentHandN0(s.GetCurrentTurn()),
				IsUpdateLegalAction:  true,
				Actions: &pb.BlackjackLegalActions{
					UserId:  s.GetCurrentTurn(),
					Actions: s.GetLegalActions(),
				},
				PlayersBet: s.GetPlayersBet(),
			}
		} else {
			messages[pb.OpCodeUpdate_OPCODE_UPDATE_TABLE] = &pb.BlackjackUpdateDesk{
				IsInsuranceTurnEnter: s.IsAllowInsurance(),
				InTurn:               s.GetCurrentTurn(),
				Hand_N0:              s.GetCurrentHandN0(s.GetCurrentTurn()),
				IsUpdateLegalAction:  false,
				Actions:              nil,
			}
		}
	}
	if s.GetGameState() == pb.GameState_GameStateReward {
		messages[pb.OpCodeUpdate_OPCODE_UPDATE_WALLET] = s.GetBalanceResult()
	}
	return messages
}

func (m *Engine) Finish(s *entity.MatchState) *pb.BlackjackUpdateFinish {
	return s.CalcGameFinish()
}

func (m *Engine) Draw(s *entity.MatchState, userId string, handN0 pb.BlackjackHandN0) {
	s.AddCards(m.Deal(1), userId, handN0)
}

func (m *Engine) DoubleDown(s *entity.MatchState, userId string, handN0 pb.BlackjackHandN0) int64 {
	s.AddCards(m.Deal(1), userId, handN0)
	return s.DoubleDownBet(userId, handN0)
}
func (m *Engine) Split(s *entity.MatchState, userId string) int64 {
	return s.SplitHand(userId)
}
func (m *Engine) Insurance(s *entity.MatchState, userId string) int64 {
	return s.InsuranceBet(userId)
}
