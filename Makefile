GO = /usr/local/go/bin/go
GOPATH = $(PWD)/.go
export GO GOPATH

all: honk docs

docs:
	$(MAKE) -C docs

honk: .preflightcheck schema.sql *.go go.mod
	env CGO_ENABLED=1 $(GO) build -mod=`ls -d vendor 2> /dev/null` -o honk

.preflightcheck: preflight.sh
	@sh ./preflight.sh

help:
	for m in docs/*.[13578] ; do \
	mandoc -T html -O style=mandoc.css,man=%N.%S.html $$m | sed -E 's/<a class="Lk" href="([[:alnum:]._-]*)">/<img src="\1"><br>/g' > $$m.html ; \
	done

clean:
	rm -f honk

test:
	$(GO) test

.PHONY: clean test docs
