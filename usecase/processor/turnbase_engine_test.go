package processor

import (
	"context"
	"testing"
	"time"
)

func TestTurnBaseEngine(t *testing.T) {
	rounds := []*Round{
		{
			code: "bet",
			phases: []*Phase{
				{
					code:     "main",
					duration: time.Second * 5,
				},
			},
			isGlob: true,
		},
		{
			code: "insurance",
			phases: []*Phase{
				{
					code:     "main",
					duration: time.Second * 3,
				},
			},
			isGlob: true,
		},
		{
			code: "playing",
			phases: []*Phase{
				{
					code:     "main",
					duration: time.Second * 1,
				},
			},
			isGlob: false,
		},
	}
	players := []string{
		"A", "B", "C", "D", "E",
	}
	type args struct {
		players []string
		rounds  []*Round
	}
	tests := []struct {
		name string
		args args
		want *TurnBaseEngine
	}{
		{
			name: "T1",
			args: args{
				players: players,
				rounds:  rounds,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTurnBaseEngine()
			got.Config(tt.args.players, tt.args.rounds)
			got.SetCurrentPlayer("B")
			if got == nil {
				t.Errorf("Engine is nil")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			for {
				time.Sleep(100 * time.Millisecond)
				info := got.Loop()
				if info.isNewRound {
					t.Logf("======= Round %v ========\n", info.roundCode)
				}
				if info.isNewTurn {
					t.Logf("-->> %v Turn\n", info.userId)
				}
				if info.isNewPhase {
					t.Logf("+++ Phase %v - cd %v\n", info.phaseCode, info.countDown)
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		})
	}
}
