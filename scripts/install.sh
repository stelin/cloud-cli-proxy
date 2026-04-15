#!/usr/bin/env bash
set -euo pipefail

REPO="ZaneL1u/cloud-cli-proxy"
BINARY="cloud-claude"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m警告:\033[0m %s\n' "$*"; }
error() { printf '\033[1;31m错误:\033[0m %s\n' "$*" >&2; exit 1; }

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "${os}" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)      error "不支持的操作系统: ${os}" ;;
  esac

  case "${arch}" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *)              error "不支持的架构: ${arch}" ;;
  esac
}

get_latest_version() {
  local url="https://api.github.com/repos/${REPO}/releases/latest"
  VERSION="$(curl -fsSL "${url}" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')"
  if [ -z "${VERSION}" ]; then
    error "无法获取最新版本号，请检查网络连接"
  fi
}

download_and_install() {
  local archive="${BINARY}-${OS}-${ARCH}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${VERSION}/${archive}"
  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "${tmpdir}"' EXIT

  info "下载 ${BINARY} ${VERSION} (${OS}/${ARCH})..."
  curl -fsSL -o "${tmpdir}/${archive}" "${url}" \
    || error "下载失败: ${url}\n请确认该版本存在: https://github.com/${REPO}/releases"

  info "校验完整性..."
  local sha_url="${url}.sha256"
  if curl -fsSL -o "${tmpdir}/${archive}.sha256" "${sha_url}" 2>/dev/null; then
    (cd "${tmpdir}" && sha256sum -c "${archive}.sha256" --quiet 2>/dev/null) \
      || (cd "${tmpdir}" && shasum -a 256 -c "${archive}.sha256" --quiet 2>/dev/null) \
      || warn "sha256 校验失败，继续安装"
  fi

  info "解压..."
  tar xzf "${tmpdir}/${archive}" -C "${tmpdir}"

  local src="${tmpdir}/${BINARY}-${OS}-${ARCH}"
  chmod +x "${src}"

  info "安装到 ${INSTALL_DIR}/${BINARY}..."
  if [ -w "${INSTALL_DIR}" ]; then
    mv "${src}" "${INSTALL_DIR}/${BINARY}"
  else
    sudo mv "${src}" "${INSTALL_DIR}/${BINARY}"
  fi
}

verify_install() {
  if command -v "${BINARY}" &>/dev/null; then
    local installed_version
    installed_version="$("${BINARY}" --version 2>/dev/null || echo "unknown")"
    info "安装成功! ${installed_version}"
  else
    warn "${BINARY} 已安装到 ${INSTALL_DIR}/${BINARY}，但不在 PATH 中"
    warn "请将 ${INSTALL_DIR} 加入 PATH，或运行: export PATH=\"${INSTALL_DIR}:\$PATH\""
  fi
}

main() {
  info "Cloud CLI Proxy — cloud-claude 安装脚本"
  echo ""

  detect_platform

  if [ -n "${1:-}" ]; then
    VERSION="$1"
  else
    get_latest_version
  fi

  download_and_install
  verify_install

  echo ""
  info "快速开始："
  echo "  cloud-claude init        # 配置网关地址与凭证"
  echo "  alias claude=cloud-claude"
  echo "  claude                   # 像本地一样使用 Claude Code"
}

main "$@"
