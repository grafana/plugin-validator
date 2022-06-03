FROM golang:1.17 as builder

ENV DEBIAN_FRONTEND=noninteractive
WORKDIR /go/src/github.com/grafana/plugin-validator
ADD . /go/src/github.com/grafana/plugin-validator

RUN apt-get update && \
    apt-get install ca-certificates -y && \
    apt-get upgrade -y && \
    git clone https://github.com/magefile/mage && \
    cd mage && \
    go run bootstrap.go && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.43.0 && \
    cd /go/src/github.com/grafana/plugin-validator && \
    mage -v && \
    ls -al bin
ENV DEBIAN_FRONTEND=newt

FROM alpine:3.15
RUN apk update && \
    apk upgrade --available && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/*
WORKDIR /app
COPY --from=builder /go/src/github.com/grafana/plugin-validator/bin bin
COPY --from=builder /go/src/github.com/grafana/plugin-validator/config config
ENTRYPOINT ["/app/bin/linux_amd64/plugincheck2"]
