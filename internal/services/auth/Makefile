CRATE_DIR := ../../../deploy/lexigo/auth
SERVICES_DIR := ../..

.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/auth:${TAG} ${SERVICES_DIR} && \
	docker build -f db/Dockerfile -t lexi-go/auth-db-migrations:${TAG} ./db
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@helm upgrade -n lexigo --install lexigo-auth $(CRATE_DIR) \
		-f $(CRATE_DIR)/values.${TAG}.yaml \
		--set container.tag=${TAG} \
		--set migrator.tag=${TAG}