#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
check_i18n.py — keep README translations consistent with the English canonical.

Checks:
  1. Every expected README.<locale>.md exists.
  2. Every README contains the canonical language switcher, verbatim.
  3. Every non-English README carries the translation banner ("Last synced:").
  4. Every translation has the same number of Markdown headings as English
     (catches dropped/merged sections), ignoring fenced code blocks.
  5. README.zh-CN.md exists as a redirect stub to README.zh-Hans.md.
  6. All README files have identical badge URLs (Status, Go version, Endpoints,
     License, etc.). Badge text labels may be translated but shield.io URLs
     must match across all files to catch version/status drift.
  7. All README files name the same release version, in the status table and in
     the translation banner, and those two agree within a file.

Rule 7 exists because rule 6 did not cover the version. The badge URLs carry no
version, so a release that updated the English status table and the banners left
five translations claiming v1.1.7 for twenty releases while the gate stayed
green - the exact failure a documentation gate is supposed to prevent.

Exits non-zero on any failure. See TRANSLATING.md.
"""

import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# locale -> (native label, filename)
LANGUAGES = [
    ("en", "English", "README.md"),
    ("zh-Hans", "简体中文", "README.zh-Hans.md"),
    ("zh-Hant", "繁體中文", "README.zh-Hant.md"),
    ("ja", "日本語", "README.ja.md"),
    ("ko", "한국어", "README.ko.md"),
    ("es", "Español", "README.es.md"),
]
CANONICAL = "README.md"
STUB = "README.zh-CN.md"
STUB_TARGET = "README.zh-Hans.md"

SWITCHER_RE = re.compile(r"^\[[^\]]+\]\(\./README[^)]*\)( · \[[^\]]+\]\(\./README[^)]*\))+$", re.M)
HEADING_RE = re.compile(r"^(#{1,6})\s")
BANNER_RE = re.compile(r"Last synced:\s*\S+")
BADGE_IMG_RE = re.compile(r'<img[^>]+src="(https://img\.shields\.io[^"]+)"[^>]*/?>')

# The release version, from the status table and the translation banner.
#
# Rule 6 compares badge *URLs*, which is why the version text in a prose table
# could drift: the badge URL did not carry a version, so five translated READMEs
# sat at v1.1.7 for twenty releases while every banner said v1.1.27, and the gate
# was green throughout. Both places the version appears are matched here, so a
# release that updates one and not the other fails instead of shipping.
#
# The status row is a Markdown table cell ending in `(GitHub + Gitee)`. The
# parentheses are matched as a character class rather than literally because the
# CJK translations use fullwidth ones - （GitHub + Gitee） - and a literal match
# would have reported those three files as having no release version at all,
# which is the failure mode this rule exists to end. The row label itself is
# deliberately not matched: it is translated ("Release", "发布", "發佈",
# "リリース", "릴리스", "Lanzamiento"), and the ASCII tail is not.
RELEASE_ROW_RE = re.compile(r"v(\d+\.\d+\.\d+)\s*[（(]GitHub \+ Gitee[）)]")
BANNER_VERSION_RE = re.compile(r"Last synced:\s*v?(\d+\.\d+\.\d+)")


def switcher_line() -> str:
    return " · ".join(f"[{label}](./{file})" for _, label, file in LANGUAGES)


def read(path: str) -> str:
    with open(os.path.join(ROOT, path), encoding="utf-8") as fh:
        return fh.read()


def heading_count(text: str) -> int:
    count = 0
    fence = False
    for line in text.splitlines():
        if line.lstrip().startswith("```"):
            fence = not fence
            continue
        if not fence and HEADING_RE.match(line):
            count += 1
    return count


def extract_badges(text: str) -> set[str]:
    """Return the set of shield.io badge URLs from a README."""
    return set(BADGE_IMG_RE.findall(text))


def main() -> int:
    failures = []
    expected_switcher = switcher_line()

    # 1 + 2 + 3
    base_headings = None
    all_badges: dict[str, set[str]] = {}  # filename -> badge URLs
    release_versions: dict[str, str] = {}  # filename -> version from the status row

    for locale, _label, filename in LANGUAGES:
        path = os.path.join(ROOT, filename)
        if not os.path.exists(path):
            failures.append(f"missing file: {filename}")
            continue
        text = read(filename)
        all_badges[filename] = extract_badges(text)

        if expected_switcher not in text:
            failures.append(f"{filename}: canonical language switcher not found")

        if locale != "en" and not BANNER_RE.search(text):
            failures.append(f"{filename}: missing translation banner (Last synced:)")
            banner_version = None
        else:
            banner_match = BANNER_VERSION_RE.search(text)
            banner_version = banner_match.group(1) if banner_match else None
            if locale != "en" and banner_version is None:
                failures.append(
                    f"{filename}: translation banner has no version; "
                    f"expected 'Last synced: vX.Y.Z' so a stale banner is visible"
                )

        # The status-table version, compared against the banner's. The banner is
        # the more reliable of the two because it was updated every release; the
        # row is what drifted, which is why this comparison is the check.
        row_match = RELEASE_ROW_RE.search(text)
        if row_match is None:
            failures.append(
                f"{filename}: no release version found in the status table; "
                f"expected a row ending 'vX.Y.Z (GitHub + Gitee)'"
            )
            continue
        release_versions[filename] = row_match.group(1)
        if banner_version is not None and banner_version != row_match.group(1):
            failures.append(
                f"{filename}: status table says v{row_match.group(1)} but the "
                f"translation banner says v{banner_version}; these must agree"
            )

        if locale == "en":
            base_headings = heading_count(text)
        elif base_headings is not None and heading_count(text) != base_headings:
            failures.append(
                f"{filename}: heading count {heading_count(text)} != English {base_headings}"
            )

    # 4
    if os.path.exists(os.path.join(ROOT, STUB)):
        stub = read(STUB)
        if STUB_TARGET not in stub:
            failures.append(f"{STUB}: does not redirect to {STUB_TARGET}")
    else:
        failures.append(f"missing redirect stub: {STUB}")

    # 6: badge URL consistency
    if all_badges:
        canonical = all_badges.get(CANONICAL, set())
        for filename, badges in all_badges.items():
            if filename == CANONICAL:
                continue
            diff = badges ^ canonical  # symmetric difference
            if diff:
                # Group by the label portion of the badge URL for readable output
                differing = sorted(diff)
                failures.append(
                    f"{filename}: badge URLs differ from {CANONICAL}: {differing}"
                )

    # 7: every README names the same release, so a translation cannot ship
    # claiming a version the canonical README does not.
    if release_versions:
        canonical_version = release_versions.get(CANONICAL)
        if canonical_version is not None:
            for filename, version in sorted(release_versions.items()):
                if filename == CANONICAL:
                    continue
                if version != canonical_version:
                    failures.append(
                        f"{filename}: release v{version} != {CANONICAL} v{canonical_version}"
                    )

    if failures:
        print("i18n check failed:", file=sys.stderr)
        for item in failures:
            print(f"  {item}", file=sys.stderr)
        return 1

    print(f"i18n OK: {len(LANGUAGES)} languages consistent")
    return 0


if __name__ == "__main__":
    sys.exit(main())
