# builder image
FROM golang:1.11-alpine3.8 as builder

ENV CGO_ENABLED 0
RUN apk --no-cache add git
RUN go get github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/linki/chaoskube
COPY . .
RUN dep ensure -vendor-only
RUN go test -v ./...
ENV GOARCH amd64
RUN go build -o /bin/chaoskube -v \
  -ldflags "-X main.version=$(git describe --tags --always --dirty) -w -s"

# final image
FROM gcr.io/distroless/base
MAINTAINER Linki <linki+docker.com@posteo.de>

COPY --from=builder /bin/chaoskube /chaoskube

USER 65534
ENTRYPOINT ["/chaoskube"]
