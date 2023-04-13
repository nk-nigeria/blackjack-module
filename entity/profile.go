package entity

import (
	"context"
	"encoding/json"
	"errors"

	pb "github.com/ciaolink-game-platform/cgp-common/proto"
	"github.com/heroiclabs/nakama-common/runtime"
)

type ListProfile []*pb.SimpleProfile

func (l ListProfile) ToMap() map[string]*pb.SimpleProfile {
	mapProfile := make(map[string]*pb.SimpleProfile)
	for _, p := range l {
		mapProfile[p.GetUserId()] = p
	}
	return mapProfile
}

func GetProfileUsers(ctx context.Context, nk runtime.NakamaModule, userIDs ...string) (ListProfile, error) {
	accounts, err := nk.AccountsGetId(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	listProfile := make(ListProfile, 0, len(accounts))
	for _, acc := range accounts {
		u := acc.GetUser()
		var metadata map[string]interface{}
		json.Unmarshal([]byte(u.GetMetadata()), &metadata)
		profile := pb.SimpleProfile{
			UserId:      u.GetId(),
			UserName:    u.GetUsername(),
			DisplayName: u.GetDisplayName(),
			Status:      InterfaceToString(metadata["status"]),
			AvatarId:    InterfaceToString(metadata["avatar_id"]),
			VipLevel:    ToInt64(metadata["vip_level"], 0),
		}
		playingMatchJson := InterfaceToString(metadata["playing_in_match"])

		if playingMatchJson == "" {
			profile.PlayingMatch = nil
		} else {
			profile.PlayingMatch = &pb.PlayingMatch{}
			json.Unmarshal([]byte(playingMatchJson), profile.PlayingMatch)
		}
		if acc.GetWallet() != "" {
			wallet, err := ParseWallet(acc.GetWallet())
			if err == nil {
				profile.AccountChip = wallet.Chips
			}
		}
		listProfile = append(listProfile, &profile)
	}
	return listProfile, nil
}

func GetProfileUser(ctx context.Context, nk runtime.NakamaModule, userID string) (*pb.SimpleProfile, error) {
	listProfile, err := GetProfileUsers(ctx, nk, userID)
	if err != nil {
		return nil, err
	}
	if len(listProfile) == 0 {
		return nil, errors.New("profile not found")
	}
	return listProfile[0], nil
}
