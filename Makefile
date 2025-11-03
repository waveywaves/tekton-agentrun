.PHONY: help
help:
	@echo "Available targets:"
	@echo "  generate    - Generate CRD manifests and Go code"
	@echo "  manifests   - Generate CRD manifests only"
	@echo "  build       - Build controller binary"
	@echo "  test        - Run unit tests"
	@echo "  install     - Install CRDs to cluster"
	@echo "  deploy      - Deploy controller to cluster"
	@echo "  undeploy    - Remove controller from cluster"
	@echo "  docker-build - Build container image"

.PHONY: generate
generate:
	go generate ./...

.PHONY: manifests
manifests:
	controller-gen crd paths="./api/..." output:crd:dir=config/crd/bases

.PHONY: build
build:
	go build -o bin/controller ./cmd/controller

.PHONY: test
test:
	go test -v ./...

.PHONY: install
install: manifests
	kubectl apply -f config/crd/bases

.PHONY: deploy
deploy: manifests
	kubectl apply -k config/default

.PHONY: undeploy
undeploy:
	kubectl delete -k config/default

.PHONY: docker-build
docker-build:
	docker build -t agentrun-controller:latest .
