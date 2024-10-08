FROM golang:1.23.2-alpine3.20 AS builder
WORKDIR /go/src/int-email
COPY . .
RUN \
    apk add protoc protobuf-dev make git && \
    make build

FROM alpine:3.20
RUN apk --no-cache add ca-certificates \
    && update-ca-certificates
COPY --from=builder /go/src/int-email/int-email /bin/int-email
ENTRYPOINT ["/bin/int-email"]
