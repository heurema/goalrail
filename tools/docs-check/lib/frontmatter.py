from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass
class FrontmatterParseResult:
    data: dict[str, Any] | None
    body: str
    start_line: int


TRUE_VALUES = {"true", "yes"}
FALSE_VALUES = {"false", "no"}
NULL_VALUES = {"null", "none"}


def parse_frontmatter(text: str) -> FrontmatterParseResult:
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return FrontmatterParseResult(data=None, body=text, start_line=1)

    end_index = None
    for index in range(1, len(lines)):
        if lines[index].strip() == "---":
            end_index = index
            break
    if end_index is None:
        return FrontmatterParseResult(data=None, body=text, start_line=1)

    raw_frontmatter = lines[1:end_index]
    data = _parse_simple_yaml_block(raw_frontmatter)
    body = "\n".join(lines[end_index + 1 :])
    return FrontmatterParseResult(data=data, body=body, start_line=1)


def _parse_simple_yaml_block(lines: list[str]) -> dict[str, Any]:
    data: dict[str, Any] = {}
    current_key: str | None = None
    current_list: list[Any] | None = None

    for raw_line in lines:
        line = raw_line.rstrip()
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue

        if stripped.startswith("- "):
            if current_key is None or current_list is None:
                raise ValueError(f"list item without owning key: {raw_line}")
            current_list.append(_parse_scalar(stripped[2:].strip()))
            continue

        if ":" not in line:
            raise ValueError(f"unsupported frontmatter line: {raw_line}")

        key, raw_value = line.split(":", 1)
        key = key.strip()
        value = raw_value.strip()
        if not key:
            raise ValueError(f"invalid empty key in frontmatter line: {raw_line}")

        if not value:
            current_key = key
            current_list = []
            data[key] = current_list
            continue

        data[key] = _parse_scalar(value)
        current_key = None
        current_list = None

    return data


def _parse_scalar(value: str) -> Any:
    if value.startswith("[") and value.endswith("]"):
        inner = value[1:-1].strip()
        if not inner:
            return []
        return [_parse_scalar(part.strip()) for part in inner.split(",")]

    lowered = value.lower()
    if lowered in TRUE_VALUES:
        return True
    if lowered in FALSE_VALUES:
        return False
    if lowered in NULL_VALUES:
        return None

    if (value.startswith('"') and value.endswith('"')) or (value.startswith("'") and value.endswith("'")):
        return value[1:-1]

    if value.isdigit():
        return int(value)

    return value
