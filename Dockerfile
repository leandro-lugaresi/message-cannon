FROM alpine

RUN apk add --no-cache ca-certificates openssl bash
COPY message-cannon /

ENTRYPOINT ["/message-cannon"]