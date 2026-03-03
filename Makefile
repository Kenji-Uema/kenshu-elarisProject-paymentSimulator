build: generate
	go build .

test:
	GOCACHE=$(PWD)/.gocache GOMODCACHE=$(PWD)/.gomodcache go test ./...

generate:
	npx buf generate

docker-build:
	 docker build --build-arg SERVICE_NAME=payment-simulator --build-arg VERSION=latest -t payment-simulator:latest .
