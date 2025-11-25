.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/words:${TAG} ../.. && \
	eval "$$(minikube docker-env -u)"

.PHONY: deploy
deploy: image
	@kubectl apply -f ../../../deploy/lexigo/words/config.yml -n lexigo
	@kubectl apply -f ../../../deploy/lexigo/words/postgres.yml -n lexigo
	@kubectl apply -f ../../../deploy/lexigo/words/postgres-service.yml -n lexigo
	@kubectl apply -f ../../../deploy/lexigo/words/ -n lexigo
	@kubectl rollout restart deployment/lexigo-words -n lexigo