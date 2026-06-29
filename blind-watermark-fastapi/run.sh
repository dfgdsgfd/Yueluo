#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

python_bin="${PYTHON:-}"
if [ -z "$python_bin" ]; then
  if command -v python3 >/dev/null 2>&1; then
    python_bin="python3"
  else
    python_bin="python"
  fi
fi

apt_install() {
  if ! command -v apt-get >/dev/null 2>&1; then
    return 1
  fi
  if [ "$(id -u)" -ne 0 ]; then
    return 1
  fi
  export DEBIAN_FRONTEND="${DEBIAN_FRONTEND:-noninteractive}"
  apt-get update
  apt-get install -y --no-install-recommends "$@"
  rm -rf /var/lib/apt/lists/*
}

install_debian_runtime_libs() {
  if ! command -v apt-get >/dev/null 2>&1; then
    return
  fi
  if command -v ldconfig >/dev/null 2>&1 && ldconfig -p 2>/dev/null | grep -q 'libGL\.so\.1'; then
    return
  fi
  if ! apt_install libgl1 libglib2.0-0; then
    echo "libGL.so.1 is missing; run as root or install libgl1 libglib2.0-0 in the image." >&2
  fi
}

install_uv() {
  if command -v uv >/dev/null 2>&1; then
    return
  fi
  if "$python_bin" -m pip --version >/dev/null 2>&1; then
    "$python_bin" -m pip install --no-cache-dir uv
    return
  fi
  if command -v pip3 >/dev/null 2>&1; then
    pip3 install --no-cache-dir uv
    return
  fi
  if command -v pip >/dev/null 2>&1; then
    pip install --no-cache-dir uv
    return
  fi
  if "$python_bin" -m ensurepip --upgrade >/dev/null 2>&1; then
    "$python_bin" -m pip install --no-cache-dir uv
    return
  fi
  if apt_install python3-pip; then
    "$python_bin" -m pip install --no-cache-dir uv
    return
  fi
  echo "uv is missing and no pip installer is available; install pip or uv in the image." >&2
  exit 1
}

install_debian_runtime_libs

install_uv

sync_args=()
if [ -n "${UV_SYNC_ARGS:-}" ]; then
  # shellcheck disable=SC2206
  sync_args=(${UV_SYNC_ARGS})
else
  sync_args=(--no-dev)
fi

uv sync "${sync_args[@]}"

host="${BLIND_WATERMARK_HOST:-0.0.0.0}"
port="${BLIND_WATERMARK_PORT:-8090}"

exec uv run --no-dev uvicorn blind_watermark_fastapi.main:app --host "$host" --port "$port"
