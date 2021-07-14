FROM golang:1.14 as build-env

COPY . /app

WORKDIR /app/cmd/plugincheck

RUN go build

###

FROM gcr.io/distroless/base

COPY --from=build-env /app/cmd/plugincheck/plugincheck /

ENTRYPOINT ["/plugincheck"]
