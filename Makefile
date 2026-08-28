BINARY := go-fz
PKG    := .

.PHONY: all build run test vet fmt fmt-check clean install

all: build

build:
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY) $(ARGS)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needs to be run on:" && gofmt -l . && exit 1)

clean:
	rm -f $(BINARY)

install:
	go install $(PKG)
