#!/bin/sh
set -e

# Ensure runtime directories exist.
mkdir -p /opt/homelytics/run /opt/homelytics/log /opt/homelytics/lib/containerd /run/containerd

# Prepare cgroup v2 hierarchy for nested containers.
# Move this shell (and therefore containerd + its children) into a delegated
# cgroup so runc can create per-container cgroups below it.
CGROUP_ROOT=/sys/fs/cgroup
if [ -d "$CGROUP_ROOT" ] && [ -f "$CGROUP_ROOT/cgroup.controllers" ]; then
    mkdir -p "$CGROUP_ROOT/homelytics"
    # Enable controllers in the root so the homelytics cgroup can use them.
    for ctrl in cpuset cpu io memory pids; do
        echo "+$ctrl" > "$CGROUP_ROOT/cgroup.subtree_control" 2>/dev/null || true
    done
    # Enable controllers for the homelytics subtree.
    for ctrl in cpuset cpu io memory pids; do
        echo "+$ctrl" > "$CGROUP_ROOT/homelytics/cgroup.subtree_control" 2>/dev/null || true
    done
    # Move the current shell into the homelytics cgroup.
    echo $$ > "$CGROUP_ROOT/homelytics/cgroup.procs" 2>/dev/null || true
fi

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
