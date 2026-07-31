#!/usr/bin/env python3
"""Create a character-neutral portable desktop pet source project."""

import argparse
import json
from pathlib import Path
import shutil
import tempfile
import unicodedata
from xml.sax.saxutils import escape


WINDOWS_FILENAME_CHARACTERS = frozenset('<>:"/\\|?*')
WINDOWS_RESERVED_BASENAMES = frozenset(
    {"CON", "PRN", "AUX", "NUL"}
    | {f"COM{number}" for number in range(1, 10)}
    | {f"LPT{number}" for number in range(1, 10)}
    | {f"COM{number}" for number in "\u00b9\u00b2\u00b3"}
    | {f"LPT{number}" for number in "\u00b9\u00b2\u00b3"}
)
TEXT_SUFFIXES = frozenset({".go", ".manifest", ".mod", ".py", ".sh"})


def sanitize_windows_filename(name: str) -> str:
    """Return a Windows-safe executable basename derived from the pet name."""
    if any(unicodedata.category(character) == "Cc" for character in name):
        raise SystemExit("name must not contain control characters")
    sanitized = "".join(
        "_" if character in WINDOWS_FILENAME_CHARACTERS else character
        for character in name
    ).rstrip(" .")
    if not sanitized:
        raise SystemExit("name must produce a non-empty Windows filename")
    basename = sanitized.split(".", 1)[0].upper()
    if basename in WINDOWS_RESERVED_BASENAMES:
        sanitized = f"_{sanitized}"
    return sanitized


def text_files(root: Path):
    """Yield deterministic, UTF-8 template text files."""
    for path in sorted(root.rglob("*")):
        if path.is_file() and path.suffix in TEXT_SUFFIXES:
            yield path


def escaped_title(path: Path, name: str) -> str:
    """Encode the visible title for the text context containing its placeholder."""
    title = f"{name}桌宠"
    if path.suffix == ".go":
        return json.dumps(title, ensure_ascii=False)[1:-1]
    if path.suffix == ".manifest":
        return escape(title, {'"': "&quot;", "'": "&apos;"})
    return title


def shell_double_quoted(value: str) -> str:
    """Escape a value inserted inside a POSIX shell double-quoted string."""
    return value.replace("\\", "\\\\").replace("$", "\\$").replace("`", "\\`")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Scaffold a portable Windows desktop pet source project."
    )
    parser.add_argument("--name", required=True)
    parser.add_argument("--atlas", type=Path, required=True)
    parser.add_argument("--icon", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    output_dir = args.output_dir
    atlas = args.atlas
    icon = args.icon
    exe_name = sanitize_windows_filename(args.name)

    if output_dir.is_symlink():
        raise SystemExit("output directory must not be a symlink")
    if output_dir.exists() and (
        not output_dir.is_dir() or any(output_dir.iterdir())
    ):
        raise SystemExit("output directory must be empty")
    if not atlas.is_file() or not icon.is_file():
        raise SystemExit("atlas and icon must be existing files")

    template_dir = (
        Path(__file__).resolve().parent.parent / "assets" / "windows-go-template"
    )
    output_dir.parent.mkdir(parents=True, exist_ok=True)
    staging_dir = Path(
        tempfile.mkdtemp(
            prefix=f".{output_dir.name}.staging.",
            dir=output_dir.parent,
        )
    )
    try:
        shutil.copytree(template_dir, staging_dir, dirs_exist_ok=True)
        runtime = staging_dir / "assets" / "runtime"
        runtime.mkdir(parents=True, exist_ok=True)
        shutil.copy2(atlas, runtime / "spritesheet.png")
        shutil.copy2(icon, runtime / "app-icon.png")

        for path in text_files(staging_dir):
            text = path.read_text(encoding="utf-8")
            text = text.replace("__PET_TITLE__", escaped_title(path, args.name))
            executable = (
                shell_double_quoted(exe_name) if path.suffix == ".sh" else exe_name
            )
            text = text.replace("__PET_EXE__", executable)
            path.write_text(text, encoding="utf-8")
        staging_dir.replace(output_dir)
    finally:
        if staging_dir.exists():
            shutil.rmtree(staging_dir)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
