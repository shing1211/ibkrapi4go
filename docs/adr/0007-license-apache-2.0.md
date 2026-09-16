# 0007 — License under Apache-2.0 with DCO

- Status: Accepted
- Date: 2026-09-16

## Context

The project is an open-source SDK intended for broad commercial and personal use.
It must be permissively licensed with an explicit patent grant, and contributions
must be clearly licensed too. An earlier `LICENSE` was only a 17-line notice, not
the actual license text, so automated tooling could not detect Apache-2.0.

## Decision

- License the project under the **Apache License 2.0**, with the **full canonical
  text** in `LICENSE` (not a notice stub).
- Record copyright in `NOTICE` (`Copyright 2026 shing1211`).
- Apply `SPDX-License-Identifier: Apache-2.0` headers to all source files.
- Require **DCO sign-off** on every commit (`git commit -s`); no CLA.
- Add `DISCLAIMER.md` for trademark non-affiliation and financial risk.

## Consequences

- Apache-2.0 is detected by GitHub/Gitee/SPDX tooling.
- Contributors retain copyright while licensing contributions under Apache-2.0.
- Maintainers must enforce sign-off; the DCO bot/check should be enabled.
- Apache-2.0's `NOTICE` obligation is satisfied for redistributors.
