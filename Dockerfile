# builder image
FROM golang:1.9-alpine as builder

WORKDIR /go/src/github.com/linki/chaoskube
COPY . .
RUN go test -v ./chaoskube ./util
RUN go install -v -ldflags "-w -s"

# final image
FROM alpine:3.6
MAINTAINER Linki <linki+docker.com@posteo.de>

RUN addgroup -S chaoskube && adduser -S -g chaoskube chaoskube
COPY --from=builder /go/bin/chaoskube /go/bin/chaoskube

USER chaoskube
ENTRYPOINT ["/go/bin/chaoskube"]
