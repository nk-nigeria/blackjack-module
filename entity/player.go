package entity

import (
	"strconv"

	pb "github.com/ciaolink-game-platform/cgp-common/proto"

	"github.com/heroiclabs/nakama-common/runtime"
)

type ArrPbPlayer []*pb.Player

func NewPlayer(presence runtime.Presence) *pb.Player {
	p := pb.Player{
		Id:       presence.GetUserId(),
		UserName: presence.GetUsername(),
	}
	m, ok := presence.(MyPrecense)
	if ok {
		p.AvatarId = m.AvatarId
		p.VipLevel = m.VipLevel
		p.Wallet = strconv.FormatInt(m.Chips, 10)
	}
	return &p
}

func NewListPlayer(presences []runtime.Presence) ArrPbPlayer {
	listPlayer := make([]*pb.Player, 0, len(presences))
	for _, presence := range presences {
		p := NewPlayer(presence)
		listPlayer = append(listPlayer, p)
	}
	return listPlayer
}
