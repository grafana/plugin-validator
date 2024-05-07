ARG GOLANGCI_LINT_VERSION=v1.58.0
ARG GOSEC_VERSION=v2.19.0
ARG SEMGREP_VERSION=1.71.0

FROM golang:1.22-alpine3.19 as builder

ARG GOLANGCI_LINT_VERSION
ARG GOSEC_VERSION
ARG SEMGREP_VERSION

WORKDIR /go/src/github.com/grafana/plugin-validator
ADD . /go/src/github.com/grafana/plugin-validator

RUN apk add --no-cache git ca-certificates curl python3 python3-dev py3-pip && \
    update-ca-certificates

RUN git clone https://github.com/magefile/mage --depth 1 && \
    cd mage && \
    go run bootstrap.go && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION} && \
    curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b /usr/local/bin ${GOSEC_VERSION} && \
    python3 -m pip install semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages

RUN cd /go/src/github.com/grafana/plugin-validator && \
    mage -v build:ci && \
    ls -al bin

FROM alpine:3.19

ARG GOSEC_VERSION
ARG SEMGREP_VERSION

RUN apk add --no-cache git ca-certificates curl wget python3 python3-dev py3-pip alpine-sdk && \
    update-ca-certificates

RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b /usr/local/bin ${GOSEC_VERSION}

# install semgrep
RUN python3 -m pip install semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages

WORKDIR /app
COPY --from=builder /go/src/github.com/grafana/plugin-validator/bin bin
COPY --from=builder /go/src/github.com/grafana/plugin-validator/config config
ENTRYPOINT ["/app/bin/linux_amd64/plugincheck2"]
