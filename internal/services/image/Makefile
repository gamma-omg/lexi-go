CRATE_DIR := ../../../deploy/lexigo/image
SERVICES_DIR := ../..

.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/image:${TAG} $(SERVICES_DIR) && \
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@helm upgrade -n lexigo --install lexigo-image $(CRATE_DIR) \
		-f $(CRATE_DIR)/values.${TAG}.yaml \
		--set container.tag=${TAG}