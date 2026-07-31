#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
downloads_dir="$project_root/work/downloads"
toolchains_dir="$project_root/work/toolchains"

case "$(uname -s)" in
    Darwin) go_os=darwin ;;
    Linux) go_os=linux ;;
    *)
        echo "unsupported build host: $(uname -s); expected Darwin or Linux" >&2
        exit 1
        ;;
esac
case "$(uname -m)" in
    arm64|aarch64) go_arch=arm64 ;;
    x86_64|amd64) go_arch=amd64 ;;
    *)
        echo "unsupported build architecture: $(uname -m)" >&2
        exit 1
        ;;
esac
go_platform=$go_os-$go_arch

if [ -n "${PORTABLE_PET_GO_RUNTIME_DIR:-}" ]; then
    if [ ! -x "$PORTABLE_PET_GO_RUNTIME_DIR/bin/go" ]; then
        echo "PORTABLE_PET_GO_RUNTIME_DIR must contain executable bin/go" >&2
        exit 1
    fi
    printf '%s\n' "$PORTABLE_PET_GO_RUNTIME_DIR"
    exit 0
fi

workspace_python=${WORKSPACE_PYTHON:-}
if [ -z "$workspace_python" ] || [ ! -x "$workspace_python" ]; then
    workspace_python=$(command -v python3 || true)
fi
if [ -z "$workspace_python" ] || [ ! -x "$workspace_python" ]; then
    echo "Python 3 is required to parse the official Go download manifest." >&2
    exit 1
fi

mkdir -p "$downloads_dir" "$toolchains_dir"

# The current checked download can be reused without a network request.
known_version=1.26.5
known_arm64_sha=efb87ff28af9a188d0536ef5d42e63dd52ba8263cd7344a993cc48dd11dedb6a
archive="$toolchains_dir/go${known_version}.${go_platform}.tar.gz"
version=$known_version
expected_sha=
if [ "$go_platform" = darwin-arm64 ] && [ -f "$archive" ]; then
    expected_sha=$known_arm64_sha
else
    manifest=$(mktemp "${TMPDIR:-/private/tmp}/portable-pet-go-downloads.XXXXXX.json")
    trap 'rm -f "$manifest"' EXIT HUP INT TERM
    curl --fail --location --silent --show-error \
        'https://go.dev/dl/?mode=json' -o "$manifest"
    metadata=$(
        "$workspace_python" - "$manifest" "$go_platform" <<'PY'
import json
import sys

manifest_path, platform = sys.argv[1:]
os_name, arch = platform.split("-", 1)
with open(manifest_path, "r", encoding="utf-8") as source:
    releases = json.load(source)
for release in releases:
    if not release.get("stable"):
        continue
    for item in release.get("files", []):
        if (
            item.get("os") == os_name
            and item.get("arch") == arch
            and item.get("kind") == "archive"
        ):
            print(release["version"].removeprefix("go"))
            print(item["filename"])
            print(item["sha256"])
            raise SystemExit(0)
raise SystemExit(f"no stable Go archive found for {platform}")
PY
    )
    version=$(printf '%s\n' "$metadata" | sed -n '1p')
    filename=$(printf '%s\n' "$metadata" | sed -n '2p')
    expected_sha=$(printf '%s\n' "$metadata" | sed -n '3p')
    archive="$downloads_dir/$filename"
    if [ ! -f "$archive" ]; then
        curl --fail --location --show-error \
            "https://go.dev/dl/$filename" -o "$archive"
    fi
fi

if command -v shasum >/dev/null 2>&1; then
    actual_sha=$(shasum -a 256 "$archive" | awk '{print $1}')
elif command -v sha256sum >/dev/null 2>&1; then
    actual_sha=$(sha256sum "$archive" | awk '{print $1}')
else
    echo "shasum or sha256sum is required to verify the Go archive." >&2
    exit 1
fi
if [ "$actual_sha" != "$expected_sha" ]; then
    echo "Go archive SHA256 mismatch: expected $expected_sha, got $actual_sha" >&2
    exit 1
fi

runtime_parent=${PORTABLE_PET_GO_RUNTIME_PARENT:-${TMPDIR:-/private/tmp}}
runtime_dir="$runtime_parent/portable-pet-go-toolchain-go${version}-${go_platform}"
if [ ! -x "$runtime_dir/bin/go" ] || [ "$(tr -d '\r\n' < "$runtime_dir/VERSION" 2>/dev/null || true)" != "go${version}" ]; then
    staging_dir=$(mktemp -d "$runtime_parent/portable-pet-go-stage.XXXXXX")
    trap 'rm -rf "$staging_dir"; rm -f "${manifest:-}"' EXIT HUP INT TERM
    tar -xzf "$archive" -C "$staging_dir"
    if [ ! -x "$staging_dir/go/bin/go" ]; then
        echo "Go archive did not contain go/bin/go" >&2
        exit 1
    fi
    if [ -e "$runtime_dir" ]; then
        case "$runtime_dir" in
            "$runtime_parent"/portable-pet-go-toolchain-*) rm -rf "$runtime_dir" ;;
            *)
                echo "refusing to replace unexpected runtime path: $runtime_dir" >&2
                exit 1
                ;;
        esac
    fi
    mv "$staging_dir/go" "$runtime_dir"
fi

# Mach-O binaries launched from Documents can stall at dyld_start on newer
# macOS releases. Clear attributes when the host provides xattr.
if command -v xattr >/dev/null 2>&1; then
    xattr -cr "$runtime_dir" 2>/dev/null || true
fi
printf '%s\n' "$runtime_dir"
