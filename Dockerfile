FROM golang:1.26.4-bookworm AS builder
ARG BUILD_VERSION
ENV BUILD_VERSION=${BUILD_VERSION}
ADD . /ffsigner
WORKDIR /ffsigner
RUN make

FROM debian:bookworm-slim
WORKDIR /ffsigner
RUN apt update -y \
    && apt install -y curl jq \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /ffsigner/firefly-signer /usr/bin/ffsigner
USER 1001

ENTRYPOINT [ "/usr/bin/ffsigner" ]
