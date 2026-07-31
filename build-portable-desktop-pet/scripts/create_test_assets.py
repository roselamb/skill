#!/usr/bin/env python3
"""Create deterministic RGBA PNG fixtures for the installed Skill self-test."""

import struct
import sys
from pathlib import Path
import zlib


PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
TRANSPARENT = b"\x00\x00\x00\x00"


def chunk(kind: bytes, data: bytes) -> bytes:
    checksum = zlib.crc32(kind + data) & 0xFFFFFFFF
    return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", checksum)


def write_rgba_png(path: Path, width: int, height: int, rows: list[bytes]) -> None:
    raw = b"".join(b"\x00" + row for row in rows)
    header = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)
    path.write_bytes(
        PNG_SIGNATURE
        + chunk(b"IHDR", header)
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b"")
    )


def atlas_rows() -> list[bytes]:
    opaque = b"\x52\xa8\xff\xff"
    clear_row = TRANSPARENT * 1536
    body_row = (TRANSPARENT * 8 + opaque * 176 + TRANSPARENT * 8) * 8
    return [
        clear_row if y % 208 < 8 or y % 208 >= 200 else body_row
        for y in range(2288)
    ]


def icon_rows() -> list[bytes]:
    opaque = b"\xff\xa8\x52\xff"
    clear_row = TRANSPARENT * 256
    body_row = TRANSPARENT * 16 + opaque * 224 + TRANSPARENT * 16
    return [clear_row if y < 16 or y >= 240 else body_row for y in range(256)]


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: create_test_assets.py OUTPUT_DIR")
    output_dir = Path(sys.argv[1])
    output_dir.mkdir(parents=True, exist_ok=True)
    write_rgba_png(output_dir / "atlas.png", 1536, 2288, atlas_rows())
    write_rgba_png(output_dir / "icon.png", 256, 256, icon_rows())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
