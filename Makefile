TAG ?= dev

.PHONY: up
up: install-tools init-k8s init-observability proto deploy
	@echo "Initialization complete."

.PHONY: init-k8s
init-k8s:
	sh scripts/init-k8s.sh

.PHONY: init-observability
init-observability:
	sh scripts/init-observability.sh

.PHONY: install-tools
install-tools:
	sh scripts/install-tools.sh

.PHONY: proto
proto:
	sh scripts/proto.sh api

.PHONY: image
image: proto
	make -C internal/services/gateway image TAG=${TAG}
	make -C internal/services/words image TAG=${TAG}

.PHONY: deploy
deploy:
	kubectl apply -f deploy/lexigo/common -n lexigo
	make -C internal/services/gateway deploy TAG=${TAG}
	make -C internal/services/words deploy TAG=${TAG}