API = $(realpath ./api)
TAG = goca

all: pull-docker build

run:
	@docker run --rm --name $(TAG) -p 8080:8080 $(TAG)/server

.PHONY: build
build: build-swagger swagger build-app

build-swagger:
	@docker build -f build/swagger.Dockerfile -t $(TAG)/swagger:latest .

build-app:
	@docker build -f build/main.Dockerfile -t $(TAG)/server:latest .

pull-docker:
	@docker pull mikefarah/yq
	@docker pull swaggerapi/swagger-codegen-cli
	@docker pull golang:alpine

swagger:
	@docker run --rm -v $(API):/workdir mikefarah/yq e -j openapi.yaml > $(API)/openapi.json
	@touch $(API)/openapi.html && docker run --rm -v $(API):/local goca/swagger /local/openapi.json
