DOCKER := docker

.PHONY: all build

all: build

build:
	$(DOCKER) build -t navikt/webhookproxy .
