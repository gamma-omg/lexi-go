.PHONY: init
init:
	sh scripts/install-tools.sh

.PHONY: proto
proto:
	sh scripts/proto.sh api

.PHONY: image
image:
	make -C internal/services/gateway image
	make -C internal/services/words image

.PHONY: deploy
deploy:
	kubectl apply -f deploy/gateway/
	kubectl apply -f deploy/words/