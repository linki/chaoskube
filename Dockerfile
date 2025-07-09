# builder image
FROM golang:1.24.4-alpine3.22 AS builder

ENV CGO_ENABLED=0
RUN apk --no-cache add alpine-sdk
WORKDIR /go/src/github.com/linki/chaoskube
COPY . .
RUN GODEBUG=randseednop=0 go test -v ./...
RUN go build -o /usr/local/bin/chaoskube -v \
  -ldflags "-X main.version=$(git describe --tags --always --dirty) -w -s"
RUN /usr/local/bin/chaoskube --version

# final image
FROM alpine:3.22.0

RUN apk --no-cache add ca-certificates dumb-init tzdata
COPY --from=builder /usr/local/bin/chaoskube /usr/local/bin/chaoskube

USER 65534
ENTRYPOINT ["dumb-init", "--", "/usr/local/bin/chaoskube"]
