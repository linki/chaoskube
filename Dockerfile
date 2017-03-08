FROM alpine:3.5

COPY build/chaoskube /chaoskube

ENTRYPOINT ["/chaoskube"]
