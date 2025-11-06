ARG GOLANGCI_LINT_VERSION=v2.5.0
ARG GOSEC_VERSION=v2.22.8
ARG SEMGREP_VERSION=1.84.1

FROM golang:1.25-alpine3.21@sha256:8f507c423fa5cb5b96dc8e31aec8dba1549f3df03eb42865f2a2469ef3fded7f AS builder

ARG GOLANGCI_LINT_VERSION
ARG GOSEC_VERSION
ARG SEMGREP_VERSION

WORKDIR /go/src/github.com/grafana/plugin-validator
ADD . /go/src/github.com/grafana/plugin-validator

RUN apk add --no-cache git ca-certificates curl python3 python3-dev py3-pip clamav
RUN update-ca-certificates
RUN freshclam

RUN git clone https://github.com/magefile/mage --depth 1 && \
    cd mage && \
    go run bootstrap.go && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION} && \
    curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b /usr/local/bin ${GOSEC_VERSION} && \
    python3 -m pip install semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages

RUN mage -v build:lint

RUN mage -v build:ci

FROM alpine:3.22@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412

ARG GOSEC_VERSION
ARG SEMGREP_VERSION

RUN apk add --no-cache git ca-certificates curl wget python3 python3-dev py3-pip alpine-sdk clamav
RUN update-ca-certificates
RUN freshclam

RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b /usr/local/bin ${GOSEC_VERSION}

# install semgrep
RUN python3 -m pip install semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages --no-cache-dir


WORKDIR /app
COPY --from=builder /go/src/github.com/grafana/plugin-validator/bin bin
COPY --from=builder /go/src/github.com/grafana/plugin-validator/config config
ENTRYPOINT ["/app/bin/linux_amd64/plugincheck2"]
