from __future__ import annotations

import csv
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable


FRONTMATTER_DELIM = "---"
KEY_VALUE_RE = re.compile(r"^([A-Za-z0-9_-]+):(?:\s*(.*))?$")
MARKDOWN_LINK_RE = re.compile(r"(?<!!)\[([^\]]+)\]\(([^)]+)\)")
WIKILINK_RE = re.compile(r"(?<!\!)\[\[([^\]]+)\]\]")


@dataclass(frozen=True)
class MarkdownDoc:
    path: Path
    relpath: str
    frontmatter: dict[str, Any]
    body: str
    raw_text: str


def iter_markdown_files(content_dir: Path) -> list[Path]:
    return sorted(
        (path for path in content_dir.rglob("*.md") if path.is_file()),
        key=lambda path: path.as_posix(),
    )


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def relative_path(path: Path, root: Path) -> str:
    try:
        return path.relative_to(root).as_posix()
    except ValueError:
        return path.as_posix()


def split_frontmatter(text: str) -> tuple[list[str], str, bool]:
    lines = text.splitlines()
    if not lines or lines[0].strip() != FRONTMATTER_DELIM:
        return [], text, False

    fm_lines: list[str] = []
    for index in range(1, len(lines)):
        line = lines[index]
        if line.strip() == FRONTMATTER_DELIM:
            body = "\n".join(lines[index + 1 :])
            return fm_lines, body, True
        fm_lines.append(line)

    return [], text, False


def parse_inline_list(value: str) -> list[Any] | None:
    value = value.strip()
    if not (value.startswith("[") and value.endswith("]")):
        return None

    inner = value[1:-1].strip()
    if not inner:
        return []

    reader = csv.reader([inner], skipinitialspace=True)
    try:
        items = next(reader)
    except StopIteration:
        return []
    return [parse_scalar(item) for item in items]


def parse_scalar(value: str) -> Any:
    value = value.strip()
    if value == "":
        return ""

    if value[:1] == value[-1:] and value[:1] in {"'", '"'} and len(value) >= 2:
        return value[1:-1]

    lowered = value.lower()
    if lowered == "true":
        return True
    if lowered == "false":
        return False
    if lowered in {"null", "none", "~"}:
        return None

    inline_list = parse_inline_list(value)
    if inline_list is not None:
        return inline_list

    if re.fullmatch(r"[+-]?\d+", value):
        try:
            return int(value)
        except ValueError:
            pass

    return value


def parse_frontmatter(lines: list[str]) -> dict[str, Any]:
    data: dict[str, Any] = {}
    i = 0
    while i < len(lines):
        raw_line = lines[i].rstrip("\n")
        stripped = raw_line.strip()
        if not stripped or stripped.startswith("#"):
            i += 1
            continue

        match = KEY_VALUE_RE.match(stripped)
        if not match:
            i += 1
            continue

        key = match.group(1)
        value = match.group(2) or ""

        if value in {"|", ">"}:
            block_lines: list[str] = []
            i += 1
            while i < len(lines):
                next_line = lines[i]
                if not next_line.strip():
                    block_lines.append("")
                    i += 1
                    continue
                if next_line.startswith(" ") or next_line.startswith("\t"):
                    block_lines.append(next_line.lstrip())
                    i += 1
                    continue
                break
            data[key] = "\n".join(block_lines).rstrip()
            continue

        if value == "":
            list_items: list[Any] = []
            j = i + 1
            while j < len(lines):
                next_line = lines[j]
                if not next_line.strip():
                    j += 1
                    continue
                if re.match(r"^\s*-\s+", next_line):
                    list_items.append(parse_scalar(re.sub(r"^\s*-\s+", "", next_line, count=1)))
                    j += 1
                    continue
                break
            if list_items:
                data[key] = list_items
                i = j
                continue
            data[key] = ""
            i += 1
            continue

        data[key] = parse_scalar(value)
        i += 1

    return data


def load_markdown_doc(path: Path, root: Path) -> MarkdownDoc:
    raw_text = read_text(path)
    fm_lines, body, has_frontmatter = split_frontmatter(raw_text)
    frontmatter = parse_frontmatter(fm_lines) if has_frontmatter else {}
    return MarkdownDoc(
        path=path,
        relpath=relative_path(path, root),
        frontmatter=frontmatter,
        body=body,
        raw_text=raw_text,
    )


def is_internal_link(target: str) -> bool:
    lowered = target.strip().lower()
    return not (
        lowered.startswith("http://")
        or lowered.startswith("https://")
        or lowered.startswith("mailto:")
        or lowered.startswith("tel:")
        or lowered.startswith("javascript:")
        or lowered.startswith("data:")
    )


def strip_link_target(target: str) -> str:
    target = target.strip().strip("<>").strip()
    if target.startswith(("'", '"')) and target.endswith(("'", '"')) and len(target) >= 2:
        target = target[1:-1]
    target = target.split("?", 1)[0]
    target = target.split("#", 1)[0]
    return target.strip()


def markdown_link_targets(text: str) -> Iterable[tuple[int, str, str]]:
    for match in MARKDOWN_LINK_RE.finditer(text):
        line = text.count("\n", 0, match.start()) + 1
        label = match.group(1)
        target = strip_link_target(match.group(2))
        if is_internal_link(target):
            yield line, label, target


def wikilink_targets(text: str) -> Iterable[tuple[int, str, str]]:
    for match in WIKILINK_RE.finditer(text):
        line = text.count("\n", 0, match.start()) + 1
        raw = match.group(1).strip()
        target = raw.split("|", 1)[0].strip()
        target = strip_link_target(target)
        if target and is_internal_link(target):
            yield line, raw, target


def unique_preserving_order(items: Iterable[Any]) -> list[Any]:
    seen: set[Any] = set()
    result: list[Any] = []
    for item in items:
        if item in seen:
            continue
        seen.add(item)
        result.append(item)
    return result
