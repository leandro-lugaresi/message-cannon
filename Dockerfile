FROM alpine

RUN apk add --no-cache ca-certificates openssl
COPY message-cannon /

ENTRYPOINT ["/message-cannon"]