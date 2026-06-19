ARG GOLANGCI_LINT_VERSION=v2.12.2
# SHA256 of golangci-lint-${GOLANGCI_LINT_VERSION#v}-linux-amd64.tar.gz from the upstream checksums file.
# Update whenever GOLANGCI_LINT_VERSION changes.
ARG GOLANGCI_LINT_SHA256=8df580d2670fed8fa984aac0507099af8df275e665215f5c7a2ae3943893a553
ARG GOSEC_VERSION=v2.22.8
ARG GOVULNCHECK_VERSION=v1.1.4
ARG SEMGREP_VERSION=1.84.1

FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

ARG GOLANGCI_LINT_VERSION
ARG GOLANGCI_LINT_SHA256
ARG GOSEC_VERSION
ARG GOVULNCHECK_VERSION
ARG SEMGREP_VERSION

WORKDIR /go/src/github.com/grafana/plugin-validator
ADD . /go/src/github.com/grafana/plugin-validator

# nodejs/npm are required by the reactcompat analyzer (npx @grafana/react-detect).
# Pinned to Node 24.x to match the version used in release workflows.
RUN apk add --no-cache git ca-certificates curl python3 python3-dev py3-pip clamav nodejs=24.16.0-r0 npm
RUN update-ca-certificates
RUN freshclam

# Split into separate layers so each network operation is independently
# cacheable and surfaces its own failure (instead of being lost in a 4-in-1 step).
RUN git clone https://github.com/magefile/mage --depth 1 && \
    cd mage && \
    go run bootstrap.go

# Install golangci-lint by downloading the binary directly + verifying the sha256.
# The upstream install.sh has had recurring checksum-validation bugs; downloading
# the tarball ourselves is more reliable.
RUN set -eux; \
    VER="${GOLANGCI_LINT_VERSION#v}"; \
    curl -sSfL "https://github.com/golangci/golangci-lint/releases/download/${GOLANGCI_LINT_VERSION}/golangci-lint-${VER}-linux-amd64.tar.gz" -o /tmp/golangci-lint.tar.gz; \
    echo "${GOLANGCI_LINT_SHA256}  /tmp/golangci-lint.tar.gz" | sha256sum -c; \
    tar -xzf /tmp/golangci-lint.tar.gz -C /tmp; \
    mv "/tmp/golangci-lint-${VER}-linux-amd64/golangci-lint" "$(go env GOPATH)/bin/golangci-lint"; \
    rm -rf /tmp/golangci-lint.tar.gz "/tmp/golangci-lint-${VER}-linux-amd64"

RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | \
    sh -s -- -b /usr/local/bin ${GOSEC_VERSION}

# govulncheck is distributed as a Go module — install with `go install` rather
# than a binary tarball. Pinned version is fixed via the ARG above.
RUN go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} && \
    mv "$(go env GOPATH)/bin/govulncheck" /usr/local/bin/govulncheck

# setuptools<81 provides pkg_resources, which semgrep 1.84.1 imports but
# Python 3.14 (alpine 3.24) no longer bundles. semgrep is pinned to the
# 1.84.x line on purpose: its OCaml 4 core runs in the restricted buildkit
# sandbox, whereas the OCaml 5 core in newer semgrep crashes there
# ("Failed to allocate signal stack for domain 0").
RUN python3 -m pip install "setuptools<81" semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages

RUN mage -v build:lint

RUN mage -v build:ci

FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648

ARG GOSEC_VERSION
ARG SEMGREP_VERSION

# govulncheck source mode shells out to the Go command to load packages.
RUN apk add --no-cache git go ca-certificates curl wget python3 python3-dev py3-pip alpine-sdk clamav nodejs=24.16.0-r0 npm
RUN update-ca-certificates
RUN freshclam

RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b /usr/local/bin ${GOSEC_VERSION}

# govulncheck is built in the builder stage; copy the static binary in.
COPY --from=builder /usr/local/bin/govulncheck /usr/local/bin/govulncheck

# install semgrep
RUN python3 -m pip install "setuptools<81" semgrep==${SEMGREP_VERSION} --ignore-installed --break-system-packages --no-cache-dir


WORKDIR /app
COPY --from=builder /go/src/github.com/grafana/plugin-validator/bin bin
COPY --from=builder /go/src/github.com/grafana/plugin-validator/config config
ENTRYPOINT ["/app/bin/linux_amd64/plugincheck2"]
