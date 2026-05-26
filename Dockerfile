ARG GOLANGCI_LINT_VERSION=v2.5.0
ARG GOSEC_VERSION=v2.22.8
ARG SEMGREP_VERSION=1.84.1

FROM golang:1.26.3-alpine3.23@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

ARG GOLANGCI_LINT_VERSION
ARG GOSEC_VERSION
ARG SEMGREP_VERSION

WORKDIR /go/src/github.com/grafana/plugin-validator
ADD . /go/src/github.com/grafana/plugin-validator

# nodejs/npm are required by the reactcompat analyzer (npx @grafana/react-detect).
# Pinned to Node 24.x to match the version used in release workflows.
RUN apk add --no-cache git ca-certificates curl python3 python3-dev py3-pip clamav nodejs=24.14.1-r0 npm
RUN update-ca-certificates
RUN freshclam

# Split into separate layers so each network operation is independently
# cacheable and surfaces its own failure (instead of being lost in a 4-in-1 step).
RUN git clone https://github.com/magefile/mage --depth 1 && \
    cd mage && \
    go run bootstrap.go

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
    sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION}

RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | \
    sh -s -- -b /usr/local/bin ${GOSEC_VERSION}

RUN python3 -m pip install semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages

RUN mage -v build:lint

RUN mage -v build:ci

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

ARG GOSEC_VERSION
ARG SEMGREP_VERSION

RUN apk add --no-cache git ca-certificates curl wget python3 python3-dev py3-pip alpine-sdk clamav nodejs=24.14.1-r0 npm
RUN update-ca-certificates
RUN freshclam

RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b /usr/local/bin ${GOSEC_VERSION}

# install semgrep
RUN python3 -m pip install semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages --no-cache-dir


WORKDIR /app
COPY --from=builder /go/src/github.com/grafana/plugin-validator/bin bin
COPY --from=builder /go/src/github.com/grafana/plugin-validator/config config
ENTRYPOINT ["/app/bin/linux_amd64/plugincheck2"]
