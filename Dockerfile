FROM golang:1.9-alpine

COPY . /go/src/github.com/linki/chaoskube
RUN go install -v github.com/linki/chaoskube
RUN addgroup -S chaoskube && adduser -S -g chaoskube chaoskube

USER chaoskube
ENTRYPOINT ["/go/bin/chaoskube"]
