FROM golang:1.20.1-alpine as builder

WORKDIR /go/src/github.com/grafana/plugin-validator
ADD . /go/src/github.com/grafana/plugin-validator

RUN apk update && \
    apk add git curl ca-certificates

RUN git clone https://github.com/magefile/mage --depth 1 && \
    cd mage && \
    go run bootstrap.go && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.51.1

RUN mage -v && \
    ls -al bin

FROM alpine:3.15
RUN apk update && \
    apk upgrade --available && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/*
RUN wget -O - -q https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s v2.14.0
WORKDIR /app
COPY --from=builder /go/src/github.com/grafana/plugin-validator/bin bin
COPY --from=builder /go/src/github.com/grafana/plugin-validator/config config
ENTRYPOINT ["/app/bin/linux_amd64/plugincheck2"]
