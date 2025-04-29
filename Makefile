TPARSE := $(shell tparse --version 2>/dev/null)

check:
ifdef TPARSE
	GODEBUG=randseednop=0 go test ./... -race -cover -json | tparse -all
else
	GODEBUG=randseednop=0 go test ./... -race -cover
endif

test: check

build: 
	go build -o bin/chaoskube -v
