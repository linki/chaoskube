FROM golang:1.7.4-alpine

COPY . /go/src/github.com/linki/chaoskube
RUN go install -v github.com/linki/chaoskube

ENTRYPOINT ["/go/bin/chaoskube"]
