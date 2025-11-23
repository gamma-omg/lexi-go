.PHONY: image
image:
	docker build -t lexi-go/gateway:latest .
	minikube image load lexi-go/gateway:latest

.PHONY: deploy
deploy: image
	kubectl apply -f ../../../deploy/lexigo/gateway/ -n lexigo
	kubectl rollout restart deployment/lexigo-gateway -n lexigo