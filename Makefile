.PHONY: all
all: jr

jr:
	go build ./...

.PHONY: install
install: jr
	sudo cp jr /usr/local/bin/

