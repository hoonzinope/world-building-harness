#!/usr/bin/env python3
from __future__ import annotations

import argparse
from collections import defaultdict
import sys
from pathlib import Path
from typing import Any

from _mdutils import (
    load_markdown_doc,
    markdown_link_targets,
    iter_markdown_files,
    relative_path,
    wikilink_targets,
)


def build_path_variants(target: str, source_path: Path, content_dir: Path) -> list[Path]:
    target_path = Path(target)
    bases: list[Path]
    if target.startswith("/"):
        bases = [content_dir, Path.cwd()]
        target_path = Path(target.lstrip("/"))
    else:
        bases = [source_path.parent, content_dir]

    variants: list[Path] = []
    suffix = target_path.suffix.lower()
    for base in bases:
        candidate = base / target_path
        variants.append(candidate)
        if suffix != ".md":
            variants.append(candidate.with_suffix(".md"))
        if candidate.suffix:
            variants.append(candidate)
        if suffix == "":
            variants.append(candidate / "index.md")
    return variants


def build_slug_maps(docs: list[Any]) -> tuple[dict[str, list[Path]], dict[str, list[Path]]]:
    slug_map: dict[str, list[Path]] = defaultdict(list)
    stem_map: dict[str, list[Path]] = defaultdict(list)
    for doc in docs:
        slug = doc.frontmatter.get("slug")
        if isinstance(slug, str) and slug:
            slug_map[slug].append(doc.path)
        stem_map[doc.path.stem].append(doc.path)
    return slug_map, stem_map


def resolve_markdown_target(target: str, source_path: Path, content_dir: Path) -> Path | None:
    content_root = content_dir.resolve()
    for candidate in build_path_variants(target, source_path, content_dir):
        resolved = candidate.resolve()
        if resolved.exists() and (resolved == content_root or content_root in resolved.parents):
            if resolved.suffix.lower() == ".md":
                return resolved
    return None


def resolve_wikilink_target(
    target: str,
    source_path: Path,
    content_dir: Path,
    slug_map: dict[str, list[Path]],
    stem_map: dict[str, list[Path]],
) -> tuple[Path | None, str | None]:
    if "/" in target or target.endswith(".md"):
        resolved = resolve_markdown_target(target, source_path, content_dir)
        if resolved is not None:
            return resolved, None

    if target in slug_map:
        paths = slug_map[target]
        if len(paths) == 1:
            return paths[0].resolve(), None
        return None, f"ambiguous slug matches {len(paths)} files"

    if target in stem_map:
        paths = stem_map[target]
        if len(paths) == 1:
            return paths[0].resolve(), None
        return None, f"ambiguous filename stem matches {len(paths)} files"

    return None, "no matching slug or file"


def is_excluded_from_orphans(path: Path) -> bool:
    return path.name.lower() == "index.md"


def main() -> int:
    parser = argparse.ArgumentParser(description="Check internal wiki links and orphan pages.")
    parser.add_argument(
        "--content-dir",
        default="contents",
        help="Root directory to scan for wiki content (default: contents).",
    )
    args = parser.parse_args()

    content_dir = Path(args.content_dir).expanduser()
    if not content_dir.exists():
        print(f"content directory not found: {content_dir}", file=sys.stderr)
        return 1

    docs = [load_markdown_doc(path, content_dir) for path in iter_markdown_files(content_dir)]
    slug_map, stem_map = build_slug_maps(docs)

    incoming_counts: dict[Path, int] = defaultdict(int)
    broken_links: list[dict[str, Any]] = []

    for doc in docs:
        for line, label, target in markdown_link_targets(doc.body):
            resolved = resolve_markdown_target(target, doc.path, content_dir)
            if resolved is None:
                broken_links.append(
                    {
                        "path": doc.relpath,
                        "line": line,
                        "label": label,
                        "target": target,
                        "reason": "unresolved markdown link",
                    }
                )
                continue
            incoming_counts[resolved] += 1

        for line, raw, target in wikilink_targets(doc.body):
            resolved, reason = resolve_wikilink_target(target, doc.path, content_dir, slug_map, stem_map)
            if resolved is None:
                broken_links.append(
                    {
                        "path": doc.relpath,
                        "line": line,
                        "label": raw,
                        "target": target,
                        "reason": reason or "unresolved wiki link",
                    }
                )
                continue
            incoming_counts[resolved] += 1

    duplicate_slugs: list[tuple[str, list[str]]] = []
    for slug, paths in sorted(slug_map.items()):
        if len(paths) > 1:
            duplicate_slugs.append((slug, [relative_path(path, content_dir) for path in paths]))

    orphans = [
        doc.relpath
        for doc in docs
        if not is_excluded_from_orphans(doc.path) and incoming_counts.get(doc.path.resolve(), 0) == 0
    ]

    has_issues = bool(broken_links or duplicate_slugs or orphans)
    if not has_issues:
        print("No link issues found.")
        return 0

    if broken_links:
        print("Broken links:")
        for item in broken_links:
            print(
                f"- {item['path']}:{item['line']} -> {item['target']} "
                f"({item['reason']})"
            )

    if duplicate_slugs:
        print("Duplicate slugs:")
        for slug, paths in duplicate_slugs:
            print(f"- {slug}")
            for path in paths:
                print(f"  - {path}")

    if orphans:
        print("Orphan pages:")
        for relpath in orphans:
            print(f"- {relpath}")

    return 1


if __name__ == "__main__":
    import sys

    raise SystemExit(main())
