#!/bin/sh
set -eu

skill_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
template="$skill_dir/assets/windows-go-template"

test -x "$skill_dir/scripts/scaffold_pet.py"
test -x "$skill_dir/scripts/verify-delivery.sh"
test -x "$skill_dir/scripts/create_test_assets.py"
test -f "$template/internal/win32/window_windows.go"
test -f "$template/internal/app/app_windows.go"
test -f "$template/build/app.manifest"

contract="$skill_dir/SKILL.md"
rg -Fq 'Windows 10/11 x64' "$contract"
rg -Fq 'silent, offline, no-install, no-admin, and no-autostart' "$contract"
rg -Fq 'Do not invent lore, body parts, markings, clothes, or accessories' "$contract"
rg -Fq 'single click -> Happy' "$contract"
rg -Fq 'double click -> Sleep/Wake' "$contract"
rg -Fq 'left-drag -> Drag/move' "$contract"
rg -Fq 'right click -> menu' "$contract"
rg -Fq 'Darwin or Linux build host' "$contract"
rg -Fq 'Python 3 with Pillow' "$contract"
rg -Fq 'Run reuses the directional Walk rows' "$contract"
rg -Fq 'Sleep reuses Idle at a slower rate' "$contract"
rg -Fq 'Drag holds the first Idle frame' "$contract"

if find "$template" -type f \( -name '*.png' -o -name '*.jpg' -o -name '*.webp' -o -name '*.exe' \) | grep .; then
  echo "template contains character or binary artifacts" >&2
  exit 1
fi
if rg -n '蓝团|cat-photo|character-sheet' "$template"; then
  echo "template contains project-specific identity" >&2
  exit 1
fi
if rg -n '/Users/quinnpan' "$template"; then
  echo "template contains a personal absolute path" >&2
  exit 1
fi

test_root=$(mktemp -d "${TMPDIR:-/private/tmp}/portable-pet-skill-test.XXXXXX")
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

forbidden_worktree=".work""trees"
forbidden_fixture="/Users/quinnpan/Documents/""Codex"
if rg -n "$forbidden_worktree|$forbidden_fixture" "$0"; then
  echo "self-test contains an external fixture path" >&2
  exit 1
fi

test_python=${WORKSPACE_PYTHON:-}
if [ -z "$test_python" ]; then
  test_python=$(command -v python3 || true)
fi
test -n "$test_python"
if [ "${PORTABLE_PET_SKIP_REGRESSION_TESTS:-0}" != 1 ]; then
  "$test_python" "$skill_dir/scripts/test_final_review.py"
fi

test_assets="$test_root/test-assets"
"$test_python" "$skill_dir/scripts/create_test_assets.py" "$test_assets"
atlas="$test_assets/atlas.png"
icon="$test_assets/icon.png"
"$test_python" -c '
import struct, sys
for path, expected in zip(sys.argv[1:], [(1536, 2288), (256, 256)]):
    data = open(path, "rb").read(33)
    assert data[:8] == b"\x89PNG\r\n\x1a\n"
    width, height, depth, color = struct.unpack(">IIBB", data[16:26])
    assert (width, height, depth, color) == (*expected, 8, 6)
' "$atlas" "$icon"

if "$skill_dir/scripts/scaffold_pet.py" --name TestPet --output-dir "$test_root/project"; then
  echo "scaffold accepted missing atlas/icon" >&2
  exit 1
fi

"$skill_dir/scripts/scaffold_pet.py" \
  --name '测试宠物' \
  --atlas "$atlas" \
  --icon "$icon" \
  --output-dir "$test_root/project"

test -f "$test_root/project/assets/runtime/spritesheet.png"
test -f "$test_root/project/assets/runtime/app-icon.png"
rg -q '测试宠物桌宠' "$test_root/project/internal/app/app_windows.go"
! rg -n '__PET_TITLE__|__PET_EXE__|蓝团' "$test_root/project"

special_project="$test_root/special-project"
"$skill_dir/scripts/scaffold_pet.py" \
  --name 'A"B<&' \
  --atlas "$atlas" \
  --icon "$icon" \
  --output-dir "$special_project"

"$test_python" -c \
  'import sys, xml.etree.ElementTree as ET; ET.parse(sys.argv[1])' \
  "$special_project/build/app.manifest"

test_go=${PORTABLE_PET_TEST_GO:-}
if [ -z "$test_go" ]; then
  test_go=$(command -v go || true)
fi
if [ -z "$test_go" ]; then
  for candidate in "${TMPDIR:-/private/tmp}"/*go-toolchain-go*-darwin-*/bin/go; do
    if [ -x "$candidate" ]; then
      test_go=$candidate
      break
    fi
  done
fi
if [ -z "$test_go" ]; then
  runtime_dir=$("$special_project/scripts/fetch-go-toolchain.sh")
  test_go="$runtime_dir/bin/go"
fi
test -x "$test_go"
mkdir -p "$test_root/go-cache" "$test_root/go-path/pkg/mod"
(
  cd "$test_root/project"
  GOCACHE="$test_root/go-cache" \
    GOPATH="$test_root/go-path" \
    GOMODCACHE="$test_root/go-path/pkg/mod" \
    GOENV=off GOTELEMETRY=off GOTOOLCHAIN=local \
    "$test_go" test ./internal/...
)
(
  cd "$special_project"
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    GOCACHE="$test_root/go-cache" \
    GOPATH="$test_root/go-path" \
    GOMODCACHE="$test_root/go-path/pkg/mod" \
    GOENV=off GOTELEMETRY=off GOTOOLCHAIN=local \
    "$test_go" build -trimpath -ldflags='-s -w -H=windowsgui' \
    -o "$test_root/A_B__.exe" ./cmd/pet
)

test_go_runtime=${PORTABLE_PET_TEST_GO_RUNTIME_DIR:-}
if [ -z "$test_go_runtime" ]; then
  test_go_runtime=$("$test_go" env GOROOT 2>/dev/null || true)
fi
(
  if [ -n "$test_go_runtime" ]; then
    export PORTABLE_PET_GO_RUNTIME_DIR="$test_go_runtime"
  fi
  if [ -n "${PORTABLE_PET_TEST_RSRC:-}" ]; then
    export PORTABLE_PET_RSRC="$PORTABLE_PET_TEST_RSRC"
  fi
  export WORKSPACE_PYTHON="$test_python"
  cd "$test_root/project"
  ./scripts/build-windows.sh
)
built_exe="$test_root/project/outputs/测试宠物.exe"
test -f "$built_exe"
section_headers=$(objdump -h "$built_exe")
printf '%s\n' "$section_headers" | rg -q '[[:space:]]\.rsrc[[:space:]]'
verification=$(
  "$skill_dir/scripts/verify-delivery.sh" \
    "$built_exe" \
    "$test_root/project"
)
printf '%s\n' "$verification" | rg -q '^sha256=[0-9a-fA-F]{64}$'

control_output="$test_root/control-output"
if "$skill_dir/scripts/scaffold_pet.py" \
  --name "$(printf 'bad\nname')" \
  --atlas "$atlas" \
  --icon "$icon" \
  --output-dir "$control_output"; then
  echo "scaffold accepted a control character" >&2
  exit 1
fi
if [ -d "$control_output" ] &&
  find "$control_output" -mindepth 1 -print -quit | grep .; then
  echo "rejected name populated the output directory" >&2
  exit 1
fi
"$skill_dir/scripts/scaffold_pet.py" \
  --name Corrected \
  --atlas "$atlas" \
  --icon "$icon" \
  --output-dir "$control_output"

for reserved_name in CON AUX COM1; do
  reserved_project="$test_root/reserved-$reserved_name"
  "$skill_dir/scripts/scaffold_pet.py" \
    --name "$reserved_name" \
    --atlas "$atlas" \
    --icon "$icon" \
    --output-dir "$reserved_project"
  if rg -iq "outputs/$reserved_name\\.exe" \
    "$reserved_project/scripts/build-windows.sh"; then
    echo "scaffold emitted reserved Windows filename: $reserved_name" >&2
    exit 1
  fi
done

echo "SELF_TEST_OK"
