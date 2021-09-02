FROM golang:1.17.0-alpine3.14 as builder
COPY . /go/src/api
WORKDIR /go/src/api
RUN go build -v

FROM alpine:3.14.2
COPY --from=builder /go/src/api/api /usr/local/bin/server
ENTRYPOINT ["/usr/local/bin/server"]