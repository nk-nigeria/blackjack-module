PROJECT_NAME=github.com/nakamaFramework/blackjack-module
APP_NAME=blackjack.so
APP_PATH=$(PWD)
NAKAMA_VER=3.19.0

update-submodule-dev:
	git checkout develop && git pull
	git submodule update --init
	git submodule update --remote
	cd ./cgp-common && git checkout develop && git pull && cd ..
	go get github.com/nakamaFramework/cgp-common@develop
update-submodule-stg:
	git checkout staging && git pull
	git submodule update --init
	git submodule update --remote
	cd ./cgp-common && git checkout staging && cd ..
	go get github.com/nakamaFramework/cgp-common@staging

cpdev:
	scp ./bin/${APP_NAME} nakama:/root/cgp-server-dev/dist/data/modules/

build:
	# ./sync_pkg_3.11.sh
	go mod tidy
	go mod vendor
	docker run --rm -w "/app" -v "${APP_PATH}:/app" heroiclabs/nakama-pluginbuilder:${NAKAMA_VER} build -buildvcs=false --trimpath --buildmode=plugin -o ./bin/${APP_NAME}

syncstg:
	rsync -aurv --delete ./bin/${APP_NAME} root@cgpdev:/root/cgp-server/dist/data/modules/bin/
	ssh root@cgpdev 'cd /root/cgp-server && docker restart nakama'

syncdev:
	rsync -aurv --delete ./bin/${APP_NAME} root@cgpdev:/root/cgp-server-dev/dist/data/modules/bin/
	ssh root@cgpdev 'cd /root/cgp-server-dev && docker restart nakama_dev'

bsync: build sync

dev: update-submodule-dev build

stg: update-submodule-stg build

local: 
	./sync_pkg_3.11.sh
	go mod tidy
	go mod vendor
	go build --trimpath --mod=vendor --buildmode=plugin -o ./bin/${APP_NAME}

proto:
	protoc -I ./ --go_out=$(pwd)/proto  ./proto/blackjack_api.proto
