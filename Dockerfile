FROM alpine:3

USER nobody
COPY vault-token-injector /

WORKDIR /
ENTRYPOINT ["/vault-token-injector"]
