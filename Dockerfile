# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Install build dependencies for containerd and tsnet.
RUN apk add --no-cache git gcc musl-dev linux-headers

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build both binaries.
COPY . .
RUN go build -o bin/homelytics-daemon ./cmd/daemon \
    && go build -o bin/homelytics-agent ./cmd/cli

# Runtime stage
FROM alpine:3.21

WORKDIR /opt/homelytics

# Install containerd and runc plus utilities.
RUN apk add --no-cache containerd runc ca-certificates curl iptables ip6tables

# Create the dedicated system user and required directories.
# Alpine busybox adduser: -S = system, -D = no password, -h = home directory.
RUN addgroup -S homelytics \
    && adduser -S -D -h /opt/homelytics -G homelytics homelytics \
    && mkdir -p /opt/homelytics/run /opt/homelytics/etc /opt/homelytics/log /opt/homelytics/lib/containerd /run/containerd \
    && chown -R homelytics:homelytics /opt/homelytics

# Copy binaries from builder.
COPY --from=builder --chown=homelytics:homelytics /build/bin/homelytics-daemon /usr/local/bin/homelytics-daemon
COPY --from=builder --chown=homelytics:homelytics /build/bin/homelytics-agent /usr/local/bin/homelytics-agent
COPY --from=builder --chown=homelytics:homelytics /build/files/config/app.yaml /opt/homelytics/etc/config.yaml

# Copy containerd configuration and entrypoint.
COPY --from=builder --chown=root:root /build/files/containerd/config.toml /etc/containerd/config.toml
COPY --from=builder --chown=root:root /build/files/containerd/entrypoint.sh /opt/homelytics/bin/entrypoint.sh
RUN chmod +x /opt/homelytics/bin/entrypoint.sh

# Optional local override file is mounted at runtime, not baked into the image.

# Socket directory is world-writable so the CLI can reach it when mounted.
RUN chmod 0777 /opt/homelytics/run

# Run as root inside Docker so containerd can manage cgroups and namespaces.
# Native installs should use the systemd service to drop privileges.
USER root

ENTRYPOINT ["/opt/homelytics/bin/entrypoint.sh"]
CMD ["--config", "/opt/homelytics/etc/config.yaml"]
