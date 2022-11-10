# Copyright 2020-2021 Authors of Cilium
# SPDX-License-Identifier: Apache-2.0

ARG GOLANG_IMAGE=docker.io/library/golang:1.16.8-alpine3.12@sha256:cb1a7482cb5cfc52527c5cdea5159419292360087d5249e3fe5472f3477be642
ARG ALPINE_IMAGE=docker.io/library/alpine:3.16.2@sha256:65a2763f593ae85fab3b5406dc9e80f744ec5b449f269b699b5efd37a07ad32e

FROM ${GOLANG_IMAGE} as builder

COPY install-protoplugins.sh /tmp/install-protoplugins.sh
RUN /tmp/install-protoplugins.sh

#------------------------------------------------------------------------------

FROM ${ALPINE_IMAGE}

WORKDIR /proto

RUN apk add --no-cache \
  curl \
  bash \
  make \
  && true

COPY install-glibc.sh /tmp/install-glibc.sh
RUN /tmp/install-glibc.sh

COPY install-protoc.sh /tmp/install-protoc.sh
RUN /tmp/install-protoc.sh

COPY --from=builder /proto/protoc-gen-go .
COPY --from=builder /proto/protoc-gen-go-grpc .
COPY --from=builder /proto/protoc-gen-go-json .

COPY Makefile.docker ./Makefile

WORKDIR /source

CMD ["make", "-f", "/proto/Makefile"]
