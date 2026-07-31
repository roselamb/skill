#!/usr/bin/env python3
"""Check that the public README workflow example is safe to publish."""

from pathlib import Path


def main() -> None:
    asset = Path("assets/readme/reference-to-pet.svg")
    readme = Path("README.md")
    assert asset.is_file(), f"missing README asset: {asset}"
    assert asset.as_posix() in readme.read_text(encoding="utf-8"), (
        f"README does not reference {asset}"
    )
    forbidden = {".jpg", ".jpeg", ".png", ".webp", ".exe"}
    leaked = [
        path for path in Path("assets/readme").rglob("*") if path.suffix.lower() in forbidden
    ]
    assert not leaked, f"unsafe README assets found: {leaked}"


if __name__ == "__main__":
    main()
