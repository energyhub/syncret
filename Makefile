.PHONY: clean install test dist

clean:
	rm -rf dist

test:
	go test -coverprofile=coverage.out -v ./...

lint:
	golint $(shell go list ./... | grep -v /vendor)

vet:
	go vet ./...

dist:
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o dist/syncret-darwin-amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/syncret-linux-amd64
