package entity

import (
	"strconv"

	"github.com/ciaolink-game-platform/cgp-common/bot"
	pb "github.com/ciaolink-game-platform/cgp-common/proto"

	"github.com/heroiclabs/nakama-common/runtime"
)

type ArrPbPlayer []*pb.Player

func NewPlayer(presence runtime.Presence) *pb.Player {
	p := pb.Player{
		Id:       presence.GetUserId(),
		UserName: presence.GetUsername(),
	}
	if m, ok := presence.(MyPrecense); ok {
		p.AvatarId = m.AvatarId
		p.VipLevel = m.VipLevel
		p.Wallet = strconv.FormatInt(m.Chips, 10)
		p.Sid = m.Sid
	}
	if m, ok := presence.(*bot.BotPresence); ok {
		account := &m.Account
		profile := ParseProfile(account)
		// p.VipLevel profile.VipLevel
		p.Wallet = strconv.FormatInt(profile.AccountChip, 10)
		p.Sid = m.Sid
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
