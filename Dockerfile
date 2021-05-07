FROM golang:1.16.4-alpine3.13 as builder
COPY . /go/src/api
WORKDIR /go/src/api
RUN go build -v

FROM alpine:3.13.5
COPY --from=builder /go/src/api/api /usr/local/bin/server
ENTRYPOINT ["/usr/local/bin/server"]