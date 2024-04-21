package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/ciaolink-game-platform/cgp-common/proto"
	"github.com/heroiclabs/nakama-common/api"
	"github.com/jackc/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Account struct {
	api.Account
	LastOnlineTimeUnix int64
	Sid                int64
}

type ListProfile []*pb.Profile

func (l ListProfile) ToMap() map[string]*pb.Profile {
	mapProfile := make(map[string]*pb.Profile)
	for _, p := range l {
		mapProfile[p.GetUserId()] = p
	}
	return mapProfile
}

func GetProfileUsers(ctx context.Context, db *sql.DB, userIDs ...string) (ListProfile, error) {
	// accounts, err := nk.AccountsGetId(ctx, userIDs)
	accounts, err := GetAccounts(ctx, db, userIDs...)
	if err != nil {
		return nil, err
	}
	listProfile := make(ListProfile, 0, len(accounts))
	for _, acc := range accounts {
		u := acc.GetUser()
		var metadata map[string]interface{}
		json.Unmarshal([]byte(u.GetMetadata()), &metadata)
		profile := pb.Profile{
			UserId:      u.GetId(),
			UserName:    u.GetUsername(),
			DisplayName: u.GetDisplayName(),
			Status:      InterfaceToString(metadata["status"]),
			AvatarId:    InterfaceToString(metadata["avatar_id"]),
			VipLevel:    ToInt64(metadata["vip_level"], 0),
			UserSid:     acc.Sid,
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

func ParseProfile(user *api.Account) *pb.Profile {
	u := user.User
	var metadata map[string]interface{}
	json.Unmarshal([]byte(u.GetMetadata()), &metadata)
	profile := &pb.Profile{
		UserId:      u.GetId(),
		UserName:    u.GetUsername(),
		DisplayName: u.GetDisplayName(),
		Status:      InterfaceToString(metadata["status"]),
		AvatarId:    InterfaceToString(metadata["avatar_id"]),
		VipLevel:    ToInt64(metadata["vip_level"], 0),
		// UserSid:     user.Sid,
	}
	playingMatchJson := InterfaceToString(metadata["playing_in_match"])

	if playingMatchJson == "" {
		profile.PlayingMatch = nil
	} else {
		profile.PlayingMatch = &pb.PlayingMatch{}
		json.Unmarshal([]byte(playingMatchJson), profile.PlayingMatch)
	}
	if user.GetWallet() != "" {
		wallet, err := ParseWallet(user.GetWallet())
		if err == nil {
			profile.AccountChip = wallet.Chips
		}
	}
	return profile
}

func GetProfileUser(ctx context.Context, db *sql.DB, userID string) (*pb.Profile, error) {
	listProfile, err := GetProfileUsers(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if len(listProfile) == 0 {
		return nil, errors.New("profile not found")
	}
	return listProfile[0], nil
}

func GetAccounts(ctx context.Context, db *sql.DB, userIds ...string) ([]*Account, error) {
	if len(userIds) == 0 {
		return make([]*Account, 0), nil
	}
	query := `
SELECT u.id, u.username, u.display_name, u.avatar_url, u.lang_tag, u.location, u.timezone, u.metadata, u.wallet,
	u.email, u.apple_id, u.facebook_id, u.facebook_instant_game_id, u.google_id, u.gamecenter_id, u.steam_id, u.custom_id, u.edge_count,
	u.create_time, u.update_time, u.verify_time, u.disable_time, array(select ud.id from user_device ud where u.id = ud.user_id),
	u.sid
FROM users u
WHERE u.id::text IN (` + "'" + strings.Join(userIds, "','") + "'" + `)`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	ml := make([]*Account, 0)
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			continue
		}
		ml = append(ml, account)
	}
	return ml, nil
}

type DBScan interface {
	Scan(dest ...any) error
}

func scanAccount(row DBScan) (*Account, error) {
	var userId sql.NullString
	var displayName sql.NullString
	var username sql.NullString
	var avatarURL sql.NullString
	var langTag sql.NullString
	var location sql.NullString
	var timezone sql.NullString
	var metadata sql.NullString
	var wallet sql.NullString
	var email sql.NullString
	var apple sql.NullString
	var facebook sql.NullString
	var facebookInstantGame sql.NullString
	var google sql.NullString
	var gamecenter sql.NullString
	var steam sql.NullString
	var customID sql.NullString
	var edgeCount int
	var createTime pgtype.Timestamptz
	var updateTime pgtype.Timestamptz
	var verifyTime pgtype.Timestamptz
	var disableTime pgtype.Timestamptz
	var deviceIDs pgtype.VarcharArray
	// var lastOnlineTime pgtype.Timestamptz
	var sID sql.NullInt64
	err := row.Scan(&userId, &username, &displayName, &avatarURL,
		&langTag, &location, &timezone,
		&metadata, &wallet, &email,
		&apple, &facebook, &facebookInstantGame,
		&google, &gamecenter, &steam,
		&customID, &edgeCount, &createTime,
		&updateTime, &verifyTime, &disableTime,
		&deviceIDs, &sID)
	if err != nil {
		return nil, err
	}
	devices := make([]*api.AccountDevice, 0, len(deviceIDs.Elements))
	for _, deviceID := range deviceIDs.Elements {
		devices = append(devices, &api.AccountDevice{Id: deviceID.String})
	}

	var verifyTimestamp *timestamppb.Timestamp
	if verifyTime.Status == pgtype.Present && verifyTime.Time.Unix() != 0 {
		verifyTimestamp = &timestamppb.Timestamp{Seconds: verifyTime.Time.Unix()}
	}
	var disableTimestamp *timestamppb.Timestamp
	if disableTime.Status == pgtype.Present && disableTime.Time.Unix() != 0 {
		disableTimestamp = &timestamppb.Timestamp{Seconds: disableTime.Time.Unix()}
	}
	account := &Account{
		Account: api.Account{
			User: &api.User{
				Id:                    userId.String,
				Username:              username.String,
				DisplayName:           displayName.String,
				AvatarUrl:             avatarURL.String,
				LangTag:               langTag.String,
				Location:              location.String,
				Timezone:              timezone.String,
				Metadata:              metadata.String,
				AppleId:               apple.String,
				FacebookId:            facebook.String,
				FacebookInstantGameId: facebookInstantGame.String,
				GoogleId:              google.String,
				GamecenterId:          gamecenter.String,
				SteamId:               steam.String,
				EdgeCount:             int32(edgeCount),
				CreateTime:            &timestamppb.Timestamp{Seconds: createTime.Time.Unix()},
				UpdateTime:            &timestamppb.Timestamp{Seconds: updateTime.Time.Unix()},
			},
			Wallet:      wallet.String,
			Email:       email.String,
			Devices:     devices,
			CustomId:    customID.String,
			VerifyTime:  verifyTimestamp,
			DisableTime: disableTimestamp,
		},
		Sid: sID.Int64,
	}
	return account, nil
}
