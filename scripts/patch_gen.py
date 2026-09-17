#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
#
# patch_gen.py - post-generation fixups for client/client.gen.go.
#
# DEPRECATED: This script is a no-op. The root cause (bare `interface{}` in
# query parameter schemas) is now fixed at the spec level by patch_spec.py,
# which maps `type: null` to `type: object, additionalProperties: {}`, producing
# `map[string]any` in Go — a safe, nil-checkable type. A fresh generation from
# the patched spec requires no post-generation patching.
#
# Kept for backward compatibility; calling it has no effect.
#
# Usage:
#   python3 scripts/patch_gen.py <path-to-gen.go>

import sys


def main():
    if len(sys.argv) != 2:
        print("usage: patch_gen.py <gen.go>", file=sys.stderr)
        return 2
    print("patch_gen: no-op (root cause fixed in patch_spec.py)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
