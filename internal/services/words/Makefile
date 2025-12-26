CRATE_DIR := ../../../deploy/lexigo/words
SERVICES_DIR := ../..

.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/words:${TAG} $(SERVICES_DIR) && \
	docker build -f db/Dockerfile -t lexi-go/words-db-migrations:${TAG} ./db
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@helm upgrade -n lexigo --install lexigo-words $(CRATE_DIR) \
		-f $(CRATE_DIR)/values.${TAG}.yaml \
	    --set container.tag=${TAG} \
		--set migrator.tag=${TAG} 

