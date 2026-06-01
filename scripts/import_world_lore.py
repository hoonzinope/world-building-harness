#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
import shutil
from pathlib import Path


TYPE_MAP = {
    "public-index": "glossary",
    "world": "nation",
    "city": "place",
    "faction": "organization",
    "system": "glossary",
    "glossary": "glossary",
    "anomaly": "magic",
    "author-note": "glossary",
}

PREFIX = {
    "character": "character_",
    "nation": "nation_",
    "organization": "org_",
    "place": "place_",
    "event": "event_",
    "timeline": "timeline_",
    "magic": "magic_",
    "glossary": "term_",
}


def split_frontmatter(text: str) -> tuple[dict[str, object], str]:
    if not text.startswith("---\n"):
        return {}, text
    end = text.find("\n---", 4)
    if end < 0:
        return {}, text
    raw = text[4:end]
    body = text[end + 4 :].lstrip("\n")
    return parse_simple_yaml(raw), body


def parse_simple_yaml(raw: str) -> dict[str, object]:
    data: dict[str, object] = {}
    current_key: str | None = None
    for line in raw.splitlines():
        if not line.strip():
            continue
        if line.startswith("  - ") and current_key:
            data.setdefault(current_key, [])
            if isinstance(data[current_key], list):
                data[current_key].append(line[4:].strip().strip('"'))
            continue
        if ":" in line and not line.startswith(" "):
            key, value = line.split(":", 1)
            key = key.strip()
            value = value.strip()
            current_key = key
            if value == "":
                data[key] = []
            elif value.startswith("[") and value.endswith("]"):
                inner = value[1:-1].strip()
                data[key] = [part.strip().strip('"') for part in inner.split(",") if part.strip()]
            else:
                data[key] = value.strip('"')
    return data


def dump_yaml(data: dict[str, object]) -> str:
    lines: list[str] = []
    for key, value in data.items():
        if value is None:
            lines.append(f"{key}: null")
        elif isinstance(value, list):
            if value:
                lines.append(f"{key}:")
                for item in value:
                    lines.append(f"  - {quote(str(item))}")
            else:
                lines.append(f"{key}: []")
        elif isinstance(value, (int, float)):
            lines.append(f"{key}: {value}")
        else:
            lines.append(f"{key}: {quote(str(value))}")
    return "\n".join(lines) + "\n"


def quote(value: str) -> str:
    if value == "" or any(ch in value for ch in ":#[]{}&,*?!|>'\"%@`"):
        return '"' + value.replace('"', '\\"') + '"'
    return value


def slugify(value: str) -> str:
    value = value.lower().replace("-", "_")
    value = re.sub(r"[^a-z0-9_]+", "_", value)
    value = re.sub(r"_+", "_", value).strip("_")
    return value or "untitled"


def migrate_markdown(src: Path, dst: Path, root: Path) -> None:
    text = src.read_text(encoding="utf-8")
    old, body = split_frontmatter(text)
    old_type = str(old.get("type") or infer_type(src, root))
    doc_type = TYPE_MAP.get(old_type, "glossary")
    slug = str(old.get("slug") or src.stem)
    doc_id = PREFIX[doc_type] + slugify(slug)
    updated = str(old.get("updated_at") or old.get("modified") or "2026-05-24")
    title = str(old.get("title") or heading_title(body) or src.stem)
    tags = old.get("tags") if isinstance(old.get("tags"), list) else []
    aliases = old.get("aliases") if isinstance(old.get("aliases"), list) else []
    meta: dict[str, object] = {
        "schema_version": "world-doc.v1",
        "id": doc_id,
        "type": doc_type,
        "status": "canon",
        "title": title,
        "tags": tags,
        "created_at": updated,
        "updated_at": updated,
        "aliases": aliases,
        "related": [],
        "relationships": [],
        "source_run_id": None,
        "legacy_type": old_type,
        "legacy_path": src.relative_to(root).as_posix(),
    }
    dst.parent.mkdir(parents=True, exist_ok=True)
    dst.write_text("---\n" + dump_yaml(meta) + "---\n\n" + body.strip() + "\n", encoding="utf-8")


def infer_type(path: Path, root: Path) -> str:
    parts = path.relative_to(root).parts
    if len(parts) > 1:
        return parts[0].rstrip("s")
    return "glossary"


def heading_title(body: str) -> str:
    for line in body.splitlines():
        if line.startswith("# "):
            return line[2:].strip()
    return ""


def copytree(src: Path, dst: Path) -> None:
    if not src.exists():
        return
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(src, dst)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True)
    parser.add_argument("--dest", required=True)
    parser.add_argument("--pack-id", default="lumen-federation")
    args = parser.parse_args()
    source = Path(args.source).resolve()
    dest = Path(args.dest).resolve()
    if dest.exists():
        shutil.rmtree(dest)
    for rel in [
        "content",
        "drafts/characters",
        "drafts/nations",
        "drafts/organizations",
        "drafts/places",
        "drafts/events",
        "drafts/timeline",
        "drafts/magic",
        "drafts/glossary",
        "drafts/storylets",
        "runs/inbox",
        "archive/accepted",
        "archive/rejected",
        "archive/deprecated",
        "graph",
        "schema",
        "raw",
        "resources",
    ]:
        (dest / rel).mkdir(parents=True, exist_ok=True)

    contents = source / "contents"
    for src in sorted(contents.rglob("*.md")):
        rel = src.relative_to(contents)
        migrate_markdown(src, dest / "content" / rel, contents)

    copytree(source / "raw", dest / "raw" / "legacy-raw")
    copytree(source / "drafts", dest / "raw" / "legacy-drafts")
    copytree(source / "schema", dest / "resources" / "legacy-schema")
    copytree(source / "prompts", dest / "resources" / "legacy-prompts")
    copytree(source / "tools", dest / "resources" / "legacy-tools")
    for name in ["README.md", "AGENTS.md", "quartz.config.ts", "Dockerfile", "docker-compose.yml"]:
        src = source / name
        if src.exists():
            shutil.copy2(src, dest / "resources" / f"legacy-{name}")

    harness = f"""schema_version: world-harness.v1
world_id: {args.pack_id}
world_root: .
content_dir: content
draft_dir: drafts
run_dir: runs
inbox_dir: runs/inbox
graph_dir: graph
archive_dir: archive
approval:
  require_accept: true
  allow_force_accept: true
security:
  deny_outside_root: true
  allow_network: false
locking:
  enabled: true
  lock_file: runs/.lock
"""
    (dest / "harness.yaml").write_text(harness, encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
