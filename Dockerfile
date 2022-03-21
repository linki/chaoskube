# builder image
FROM golang:1.18.0-alpine3.14 as builder

ENV CGO_ENABLED 0
ENV GO111MODULE on
RUN apk --no-cache add git
WORKDIR /go/src/github.com/linki/chaoskube
COPY . .
RUN go test -v ./...
RUN go run main.go --version
ENV GOARCH amd64
RUN go build -o /bin/chaoskube -v \
  -ldflags "-X main.version=$(git describe --tags --always --dirty) -w -s"

# final image
FROM alpine:3.14.3
MAINTAINER Linki <linki+docker.com@posteo.de>

RUN apk --no-cache add ca-certificates dumb-init tzdata
COPY --from=builder /bin/chaoskube /bin/chaoskube

USER 65534
ENTRYPOINT ["dumb-init", "--", "/bin/chaoskube"]
