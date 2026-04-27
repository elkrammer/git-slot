.PHONY: build

BINARY := git-slot

build:
	go build -o $(BINARY)
