PROJECT_NAME=github.com/nk-nigeria/blackjack-module
APP_NAME=blackjack_plugin.so
APP_PATH=$(PWD)
NAKAMA_VER=3.27.0

GOPRIVATE="github.com/nk-nigeria/*"

update-common:
	GOPROXY=direct go get github.com/nk-nigeria/cgp-common@main
update-common-stg:
	GOPROXY=direct go get github.com/nk-nigeria/cgp-common@staging

cpdev:
	scp ./bin/${APP_NAME} nakama:/root/nk-nigeria/dist/data/modules/

build:
	go mod tidy
	go mod vendor
	docker run --rm -w "/app" -v "${APP_PATH}:/app" "heroiclabs/nakama-pluginbuilder:${NAKAMA_VER}" build -buildvcs=false --trimpath --buildmode=plugin -o ./bin/${APP_NAME} . && cp ./bin/${APP_NAME} ../bin/

build_dev: build cpdev

syncstg:
	rsync -aurv --delete ./bin/${APP_NAME} root@cgpdev:/root/nk-nigeria-stg/dist/data/modules/bin/
	ssh root@cgpdev 'cd /root/nk-nigeria-stg && docker restart nakama'

syncdev:
	rsync -aurv --delete ./bin/${APP_NAME} root@cgpdev:/root/nk-nigeria/dist/data/modules/bin/
	ssh root@cgpdev 'cd /root/nk-nigeria && docker restart nakama'

bsync: build sync

dev: update-common build

stg: update-common-stg build

local: 
	./sync_pkg_3.11.sh
	go mod tidy
	go mod vendor
	go build --trimpath --mod=vendor --buildmode=plugin -o ./bin/${APP_NAME}
