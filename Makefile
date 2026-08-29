BINARY := go-fz
PKG    := ./cmd/go-fz
BINDIR := bin

.PHONY: all build run test vet fmt fmt-check clean install

all: build

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/$(BINARY) $(PKG)

run: build
	./$(BINDIR)/$(BINARY) $(ARGS)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needs to be run on:" && gofmt -l . && exit 1)

clean:
	rm -rf $(BINDIR)

install:
	go install $(PKG)
