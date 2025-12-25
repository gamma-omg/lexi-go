include scripts/auth-keys.mk
include scripts/tls-certs.mk

TAG ?= dev

.PHONY: up
up: init-k8s init-observability init-traefik deploy
	@echo "Initialization complete."

.PHONY: init-k8s
init-k8s:
	sh scripts/init-k8s.sh

.PHONY: init-observability
init-observability:
	sh scripts/init-observability.sh

.PHONY: init-traefik
init-traefik:
	sh scripts/init-traefik.sh

.PHONY: image
image: 
	make -C internal/services/words image TAG=${TAG}
	make -C internal/services/image image TAG=${TAG}
	make -C internal/services/auth image TAG=${TAG}

.PHONY: deploy
deploy: auth-keys tls-certs image
	@helm upgrade --install traefik traefik/traefik \
  		-n traefik \
  		--create-namespace \
  		-f deploy/lexigo/gateway/values.${TAG}.yaml
	
	@helm upgrade -n lexigo --install lexigo-common deploy/lexigo/common \
		-f deploy/lexigo/common/values.${TAG}.yaml

	make -C internal/services/auth deploy TAG=${TAG}
	make -C internal/services/words deploy TAG=${TAG}
	make -C internal/services/image deploy TAG=${TAG}
	make -C internal/services/auth deploy TAG=${TAG}