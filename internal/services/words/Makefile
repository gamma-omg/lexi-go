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
	@helm upgrade -n lexigo --install lexigo-words ../../../deploy/lexigo/words/ \
		-f ../../../deploy/lexigo/words/values.${TAG}.yaml \
	    --set container.tag=${TAG} \
		--set migrator.tag=${TAG} \

