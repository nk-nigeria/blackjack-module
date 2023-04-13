package entity

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

type MyPrecense struct {
	runtime.Presence
	AvatarId string
	Chips    int64
	VipLevel int64
}

// type ListMyPrecense []MyPrecense

func NewMyPrecense(ctx context.Context, nk runtime.NakamaModule, precense runtime.Presence) MyPrecense {
	m := MyPrecense{
		Presence: precense,
	}
	profiles, err := GetProfileUsers(ctx, nk, precense.GetUserId())
	if err != nil {
		return m
	}
	if len(profiles) == 0 {
		return m
	}
	p := profiles[0]
	m.AvatarId = p.AvatarId
	m.Chips = p.AccountChip
	m.VipLevel = p.VipLevel
	return m
}

type FakePrecense struct {
	runtime.Presence
	UserId string
}

func (f *FakePrecense) GetUserId() string {
	return f.UserId
}
func (f *FakePrecense) GetSessionId() string {
	return ""
}
func (f *FakePrecense) GetNodeId() string {
	return ""
}

func (f *FakePrecense) GetHidden() bool {
	return false
}
func (f *FakePrecense) GetPersistence() bool {
	return false
}
func (f *FakePrecense) GetUsername() string {
	return ""
}
func (f *FakePrecense) GetStatus() string {
	return ""
}
func (f *FakePrecense) GetReason() runtime.PresenceReason {
	return runtime.PresenceReasonUpdate
}
