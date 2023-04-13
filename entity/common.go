package entity

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand"
	"strconv"
	"time"

	"github.com/bwmarrin/snowflake"
)

const (
	ModuleName = "blackjack"
)

// free game by lv
// [level]=%
// https://docs.google.com/spreadsheets/d/1OKPtCzTGe5Da-HRUKe37rS3bIGYw4F_B/edit#gid=1754766987
var feeGameByLvPercent = map[int]int{0: 7, 1: 7, 2: 6, 3: 5, 4: 5, 6: 4, 7: 4, 8: 4, 9: 4, 10: 4}

func GetFeeGameByLevel(lv int) int {
	val, exist := feeGameByLvPercent[lv]
	if exist {
		return val
	}
	return 5
}

var SnowlakeNode, _ = snowflake.NewNode(1)

type WalletAction string

const (
	WalletActionWinGameJackpot WalletAction = "win_game_jackpot"
)

func InterfaceToString(inf interface{}) string {
	if inf == nil {
		return ""
	}
	str, ok := inf.(string)
	if !ok {
		return ""
	}
	return str
}

func ToInt64(inf interface{}, def int64) int64 {
	if inf == nil {
		return def
	}
	switch v := inf.(type) {
	case int:
		return int64(inf.(int))
	case int64:
		return inf.(int64)
	case string:
		str := inf.(string)
		i, _ := strconv.ParseInt(str, 10, 64)
		return i
	case float64:
		return int64(inf.(float64))
	default:
		fmt.Printf("I don't know about type %T!\n", v)
	}
	return def
}

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Random function that return <int> in range [min, max]
func Random(min int, max int) int {
	if min < 0 {
		min = 0
	}
	if max <= min {
		max = min + 1
	}
	n := max - min
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		mrand.Seed(time.Now().UTC().UnixNano())
		return mrand.Intn(n) + min
	}
	return int(nBig.Int64()) + min
}
