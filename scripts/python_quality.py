#!/usr/bin/env python3
"""Provision or check the complete repository Python inventory with pinned tools.

This bootstrap deliberately parses on Python 3.9 to explain the 3.12 floor.
Operational scripts remain standard-library-only; tools live in a separate venv.
"""

from __future__ import annotations

import argparse
import hashlib
import importlib.metadata
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import sysconfig
import venv
from collections.abc import Iterable
from typing import NoReturn, TypedDict

ROOT = Path(__file__).resolve().parents[1]
ENVIRONMENT = Path(".cache/python-quality")
CONFIG_INPUTS = {
    "pyproject.toml",
    "requirements-quality.in",
    "requirements-quality.lock",
    "Makefile",
    ".gitignore",
    ".gitattributes",
}
EXCLUDED = {"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache"}


class Component(TypedDict):
    """A content identity with an explicit local implementation location."""

    path: str
    sha256: str


class Identity(TypedDict):
    """Versioned identity consumed by external validation receipt owners."""

    schema_version: int
    python_version: str
    interpreter: Component
    environment: Component
    stdlib: Component
    dependencies: Component
    distributions: dict[str, str]
    configuration: str
    lock: str
    helper: Component
    sha256: str


def require_modern_python() -> None:
    """Fail before provisioning or checking under an unsupported interpreter."""
    if sys.version_info < (3, 12):
        raise ValueError(
            "Python 3.12+ is required; run make python-tools "
            "PYTHON_BOOTSTRAP=python3.12 (or another installed modern Python). "
            "Checks use .cache/python-quality, not the ambient python3."
        )


def python_path(root: Path) -> Path:
    """Locate the environment interpreter on POSIX and Windows."""
    return (
        root / ENVIRONMENT / ("Scripts/python.exe" if os.name == "nt" else "bin/python")
    )


def git(root: Path, *arguments: str) -> bytes:
    """Run a read-only Git query without text/path quoting ambiguity."""
    return subprocess.run(
        ["git", "-C", str(root), *arguments], check=True, capture_output=True
    ).stdout


def relevant(name: str) -> bool:
    """Include Python and every repository input that can select its checks."""
    return (
        name.endswith(".py")
        or name in CONFIG_INPUTS
        or Path(name).name in {".gitignore", ".gitattributes"}
        or name.startswith(".githooks/")
        or name.startswith(".github/workflows/")
        or (name.startswith("scripts/") and name.endswith(".sh"))
    )


def safe_path(root: Path, name: str) -> Path:
    """Reject escaping paths and symlinks in every path component."""
    relative = Path(name)
    if relative.is_absolute() or ".." in relative.parts:
        raise ValueError(f"unsafe repository path: {name}")
    path = root
    for part in relative.parts:
        path /= part
        if path.is_symlink():
            raise ValueError(f"symlink is not a quality input: {name}")
    return path


def safe_file(root: Path, name: str) -> Path:
    """Require a regular repository input without symlink traversal."""
    path = safe_path(root, name)
    if not path.is_file():
        raise ValueError(f"quality input is not a regular file: {name}")
    return path


def inventory(root: Path) -> list[str]:
    """Enumerate tracked and nonignored new files, excluding deleted files."""
    names = {
        os.fsdecode(name)
        for name in git(
            root, "ls-files", "-z", "--cached", "--others", "--exclude-standard"
        ).split(b"\0")
        if name
    }
    result = []
    for name in sorted(names):
        path = root / name
        # Git lists a directory symlink as one entry, hiding Python below it.
        if path.is_symlink() and (path.is_dir() or not path.exists()):
            raise ValueError(f"directory or broken symlink in inventory: {name}")
        if not relevant(name):
            continue
        safe_path(root, name)
        if not path.exists() and not path.is_symlink():
            continue
        safe_file(root, name)
        result.append(name)
    return result


def staged_check(root: Path) -> None:
    """Require exact index bytes/modes for the complete relevant working inventory."""
    indexed: dict[str, tuple[str, str]] = {}
    for entry in git(root, "ls-files", "--stage", "-z").split(b"\0"):
        if not entry:
            continue
        header, raw_name = entry.split(b"\t", 1)
        name = os.fsdecode(raw_name)
        if not relevant(name):
            continue
        mode, blob, stage = header.decode("ascii").split()
        if stage != "0" or mode not in {"100644", "100755"}:
            raise ValueError(f"unmerged or unsupported staged quality input: {name}")
        indexed[name] = (mode, blob)
    working = set(inventory(root))
    if working != set(indexed):
        raise ValueError(
            "stage the complete Python/config/runner candidate before committing: "
            + ", ".join(sorted(working ^ set(indexed)))
        )
    for name, (mode, blob) in indexed.items():
        path = safe_file(root, name)
        content = git(root, "cat-file", "blob", blob)
        executable = bool(path.stat().st_mode & stat.S_IXUSR)
        if path.read_bytes() != content or (
            os.name != "nt" and executable != (mode == "100755")
        ):
            raise ValueError(
                f"staged/worktree quality input differs: {name}; "
                "stage the reviewed file"
            )


def provision(root: Path) -> None:
    """Create a dedicated venv and install only the reviewed hashed requirements."""
    require_modern_python()
    directory = root / ENVIRONMENT
    for path in (root / ".cache", directory):
        if path.is_symlink():
            raise ValueError(f"refusing symlinked tool environment: {path}")
    if directory.exists():
        raise ValueError(
            "tool environment already exists; remove .cache/python-quality "
            "deliberately "
            "before reprovisioning the lock"
        )
    safe_file(root, "requirements-quality.lock")
    venv.EnvBuilder(with_pip=True).create(directory)
    subprocess.run(
        [
            str(python_path(root)),
            "-I",
            "-m",
            "pip",
            "install",
            "--require-hashes",
            "--only-binary=:all:",
            "--disable-pip-version-check",
            "-r",
            str(root / "requirements-quality.lock"),
        ],
        check=True,
    )


def verify_environment(root: Path) -> None:
    """Refuse ambient Python, incomplete pins, and inherited import injection."""
    require_modern_python()
    expected = root / ENVIRONMENT
    if expected.is_symlink() or (root / ".cache").is_symlink():
        raise ValueError("tool environment must not be a symlink")
    if Path(sys.prefix).resolve() != expected.resolve():
        raise ValueError(
            "use .cache/python-quality/bin/python; provision with make python-tools"
        )
    safe_file(root, "pyproject.toml")
    safe_file(root, "requirements-quality.in")
    safe_file(root, str(ENVIRONMENT / "pyvenv.cfg"))
    packages = safe_path(
        root, str(Path(sysconfig.get_path("purelib")).relative_to(root))
    )
    for line in (
        safe_file(root, "requirements-quality.lock")
        .read_text(encoding="utf-8")
        .splitlines()
    ):
        match = re.match(r"([a-zA-Z0-9_-]+)==([^ ;\\]+)", line)
        if not match or ("sys_platform == 'win32'" in line and sys.platform != "win32"):
            continue
        name, version = match.groups()
        try:
            distribution = importlib.metadata.distribution(name)
        except importlib.metadata.PackageNotFoundError as error:
            raise ValueError(
                f"missing {name}; reprovision with make python-tools"
            ) from error
        if distribution.version != version:
            raise ValueError(
                f"{name} does not match the lock; reprovision with make python-tools"
            )
        if Path(distribution.locate_file("")).resolve() != packages:
            raise ValueError(f"{name} is outside the dedicated environment")
    for distribution in importlib.metadata.distributions():
        if Path(distribution.locate_file("")).resolve() != packages:
            raise ValueError("external installed packages are not quality inputs")


def run_tools(root: Path, action: str) -> None:
    """Run real tools on explicitly enumerated files; never provision during checks."""
    if action not in {"format", "format-check", "lint", "check"}:
        raise ValueError(f"unsupported Python tool action: {action}")
    verify_environment(root)
    files = [name for name in inventory(root) if name.endswith(".py")]
    if not files:
        raise ValueError("Python inventory is empty")
    config = str(safe_file(root, "pyproject.toml"))
    environment = dict(os.environ, PYTHONDONTWRITEBYTECODE="1")
    if action in {"format", "format-check", "check"}:
        subprocess.run(
            [
                sys.executable,
                "-I",
                "-B",
                "-m",
                "black",
                "--config",
                config,
                "--no-cache",
                *([] if action == "format" else ["--check", "--diff"]),
                "--",
                *files,
            ],
            cwd=root,
            env=environment,
            check=True,
        )
    if action in {"lint", "check"}:
        run_pylint(root, config, files, environment)


def run_pylint(
    root: Path, config: str, files: list[str], environment: dict[str, str]
) -> None:
    """Reject diagnostics even when Pylint exits zero for configuration errors."""
    command = [
        sys.executable,
        "-I",
        "-B",
        "-m",
        "pylint",
        "--rcfile",
        config,
        "--score=no",
        "--reports=no",
        "--output-format=json",
        "--",
        *files,
    ]
    result = subprocess.run(
        command, cwd=root, env=environment, capture_output=True, text=True, check=False
    )
    if result.stderr:
        print(result.stderr, end="", file=sys.stderr)
    try:
        messages = json.loads(result.stdout)
    except ValueError as error:
        print(result.stdout, end="", file=sys.stderr)
        raise ValueError("Pylint did not produce a valid diagnostic payload") from error
    if not isinstance(messages, list):
        raise ValueError("Pylint diagnostic payload must be a list")
    if messages or result.returncode:
        print(result.stdout, end="", file=sys.stderr)
        raise subprocess.CalledProcessError(result.returncode or 1, command)


def file_digest(path: Path) -> str:
    """Hash implementation bytes, not distribution metadata claims."""
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def tree_digest(directory: Path, excluded: frozenset[str] = frozenset()) -> str:
    """Hash names, modes, link targets and contents without bytecode or caches."""
    entries: list[tuple[str, int, str, str]] = []
    for parent, directories, filenames in os.walk(
        directory, followlinks=False, onerror=walk_error
    ):
        directories[:] = sorted(
            name for name in directories if name not in EXCLUDED | set(excluded)
        )
        base = Path(parent)
        for name in directories:
            if (base / name).is_symlink():
                raise ValueError(
                    f"unsupported implementation directory symlink: {base / name}"
                )
        for name in sorted(filenames):
            if name.endswith((".pyc", ".pyo")):
                continue
            path = base / name
            if not path.is_file():
                raise ValueError(f"unreadable implementation file: {path}")
            entries.append(
                (
                    str(path.relative_to(directory)),
                    stat.S_IMODE(path.stat().st_mode),
                    os.readlink(path) if path.is_symlink() else "",
                    file_digest(path),
                )
            )
    if not entries:
        raise ValueError(f"empty implementation directory: {directory}")
    return hash_records(entries)


def walk_error(error: OSError) -> NoReturn:
    """Never turn an unreadable implementation directory into partial identity."""
    raise error


def hash_records(records: object) -> str:
    """Produce an unambiguous deterministic digest of JSON-compatible records."""
    return hashlib.sha256(
        json.dumps(records, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest()


def configuration_digest(root: Path, names: Iterable[str]) -> str:
    """Bind repository runner and configuration bytes used by Python validation."""
    return hash_records(
        [
            (
                name,
                stat.S_IMODE(safe_file(root, name).stat().st_mode),
                file_digest(safe_file(root, name)),
            )
            for name in sorted(names)
        ]
    )


def identity(root: Path) -> Identity:
    """Describe actual interpreter, stdlib, installed code, lock and configuration."""
    verify_environment(root)
    executable = Path(sys.executable).resolve(strict=True)
    stdlib = Path(sysconfig.get_path("stdlib")).resolve(strict=True)
    packages = Path(sysconfig.get_path("purelib")).resolve(strict=True)
    interpreter_files = [executable]
    library = Path(sysconfig.get_config_var("LIBDIR") or executable.parent) / (
        sysconfig.get_config_var("LDLIBRARY") or "missing"
    )
    if library.is_file() and library.resolve() != executable:
        interpreter_files.append(library.resolve())
    # CPython can also import its stdlib from a zip beside the ordinary tree.
    interpreter_files.extend(
        Path(path)
        for path in sys.path
        if path.endswith(".zip") and Path(path).is_file()
    )
    result: Identity = {
        "schema_version": 1,
        "python_version": sys.version,
        "environment": {
            "path": str(root / ENVIRONMENT / "pyvenv.cfg"),
            "sha256": file_digest(safe_file(root, str(ENVIRONMENT / "pyvenv.cfg"))),
        },
        "interpreter": {
            "path": str(executable),
            "sha256": hash_records(
                [(str(path), file_digest(path)) for path in interpreter_files]
            ),
        },
        "stdlib": {
            "path": str(stdlib),
            "sha256": tree_digest(
                stdlib, frozenset({"site-packages", "dist-packages"})
            ),
        },
        "dependencies": {"path": str(packages), "sha256": tree_digest(packages)},
        "distributions": {
            item.metadata["Name"]: item.version
            for item in importlib.metadata.distributions()
        },
        "configuration": configuration_digest(root, inventory(root)),
        "lock": file_digest(safe_file(root, "requirements-quality.lock")),
        "helper": {
            "path": str(Path(__file__).resolve()),
            "sha256": file_digest(Path(__file__)),
        },
        "sha256": "",
    }
    result["sha256"] = hash_records(
        {key: value for key, value in result.items() if key != "sha256"}
    )
    return result


def main() -> int:
    """Dispatch the public helper CLI and preserve failing tool exit codes."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "action",
        choices=(
            "provision",
            "ready",
            "format",
            "format-check",
            "lint",
            "check",
            "staged-check",
            "identity",
        ),
    )
    args = parser.parse_args()
    try:
        require_modern_python()
        if args.action == "provision":
            provision(ROOT)
        elif args.action == "ready":
            verify_environment(ROOT)
        elif args.action == "staged-check":
            staged_check(ROOT)
        elif args.action == "identity":
            print(json.dumps(identity(ROOT), sort_keys=True))
        else:
            run_tools(ROOT, args.action)
    except subprocess.CalledProcessError as error:
        return error.returncode if error.returncode > 0 else 128 - error.returncode
    except (OSError, ValueError, importlib.metadata.PackageNotFoundError) as error:
        print(f"Python quality: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
