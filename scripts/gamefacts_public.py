"""Pure public Game Facts schema, parsing, and dependency-graph API (Python 3.12+).

Section text and preambles are returned verbatim so callers can fingerprint them.
This module has no workspace, evidence, Git, or review-state dependencies.
load_catalog validates the public graph; validate_repository additionally checks
the application checkout's catalog completeness, locators, and ref: annotations.
"""

import json
import re
from pathlib import Path
from typing import NotRequired, TypedDict, cast
from urllib.parse import unquote, urlsplit

ID = re.compile(r"[A-Z][A-Z0-9]*-\d{4}")
HEADING = re.compile(r"### ([A-Z][A-Z0-9]*-\d{4})[ \t]*$")
MAX_RULES = 2048
MAX_DEPTH = 20
ORIGINAL_STATES = {"Established", "Incomplete", "Conflicting"}
REDUX_STATES = {
    "Matches",
    "Intentional deviation",
    "Known discrepancy",
    "Unverified",
    "Not implemented",
}


class Rule(TypedDict):
    """Verbatim rule and its validated graph edges, optionally catalog-located."""

    text: str
    kind: str
    uses: list[str]
    document: NotRequired[str]


class Document(TypedDict):
    """Verbatim preamble and ordered rules for one public topic."""

    preamble: str
    rules: dict[str, Rule]


class Catalog(TypedDict):
    """Public catalog schema; contents are validated when loaded."""

    schema_version: int
    documents: list[str]


type JSONValue = None | bool | int | float | str | list[JSONValue] | dict[
    str, JSONValue
]


def require(condition: object, message: str) -> None:
    """Raise the public validation error when a condition is false."""
    if not condition:
        raise ValueError(message)


def within(root: Path, relative: str) -> Path:
    """Resolve a repository-relative locator without escaping its root."""
    require(isinstance(relative, str) and relative, "path must be a nonempty string")
    require(
        not Path(relative).is_absolute(),
        f"absolute path is not a repository locator: {relative}",
    )
    path = (root / relative).resolve()
    require(path.is_relative_to(root.resolve()), f"path escapes repository: {relative}")
    return path


def decode_json(text: str) -> JSONValue:
    """Decode JSON while rejecting ambiguous duplicate object keys."""

    def unique(pairs: list[tuple[str, JSONValue]]) -> dict[str, JSONValue]:
        result: dict[str, JSONValue] = {}
        for key, value in pairs:
            require(key not in result, f"duplicate JSON key: {key}")
            result[key] = value
        return result

    return json.loads(text, object_pairs_hook=unique)


def field(text: str, name: str, required: bool = True) -> str:
    """Read a unique public declaration, treating omitted optional fields as none."""
    values = re.findall(
        r"^\*\*" + re.escape(name) + r":\*\*[ \t]*([^\n]+)$", text, re.MULTILINE
    )
    require(len(values) <= 1, f"duplicate {name} field")
    require(bool(values) or not required, f"missing {name} field")
    return values[0].strip() if values else "none"


def parse_public(
    text: str, validate_relationships: bool = True
) -> tuple[str, dict[str, Rule]]:
    """Parse declarations while retaining exact section bytes for consumers."""
    # One pass preserves fence/offset state; splitting it obscures verbatim boundaries.
    # pylint: disable=too-many-locals
    starts = []
    offset = 0
    fenced = False
    for line in text.splitlines(keepends=True):
        if line.startswith("```"):
            fenced = not fenced
        if not fenced:
            match = HEADING.fullmatch(line.rstrip("\r\n"))
            if re.match(r"^###\s", line):
                require(
                    match is not None, f"malformed public ID heading: {line.strip()}"
                )
            if match:
                starts.append((match.group(1), offset))
        offset += len(line)
    require(not fenced, "unclosed code fence")
    require(
        0 < len(starts) <= MAX_RULES,
        "public document has no rules or exceeds the catalog limit",
    )
    rules: dict[str, Rule] = {}
    for index, (identifier, start) in enumerate(starts):
        require(identifier not in rules, f"duplicate public ID: {identifier}")
        end = starts[index + 1][1] if index + 1 < len(starts) else len(text)
        section = text[start:end]
        kind = field(section, "Kind")
        require(kind in {"rule", "group"}, f"{identifier}: invalid Kind")
        uses_text = field(section, "Uses")
        uses = (
            []
            if uses_text == "none"
            else [value.strip() for value in uses_text.split(",")]
        )
        require(
            all(ID.fullmatch(value) for value in uses),
            f"{identifier}: malformed dependency ID",
        )
        require(len(uses) == len(set(uses)), f"{identifier}: duplicate dependency")
        if kind == "rule":
            require(
                field(section, "Original") in ORIGINAL_STATES,
                f"{identifier}: invalid Original state",
            )
            require(
                field(section, "Redux") in REDUX_STATES,
                f"{identifier}: invalid Redux state",
            )
        rules[identifier] = {"text": section, "kind": kind, "uses": uses}

    if validate_relationships:
        validate_graph(rules)
    for anchor in re.findall(r"\]\(#([a-z][a-z0-9]*-\d{4})\)", text):
        require(anchor.upper() in rules, f"missing public anchor: {anchor}")
    return text[: starts[0][1]], rules


def load_public(
    game: Path, document: str, validate_relationships: bool = True
) -> tuple[str, dict[str, Rule]]:
    """Load one topic with optional deferred cross-topic graph validation."""
    return parse_public(
        within(game, document).read_text(encoding="utf-8"), validate_relationships
    )


def validate_graph(rules: dict[str, Rule]) -> None:
    """Check dependency existence, leaf-only membership, cycles and depth."""
    for identifier, rule in rules.items():
        for dependency in rule["uses"]:
            require(
                dependency in rules, f"{identifier}: missing dependency {dependency}"
            )
            require(
                rules[dependency]["kind"] == "rule",
                f"{identifier}: dependencies must be leaf rules",
            )
        expand(rules, identifier)


def load_catalog(
    game: Path, catalog_path: str
) -> tuple[dict[str, Document], dict[str, Rule]]:
    """Load all public topics and validate their cross-topic references."""
    catalog = cast(
        Catalog, decode_json(within(game, catalog_path).read_text(encoding="utf-8"))
    )
    require(catalog["schema_version"] == 1, "unsupported public catalog schema")
    paths = catalog["documents"]
    require(
        isinstance(paths, list)
        and paths
        and all(isinstance(path, str) for path in paths),
        "catalog must list public document paths",
    )
    require(len(paths) == len(set(paths)), "duplicate document in public catalog")
    documents: dict[str, Document] = {}
    rules: dict[str, Rule] = {}
    for document in paths:
        require(
            document.startswith("docs/gamefacts/") and document.endswith(".md"),
            "catalog documents must be Game Facts Markdown",
        )
        preamble, local = load_public(game, document, validate_relationships=False)
        require(
            not (set(rules) & set(local)),
            f"duplicate public IDs across topics: {document}",
        )
        for item in local.values():
            item["document"] = document
        documents[document] = {"preamble": preamble, "rules": local}
        rules.update(local)
    require(len(rules) <= MAX_RULES, "catalog exceeds reference limit")
    validate_graph(rules)
    for document, topic in documents.items():
        text = topic["preamble"] + "".join(
            item["text"] for item in topic["rules"].values()
        )
        for link in re.findall(r"\]\(([^)\n]+)\)", text):
            parsed = urlsplit(link)
            if (
                parsed.scheme
                or parsed.netloc
                or not ID.fullmatch(parsed.fragment.upper())
            ):
                continue
            destination = (
                within(game, str(Path(document).parent / unquote(parsed.path)))
                .relative_to(game.resolve())
                .as_posix()
                if parsed.path
                else document
            )
            require(
                destination in documents,
                f"reference link targets uncataloged document: {link}",
            )
            require(
                parsed.fragment.upper() in documents[destination]["rules"],
                f"reference link has wrong anchor: {link}",
            )
    return documents, rules


def expand(rules: dict[str, Rule], identifier: str) -> list[str]:
    """Expand dependencies in deterministic postorder with bounded recursion."""
    require(identifier in rules, f"unknown reference: {identifier}")
    ordered = []
    visited = set()
    active = set()

    def visit(current: str, depth: int) -> None:
        require(depth <= MAX_DEPTH, "reference dependency depth exceeds pilot limit")
        require(current not in active, f"dependency cycle at {current}")
        require(current in rules, f"missing dependency {current}")
        if current in visited:
            return
        active.add(current)
        for dependency in rules[current]["uses"]:
            visit(dependency, depth + 1)
        active.remove(current)
        visited.add(current)
        ordered.append(current)

    visit(identifier, 0)
    return ordered


def implementation_paths(rules: dict[str, Rule]) -> list[str]:
    """Collect unique implementation/test Go locators from public declarations."""
    paths = set()
    for identifier, rule in rules.items():
        for name in ("Implementation", "Tests"):
            value = field(rule["text"], name, required=False)
            if value == "none":
                continue
            entries = value.split(", ")
            require(
                all(re.fullmatch(r"`[^`]+\.go`", item) for item in entries),
                f"{identifier}: malformed {name} locators",
            )
            paths.update(item[1:-1] for item in entries)
    return sorted(paths)


def deviation_ids(text: str) -> list[list[str]]:
    """Return the rule IDs named in each `REDUX DEVIATION` comment block."""
    lines = text.split("\n")
    blocks = []
    for index, line in enumerate(lines):
        if "REDUX DEVIATION" not in line:
            continue
        start = end = index
        while start > 0 and lines[start - 1].lstrip().startswith("//"):
            start -= 1
        while end + 1 < len(lines) and lines[end + 1].lstrip().startswith("//"):
            end += 1
        block = "\n".join(lines[start : end + 1])
        blocks.append(re.findall(r"\b[A-Z]+-\d{4}\b", block))
    return blocks


def validate_repository(
    game: Path, catalog_path: str = "docs/gamefacts/catalog.json"
) -> tuple[dict[str, Document], dict[str, Rule]]:
    """Validate public application references; general Markdown links use docs-check."""
    game = game.resolve()
    documents, rules = load_catalog(game, catalog_path)
    # These two navigation/schema documents declare no cataloged behavior.
    guides = {"docs/gamefacts/README.md", "docs/gamefacts/CONVENTIONS.md"}
    actual = {
        path.relative_to(game).as_posix()
        for path in (game / "docs/gamefacts").rglob("*.md")
    } - guides
    require(
        set(documents) == actual,
        f"Game Facts catalog mismatch: uncataloged={sorted(actual - set(documents))}; "
        f"missing={sorted(set(documents) - actual)}",
    )
    for path in implementation_paths(rules):
        require(
            within(game, path).is_file(), f"missing implementation/test locator: {path}"
        )
    # Scan maintained Go code only, not caches, generated output, or original data.
    for directory in ("cmd", "internal", "pkg", "test", "scripts", "u5dump"):
        for path in sorted((game / directory).rglob("*.go")):
            relative = path.relative_to(game).as_posix()
            text = within(game, relative).read_text(encoding="utf-8")
            for match in re.finditer(r"\bref:([^\s`\"'<>]*)", text):
                identifier = match.group(1).rstrip(".,;:)")
                require(
                    ID.fullmatch(identifier) is not None,
                    f"{relative}: malformed ref:{identifier}",
                )
                require(identifier in rules, f"{relative}: unknown ref:{identifier}")
            for identifiers in deviation_ids(text):
                require(
                    any(identifier in rules for identifier in identifiers),
                    f"{relative}: REDUX DEVIATION comment names no Game Facts ID",
                )
    return documents, rules
