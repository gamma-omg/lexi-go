.PHONY: image
image:
	@eval "$$(minikube docker-env)" && \
	docker build -f Dockerfile -t lexi-go/auth:${TAG} ../.. && \
	docker build -f db/Dockerfile -t lexi-go/auth-db-migrations:${TAG} ./db
	eval "$$(minikube docker-env -u)"

.PHONY: migrate
migrate: image
	@kubectl apply -f ../../../deploy/lexigo/auth/migrate.yml -n lexigo

.PHONY: deploy
deploy: image
	@helm upgrade -n lexigo --install lexigo-auth ../../../deploy/lexigo/auth \
		--set container.tag=${TAG} \
		--set migrator.tag=${TAG} \
		-f ../../../deploy/lexigo/auth/values.${TAG}.yaml 