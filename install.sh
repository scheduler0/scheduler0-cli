#!/usr/bin/env bash
set -euo pipefail

REPO="scheduler0/scheduler0-cli"
BINARY="scheduler0"
RELEASES_URL="https://github.com/${REPO}/releases"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { printf '\033[0;32m[info]\033[0m  %s\n' "$*" >&2; }
warn()  { printf '\033[0;33m[warn]\033[0m  %s\n' "$*" >&2; }
error() { printf '\033[0;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || error "Required command not found: $1"
}

# ---------------------------------------------------------------------------
# Platform detection
# ---------------------------------------------------------------------------
detect_os() {
  local raw
  raw="$(uname -s)"
  case "$raw" in
    Darwin) echo "darwin" ;;
    Linux)  echo "linux"  ;;
    *)
      error "Unsupported OS: $raw. Download manually from ${RELEASES_URL}"
      ;;
  esac
}

detect_arch() {
  local raw
  raw="$(uname -m)"
  case "$raw" in
    x86_64)          echo "amd64" ;;
    arm64|aarch64)   echo "arm64" ;;
    *)
      error "Unsupported architecture: $raw. Download manually from ${RELEASES_URL}"
      ;;
  esac
}

# ---------------------------------------------------------------------------
# Download helper (curl with wget fallback)
# ---------------------------------------------------------------------------
download() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fSL --progress-bar -o "$dest" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q --show-progress -O "$dest" "$url"
  else
    error "Neither curl nor wget found. Install one of them and re-run."
  fi
}

# ---------------------------------------------------------------------------
# Version resolution
# ---------------------------------------------------------------------------
resolve_version() {
  # Priority: CLI arg -> VERSION env var -> latest release from GitHub API
  if [[ "${1:-}" =~ ^v?[0-9] ]]; then
    echo "${1#v}"
    return
  fi
  if [[ -n "${VERSION:-}" ]]; then
    echo "${VERSION#v}"
    return
  fi

  info "Fetching latest release version..."
  local tag
  if command -v curl >/dev/null 2>&1; then
    tag="$(curl -fsSL "$API_URL" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')"
  elif command -v wget >/dev/null 2>&1; then
    tag="$(wget -qO- "$API_URL" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')"
  else
    error "Neither curl nor wget found. Install one of them and re-run."
  fi

  [[ -n "$tag" ]] || error "Could not determine latest version from GitHub API. Set VERSION= and retry."
  echo "${tag#v}"
}

# ---------------------------------------------------------------------------
# Checksum verification
# ---------------------------------------------------------------------------
verify_checksum() {
  local archive="$1" checksums="$2" dir="$3"
  local basename
  basename="$(basename "$archive")"

  info "Verifying checksum..."
  # Extract only the line for this archive so the checker doesn't fail on other entries
  local tmp_check="${dir}/check.txt"
  grep "${basename}" "$checksums" > "$tmp_check" || error "Checksum entry for ${basename} not found in checksums.txt"

  # Prefer shasum (macOS) then sha256sum (Linux)
  if command -v shasum >/dev/null 2>&1; then
    (cd "$dir" && shasum -a 256 -c "$tmp_check") || error "Checksum verification failed."
  elif command -v sha256sum >/dev/null 2>&1; then
    (cd "$dir" && sha256sum -c "$tmp_check") || error "Checksum verification failed."
  else
    warn "No sha256 tool found (shasum / sha256sum). Skipping checksum verification."
  fi
  info "Checksum OK."
}

# ---------------------------------------------------------------------------
# Install dir resolution
# ---------------------------------------------------------------------------
resolve_install_dir() {
  if [[ -n "${INSTALL_DIR:-}" ]]; then
    echo "$INSTALL_DIR"
    return
  fi
  echo "/usr/local/bin"
}

install_binary() {
  local src="$1" dir="$2"
  chmod +x "$src"

  if [[ -w "$dir" ]]; then
    mv "$src" "${dir}/${BINARY}"
  elif command -v sudo >/dev/null 2>&1; then
    info "Writing to ${dir} requires sudo..."
    sudo mv "$src" "${dir}/${BINARY}"
    sudo chmod +x "${dir}/${BINARY}"
  else
    warn "Cannot write to ${dir} and sudo is unavailable."
    local fallback="${HOME}/.local/bin"
    mkdir -p "$fallback"
    mv "$src" "${fallback}/${BINARY}"
    chmod +x "${fallback}/${BINARY}"
    dir="$fallback"
    warn "Installed to ${fallback}."
    if ! echo ":${PATH}:" | grep -q ":${fallback}:"; then
      warn "${fallback} is not in PATH. Add it to your shell profile:"
      warn "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
  fi
  echo "$dir"
}

# ---------------------------------------------------------------------------
# Gatekeeper (macOS)
# ---------------------------------------------------------------------------
clear_quarantine() {
  local bin_path="$1"
  if [[ "$(uname -s)" == "Darwin" ]]; then
    if xattr "$bin_path" 2>/dev/null | grep -q "com.apple.quarantine"; then
      info "Clearing macOS quarantine flag..."
      xattr -d com.apple.quarantine "$bin_path" 2>/dev/null || true
    fi
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
# Declared at global scope so the EXIT trap can reference it after main returns
tmpdir=""

main() {
  local version
  version="$(resolve_version "${1:-}")"

  local os arch
  os="$(detect_os)"
  arch="$(detect_arch)"

  local archive_name="scheduler0_${version}_${os}_${arch}.tar.gz"
  local base_url="${RELEASES_URL}/download/v${version}"
  local archive_url="${base_url}/${archive_name}"
  local checksums_url="${base_url}/checksums.txt"

  info "Installing scheduler0 v${version} (${os}/${arch})..."

  # Temp workspace, cleaned up on exit
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  local archive_path="${tmpdir}/${archive_name}"
  local checksums_path="${tmpdir}/checksums.txt"

  info "Downloading ${archive_name}..."
  download "$archive_url" "$archive_path"

  info "Downloading checksums.txt..."
  download "$checksums_url" "$checksums_path"

  verify_checksum "$archive_path" "$checksums_path" "$tmpdir"

  info "Extracting binary..."
  tar -xzf "$archive_path" -C "$tmpdir" "$BINARY"

  local install_dir
  install_dir="$(resolve_install_dir)"
  mkdir -p "$install_dir"

  local final_dir
  final_dir="$(install_binary "${tmpdir}/${BINARY}" "$install_dir")"

  local bin_path="${final_dir}/${BINARY}"
  clear_quarantine "$bin_path"

  info "Installed to ${bin_path}"
  info "Version: $("$bin_path" --version 2>&1 || true)"
  info "Run 'scheduler0 --help' to get started."
}

main "$@"
