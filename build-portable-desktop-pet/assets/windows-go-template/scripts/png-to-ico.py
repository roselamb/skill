#!/usr/bin/env python3
"""Convert the approved PNG icon into a deterministic multi-size ICO."""

from pathlib import Path
import sys

from PIL import Image


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: png-to-ico.py INPUT.png OUTPUT.ico", file=sys.stderr)
        return 2

    source = Path(sys.argv[1])
    destination = Path(sys.argv[2])
    if not source.is_file():
        print(f"missing icon source: {source}", file=sys.stderr)
        return 1

    destination.parent.mkdir(parents=True, exist_ok=True)
    with Image.open(source) as image:
        rgba = image.convert("RGBA")
        rgba.save(
            destination,
            format="ICO",
            sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)],
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
