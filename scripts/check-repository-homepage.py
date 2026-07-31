#!/usr/bin/env python3
"""Check that the public README workflow example is safe to publish."""

from pathlib import Path


def main() -> None:
    asset = Path("assets/readme/reference-to-pet.svg")
    readme = Path("README.md")
    assert asset.is_file(), f"missing README asset: {asset}"
    text = readme.read_text(encoding="utf-8")
    required = {
        "public positioning": "portable single-file Windows desktop pets",
        "bilingual quick start": "中文快速开始",
        "target platform": "Windows 10/11 x64",
        "scaffold command": "scripts/scaffold_pet.py",
        "self-test command": "scripts/self-test.sh",
        "delivery verifier": "scripts/verify-delivery.sh",
        "README example": asset.as_posix(),
        "license label": "MIT License",
        "real-machine boundary": "Real-machine validation on Windows 10/11 remains separate",
    }
    missing = [label for label, needle in required.items() if needle not in text]
    assert not missing, f"README is missing required content: {', '.join(missing)}"
    assert "[MIT License](LICENSE)" in text, "README does not link to LICENSE"
    license_text = Path("LICENSE").read_text(encoding="utf-8")
    assert "MIT License" in license_text, "LICENSE is not an MIT license"
    assert "Copyright (c) 2026 roselamb" in license_text, "LICENSE copyright is incorrect"
    forbidden = {".jpg", ".jpeg", ".png", ".webp", ".exe"}
    leaked = [
        path for path in Path("assets/readme").rglob("*") if path.suffix.lower() in forbidden
    ]
    assert not leaked, f"unsafe README assets found: {leaked}"


if __name__ == "__main__":
    main()
