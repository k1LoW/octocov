FROM node:24 AS bundler

WORKDIR /workdir/
COPY . /workdir/

RUN make page_bundle

FROM golang:1.26.8 AS builder

WORKDIR /workdir/
COPY . /workdir/
COPY --from=bundler /workdir/internal/page/bundle.js /workdir/internal/page/ssr.css /workdir/internal/page/

RUN apt-get update

RUN update-ca-certificates

RUN make go_build

FROM debian:trixie-slim

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /workdir/octocov ./usr/bin

ENTRYPOINT ["/entrypoint.sh"]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
