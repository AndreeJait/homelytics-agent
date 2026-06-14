#!/bin/sh
set -e

# Ensure runtime directories exist.
mkdir -p /opt/homelytics/run /opt/homelytics/log /opt/homelytics/lib/containerd /run/containerd

# Start containerd in the background.
containerd -c /etc/containerd/config.toml &
CONTAINERD_PID=$!

# Wait for the containerd socket to become available.
for _ in $(seq 1 30); do
    if [ -S /run/containerd/containerd.sock ]; then
        break
    fi
    sleep 1
done

if ! [ -S /run/containerd/containerd.sock ]; then
    echo "containerd socket did not appear" >&2
    kill $CONTAINERD_PID || true
    exit 1
fi

# Start the homelytics daemon.
exec homelytics-daemon "$@"
