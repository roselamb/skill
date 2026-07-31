#!/usr/bin/env python3
"""Regression tests for the final delivery hardening pass."""

from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import unittest


SKILL_DIR = Path(__file__).resolve().parent.parent
SCAFFOLD = SKILL_DIR / "scripts" / "scaffold_pet.py"
VERIFY = SKILL_DIR / "scripts" / "verify-delivery.sh"
CREATE_ASSETS = SKILL_DIR / "scripts" / "create_test_assets.py"


def run(*args: object, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [os.fspath(arg) for arg in args],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
        check=False,
    )


def write_executable(path: Path, source: str) -> None:
    path.write_text(source, encoding="utf-8")
    path.chmod(path.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)


class ScaffoldTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="portable-pet-scaffold-test.")
        self.root = Path(self.temporary.name)
        self.assets = self.root / "assets"
        result = run(sys.executable, CREATE_ASSETS, self.assets)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.atlas = self.assets / "atlas.png"
        self.icon = self.assets / "icon.png"

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def scaffold(self, name: str, output: Path, script: Path = SCAFFOLD) -> subprocess.CompletedProcess[str]:
        return run(
            sys.executable,
            script,
            "--name",
            name,
            "--atlas",
            self.atlas,
            "--icon",
            self.icon,
            "--output-dir",
            output,
        )

    def test_rejects_symlink_output_without_populating_target(self) -> None:
        target = self.root / "empty-target"
        target.mkdir()
        output = self.root / "project"
        output.symlink_to(target, target_is_directory=True)

        result = self.scaffold("TestPet", output)

        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(output.is_symlink())
        self.assertEqual(list(target.iterdir()), [])

    def test_mid_generation_failure_leaves_requested_output_untouched(self) -> None:
        copied_skill = self.root / "skill-copy"
        shutil.copytree(SKILL_DIR, copied_skill)
        corrupt = copied_skill / "assets" / "windows-go-template" / "internal" / "app" / "logic.go"
        corrupt.write_bytes(b"\xff")
        output = self.root / "project"

        result = self.scaffold(
            "TestPet",
            output,
            copied_skill / "scripts" / "scaffold_pet.py",
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(output.exists())
        self.assertFalse(output.is_symlink())

    def test_superscript_device_names_are_made_safe(self) -> None:
        for name in ("COM\u00b9", "COM\u00b2", "COM\u00b3", "LPT\u00b9", "LPT\u00b2", "LPT\u00b3"):
            with self.subTest(name=name):
                output = self.root / name
                result = self.scaffold(name, output)
                self.assertEqual(result.returncode, 0, result.stderr)
                build = (output / "scripts" / "build-windows.sh").read_text(encoding="utf-8")
                self.assertNotIn(f'outputs/{name}.exe', build)


class VerifierTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="portable-pet-verifier-test.")
        self.root = Path(self.temporary.name)
        self.fake_bin = self.root / "bin"
        self.fake_bin.mkdir()
        self.exe = self.root / "pet.exe"
        self.exe.write_bytes(b"MZ synthetic fixture")
        self.project = self.root / "project"
        self.project.mkdir()
        self.env = os.environ.copy()
        self.env["PATH"] = os.pathsep.join((os.fspath(self.fake_bin), self.env["PATH"]))
        self.env["FAKE_PROJECT_ROOT"] = os.fspath(self.project.resolve())
        self._write_tools()

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def _write_tools(self) -> None:
        write_executable(
            self.fake_bin / "file",
            """#!/bin/sh
printf '%s: PE32+ executable (GUI) x86-64, for MS Windows\\n' "$1"
""",
        )
        write_executable(
            self.fake_bin / "objdump",
            """#!/bin/sh
mode=${FAKE_OBJDUMP_MODE:-valid}
case "$1" in
  -p)
    printf 'Subsystem 00000002 (Windows GUI)\\n'
    case "$mode" in
      empty) ;;
      unexpected) printf 'DLL Name: EVIL.dll\\n' ;;
      *) printf 'DLL Name: KERNEL32.dll\\nDLL Name: USER32.dll\\n' ;;
    esac
    ;;
  -h) printf '  3 .rsrc 00000100\\n' ;;
  *) exit 2 ;;
esac
[ "$mode" != fail ] || exit 9
""",
        )
        write_executable(
            self.fake_bin / "strings",
            """#!/bin/sh
case ${FAKE_STRINGS_MODE:-clean} in
  fail) exit 8 ;;
  leak) printf '%s/leaked/source.go\\n' "$FAKE_PROJECT_ROOT" ;;
  *) printf 'portable-pet\\n' ;;
esac
""",
        )
        write_executable(
            self.fake_bin / "shasum",
            """#!/bin/sh
printf '0000000000000000000000000000000000000000000000000000000000000000  %s\\n' "$3"
""",
        )

    def verify(self, *extra: object, **settings: str) -> subprocess.CompletedProcess[str]:
        env = self.env.copy()
        env.update(settings)
        return run(VERIFY, self.exe, *extra, env=env)

    def test_requires_absolute_project_root(self) -> None:
        self.assertNotEqual(self.verify().returncode, 0)
        self.assertNotEqual(self.verify("relative/project").returncode, 0)

    def test_fails_when_objdump_fails(self) -> None:
        self.assertNotEqual(
            self.verify(self.project, FAKE_OBJDUMP_MODE="fail").returncode,
            0,
        )

    def test_fails_when_import_list_is_empty(self) -> None:
        self.assertNotEqual(
            self.verify(self.project, FAKE_OBJDUMP_MODE="empty").returncode,
            0,
        )

    def test_fails_when_strings_fails(self) -> None:
        self.assertNotEqual(
            self.verify(self.project, FAKE_STRINGS_MODE="fail").returncode,
            0,
        )

    def test_rejects_unexpected_import(self) -> None:
        self.assertNotEqual(
            self.verify(self.project, FAKE_OBJDUMP_MODE="unexpected").returncode,
            0,
        )

    def test_rejects_canonical_project_root_leak(self) -> None:
        alias = self.root / "project-alias"
        alias.symlink_to(self.project, target_is_directory=True)
        self.assertNotEqual(
            self.verify(alias, FAKE_STRINGS_MODE="leak").returncode,
            0,
        )

    def test_accepts_valid_fixture_and_prints_sha256(self) -> None:
        result = self.verify(self.project)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(
            result.stdout.strip(),
            "sha256=" + "0" * 64,
        )

    def test_uses_sha256sum_when_shasum_is_unavailable(self) -> None:
        portable_bin = self.root / "portable-bin"
        shutil.copytree(self.fake_bin, portable_bin)
        (portable_bin / "shasum").unlink()
        write_executable(
            portable_bin / "sha256sum",
            """#!/bin/sh
printf '1111111111111111111111111111111111111111111111111111111111111111  %s\\n' "$1"
""",
        )
        for command in ("awk", "tr", "rg"):
            (portable_bin / command).symlink_to(shutil.which(command))
        env = self.env.copy()
        env["PATH"] = os.fspath(portable_bin)

        result = run(VERIFY, self.exe, self.project, env=env)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), "sha256=" + "1" * 64)


class BuildHostTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="portable-pet-host-test.")
        self.root = Path(self.temporary.name)
        self.project = self.root / "project"
        shutil.copytree(
            SKILL_DIR / "assets" / "windows-go-template",
            self.project,
        )
        self.fake_bin = self.root / "bin"
        self.fake_bin.mkdir()
        self.env = os.environ.copy()
        self.env["PATH"] = os.pathsep.join((os.fspath(self.fake_bin), self.env["PATH"]))
        self._write_download_fixture()

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def _write_download_fixture(self) -> None:
        payload = self.root / "payload"
        go_root = payload / "go"
        (go_root / "bin").mkdir(parents=True)
        write_executable(go_root / "bin" / "go", "#!/bin/sh\nexit 0\n")
        (go_root / "VERSION").write_text("go9.9.9\n", encoding="utf-8")
        archive = self.root / "go9.9.9.linux-amd64.tar.gz"
        with tarfile.open(archive, "w:gz") as destination:
            destination.add(go_root, arcname="go")
        manifest = self.root / "manifest.json"
        manifest.write_text(
            '[{"version":"go9.9.9","stable":true,"files":['
            '{"os":"linux","arch":"amd64","kind":"archive",'
            '"filename":"go9.9.9.linux-amd64.tar.gz","sha256":"' + "a" * 64 + '"}'
            "]}]",
            encoding="utf-8",
        )
        self.env["FAKE_GO_ARCHIVE"] = os.fspath(archive)
        self.env["FAKE_GO_MANIFEST"] = os.fspath(manifest)
        write_executable(
            self.fake_bin / "curl",
            """#!/bin/sh
output=
previous=
for argument in "$@"; do
  if [ "$previous" = -o ]; then output=$argument; fi
  previous=$argument
done
case "$*" in
  *mode=json*) cp "$FAKE_GO_MANIFEST" "$output" ;;
  *) cp "$FAKE_GO_ARCHIVE" "$output" ;;
esac
""",
        )
        write_executable(
            self.fake_bin / "shasum",
            "#!/bin/sh\nprintf '%s  %s\\n' '" + "a" * 64 + "' \"$3\"\n",
        )

    def write_uname(self, system: str, machine: str = "x86_64") -> None:
        write_executable(
            self.fake_bin / "uname",
            f"""#!/bin/sh
case "$1" in
  -s) printf '%s\\n' '{system}' ;;
  -m) printf '%s\\n' '{machine}' ;;
  *) printf '%s\\n' '{system}' ;;
esac
""",
        )

    def test_fetches_linux_amd64_toolchain(self) -> None:
        self.write_uname("Linux")
        result = run(
            self.project / "scripts" / "fetch-go-toolchain.sh",
            env=self.env,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        runtime = Path(result.stdout.strip())
        self.assertIn("linux-amd64", runtime.name)
        self.assertTrue((runtime / "bin" / "go").is_file())

    def test_rejects_unknown_host_before_downloading(self) -> None:
        self.write_uname("Plan9")
        result = run(
            self.project / "scripts" / "fetch-go-toolchain.sh",
            env=self.env,
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unsupported build host", result.stderr)


class SelfTestChainTests(unittest.TestCase):
    def test_self_test_runs_real_build_and_verifier_entrypoints(self) -> None:
        with tempfile.TemporaryDirectory(prefix="portable-pet-chain-test.") as temporary:
            root = Path(temporary)
            copied_skill = root / "skill"
            shutil.copytree(SKILL_DIR, copied_skill)
            build_marker = root / "build-called"
            verify_marker = root / "verify-called"
            write_executable(
                copied_skill
                / "assets"
                / "windows-go-template"
                / "scripts"
                / "build-windows.sh",
                f"""#!/bin/sh
set -eu
touch '{build_marker}'
mkdir -p outputs
printf 'MZ synthetic build' > 'outputs/测试宠物.exe'
""",
            )
            write_executable(
                copied_skill / "scripts" / "verify-delivery.sh",
                f"""#!/bin/sh
set -eu
touch '{verify_marker}'
printf 'sha256=%s\\n' '{"0" * 64}'
""",
            )
            fake_bin = root / "bin"
            fake_bin.mkdir()
            write_executable(
                fake_bin / "objdump",
                "#!/bin/sh\nprintf '  3 .rsrc 00000100\\n'\n",
            )
            env = os.environ.copy()
            env.update(
                {
                    "PATH": os.pathsep.join((os.fspath(fake_bin), env["PATH"])),
                    "PORTABLE_PET_SKIP_REGRESSION_TESTS": "1",
                    "PORTABLE_PET_TEST_GO": "/usr/bin/true",
                    "WORKSPACE_PYTHON": sys.executable,
                }
            )

            result = run(copied_skill / "scripts" / "self-test.sh", env=env)

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(build_marker.is_file(), "self-test skipped build-windows.sh")
            self.assertTrue(verify_marker.is_file(), "self-test skipped verify-delivery.sh")


if __name__ == "__main__":
    unittest.main(verbosity=2)
