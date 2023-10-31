TPARSE := $(shell tparse --version 2>/dev/null)

check:
ifdef TPARSE
	go test ./... -race -cover -json | tparse -all
else
	go test ./... -race -cover
endif

test: check

build: 
	go build -o bin/chaoskube -v
