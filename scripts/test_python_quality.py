"""Real formatter/linter failures, complete scope and check-only hook regressions."""

# Test names state the assertion; fixtures are deliberately small isolated repositories.
# pylint: disable=missing-function-docstring

import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import python_quality as quality


class PythonQualityTests(unittest.TestCase):
    """Verify failure propagation and staged candidate integrity with real tools."""

    def setUp(self) -> None:
        # unittest owns cleanup across the whole test, including assertion failures.
        # pylint: disable=consider-using-with
        self.root = Path(self.enterContext(tempfile.TemporaryDirectory())).resolve()
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)
        shutil.copyfile(quality.ROOT / "pyproject.toml", self.root / "pyproject.toml")
        self.write(".gitignore", ".cache/\n__pycache__/\n")

    def write(self, name: str, text: str) -> Path:
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
        return path

    def stage(self) -> None:
        subprocess.run(["git", "-C", str(self.root), "add", "--all"], check=True)

    def test_real_black_and_pylint_fail_without_changing_source(self) -> None:
        path = self.write("sample.py", '"""A formatter failure fixture."""\nVALUE=1\n')
        before = path.read_bytes()
        # Replace only environment location; subprocesses invoke actual pinned tools.
        with patch.object(quality, "verify_environment"):
            with self.assertRaises(subprocess.CalledProcessError):
                quality.run_tools(self.root, "format-check")
        self.assertEqual(path.read_bytes(), before)
        path.write_text(
            '"""A linter failure fixture."""\nprint(missing_name)\n', encoding="utf-8"
        )
        before = path.read_bytes()
        with patch.object(quality, "verify_environment"):
            with self.assertRaises(subprocess.CalledProcessError):
                quality.run_tools(self.root, "lint")
        self.assertEqual(path.read_bytes(), before)
        path.write_text(
            '"""A stale suppression must also fail."""\n'
            "# pylint: disable=unused-import\nVALUE = 1\n",
            encoding="utf-8",
        )
        with patch.object(quality, "verify_environment"):
            with self.assertRaises(subprocess.CalledProcessError):
                quality.run_tools(self.root, "lint")
        path.write_text(
            '"""A clean control fixture."""\n\nVALUE = 1\n', encoding="utf-8"
        )
        with patch.object(quality, "verify_environment"):
            quality.run_tools(self.root, "check")

    def test_hidden_python_is_not_silently_excluded_by_pylint(self) -> None:
        self.write(
            ".hidden.py", '"""Hidden file must still be checked."""\nprint(missing)\n'
        )
        with patch.object(quality, "verify_environment"):
            with self.assertRaises(subprocess.CalledProcessError):
                quality.run_tools(self.root, "lint")

    def test_real_pylint_config_diagnostic_fails_even_with_zero_exit(self) -> None:
        self.write("sample.py", '"""Clean source with invalid tool configuration."""\n')
        config = self.root / "pyproject.toml"
        config.write_text(
            config.read_text(encoding="utf-8")
            + "\n[tool.pylint.invalid]\nnonexistent-option = true\n",
            encoding="utf-8",
        )
        with patch.object(quality, "verify_environment"):
            with self.assertRaises(subprocess.CalledProcessError):
                quality.run_tools(self.root, "lint")

    def test_pylint_missing_or_malformed_payload_fails_closed(self) -> None:
        for output in ("", "not json", "{}"):
            with self.subTest(output=output):
                result = subprocess.CompletedProcess(["pylint"], 0, output, "")
                with patch.object(quality.subprocess, "run", return_value=result):
                    with self.assertRaises(ValueError):
                        quality.run_pylint(
                            self.root, "pyproject.toml", ["sample.py"], {}
                        )

    def test_identity_binds_actual_helper_and_aggregate(self) -> None:
        result = quality.identity(quality.ROOT)
        helper = Path(quality.__file__)
        self.assertEqual(
            result["helper"],
            {"path": str(helper.resolve()), "sha256": quality.file_digest(helper)},
        )
        self.assertEqual(
            result["sha256"],
            quality.hash_records(
                {key: value for key, value in result.items() if key != "sha256"}
            ),
        )

    def test_inventory_includes_new_and_tracked_ignored_python(self) -> None:
        self.write("tracked.py", "VALUE = 1\n")
        self.stage()
        self.write(".gitignore", ".cache/\ntracked.py\n")
        self.write("new dir/untracked.py", "VALUE = 2\n")
        self.write(".cache/ignored.py", "invalid ignored source\n")
        files = quality.inventory(self.root)
        self.assertIn("tracked.py", files)
        self.assertIn("new dir/untracked.py", files)
        self.assertNotIn(".cache/ignored.py", files)

    def test_python_and_directory_symlinks_are_rejected(self) -> None:
        target = self.write("original.txt", "VALUE = 1\n")
        link = self.root / "alias.py"
        link.symlink_to(target)
        with self.assertRaisesRegex(ValueError, "symlink"):
            quality.inventory(self.root)
        link.unlink()
        directory = self.root / "alias"
        directory.symlink_to(self.root, target_is_directory=True)
        with self.assertRaisesRegex(ValueError, "symlink"):
            quality.inventory(self.root)

    def test_staged_parity_covers_config_scripts_new_files_and_deletions(self) -> None:
        path = self.write("sample.py", "VALUE = 1\n")
        runner = self.write("scripts/quality-gate.sh", "#!/bin/sh\nexit 0\n")
        ignore = self.write("nested/.gitignore", "ignored/\n")
        self.stage()
        quality.staged_check(self.root)
        for candidate in (path, runner, ignore, self.root / "pyproject.toml"):
            before = candidate.read_bytes()
            candidate.write_bytes(before + b"\n")
            with self.assertRaisesRegex(ValueError, "differs"):
                quality.staged_check(self.root)
            self.assertEqual(candidate.read_bytes(), before + b"\n")
            candidate.write_bytes(before)
        path.unlink()
        with self.assertRaisesRegex(ValueError, "complete"):
            quality.staged_check(self.root)
        self.stage()
        quality.staged_check(self.root)
        self.write("new.py", "VALUE = 2\n")
        with self.assertRaisesRegex(ValueError, "complete"):
            quality.staged_check(self.root)

    @unittest.skipIf(os.name == "nt", "POSIX executable-bit parity")
    def test_staged_executable_mode_changes_fail(self) -> None:
        path = self.write("sample.py", "VALUE = 1\n")
        self.stage()
        path.chmod(0o755)
        with self.assertRaisesRegex(ValueError, "differs"):
            quality.staged_check(self.root)

    def test_implementation_identity_tracks_code_but_ignores_bytecode(self) -> None:
        implementation = self.root / "implementation"
        code = self.write("implementation/tool.py", "VALUE = 1\n")
        self.write("implementation/tool.dist-info/METADATA", "Version: 1\n")
        original = quality.tree_digest(implementation)
        self.write("implementation/__pycache__/tool.pyc", "cache bytes")
        self.assertEqual(quality.tree_digest(implementation), original)
        code.write_text("VALUE = 2\n", encoding="utf-8")
        self.assertNotEqual(quality.tree_digest(implementation), original)

    def test_old_bootstrap_has_actionable_error(self) -> None:
        with patch.object(sys, "version_info", (3, 9, 6)):
            with self.assertRaisesRegex(ValueError, "PYTHON_BOOTSTRAP=python3.12"):
                quality.require_modern_python()

    @unittest.skipIf(os.name == "nt", "native hook uses a POSIX shell")
    def test_native_hook_owns_one_check_only_route_and_rejects_partial_stage(
        self,
    ) -> None:
        for name in (
            ".githooks/pre-commit",
            "scripts/pre-commit-checks.sh",
            "scripts/python_quality.py",
        ):
            target = self.root / name
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(quality.ROOT / name, target)
        script = self.root / "scripts/pre-commit-checks.sh"
        script.chmod(0o755)
        sample = self.write("sample.py", "VALUE = 1\n")
        # Stub only unrelated heavy Go/Make operations; parity uses the real helper.
        make = self.write(
            "fake-bin/make",
            '#!/bin/sh\nprintf "%s\\n" "$*" >> "$CALLS"\n'
            'if [ "$1" = python-staged-check ]; then\n'
            '  exec "$TEST_PYTHON" -I -B scripts/python_quality.py staged-check\nfi\n',
        )
        go = self.write("fake-bin/go", "#!/bin/sh\nexit 0\n")
        make.chmod(0o755)
        go.chmod(0o755)
        self.stage()
        log = self.root / "calls.txt"
        environment = dict(
            os.environ,
            PATH=str(make.parent) + os.pathsep + os.environ["PATH"],
            CALLS=str(log),
            TEST_PYTHON=sys.executable,
        )
        result = subprocess.run(
            ["bash", ".githooks/pre-commit"],
            cwd=self.root,
            env=environment,
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        calls = log.read_text(encoding="utf-8").splitlines()
        self.assertEqual(calls.count("format-check lint"), 1)
        self.assertEqual(calls.count("python-staged-check"), 2)
        self.assertNotIn("format", calls)
        sample.write_text("VALUE = 2\n", encoding="utf-8")
        index = quality.git(self.root, "ls-files", "--stage", "-z")
        log.unlink()
        result = subprocess.run(
            ["bash", ".githooks/pre-commit"],
            cwd=self.root,
            env=environment,
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("staged/worktree", result.stderr)
        self.assertEqual(
            log.read_text(encoding="utf-8").splitlines(), ["python-staged-check"]
        )
        self.assertEqual(sample.read_text(encoding="utf-8"), "VALUE = 2\n")
        self.assertEqual(quality.git(self.root, "ls-files", "--stage", "-z"), index)


if __name__ == "__main__":
    unittest.main()
