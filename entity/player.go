package entity

import (
	"context"
	"strconv"

	"github.com/nk-nigeria/cgp-common/bot"
	pb "github.com/nk-nigeria/cgp-common/proto"

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

func (arr ArrPbPlayer) ReadProfile(ctx context.Context, nk runtime.NakamaModule, logger runtime.Logger) error {
	listUserId := make([]string, 0, len(arr))
	for _, player := range arr {
		listUserId = append(listUserId, player.Id)
	}
	profiles, err := GetProfileUsers(ctx, nk, listUserId...)
	if err != nil {
		return err
	}
	mapWallet := make(map[string]*pb.SimpleProfile, 0)
	for _, w := range profiles {
		mapWallet[w.UserId] = w
	}
	for i, player := range arr {
		profile := mapWallet[player.Id]
		if profile == nil {
			continue
		}
		player.Wallet = strconv.FormatInt(profile.GetAccountChip(), 10)
		player.AvatarId = profile.GetAvatarId()
		player.VipLevel = profile.GetVipLevel()
		arr[i] = player
	}
	return nil
}
