FROM golang:1.20.1-alpine as builder

WORKDIR /go/src/github.com/grafana/plugin-validator
ADD . /go/src/github.com/grafana/plugin-validator

RUN apk add --no-cache git ca-certificates curl && \
    update-ca-certificates

RUN git clone https://github.com/magefile/mage --depth 1 && \
    cd mage && \
    go run bootstrap.go && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.51.1

RUN cd /go/src/github.com/grafana/plugin-validator && \
    mage -v build:ci && \
    ls -al bin


FROM alpine:3.17
RUN apk add --no-cache git ca-certificates curl wget python3 python3-dev py3-pip alpine-sdk && \
    update-ca-certificates

# install gosec
RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b /usr/local/bin v2.14.0

# install osv-scanner
RUN wget https://github.com/google/osv-scanner/releases/download/v1.2.0/osv-scanner_1.2.0_linux_amd64
RUN mv osv-scanner_1.2.0_linux_amd64 /usr/local/bin/osv-scanner
RUN chmod +x /usr/local/bin/osv-scanner

# install semgrep
RUN python3 -m pip install semgrep --ignore-installed

WORKDIR /app
COPY --from=builder /go/src/github.com/grafana/plugin-validator/bin bin
COPY --from=builder /go/src/github.com/grafana/plugin-validator/config config
ENTRYPOINT ["/app/bin/linux_amd64/plugincheck2"]
