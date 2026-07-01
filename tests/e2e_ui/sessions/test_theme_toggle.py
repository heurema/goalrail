"""E2E: Settings → Appearance is fixed to the Dracula dark theme.

The Appearance section no longer exposes System / Light / Dark radio cards.
Goalrail uses one shared dark theme across app/admin/embed surfaces, and the
provider forces that mode regardless of OS preference or stale localStorage.

No LLM turn is involved.
"""

from __future__ import annotations

from playwright.sync_api import Page, expect


def _html_has_dark(page: Page) -> bool:
    """True when the ``dark`` class is applied to ``<html>`` (next-themes)."""
    return page.evaluate("() => document.documentElement.classList.contains('dark')")


def _stored_theme(page: Page) -> str | None:
    """The persisted legacy theme preference, if one exists."""
    return page.evaluate("() => window.localStorage.getItem('web-theme')")


def _open_appearance(page: Page, base_url: str) -> None:
    """Navigate to the Settings Appearance section and wait for the theme card."""
    page.goto(f"{base_url}/settings/appearance")
    expect(page.get_by_test_id("theme-dracula")).to_be_visible(timeout=30_000)


def _assert_fixed_dracula_appearance(page: Page) -> None:
    card = page.get_by_test_id("theme-dracula")
    expect(card).to_contain_text("Dracula")
    expect(card).to_contain_text("#282A36 / #FF79C6 / #F8F8F2")
    expect(page.get_by_role("radio")).to_have_count(0)
    assert _html_has_dark(page), "forced Dracula theme should keep the dark class on <html>"


def test_forced_dracula_theme_ignores_light_os(
    page: Page, seeded_session: tuple[str, str]
) -> None:
    """On a light OS, the app still renders the forced Dracula dark theme."""
    page.emulate_media(color_scheme="light")

    base_url, _session_id = seeded_session
    _open_appearance(page, base_url)

    assert _stored_theme(page) is None, "expected no persisted theme on a fresh load"
    _assert_fixed_dracula_appearance(page)


def test_forced_dracula_theme_overrides_stored_light_preference(
    page: Page, seeded_session: tuple[str, str]
) -> None:
    """A stale stored light preference does not change the forced theme."""
    page.emulate_media(color_scheme="dark")
    page.add_init_script("window.localStorage.setItem('web-theme', 'light')")

    base_url, _session_id = seeded_session
    _open_appearance(page, base_url)

    assert _stored_theme(page) == "light"
    _assert_fixed_dracula_appearance(page)
