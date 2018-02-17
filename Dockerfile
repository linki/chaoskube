# builder image
FROM golang:1.9-alpine as builder

RUN apk --no-cache add git
RUN go get github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/linki/chaoskube
COPY . .
RUN dep ensure
RUN go test -v ./...
RUN go build -o /bin/chaoskube -v \
  -ldflags "-X main.version=$(git describe --tags --always --dirty) -w -s"

# final image
FROM alpine:3.6
MAINTAINER Linki <linki+docker.com@posteo.de>

RUN apk --no-cache add ca-certificates dumb-init tzdata
RUN addgroup -S chaoskube && adduser -S -g chaoskube chaoskube
COPY --from=builder /bin/chaoskube /bin/chaoskube

USER chaoskube
ENTRYPOINT ["dumb-init", "--", "/bin/chaoskube"]
