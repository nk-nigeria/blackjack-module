package processor

import (
	"math"
	"time"
)

type Phase struct {
	code     string
	duration time.Duration
}

type Round struct {
	code   string
	phases []*Phase
	isGlob bool
}

type TurnInfo struct {
	userId            string
	roundCode         string
	phaseCode         string
	isNewRound        bool
	isNewPhase        bool
	isNewTurn         bool
	countDown         int
	prevTimeout       bool
	prevTimeoutUserId string
}

type TurnBaseEngine struct {
	players          []string
	size             int
	rounds           []*Round
	curRound         int
	curPhase         int
	curPlayer        int
	isNewTurn        bool
	isNewRound       bool
	isNewPhase       bool
	isInit           bool
	countdownEndTime time.Time
}

func NewTurnBaseEngine() *TurnBaseEngine {
	return &TurnBaseEngine{}
}

func (m *TurnBaseEngine) Config(players []string, rounds []*Round) {
	m.players = players
	m.size = len(players)
	m.rounds = rounds
	m.isInit = true
	m.isNewTurn = true
	m.isNewRound = true
	m.isNewPhase = true
	m.curRound = 0
	m.curPhase = 0
	m.curPlayer = 0
	m.SetCountDown()
}

func (m *TurnBaseEngine) Loop() *TurnInfo {
	if !m.isInit {
		return nil
	}
	defer func() {
		m.isNewRound = false
		m.isNewTurn = false
		m.isNewPhase = false
	}()
	// next phase (player) by timeout
	if diff := m.GetRemainCountDown(); diff < 0 {
		prevUID := m.players[m.curPlayer]
		if !m.NextPhase() {
			if m.IsGlob() {
				m.NextRound()
			} else {
				m.NextPlayer()
			}
		}
		isNewRound, isNewTurn, isNewPhase := m.isNewRound, m.isNewTurn, m.isNewPhase
		return &TurnInfo{
			userId:            m.players[m.curPlayer],
			roundCode:         m.rounds[m.curRound].code,
			phaseCode:         m.rounds[m.curRound].phases[m.curPhase].code,
			isNewRound:        isNewRound,
			isNewTurn:         isNewTurn,
			isNewPhase:        isNewPhase,
			countDown:         m.GetRemainCountDown(),
			prevTimeout:       true,
			prevTimeoutUserId: prevUID,
		}
	}
	isNewRound, isNewTurn, isNewPhase := m.isNewRound, m.isNewTurn, m.isNewPhase
	return &TurnInfo{
		userId:      m.players[m.curPlayer],
		roundCode:   m.rounds[m.curRound].code,
		phaseCode:   m.rounds[m.curRound].phases[m.curPhase].code,
		isNewRound:  isNewRound,
		isNewTurn:   isNewTurn,
		isNewPhase:  isNewPhase,
		countDown:   m.GetRemainCountDown(),
		prevTimeout: false,
	}
}

func (m *TurnBaseEngine) NextRound() bool {
	if m.curRound < len(m.rounds)-1 {
		m.curRound++
		m.isNewRound = true
		m.isNewTurn = true
		m.isNewPhase = true
		m.SetCountDown()
		return true
	} else {
		return false
	}
}

func (m *TurnBaseEngine) NextPlayer() {
	m.curPlayer++
	m.curPlayer %= m.size
	m.isNewTurn = true
	m.curPhase = 0
	m.isNewPhase = true
	m.SetCountDown()
}

func (m *TurnBaseEngine) RePhase() {
	m.SetCountDown()
	m.isNewPhase = true
}

func (m *TurnBaseEngine) NextPhase() bool {
	if m.curPhase < len(m.rounds[m.curRound].phases)-1 {
		m.curPhase++
		m.isNewPhase = true
		m.SetCountDown()
		return true
	} else {
		return false
	}
}

func (m *TurnBaseEngine) SetCountDown() {
	m.countdownEndTime = time.Now().Add(m.rounds[m.curRound].phases[m.curPhase].duration)
}

func (m *TurnBaseEngine) SetCurrentRound(code string) {
	for k, v := range m.rounds {
		if v.code == code {
			m.curRound = k
			return
		}
	}
}

func (m *TurnBaseEngine) SetCurrentPlayer(userId string) {
	for k, v := range m.players {
		if userId == v {
			m.curPlayer = k
			return
		}
	}
}

func (m *TurnBaseEngine) SetCurrentPhase(code string) {
	for k, v := range m.rounds[m.curRound].phases {
		if v.code == code {
			m.curPhase = k
			return
		}
	}
}

func (m *TurnBaseEngine) GetRemainCountDown() int {
	return int(math.Round(time.Until(m.countdownEndTime).Seconds()))
}

func (m *TurnBaseEngine) IsGlob() bool {
	return m.rounds[m.curRound].isGlob
}
