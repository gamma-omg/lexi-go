.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/gateway:${TAG} ../.. && \
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@helm upgrade -n lexigo --install lexigo-gateway ../../../deploy/lexigo/gateway \
		-f ../../../deploy/lexigo/gateway/values.${TAG}.yaml \
		--set container.tag=${TAG}