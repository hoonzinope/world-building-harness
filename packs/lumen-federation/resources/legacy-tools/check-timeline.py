#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from datetime import date
from pathlib import Path
from typing import Any

from _mdutils import iter_markdown_files, load_markdown_doc


YEAR_TOKEN_RE = r"(?:[1-9]\d{0,3}|0[1-9]\d{0,2}|00[1-9]\d?|000[1-9])"
YEAR_RE = re.compile(rf"\b{YEAR_TOKEN_RE}\b")
DATE_RE = re.compile(rf"\b({YEAR_TOKEN_RE})-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])\b")
YEAR_RANGE_RE = re.compile(rf"\b({YEAR_TOKEN_RE})\s*(?:\.\.|-)\s*({YEAR_TOKEN_RE})\b")
DATE_RANGE_RE = re.compile(
    rf"\b({YEAR_TOKEN_RE}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))\s*(?:\.\.|-)\s*"
    rf"({YEAR_TOKEN_RE}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))\b"
)

TEMPORAL_KEYS = {
    "year",
    "date",
    "event_year",
    "event_date",
    "start_year",
    "end_year",
    "start_date",
    "end_date",
}


@dataclass(frozen=True)
class TemporalValue:
    key: str
    value: Any
    line: int


def parse_date(value: str) -> date | None:
    try:
        year, month, day = map(int, value.split("-"))
        return date(year, month, day)
    except Exception:
        return None


def temporal_values_from_frontmatter(frontmatter: dict[str, Any]) -> list[TemporalValue]:
    values: list[TemporalValue] = []
    for key, value in frontmatter.items():
        if key not in TEMPORAL_KEYS:
            continue
        values.append(TemporalValue(key=key, value=value, line=0))
    return values


def extract_body_temporal_mentions(body: str) -> list[tuple[int, str]]:
    mentions: list[tuple[int, int, str]] = []
    for match in DATE_RANGE_RE.finditer(body):
        line = body.count("\n", 0, match.start()) + 1
        mentions.append((line, 0, match.group(0)))
    for match in DATE_RE.finditer(body):
        line = body.count("\n", 0, match.start()) + 1
        mentions.append((line, 1, match.group(0)))
    for match in YEAR_RANGE_RE.finditer(body):
        line = body.count("\n", 0, match.start()) + 1
        mentions.append((line, 2, match.group(0)))
    for match in YEAR_RE.finditer(body):
        line = body.count("\n", 0, match.start()) + 1
        mentions.append((line, 3, match.group(0)))
    mentions.sort(key=lambda item: (item[0], item[1], item[2]))
    return mentions


def normalize_temporal(value: Any) -> tuple[str, Any] | None:
    if isinstance(value, int):
        if 1 <= value <= 9999:
            return ("year", value)
        return None
    if isinstance(value, str):
        value = value.strip()
        if re.fullmatch(YEAR_TOKEN_RE, value):
            year = int(value)
            if 1 <= year <= 9999:
                return ("year", year)
        if re.fullmatch(rf"{YEAR_TOKEN_RE}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01])", value):
            parsed = parse_date(value)
            if parsed is not None:
                return ("date", parsed)
    return None


def format_temporal(value: Any) -> str:
    if isinstance(value, date):
        return value.isoformat()
    return str(value)


def main() -> int:
    parser = argparse.ArgumentParser(description="Check year/date consistency in timeline content.")
    parser.add_argument(
        "--content-dir",
        default="contents",
        help="Root directory to scan for Quartz public input (default: contents).",
    )
    args = parser.parse_args()

    content_dir = Path(args.content_dir).expanduser()
    if not content_dir.exists():
        print(f"content directory not found: {content_dir}", file=sys.stderr)
        return 1

    docs = [load_markdown_doc(path, content_dir) for path in iter_markdown_files(content_dir)]
    issues: list[str] = []

    for doc in docs:
        fm_values = temporal_values_from_frontmatter(doc.frontmatter)
        if not fm_values:
            continue

        normalized: dict[str, tuple[str, Any]] = {}
        for item in fm_values:
            norm = normalize_temporal(item.value)
            if norm is not None:
                normalized[item.key] = norm

        start_year = normalized.get("start_year")
        end_year = normalized.get("end_year")
        start_date = normalized.get("start_date")
        end_date = normalized.get("end_date")

        if start_year and end_year and start_year[0] == end_year[0] == "year":
            if start_year[1] > end_year[1]:
                issues.append(
                    f"{doc.relpath}: start_year {start_year[1]} is after end_year {end_year[1]}"
                )

        if start_date and end_date and start_date[0] == end_date[0] == "date":
            if start_date[1] > end_date[1]:
                issues.append(
                    f"{doc.relpath}: start_date {format_temporal(start_date[1])} is after end_date {format_temporal(end_date[1])}"
                )

        singular_year_values = [
            value
            for key, value in normalized.items()
            if key in {"year", "event_year"} and value[0] == "year"
        ]
        singular_date_values = [
            value
            for key, value in normalized.items()
            if key in {"date", "event_date"} and value[0] == "date"
        ]

        if singular_year_values:
            distinct_years = {value for _, value in singular_year_values}
            if len(distinct_years) > 1:
                issues.append(
                    f"{doc.relpath}: conflicting year values in frontmatter: "
                    + ", ".join(str(value) for value in sorted(distinct_years))
                )

        if singular_date_values:
            distinct_dates = {value for _, value in singular_date_values}
            if len(distinct_dates) > 1:
                issues.append(
                    f"{doc.relpath}: conflicting date values in frontmatter: "
                    + ", ".join(format_temporal(value) for value in sorted(distinct_dates))
                )

        if start_year and end_year and singular_year_values:
            for key, (kind, value) in normalized.items():
                if kind != "year" or key in {"start_year", "end_year"}:
                    continue
                if not (start_year[1] <= value <= end_year[1]):
                    issues.append(
                        f"{doc.relpath}: {key}={value} is outside start_year/end_year range "
                        f"{start_year[1]}..{end_year[1]}"
                    )

        if start_date and end_date and singular_date_values:
            for key, (kind, value) in normalized.items():
                if kind != "date" or key in {"start_date", "end_date"}:
                    continue
                if not (start_date[1] <= value <= end_date[1]):
                    issues.append(
                        f"{doc.relpath}: {key}={format_temporal(value)} is outside start_date/end_date range "
                        f"{format_temporal(start_date[1])}..{format_temporal(end_date[1])}"
                    )

        body_mentions = extract_body_temporal_mentions(doc.body)
        if not body_mentions:
            continue

        primary_value = None
        primary_key = None
        for key in ("event_date", "date", "event_year", "year"):
            if key in normalized:
                primary_key = key
                primary_value = normalized[key][1]
                break

        first_line, _, first_mention = body_mentions[0]
        if primary_key is not None:
            if primary_key in {"event_date", "date"}:
                parsed = parse_date(first_mention)
                if parsed is not None and parsed != primary_value:
                    issues.append(
                        f"{doc.relpath}: body date {first_mention} on line {first_line} conflicts with "
                        f"{primary_key}={format_temporal(primary_value)}"
                    )
                else:
                    year_match = YEAR_RE.search(first_mention)
                    if year_match and int(year_match.group(0)) != primary_value.year:
                        issues.append(
                            f"{doc.relpath}: body year {year_match.group(0)} on line {first_line} conflicts with "
                            f"{primary_key}={format_temporal(primary_value)}"
                        )
            else:
                match = YEAR_RE.search(first_mention)
                if match and int(match.group(0)) != int(primary_value):
                    issues.append(
                        f"{doc.relpath}: body year {match.group(0)} on line {first_line} conflicts with "
                        f"{primary_key}={primary_value}"
                    )
        elif start_year and end_year:
            year_match = YEAR_RE.search(first_mention)
            date_value = parse_date(first_mention)
            if year_match:
                year_value = int(year_match.group(0))
                if not (start_year[1] <= year_value <= end_year[1]):
                    issues.append(
                        f"{doc.relpath}: body year {year_value} on line {first_line} is outside "
                        f"start_year/end_year range {start_year[1]}..{end_year[1]}"
                    )
            elif date_value is not None:
                if not (date_value.year >= start_year[1] and date_value.year <= end_year[1]):
                    issues.append(
                        f"{doc.relpath}: body date {first_mention} on line {first_line} is outside "
                        f"start_year/end_year range {start_year[1]}..{end_year[1]}"
                    )
        elif start_date and end_date:
            date_value = parse_date(first_mention)
            if date_value is not None and not (start_date[1] <= date_value <= end_date[1]):
                issues.append(
                    f"{doc.relpath}: body date {first_mention} on line {first_line} is outside "
                    f"start_date/end_date range {format_temporal(start_date[1])}..{format_temporal(end_date[1])}"
                )

    if not issues:
        print("No timeline issues found.")
        return 0

    for issue in issues:
        print(issue)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
