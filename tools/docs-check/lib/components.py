from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass
class ComponentsParseError:
    line: int
    message: str


@dataclass
class ComponentsParseResult:
    data: dict[str, Any]
    line_map: dict[tuple[str, ...], int]
    errors: list[ComponentsParseError]


TRUE_VALUES = {"true", "yes"}
FALSE_VALUES = {"false", "no"}
NULL_VALUES = {"null", "none"}


def parse_components_document(text: str) -> ComponentsParseResult:
    data: dict[str, Any] = {}
    line_map: dict[tuple[str, ...], int] = {}
    errors: list[ComponentsParseError] = []
    in_components = False
    current_component: str | None = None
    current_list_field: str | None = None

    for line_number, raw_line in enumerate(text.splitlines(), start=1):
        stripped = raw_line.strip()
        if not stripped or stripped.startswith("#"):
            continue

        indent = len(raw_line) - len(raw_line.lstrip(" "))
        if "\t" in raw_line[:indent]:
            errors.append(ComponentsParseError(line_number, "Tabs are not supported in COMPONENTS.yaml."))
            continue

        if indent == 0:
            current_component = None
            current_list_field = None
            key, raw_value = parse_key_value(stripped, line_number, errors)
            if key is None:
                continue
            line_map[(key,)] = line_number
            in_components = key == "components"

            if key == "components":
                if raw_value:
                    errors.append(
                        ComponentsParseError(
                            line_number,
                            "Top-level `components` key must open a nested mapping.",
                        )
                    )
                data.setdefault("components", {})
                continue

            if not raw_value:
                errors.append(
                    ComponentsParseError(
                        line_number,
                        f"Top-level key `{key}` must use an inline scalar value.",
                    )
                )
                continue

            data[key] = parse_scalar(raw_value)
            continue

        if indent == 2:
            current_list_field = None
            if not in_components:
                errors.append(
                    ComponentsParseError(
                        line_number,
                        "Indented entries are only supported under the top-level `components` key.",
                    )
                )
                continue

            key, raw_value = parse_key_value(stripped, line_number, errors)
            if key is None:
                continue
            if raw_value:
                errors.append(
                    ComponentsParseError(
                        line_number,
                        "Component identifiers must open a nested mapping, not an inline value.",
                    )
                )
                continue

            components = data.setdefault("components", {})
            if not isinstance(components, dict):
                errors.append(ComponentsParseError(line_number, "`components` must be a mapping."))
                continue
            components[key] = {}
            current_component = key
            line_map[("components", key)] = line_number
            continue

        if indent == 4:
            if current_component is None:
                errors.append(
                    ComponentsParseError(
                        line_number,
                        "Component fields must belong to a component entry.",
                    )
                )
                continue

            key, raw_value = parse_key_value(stripped, line_number, errors)
            if key is None:
                continue
            component = data.setdefault("components", {}).setdefault(current_component, {})
            if not isinstance(component, dict):
                errors.append(
                    ComponentsParseError(line_number, f"Component `{current_component}` must be a mapping.")
                )
                continue

            line_map[("components", current_component, key)] = line_number
            if raw_value:
                component[key] = parse_scalar(raw_value)
                current_list_field = None
            else:
                component[key] = []
                current_list_field = key
            continue

        if indent == 6:
            if current_component is None or current_list_field is None:
                errors.append(
                    ComponentsParseError(
                        line_number,
                        "List items must belong to a list-valued component field.",
                    )
                )
                continue
            if not stripped.startswith("- "):
                errors.append(ComponentsParseError(line_number, "Only `- value` list items are supported."))
                continue

            component = data.setdefault("components", {}).setdefault(current_component, {})
            values = component.setdefault(current_list_field, [])
            if not isinstance(values, list):
                errors.append(
                    ComponentsParseError(
                        line_number,
                        f"Component field `{current_list_field}` must be a list before adding items.",
                    )
                )
                continue

            values.append(parse_scalar(stripped[2:].strip()))
            continue

        errors.append(
            ComponentsParseError(
                line_number,
                "Unsupported indentation in COMPONENTS.yaml; use 2-space nested mappings.",
            )
        )

    return ComponentsParseResult(data=data, line_map=line_map, errors=errors)


def parse_key_value(
    line: str,
    line_number: int,
    errors: list[ComponentsParseError],
) -> tuple[str | None, str]:
    if ":" not in line:
        errors.append(ComponentsParseError(line_number, f"Unsupported YAML line: {line}"))
        return None, ""

    key, raw_value = line.split(":", 1)
    key = key.strip()
    if not key:
        errors.append(ComponentsParseError(line_number, "Encountered an empty key in COMPONENTS.yaml."))
        return None, ""
    return key, raw_value.strip()


def parse_scalar(value: str) -> Any:
    if value.startswith("[") and value.endswith("]"):
        inner = value[1:-1].strip()
        if not inner:
            return []
        return [parse_scalar(part.strip()) for part in inner.split(",")]

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
