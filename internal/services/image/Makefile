.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/image:${TAG} ../.. && \
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@helm upgrade -n lexigo --install lexigo-image ../../../deploy/lexigo/image \
		-f ../../../deploy/lexigo/image/values.${TAG}.yaml 
		--set container.tag=${TAG} \