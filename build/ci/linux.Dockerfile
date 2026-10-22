# syntax=docker/dockerfile:1.7

ARG UBUNTU_VERSION=24.04

FROM ubuntu:${UBUNTU_VERSION} AS tools

ARG GO_VERSION
ARG GO_LINUX_AMD64_SHA256
ARG NODE_VERSION
ARG NODE_LINUX_X64_SHA256
ARG PNPM_VERSION
ARG TASK_VERSION
ARG WAILS_VERSION
ARG GO_LICENSES_VERSION
ARG GOLANGCI_LINT_VERSION

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
        build-essential \
        ca-certificates \
        curl \
        git \
        gzip \
        libgtk-4-dev \
        libwebkitgtk-6.0-dev \
        make \
        pkg-config \
        python3 \
        tar \
        unzip \
        xz-utils \
    && pkg-config --modversion gtk4 \
    && pkg-config --modversion webkitgtk-6.0 \
    && rm -rf /var/lib/apt/lists/*

RUN test -n "${GO_VERSION}" \
    && test -n "${GO_LINUX_AMD64_SHA256}" \
    && test -n "${NODE_VERSION}" \
    && test -n "${NODE_LINUX_X64_SHA256}" \
    && curl --fail --location --silent --show-error \
        "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
        --output /tmp/go.tar.gz \
    && echo "${GO_LINUX_AMD64_SHA256}  /tmp/go.tar.gz" | sha256sum --check --status \
    && tar --extract --gzip --file /tmp/go.tar.gz --directory /usr/local \
    && rm /tmp/go.tar.gz \
    && mkdir --parents /opt/node \
    && curl --fail --location --silent --show-error \
        "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" \
        --output /tmp/node.tar.xz \
    && echo "${NODE_LINUX_X64_SHA256}  /tmp/node.tar.xz" | sha256sum --check --status \
    && tar --extract --xz --file /tmp/node.tar.xz --directory /opt/node --strip-components=1 \
    && rm /tmp/node.tar.xz

ENV GOPATH=/go
ENV GOMODCACHE=/go/pkg/mod
ENV PNPM_HOME=/opt/pnpm
ENV NPM_CONFIG_STORE_DIR=/opt/pnpm/store
ENV PATH=/opt/flexigpt/bin:/go/bin:/opt/pnpm/bin:/usr/local/go/bin:/opt/node/bin:${PATH}

RUN npm config set prefix /opt/pnpm \
    && npm install --global "pnpm@${PNPM_VERSION}" \
    && pnpm config set store-dir /opt/pnpm/store \
    && mkdir --parents /opt/flexigpt/bin \
    && GOBIN=/opt/flexigpt/bin go install "github.com/go-task/task/v3/cmd/task@v${TASK_VERSION}" \
    && GOBIN=/opt/flexigpt/bin go install "github.com/wailsapp/wails/v3/cmd/wails3@${WAILS_VERSION}" \
    && GOBIN=/opt/flexigpt/bin go install "github.com/google/go-licenses/v2@${GO_LICENSES_VERSION}" \
    && GOBIN=/opt/flexigpt/bin go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}" \
    && task --version \
    && wails3 version \
    && go-licenses --help >/dev/null \
    && golangci-lint version

FROM tools AS dependencies

WORKDIR /workspace

COPY . /workspace

RUN task install-go-tools \
    && go mod download \
    && pnpm fetch --frozen-lockfile

FROM tools AS ci

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
        appstream \
        cloc \
        desktop-file-utils \
        graphviz \
        libnss3 \
        shared-mime-info \
        xvfb \
        zip \
    && rm -rf /var/lib/apt/lists/*

COPY --from=dependencies /go/pkg/mod /go/pkg/mod
COPY --from=dependencies /go/bin /go/bin
COPY --from=dependencies /opt/pnpm/store /opt/pnpm/store

ENV FLEXIGPT_TOOLS_DIR=/opt/flexigpt/bin

WORKDIR /workspace

RUN go version \
    && node --version \
    && pnpm --version \
    && task --version \
    && golangci-lint version \
    && pkg-config --modversion gtk4 \
    && pkg-config --modversion webkitgtk-6.0
