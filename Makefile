DOCKER  := docker
VERSION := $(shell cat ./VERSION)

.PHONY: all release build run bump-version docker-push tag

all: build
release: tag docker-push

build:
	$(DOCKER) build -t navikt/webhookproxy -t navikt/webhookproxy:$(VERSION) .

run:
	$(DOCKER) run --rm -it -p 8080:8080 navikt/webhookproxy

bump-version:
	@echo $$(($$(cat ./VERSION) + 1)) > ./VERSION

docker-push:
	$(DOCKER) push navikt/webhookproxy:latest
	$(DOCKER) push navikt/webhookproxy:$(VERSION)

tag:
	git add VERSION
	git commit -m "Bump version to $(VERSION) [skip ci]"
	git tag -a $(VERSION) -m "auto-tag from Makefile"
