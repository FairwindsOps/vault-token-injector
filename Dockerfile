FROM hashicorp/vault:1.7.1

USER nobody
COPY vault-token-injector /

WORKDIR /
ENTRYPOINT ["/vault-token-injector"]
