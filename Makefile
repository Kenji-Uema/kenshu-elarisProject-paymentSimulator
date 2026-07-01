IMAGE_TAG ?= 1.0.0

codbuild: generate
	GOCACHE=$(PWD)/.gocache GOMODCACHE=$(PWD)/.gomodcache go build -o $(PWD)/bin/payment-simulator ./internal

test:
	GOCACHE=$(PWD)/.gocache GOMODCACHE=$(PWD)/.gomodcache go test ./...

generate:
	buf generate

docker-build:
	 docker buildx build --build-arg SERVICE_NAME=payment-simulator --build-arg VERSION=$(IMAGE_TAG) -t payment-simulator:$(IMAGE_TAG) --load .
