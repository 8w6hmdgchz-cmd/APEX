.PHONY: build test lint docker clean run

APP_NAME := agent-os
VERSION := 2.0.0
GO := go

build:
	$(GO) build -o bin/$(APP_NAME) ./cmd/agent-os/

test:
	$(GO) test ./... -v -count=1

lint:
	$(GO) vet ./...

run: build
	./bin/$(APP_NAME)

clean:
	rm -rf bin/

docker:
	docker build -t $(APP_NAME):$(VERSION) -f deployments/docker/Dockerfile .

docker-run:
	docker run --rm -p 8080:8080 $(APP_NAME):$(VERSION)

k8s-deploy:
	kubectl apply -f deployments/k8s/deployment.yaml

k8s-delete:
	kubectl delete -f deployments/k8s/deployment.yaml

.DEFAULT_GOAL := build
