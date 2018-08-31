FROM golang:1.11-alpine AS builder

COPY . .

RUN mkdir -p /build && go build -v -o /build/debug-reverse-proxy

FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /

COPY --from=builder /build/ /usr/local/bin/

CMD [ "/usr/local/bin/debug-reverse-proxy" ]
