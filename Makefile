codbuild: generate
	GOCACHE=$(PWD)/.gocache GOMODCACHE=$(PWD)/.gomodcache go build -o $(PWD)/bin/payment-simulator ./internal

test:
	GOCACHE=$(PWD)/.gocache GOMODCACHE=$(PWD)/.gomodcache go test ./...

generate:
	npx buf generate

docker-build:
	 docker build --build-arg SERVICE_NAME=payment-simulator --build-arg VERSION=latest -t payment-simulator:latest .
