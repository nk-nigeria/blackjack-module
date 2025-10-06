PROJECT_NAME=github.com/nk-nigeria/blackjack-module
APP_NAME=blackjack_plugin.so
APP_PATH=$(PWD)
NAKAMA_VER=3.27.0

GOPRIVATE="github.com/nk-nigeria/*"

update-common:
	go get github.com/nk-nigeria/cgp-common
update-common-stg:
	go get github.com/nk-nigeria/cgp-common@staging

cpdev:
	scp ./bin/${APP_NAME} nakama:/root/cgp-server-dev/dist/data/modules/

build:
	go mod tidy
	go mod vendor
	docker run --rm -w "/app" -v "${APP_PATH}:/app" "heroiclabs/nakama-pluginbuilder:${NAKAMA_VER}" build -buildvcs=false --trimpath --buildmode=plugin -o ./bin/${APP_NAME} . && cp ./bin/${APP_NAME} ../bin/
build_dev: build cpdev
syncstg:
	rsync -aurv --delete ./bin/${APP_NAME} root@cgpdev:/root/cgp-server/dist/data/modules/bin/
	ssh root@cgpdev 'cd /root/cgp-server && docker restart nakama'

syncdev:
	rsync -aurv --delete ./bin/${APP_NAME} root@cgpdev:/root/cgp-server-dev/dist/data/modules/bin/
	ssh root@cgpdev 'cd /root/cgp-server-dev && docker restart nakama_dev'

bsync: build sync

dev: update-common-dev build

stg: update-common-stg build

local: 
	./sync_pkg_3.11.sh
	go mod tidy
	go mod vendor
	go build --trimpath --mod=vendor --buildmode=plugin -o ./bin/${APP_NAME}

proto:
	protoc -I ./ --go_out=$(pwd)/proto  ./proto/blackjack_api.proto
