GO = /usr/local/go/bin/go
GOPATH = $(PWD)/.go
export GO GOPATH

all: honk

honk: .preflightcheck schema.sql *.go go.mod
	$(GO) build -mod=`ls -d vendor 2> /dev/null` -o honk

.preflightcheck: preflight.sh
	@sh ./preflight.sh

clean:
	rm -f honk

test:
	$(GO) test
