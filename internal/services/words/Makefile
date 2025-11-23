.PHONY: image
image:
	docker build -t lexi-go/words:latest .
	minikube image load lexi-go/words:latest

.PHONY: deploy
deploy: image
	kubectl apply -f ../../../deploy/lexigo/words/ -n lexigo
	kubectl rollout restart deployment/lexigo-words -n lexigo