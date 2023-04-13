#!/bin/bash

go get github.com/heroiclabs/nakama-common@v1.22.0
go get github.com/bwmarrin/snowflake@v0.3.0
go get github.com/emirpasic/gods@v1.12.0
go get github.com/qmuntal/stateless@v1.5.3
go get go.uber.org/zap@v1.19.1
go get google.golang.org/grpc@v1.42.0
go get google.golang.org/protobuf@v1.27.1
go get google.golang.org/genproto@v0.0.0-20211118181313-81c1377c94b1
go get github.com/golang/protobuf@v1.5.2
go mod tidy