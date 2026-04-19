# Proto generation: outputs to gen/proto/ (module=lunar-tear/server).
# All proto files have go_package. Only protos used by the server are generated here
# (generating all would put them in one package and cause name clashes).
PROTO_USED = proto/banner.proto proto/battle.proto proto/bighunt.proto proto/cageornament.proto proto/character.proto proto/characterboard.proto proto/characterviewer.proto proto/companion.proto proto/config.proto proto/consumableitem.proto proto/contentsstory.proto proto/costume.proto proto/data.proto proto/deck.proto proto/dokan.proto proto/explore.proto proto/friend.proto proto/gacha.proto proto/gameplay.proto proto/gift.proto proto/gimmick.proto proto/labyrinth.proto proto/loginbonus.proto proto/material.proto proto/mission.proto proto/movie.proto proto/navicutin.proto proto/omikuji.proto proto/notification.proto proto/parts.proto proto/portalcage.proto proto/pvp.proto proto/quest.proto proto/reward.proto proto/shop.proto proto/sidestoryquest.proto proto/tutorial.proto proto/user.proto proto/weapon.proto

proto:
	protoc -I . $(PROTO_USED) --go_out=. --go_opt=module=lunar-tear/server --go-grpc_out=. --go-grpc_opt=module=lunar-tear/server
	@echo "Generated in gen/proto/"

build:
	go build -o lunar-tear ./cmd/lunar-tear

build-import:
	go build -o import-snapshot ./cmd/import-snapshot

migrate:
	mkdir -p db
	goose -dir migrations sqlite3 db/game.db up

import:
ifndef SNAPSHOT
	$(error SNAPSHOT is required, e.g. make import SNAPSHOT=snapshots/scene_1.json UUID=...)
endif
ifndef UUID
	$(error UUID is required, e.g. make import SNAPSHOT=snapshots/scene_1.json UUID=...)
endif
	go run ./cmd/import-snapshot --snapshot $(SNAPSHOT) --uuid $(UUID)

.PHONY: proto build build-import migrate import
