#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""Check that every package under internal/ is actually imported.

The rule is deliberately unconditional: a package under ``internal/`` that nothing
imports is dead code shipped in the module, and shipping it is a decision that belongs
in a commit message rather than in an oversight. There are no heuristics here, which
is the point - a gate with judgement calls in it gets ignored the first time it is
wrong, and a previous attempt at a similar gate was withdrawn for exactly that reason
(see docs/runs/2026-09-27-money-gate-retracted).

A package imported only by tests is fine and counts as referenced: a test-only
importer is a legitimate consumer.

``ALLOWED`` is the escape hatch, and an entry has to say why. It should be used
rarely and the reason should be a decision worth recording - not a wish to be
rid of a cleanup task."""

import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Directories whose Go files are never compiled into the module's binaries and are
# not imported by import path.
SKIP_DIRS = {".git", "vendor", "testdata", "node_modules"}

# Unreferenced packages, with the reason each is tolerated. Keep this short: every
# entry is a package someone has decided to ship without an importer, and the reason
# is the record of that decision.
ALLOWED = {}

# An import spec on a single line, as `bare "path"`, `name "path"`, `. "path"` or
# `_ "path"`.
#
# Two alternatives rather than one optional alias group, and matched per line
# rather than with a multiline regex. Both details are load-bearing:
#
#   - `^\s*(?:[.\w]+\s+)?"([^"]+)"` looks equivalent and is wrong. Its optional alias
#     group greedily eats the path's own leading word characters, cannot satisfy the
#     trailing `\s+`, and then fails because a quote was expected where a letter is.
#     The blank line inside a grouped import block hides the difference from a naive
#     test, which is why this needed the real tree to surface.
#   - `^` with re.M and a greedy `\s*` that may span newlines silently misses an
#     import that follows a blank line, which is the shape a gofmt'd file has once a
#     stdlib group is separated from a third-party one.
IMPORT = re.compile(r'^\s*(?:import\s+)?(?:"([^"]+)"|[.\w]+\s+"([^"]+)")')


def module_path() -> str:
    """The module path from go.mod, so package paths can be compared exactly."""
    try:
        with open(os.path.join(ROOT, "go.mod"), "r", encoding="utf-8") as fh:
            for line in fh:
                if line.startswith("module "):
                    return line.split(None, 1)[1].strip()
    except OSError:
        pass
    return ""


def go_packages_under(directory: str) -> list:
    """Return repo-relative paths of directories under `directory` with Go files."""
    found = []
    base = os.path.join(ROOT, directory)
    if not os.path.isdir(base):
        return found
    for entry in sorted(os.listdir(base)):
        path = os.path.join(base, entry)
        if not os.path.isdir(path) or entry in SKIP_DIRS or entry.startswith("."):
            continue
        if any(f.endswith(".go") for f in os.listdir(path)):
            found.append("%s/%s" % (directory, entry))
    return found


def go_files() -> list:
    """Every .go file in the module, tests included: a test-only import is a real one."""
    out = []
    for dirpath, dirnames, filenames in os.walk(ROOT):
        dirnames[:] = [d for d in dirnames
                       if d not in SKIP_DIRS and not d.startswith(".")]
        for name in filenames:
            if name.endswith(".go"):
                out.append(os.path.join(dirpath, name))
    return out


def imported_paths_from_text(text: str) -> set:
    found = set()
    for line in text.splitlines():
        m = IMPORT.match(line)
        if m:
            found.add((m.group(1) or m.group(2)).rstrip("/"))
    return found


def imported_paths(files: list) -> set:
    """Every import path appearing in a .go file. Tests included: a test-only
    importer is a legitimate consumer of an internal package."""
    found = set()
    for path in files:
        try:
            with open(path, "r", encoding="utf-8", errors="replace") as fh:
                found |= imported_paths_from_text(fh.read())
        except OSError:
            continue
    return found


def find_unreferenced(packages: list, imports: set, allowed: dict) -> list:
    """Packages whose import path appears nowhere in the module, minus ALLOWED.

    Comparison is exact. ``internal/a`` and ``internal/a/deeper`` are different Go
    packages and only the former can be imported as the latter's prefix, so a
    prefix match here would quietly pass a package nobody imports.
    """
    out = []
    for pkg in packages:
        if pkg in allowed:
            continue
        if pkg not in imports:
            out.append(pkg)
    return out


def _self_test() -> bool:
    # Keyed the same way main() keys them - module-qualified - so an allowed entry
    # is actually matched by the case that exercises it.
    allowed = {"m/internal/fake": "test"}

    cases = [
        # Imported by exactly its path.
        (["m/internal/a"], {"m/internal/a"}, []),
        # Nothing imports it: dead code.
        (["m/internal/a"], set(), ["m/internal/a"]),
        # An import of a deeper package is not an import of this one.
        (["m/internal/a"], {"m/internal/a/deeper"}, ["m/internal/a"]),
        # An import of a sibling does not count.
        (["m/internal/a"], {"m/internal/b"}, ["m/internal/a"]),
        # A test-only importer is a real importer.
        (["m/internal/a"], {"m/internal/a"}, []),
        # Allowed entries are skipped whatever the imports say.
        (["m/internal/fake"], set(), []),
    ]
    if not all(find_unreferenced(pkgs, imps, allowed) == want
               for pkgs, imps, want in cases):
        return False

    # The import scanner must see every form Go uses.
    forms = [
        ('import "m/internal/a"', {"m/internal/a"}),
        ('import (\n\t"net/http"\n\t"m/internal/b"\n)', {"m/internal/b", "net/http"}),
        ('import alias "m/internal/c"', {"m/internal/c"}),
        ('\t_ "m/internal/d"', {"m/internal/d"}),
    ]
    return all(imported_paths_from_text(text) == want for text, want in forms)


def main() -> int:
    if not _self_test():
        print("internal-refs-check: internal self-test failed", file=sys.stderr)
        return 1

    mod = module_path()
    if not mod:
        print("internal-refs-check: could not read the module path from go.mod",
              file=sys.stderr)
        return 1

    rel = go_packages_under("internal")
    if not rel:
        print("internal-refs-check: no packages found under internal/ - refusing to pass",
              file=sys.stderr)
        return 1

    packages = ["%s/%s" % (mod, r) for r in rel]
    allowed = {"%s/%s" % (mod, p): why for p, why in ALLOWED.items()}

    imports = imported_paths(go_files())
    unreferenced = find_unreferenced(packages, imports, allowed)

    if unreferenced:
        print("internal-refs-check: %d package(s) under internal/ are imported by "
              "nothing:" % len(unreferenced), file=sys.stderr)
        for pkg in unreferenced:
            print("  %s" % pkg, file=sys.stderr)
        print("  A package under internal/ with no importer is dead code shipped in "
              "the module, and internal/ is not importable from outside it. Import it, "
              "delete it, or add it to ALLOWED in scripts/check_internal_refs.py with "
              "a reason.", file=sys.stderr)
        return 1

    tolerated = sorted(p for p in packages if p in allowed)
    print("internal-refs-check OK (%d packages referenced%s)"
          % (len(packages),
             "" if not tolerated
             else "; %d tolerated: %s" % (len(tolerated), ", ".join(tolerated))))
    return 0


if __name__ == "__main__":
    sys.exit(main())
