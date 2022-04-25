FROM golang:1.18 AS base

ENV GOPRIVATE "git.wosai-inc.com/middleware"
ENV GO111MODULE "on"
ARG GOPROXY
ARG SSH_PRIVATE_KEY
ARG GITLAB_SSH_KNOWN_HOST

WORKDIR /src
RUN sed -i.bak -e 's/deb.debian.org/mirrors.aliyun.com/g' -e 's/security.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list
RUN apt-get update && \
    apt-get install -y \
        git \
        openssh-server

# Authorize SSH Host
RUN mkdir -p /root/.ssh && \
    chmod 0700 /root/.ssh

RUN echo "$GITLAB_SSH_KNOWN_HOST" >> ~/.ssh/known_hosts

# Add the keys and set permissions
RUN echo "$SSH_PRIVATE_KEY" > /root/.ssh/id_rsa && \
    chmod 600 /root/.ssh/id_rsa

RUN git config --global url."git@git.wosai-inc.com:".insteadOf "https://git.wosai-inc.com/"

COPY go.* ./
RUN go mod download

FROM golangci/golangci-lint:v1.45.2 AS lint-base

FROM base AS lint
RUN --mount=target=. \
            --mount=from=lint-base,src=/usr/bin/golangci-lint,target=/usr/bin/golangci-lint \
            --mount=type=cache,target=/root/.cache/go-build \
            --mount=type=cache,target=/root/.cache/golangci-lint \
            golangci-lint run --timeout 10m0s ./...


FROM base AS unit-test
RUN --mount=target=. \
            --mount=type=cache,target=/root/.cache/go-build \
            go test -v ./...

