#!/bin/sh
set -e

REPO="mekagojira/dotfilex" # Change if using a different repository
INSTALL_DIR="${HOME}/.local/bin"

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Darwin)
        PLATFORM="darwin-universal"
        ;;
    Linux)
        ARCH="$(uname -m)"
        case "${ARCH}" in
            x86_64|amd64)
                PLATFORM="linux-amd64"
                ;;
            aarch64|arm64)
                PLATFORM="linux-arm64"
                ;;
            *)
                echo "Unsupported Linux architecture: ${ARCH}"
                exit 1
                ;;
        esac
        ;;
    *)
        echo "Unsupported operating system: ${OS}"
        exit 1
        ;;
esac

echo "⚡ Installing dotsynx for ${PLATFORM}..."

# Get latest release tag if possible, or fallback to v1.0.0
LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
if [ -z "${LATEST_TAG}" ]; then
    LATEST_TAG="v1.0.0"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/dotsynx-${PLATFORM}.tar.gz"

mkdir -p "${INSTALL_DIR}"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "⬇️  Downloading ${DOWNLOAD_URL}..."
if curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/dotsynx.tar.gz"; then
    tar -xzf "${TMP_DIR}/dotsynx.tar.gz" -C "${TMP_DIR}"
    rm -f "${INSTALL_DIR}/dotsynx"
    mv "${TMP_DIR}/dotsynx" "${INSTALL_DIR}/dotsynx"
    chmod +x "${INSTALL_DIR}/dotsynx"
    if [ "${OS}" = "Darwin" ]; then
        codesign -s - -f "${INSTALL_DIR}/dotsynx" 2>/dev/null || true
    fi
    echo "✅ Successfully installed dotsynx to ${INSTALL_DIR}/dotsynx!"
    echo "👉 Make sure ${INSTALL_DIR} is in your PATH."
    echo "👉 Run 'dotsynx' to start the TUI, or 'dotsynx ui' for the Webview."
else
    echo "Could not download pre-built release binary."
    echo "You can build locally with:"
    echo "  git clone https://github.com/${REPO}.git && cd dotfilex && make install"
    exit 1
fi
