.PHONY: build-manager build-agent build-all push-multi clean

PLATFORMS=linux/amd64,linux/arm64

build-manager:
	docker build -t halyard-manager:latest -f deploy/manager.Dockerfile .

build-agent:
	docker build -t halyard-agent:latest -f deploy/agent.Dockerfile .

push-manager-multi:
	docker buildx build --push --platform $(PLATFORMS) -t splinterhead27/halyard-manager:latest -f deploy/manager.Dockerfile .

push-agent-multi:
	docker buildx build --push --platform $(PLATFORMS) -t splinterhead27/halyard-agent:latest -f deploy/agent.Dockerfile .

build-all: build-manager build-agent
push-multi: push-manager-multi push-agent-multi

compile-agent:
	go build -o /dev/null ./cmd/agent

compile-manager:
	go build -o /dev/null ./cmd/manager

compile-frontend:
	cd ui && npm run build
compile-backend: compile-manager compile-agent
compile: compile-backend compile-frontend

clean:
	rm -rf ui/dist
	go clean
