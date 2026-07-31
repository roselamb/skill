#!/bin/sh
set -eu
exe=${1:?usage: verify-delivery.sh EXE ABSOLUTE_PROJECT_ROOT}
project_root=${2:?usage: verify-delivery.sh EXE ABSOLUTE_PROJECT_ROOT}

case "$project_root" in
  /*) ;;
  *) echo "PROJECT_ROOT must be absolute" >&2; exit 1 ;;
esac
if [ ! -d "$project_root" ]; then
  echo "PROJECT_ROOT must be an existing directory" >&2
  exit 1
fi
project_root=$(CDPATH= cd -- "$project_root" && pwd -P)

if [ ! -f "$exe" ]; then
  echo "EXE must be an existing file" >&2
  exit 1
fi
file_output=$(file "$exe") || {
  echo "file inspection failed" >&2
  exit 1
}
printf '%s\n' "$file_output" |
  rg -q 'PE32\+ executable \(GUI\) x86-64' || {
    echo "EXE is not a Windows x64 GUI PE" >&2
    exit 1
  }

pe_headers=$(objdump -p "$exe") || {
  echo "objdump PE header inspection failed" >&2
  exit 1
}
printf '%s\n' "$pe_headers" |
  rg -q 'Subsystem[[:space:]]+00000002[[:space:]]+\(Windows GUI\)' || {
    echo "EXE does not use the Windows GUI subsystem" >&2
    exit 1
  }
section_headers=$(objdump -h "$exe") || {
  echo "objdump section inspection failed" >&2
  exit 1
}
printf '%s\n' "$section_headers" |
  rg -q '[[:space:]]\.rsrc[[:space:]]' || {
    echo "EXE does not contain a .rsrc section" >&2
    exit 1
  }

imports=$(printf '%s\n' "$pe_headers" | awk '/DLL Name:/ {sub(/^.*DLL Name:[[:space:]]*/, ""); print}')
if [ -z "$imports" ]; then
  echo "EXE import list is empty" >&2
  exit 1
fi
old_ifs=$IFS
IFS='
'
for dll in $imports; do
  case "$(printf '%s' "$dll" | tr '[:upper:]' '[:lower:]')" in
    kernel32.dll|user32.dll|gdi32.dll|shell32.dll|advapi32.dll|\
    ws2_32.dll|crypt32.dll|secur32.dll|dnsapi.dll|iphlpapi.dll|\
    netapi32.dll|userenv.dll|psapi.dll|ntdll.dll|winmm.dll|\
    mswsock.dll|bcryptprimitives.dll) ;;
    *) echo "unexpected DLL import: $dll" >&2; exit 1 ;;
  esac
done
IFS=$old_ifs

string_table=$(strings -a "$exe") || {
  echo "strings inspection failed" >&2
  exit 1
}
if printf '%s\n' "$string_table" | rg -Fq "$project_root"; then
  echo "project absolute path leaked into EXE" >&2
  exit 1
fi
if command -v shasum >/dev/null 2>&1; then
  digest_output=$(shasum -a 256 "$exe") || {
    echo "SHA-256 calculation failed" >&2
    exit 1
  }
elif command -v sha256sum >/dev/null 2>&1; then
  digest_output=$(sha256sum "$exe") || {
    echo "SHA-256 calculation failed" >&2
    exit 1
  }
else
  echo "shasum or sha256sum is required" >&2
  exit 1
fi
digest=$(printf '%s\n' "$digest_output" | awk '{print $1}')
printf '%s' "$digest" | rg -q '^[0-9a-fA-F]{64}$' || {
  echo "invalid SHA-256 output" >&2
  exit 1
}
printf 'sha256=%s\n' "$digest"
