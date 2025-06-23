FROM golang:1.24-alpine AS builder

WORKDIR /rss_exporter
COPY . /rss_exporter

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

RUN go mod tidy && go test ./... && go build -trimpath -ldflags="-w -s" -o rss_exporter

FROM registry-maas.maas.services.sabio.co.uk/docker/busybox-glibc:1.0.0

WORKDIR /
COPY --from=builder /rss_exporter/rss_exporter .

EXPOSE 9091/tcp

ENTRYPOINT ["/rss_exporter"]
CMD ["-config.file=/config/config.yml"]
