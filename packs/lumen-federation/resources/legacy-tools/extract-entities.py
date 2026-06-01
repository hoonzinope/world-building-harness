#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from _mdutils import iter_markdown_files, load_markdown_doc


REQUIRED_KEYS = ("title", "type", "slug")


def main() -> int:
    parser = argparse.ArgumentParser(description="Extract entity frontmatter as JSON.")
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
    entities: list[dict[str, object]] = []
    slug_index: dict[str, list[str]] = {}
    problems: list[str] = []

    for doc in docs:
        if doc.path.name.lower() == "index.md":
            continue
        if not doc.frontmatter:
            problems.append(f"{doc.relpath}: missing frontmatter")
            continue

        entity = {
            "path": doc.relpath,
            "frontmatter": doc.frontmatter,
            "title": doc.frontmatter.get("title"),
            "type": doc.frontmatter.get("type"),
            "slug": doc.frontmatter.get("slug"),
        }
        entities.append(entity)

        missing = [key for key in REQUIRED_KEYS if not doc.frontmatter.get(key)]
        if missing:
            problems.append(f"{doc.relpath}: missing metadata {', '.join(missing)}")

        slug = doc.frontmatter.get("slug")
        if isinstance(slug, str) and slug:
            slug_index.setdefault(slug, []).append(doc.relpath)

    for slug, paths in sorted(slug_index.items()):
        if len(paths) > 1:
            problems.append(f"duplicate slug {slug}: {', '.join(paths)}")

    entities.sort(key=lambda entity: (str(entity.get("slug") or ""), str(entity.get("path") or "")))
    print(json.dumps(entities, ensure_ascii=False, indent=2))

    if problems:
        for problem in problems:
            print(problem, file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
