#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

runtime_assets="$project_root/assets/runtime"
atlas_source="$runtime_assets/spritesheet.png"
icon_source="$runtime_assets/app-icon.png"
resource_output="$project_root/build/rsrc_windows_amd64.syso"
package_resource="$project_root/cmd/pet/rsrc_windows_amd64.syso"

for required in "$atlas_source" "$icon_source" "$project_root/build/app.manifest"; do
    if [ ! -f "$required" ]; then
        echo "missing required build input: $required" >&2
        exit 1
    fi
done
if [ ! -d "$project_root/cmd/pet" ]; then
    echo "missing Windows entry package: $project_root/cmd/pet" >&2
    exit 1
fi

mkdir -p "$project_root/build" "$project_root/outputs"

workspace_python=${WORKSPACE_PYTHON:-}
if [ -z "$workspace_python" ] || [ ! -x "$workspace_python" ]; then
    workspace_python=$(command -v python3 || true)
fi
if [ -z "$workspace_python" ] || [ ! -x "$workspace_python" ]; then
    echo "Python 3 with Pillow is required to create build/app.ico." >&2
    exit 1
fi
if ! "$workspace_python" -c 'import PIL' >/dev/null 2>&1; then
    echo "Python 3 with Pillow is required to create build/app.ico." >&2
    exit 1
fi
"$workspace_python" "$project_root/scripts/png-to-ico.py" \
    "$runtime_assets/app-icon.png" "$project_root/build/app.ico"

rsrc_source=${PORTABLE_PET_RSRC:-"$project_root/work/go-bin/rsrc"}
if [ -n "${PORTABLE_PET_RSRC:-}" ] && [ ! -x "$rsrc_source" ]; then
    echo "PORTABLE_PET_RSRC must be an executable rsrc tool." >&2
    exit 1
fi
if [ ! -x "$rsrc_source" ]; then
    "$project_root/scripts/project-go.sh" install github.com/akavel/rsrc@v0.10.2
fi
rsrc_source=${PORTABLE_PET_RSRC:-"$project_root/work/go-bin/rsrc"}

# Mach-O tools executed directly from the Documents worktree can stall on
# newer macOS releases. Stage the verified helper and linker output in a
# private temporary directory, then copy only completed artifacts back.
build_tmp=$(mktemp -d "${TMPDIR:-/private/tmp}/portable-pet-build.XXXXXX")
rsrc_runner="$build_tmp/rsrc"
temporary_resource="$build_tmp/rsrc_windows_amd64.syso"
temporary_exe="$build_tmp/__PET_EXE__.exe"
trap 'rm -rf "$build_tmp"; rm -f "$package_resource"' EXIT HUP INT TERM
cp "$rsrc_source" "$rsrc_runner"
chmod 755 "$rsrc_runner"
if command -v xattr >/dev/null 2>&1; then
    xattr -cr "$rsrc_runner" 2>/dev/null || true
fi
"$rsrc_runner" \
    -manifest "$project_root/build/app.manifest" \
    -ico "$project_root/build/app.ico" \
    -o "$temporary_resource"

cp "$temporary_resource" "$resource_output"
cp "$resource_output" "$package_resource"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    "$project_root/scripts/project-go.sh" build \
    -trimpath \
    -ldflags="-s -w -H=windowsgui" \
    -o "$temporary_exe" \
    ./cmd/pet
cp "$temporary_exe" "$project_root/outputs/__PET_EXE__.exe"
