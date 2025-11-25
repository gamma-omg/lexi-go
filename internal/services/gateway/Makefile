.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/gateway:${TAG} ../.. && \
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@kubectl apply -f ../../../deploy/lexigo/gateway/ -n lexigo
	@kubectl rollout restart deployment/lexigo-gateway -n lexigo