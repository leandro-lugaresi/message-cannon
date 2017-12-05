FROM alpine

ADD assets/ca-certificates.crt /etc/ssl/certs/
COPY message-cannon /

ENTRYPOINT ["/message-cannon"]