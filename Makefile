IMAGE := sun-calculator
SHA := $(shell git rev-parse --short HEAD)
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")

.PHONY: build push tag

build:
	container build \
		-t $(IMAGE):$(VERSION) \
		-t $(IMAGE):$(VERSION)-$(SHA) \
		-t $(IMAGE):latest \
		./sun-calculator-be

push:
	container push $(IMAGE):$(VERSION)
	container push $(IMAGE):$(VERSION)-$(SHA)
	container push $(IMAGE):latest

tag:
	@read -p "Version (e.g. 1.0.0): " v && git tag v$$v && echo "Tagged v$$v — run 'make build' to build"
