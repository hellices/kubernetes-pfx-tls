.PHONY: build
build:
	go build -o bin/pfx-converter main.go

.PHONY: test
test:
	go test -v ./...

.PHONY: docker-build
docker-build:
	docker build -t ghcr.io/hellices/kubernetes-pfx-tls:latest .

.PHONY: docker-push
docker-push:
	docker push ghcr.io/hellices/kubernetes-pfx-tls:latest

.PHONY: deploy
deploy:
	kubectl apply -f deploy/

.PHONY: undeploy
undeploy:
	kubectl delete -f deploy/

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin/
