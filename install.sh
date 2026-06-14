#!/usr/bin/env bash
set -euo pipefail

# Homelytics Agent installer for Linux/WSL.
# Run as root.

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/homelytics"
RUN_DIR="${INSTALL_DIR}/run"
ETC_DIR="${INSTALL_DIR}/etc"
LOG_DIR="${INSTALL_DIR}/log"
LIB_DIR="${INSTALL_DIR}/lib/containerd"
BIN_DIR="/usr/local/bin"
DAEMON_BIN="homelytics-daemon"
CLI_BIN="homelytics-agent"
SERVICE_NAME="homelytics-agent"
CONFIG_PATH="${ETC_DIR}/config.yaml"

CREATE_USER=false
START_SERVICE=false

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --create-user)
                CREATE_USER=true
                shift
                ;;
            --start)
                START_SERVICE=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

usage() {
    cat <<EOF
Usage: $0 [options]

Options:
  --create-user    Create a dedicated 'homelytics' system user.
  --start          Enable and start the systemd/OpenRC service after install.
  -h, --help       Show this help.

Example:
  sudo $0 --create-user --start
EOF
}

require_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "This installer must be run as root. Use: sudo $0"
        exit 1
    fi
}

detect_distro() {
    if [[ -f /etc/os-release ]]; then
        # shellcheck source=/dev/null
        . /etc/os-release
        echo "$ID"
    else
        echo "unknown"
    fi
}

install_go() {
    if command -v go >/dev/null 2>&1; then
        local version
        version="$(go version | awk '{print $3}' | sed 's/^go//')"
        echo "Go found: $version"
        if [[ "$(printf '%s\n' "1.25" "$version" | sort -V | head -n1)" != "1.25" ]]; then
            echo "WARNING: Go $version is older than 1.25. The build may fail."
            echo "Install a newer Go manually or upgrade the package."
        fi
        return 0
    fi

    echo "Go not found. Installing Go via package manager..."
    local distro
    distro="$(detect_distro)"
    case "$distro" in
        debian|ubuntu|pop)
            apt-get update
            apt-get install -y golang-go
            ;;
        fedora|rhel|centos|rocky|almalinux)
            dnf install -y golang
            ;;
        alpine)
            apk add go
            ;;
        arch|manjaro)
            pacman -S --noconfirm go
            ;;
        *)
            echo "Could not install Go automatically for distro: $distro"
            echo "Please install Go 1.25+ manually and re-run this script."
            exit 1
            ;;
    esac
}

install_containerd() {
    echo "Installing containerd and runc..."
    local distro
    distro="$(detect_distro)"
    case "$distro" in
        debian|ubuntu|pop)
            apt-get update
            apt-get install -y containerd runc
            ;;
        fedora|rhel|centos|rocky|almalinux)
            dnf install -y containerd runc
            ;;
        alpine)
            apk add containerd runc
            ;;
        arch|manjaro)
            pacman -S --noconfirm containerd runc
            ;;
        *)
            echo "Could not install containerd automatically for distro: $distro"
            echo "Please install containerd and runc manually."
            exit 1
            ;;
    esac

    if command -v containerd >/dev/null 2>&1; then
        echo "containerd installed at $(command -v containerd)"
    else
        echo "WARNING: containerd binary not found in PATH after install."
    fi
}

build_binaries() {
    echo "Building homelytics-agent binaries..."
    cd "$REPO_DIR"
    go mod tidy
    make build
}

install_binaries() {
    echo "Installing binaries to ${BIN_DIR}..."
    install -m 755 "${REPO_DIR}/bin/${DAEMON_BIN}" "${BIN_DIR}/${DAEMON_BIN}"
    install -m 755 "${REPO_DIR}/bin/${CLI_BIN}" "${BIN_DIR}/${CLI_BIN}"
}

create_directories() {
    echo "Creating runtime directories..."
    mkdir -p "$RUN_DIR" "$ETC_DIR" "$LOG_DIR" "$LIB_DIR"
    chmod 0750 "$INSTALL_DIR" "$RUN_DIR" "$ETC_DIR" "$LOG_DIR" "$LIB_DIR"
}

create_user() {
    if ! id -u homelytics >/dev/null 2>&1; then
        echo "Creating homelytics system user..."
        useradd --system --no-create-home --home-dir "$INSTALL_DIR" --shell /usr/sbin/nologin homelytics
    else
        echo "homelytics user already exists."
    fi

    chown -R homelytics:homelytics "$INSTALL_DIR"
    chmod 0750 "$INSTALL_DIR"
}

install_config() {
    if [[ -f "$CONFIG_PATH" ]]; then
        echo "Config already exists at ${CONFIG_PATH}, skipping."
        return 0
    fi

    echo "Installing default config to ${CONFIG_PATH}..."
    cp "${REPO_DIR}/files/config/app.yaml" "$CONFIG_PATH"

    if [[ "$CREATE_USER" == true ]]; then
        chown root:homelytics "$CONFIG_PATH"
        chmod 0640 "$CONFIG_PATH"
    else
        chown root:root "$CONFIG_PATH"
        chmod 0640 "$CONFIG_PATH"
    fi
}

install_systemd_service() {
    if ! command -v systemctl >/dev/null 2>&1; then
        return 0
    fi

    echo "Installing systemd service..."
    local service_path="/etc/systemd/system/${SERVICE_NAME}.service"

    local user group
    if [[ "$CREATE_USER" == true ]]; then
        user="homelytics"
        group="homelytics"
    else
        user="root"
        group="root"
    fi

    cat > "$service_path" <<EOF
[Unit]
Description=Homelytics edge agent daemon
After=network.target containerd.service

[Service]
Type=simple
User=${user}
Group=${group}
ExecStart=${BIN_DIR}/${DAEMON_BIN} --config ${CONFIG_PATH}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload

    if [[ "$START_SERVICE" == true ]]; then
        echo "Enabling and starting ${SERVICE_NAME} service..."
        systemctl enable --now "$SERVICE_NAME"
    fi
}

install_openrc_service() {
    if [[ ! -x /sbin/rc-service ]]; then
        return 0
    fi

    echo "Installing OpenRC service..."
    local service_path="/etc/init.d/${SERVICE_NAME}"

    cat > "$service_path" <<'EOF'
#!/sbin/openrc-run

description="Homelytics edge agent daemon"
command="/usr/local/bin/homelytics-daemon"
command_args="--config /opt/homelytics/etc/config.yaml"
command_background=true
pidfile="/run/${RC_SVCNAME}.pid"
output_log="/opt/homelytics/log/daemon.log"
error_log="/opt/homelytics/log/daemon.err"

depend() {
    need net
    after containerd
}
EOF

    chmod 755 "$service_path"

    if [[ "$START_SERVICE" == true ]]; then
        rc-service "$SERVICE_NAME" start
        rc-update add "$SERVICE_NAME" default
    fi
}

print_summary() {
    echo
    echo "========================================"
    echo "Homelytics Agent installed."
    echo "========================================"
    echo "Daemon binary:  ${BIN_DIR}/${DAEMON_BIN}"
    echo "CLI binary:     ${BIN_DIR}/${CLI_BIN}"
    echo "Config:         ${CONFIG_PATH}"
    echo "IPC socket:     ${RUN_DIR}/ipc.sock"
    echo "containerd:     $(command -v containerd 2>/dev/null || echo 'not in PATH')"
    echo
    if [[ "$CREATE_USER" == true ]]; then
        echo "Service user:   homelytics"
    else
        echo "WARNING: Running as root. Use --create-user for a dedicated account."
    fi
    echo
    if [[ "$START_SERVICE" == true ]]; then
        echo "Service status:"
        if command -v systemctl >/dev/null 2>&1; then
            systemctl status "$SERVICE_NAME" --no-pager || true
        fi
    else
        echo "Start the daemon manually with:"
        echo "  ${BIN_DIR}/${DAEMON_BIN} --config ${CONFIG_PATH}"
        echo "Or run the installer again with --start to enable the service."
    fi
    echo
    echo "CLI examples:"
    echo "  ${BIN_DIR}/${CLI_BIN} login --email merchant@example.com --password password"
    echo "  ${BIN_DIR}/${CLI_BIN} tsnet auth"
    echo "  ${BIN_DIR}/${CLI_BIN} status"
}

main() {
    parse_args "$@"
    require_root
    install_go
    install_containerd
    build_binaries
    create_directories
    install_binaries
    install_config

    if [[ "$CREATE_USER" == true ]]; then
        create_user
    fi

    install_systemd_service
    install_openrc_service
    print_summary
}

main "$@"
