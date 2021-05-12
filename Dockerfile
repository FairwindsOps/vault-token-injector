FROM golang:1.15 AS build-env
WORKDIR /go/src/github.com/fairwindsops/vault-token-injector/

ARG version=dev
ARG commit=none

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN VERSION=$version COMMIT=$commit make build-linux

FROM hashicorp/vault:1.7.1

USER nobody
COPY --from=build-env /go/src/github.com/fairwindsops/vault-token-injector/vault-token-injector /

WORKDIR /
ENTRYPOINT ["/vault-token-injector"]
