#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
check_links.py — fail if a Markdown document references a missing local file.

Checks relative links of the form [text](path) and [text](path#anchor) in
Markdown files, ignoring http(s), mailto, and pure anchors. Exits non-zero if
any referenced local path does not exist.
"""

import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LINK = re.compile(r"\]\(([^)]+)\)")
SKIP_SCHEMES = ("http://", "https://", "mailto:", "#")


def markdown_files():
    for dirpath, dirnames, filenames in os.walk(ROOT):
        dirnames[:] = [d for d in dirnames if d not in (".git", "node_modules")]
        for name in filenames:
            if name.endswith(".md"):
                yield os.path.join(dirpath, name)


def main() -> int:
    failures = []
    for doc in markdown_files():
        with open(doc, encoding="utf-8") as fh:
            text = fh.read()
        for match in LINK.finditer(text):
            target = match.group(1).strip()
            if not target or target.startswith(SKIP_SCHEMES):
                continue
            path = target.split("#", 1)[0]
            if not path:
                continue
            resolved = os.path.normpath(os.path.join(os.path.dirname(doc), path))
            if not os.path.exists(resolved):
                rel = os.path.relpath(doc, ROOT)
                failures.append(f"{rel}: {target}")

    if failures:
        print("dangling local links:", file=sys.stderr)
        for item in sorted(set(failures)):
            print(f"  {item}", file=sys.stderr)
        return 1

    print("all local markdown links resolve")
    return 0


if __name__ == "__main__":
    sys.exit(main())
