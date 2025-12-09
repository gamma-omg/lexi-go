.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/words:${TAG} ../.. && \
	docker build -f db/Dockerfile -t lexi-go/words-db-migrations:${TAG} ./db
	eval "$$(minikube docker-env -u)"

.PHONY: migrate
migrate: image
	@kubectl apply -f ../../../deploy/lexigo/words/migrate.yml -n lexigo

.PHONY: deploy
deploy: image
	@kubectl apply -f ../../../deploy/lexigo/words/ -n lexigo
	@kubectl rollout restart deployment/lexigo-words -n lexigo