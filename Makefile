export PATH := $(PATH):`go env GOPATH`/bin
export GO111MODULE=on
LDFLAGS := -s -w

all: env fmt build

build: cfddns

env:
	@go version

fmt:
	go fmt ./...

fmt-more:
	gofumpt -l -w .

gci:
	gci write -s standard -s default -s "prefix(github.com/aircross/cfddns/)" ./


cfddns:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags cfddns -o bin/cfddns ./
	
clean:
	rm -f ./bin/cfddns
