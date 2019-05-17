PHONY: all build test clean build-image push-image
.DEFAULT_GOAL := all

IMAGE_PREFIX ?= ctovena
IMAGE_TAG := 0.1

all: test build-image

build:
	go build -o logger -v main.go

test:
	go test -v ./...

clean:
	rm -f ./logger
	go clean ./...

build-image:
	docker build -t $(IMAGE_PREFIX)/logger .
	docker tag $(IMAGE_PREFIX)/logger $(IMAGE_PREFIX)/logger:$(IMAGE_TAG)

push-image:
	docker push $(IMAGE_PREFIX)/logger:$(IMAGE_TAG)
	docker push $(IMAGE_PREFIX)/logger:latest

deploy:
	kubectl apply -f deployment.yaml

delete:
	kubectl delete -f deployment.yaml
