module github.com/ciaolink-game-platform/blackjack-module

replace github.com/ciaolink-game-platform/cgp-common => ./cgp-common

go 1.19

require (
	github.com/emirpasic/gods v1.12.0
	github.com/heroiclabs/nakama-common v1.22.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	google.golang.org/genproto v0.0.0-20211118181313-81c1377c94b1 // indirect
)

require (
	github.com/bwmarrin/snowflake v0.3.0
	github.com/ciaolink-game-platform/cgp-common v0.0.0-20230428104858-728dac40ec4d
	github.com/qmuntal/stateless v1.5.3
	go.uber.org/zap v1.19.1
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
)
