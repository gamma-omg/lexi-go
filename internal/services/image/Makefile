.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/image:${TAG} ../.. && \
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@kubectl apply -f ../../../deploy/lexigo/image/ -n lexigo
	@kubectl rollout restart deployment/lexigo-image -n lexigo