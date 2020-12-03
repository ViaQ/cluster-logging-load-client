PHONY: all build test clean build-image push-image
.DEFAULT_GOAL := all

IMAGE_PREFIX ?= ctovena
IMAGE_TAG := 0.1
ES_CONTAINER_NAME=elasticsearch
ES_IMAGE_TAG=elasticsearch:6.8.12

all: test build-image

build:
	go build -o logger -v main.go es_bulk_indexer.go

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

run-es:
	docker run -d --name $(ES_CONTAINER_NAME) \
		-p 9200:9200 -p 9300:9300 \
		-e "discovery.type=single-node" \
		$(ES_IMAGE_TAG)
