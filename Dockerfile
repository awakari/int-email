FROM golang:1.24.5-alpine3.22 AS builder
WORKDIR /go/src/int-email
COPY . .
RUN \
    apk add protoc protobuf-dev make git && \
    make build

FROM alpine:3.22.1
RUN apk --no-cache add ca-certificates \
    && update-ca-certificates
COPY --from=builder /go/src/int-email/int-email /bin/int-email
ENTRYPOINT ["/bin/int-email"]
